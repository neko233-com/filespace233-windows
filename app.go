package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

//go:embed resources/icon.png
var iconPNG []byte

var logoBitmap, _ = ui.DecodeBitmap(iconPNG)

type favorite struct {
	Label string
	Path  string
	Kind  string
}

type persisted struct {
	Dir        string `json:"dir"`
	SortKey    string `json:"sortKey"`
	SortAsc    bool   `json:"sortAsc"`
	ShowHidden bool   `json:"showHidden"`
}

// App is the window's state. It lives on the UI thread.
type App struct {
	win  *mygo.Window
	tray *mygo.Tray

	dir       string
	entries   []fsx.Entry
	visible   []fsx.Entry
	stats     fsx.Stats
	drives    []fsx.Drive
	favorites []favorite

	query      string
	sortKey    string
	sortAsc    bool
	showHidden bool

	loading  bool
	err      string
	selected int
	list     ui.ListState

	history  []string
	forward  []string
	gen      int
	quitting bool
}

func NewApp() *App {
	a := &App{sortKey: "name", sortAsc: true, selected: -1}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "C:\\"
	}
	a.dir = home
	a.entries, a.stats, _ = fsx.List(home, false)
	a.drives = fsx.Drives()
	return a
}

func (a *App) start(capturePath string) {
	a.loadState()
	if a.dir != "" {
		if entries, st, err := fsx.List(a.dir, a.showHidden); err == nil {
			a.entries, a.stats = entries, st
		} else if home, e := os.UserHomeDir(); e == nil {
			a.dir = home
			a.entries, a.stats, _ = fsx.List(home, a.showHidden)
		}
	}
	a.loadFavorites()

	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     "FileSpace233",
		Width:     1180,
		Height:    760,
		MinWidth:  880,
		MinHeight: 540,
		StateKey:  "main",
		Content:   ui.View(a.view),
	})
	_ = a.win.SetIcon(iconPNG)
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if a.quitting {
			return
		}
		e.PreventDefault()
		a.win.Hide()
	})

	a.setupTray()

	if capturePath != "" {
		go func() {
			time.Sleep(1200 * time.Millisecond)
			png, err := a.win.CapturePage()
			if err == nil {
				err = os.WriteFile(capturePath, png, 0o644)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "capture:", err)
			}
			a.win.Update(func() { a.quit() })
		}()
	}
}

func (a *App) quit() {
	a.quitting = true
	mygo.App.Quit()
}

func (a *App) loadFavorites() {
	add := func(label string, name mygo.PathName, kind string) {
		p, err := mygo.App.Path(name)
		if err == nil && p != "" {
			a.favorites = append(a.favorites, favorite{Label: label, Path: p, Kind: kind})
		}
	}
	add("主目录", mygo.PathHome, "home")
	add("桌面", mygo.PathDesktop, "desktop")
	add("文档", mygo.PathDocuments, "docs")
	add("下载", mygo.PathDownloads, "downloads")
	add("图片", mygo.PathPictures, "pictures")
	add("音乐", mygo.PathMusic, "music")
	add("视频", mygo.PathVideos, "videos")
}

func (a *App) setupTray() {
	res, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return
	}
	trayPNG, err := os.ReadFile(filepath.Join(res, "tray.png"))
	if err != nil {
		trayPNG = iconPNG
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{ID: "show", Label: "打开 FileSpace233", Click: func(*mygo.MenuItem, *mygo.Window) { a.showWindow() }},
		{ID: "sep", Type: mygo.MenuItemSeparator},
		{ID: "quit", Label: "退出", Click: func(*mygo.MenuItem, *mygo.Window) { a.quit() }},
	})
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "main",
		Icon:    trayPNG,
		ToolTip: "FileSpace233 — 文件夹管理器（后台运行中）",
		Menu:    menu,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "tray:", err)
		return
	}
	tray.OnClick(func() { a.showWindow() })
	a.tray = tray
}

func (a *App) showWindow() {
	if a.win == nil {
		return
	}
	a.win.Restore()
	a.win.Show()
	a.win.Focus()
}

func stateFile() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "state.json")
}

func (a *App) loadState() {
	path := stateFile()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var p persisted
	if json.Unmarshal(data, &p) != nil {
		return
	}
	if p.Dir != "" {
		if _, err := os.Stat(p.Dir); err == nil {
			a.dir = p.Dir
		}
	}
	if p.SortKey != "" {
		a.sortKey = p.SortKey
	}
	a.sortAsc = p.SortAsc
	a.showHidden = p.ShowHidden
}

func (a *App) save() {
	path := stateFile()
	if path == "" {
		return
	}
	p := persisted{Dir: a.dir, SortKey: a.sortKey, SortAsc: a.sortAsc, ShowHidden: a.showHidden}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func (a *App) navigate(dir string, record bool) {
	abs, err := fsx.NormalizeDir(dir)
	if err != nil {
		a.err = err.Error()
		return
	}
	if abs == a.dir && !a.loading {
		return
	}
	a.gen++
	gen := a.gen
	old := a.dir
	showHidden := a.showHidden
	a.loading = true
	a.err = ""
	go func() {
		entries, st, err := fsx.List(abs, showHidden)
		a.win.Update(func() {
			if gen != a.gen {
				return
			}
			a.loading = false
			if err != nil {
				a.err = err.Error()
				return
			}
			if record && old != "" && old != abs {
				a.history = append(a.history, old)
				a.forward = nil
			}
			a.dir = abs
			a.entries = entries
			a.stats = st
			a.selected = -1
			a.list = ui.ListState{}
			a.query = ""
			a.save()
		})
	}()
}

func (a *App) refresh() {
	dir := a.dir
	if dir == "" {
		return
	}
	a.gen++
	gen := a.gen
	showHidden := a.showHidden
	a.loading = true
	a.err = ""
	go func() {
		entries, st, err := fsx.List(dir, showHidden)
		a.win.Update(func() {
			if gen != a.gen {
				return
			}
			a.loading = false
			if err != nil {
				a.err = err.Error()
				return
			}
			a.entries = entries
			a.stats = st
			a.selected = -1
			a.list = ui.ListState{}
		})
	}()
}

func (a *App) back() {
	if len(a.history) == 0 {
		return
	}
	target := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.forward = append(a.forward, a.dir)
	a.navigate(target, false)
}

func (a *App) forwardDir() {
	if len(a.forward) == 0 {
		return
	}
	target := a.forward[len(a.forward)-1]
	a.forward = a.forward[:len(a.forward)-1]
	a.history = append(a.history, a.dir)
	a.navigate(target, false)
}

func (a *App) up() {
	if a.dir == "" {
		return
	}
	a.navigate(fsx.Parent(a.dir), true)
}

func (a *App) open(path string) {
	for _, e := range a.entries {
		if e.Path == path {
			if e.IsDir {
				a.navigate(e.Path, true)
			} else if err := mygo.Shell.OpenPath(e.Path); err != nil {
				a.err = err.Error()
			}
			return
		}
	}
}

func (a *App) current() (fsx.Entry, bool) {
	if a.selected < 0 || a.selected >= len(a.visible) {
		return fsx.Entry{}, false
	}
	return a.visible[a.selected], true
}

func (a *App) openSelected() {
	if e, ok := a.current(); ok {
		a.open(e.Path)
	}
}

func (a *App) copyPath(path string) {
	if err := mygo.Clipboard.Write(transfer.TextData(path)); err != nil {
		a.err = err.Error()
	}
}

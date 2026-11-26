package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

//go:embed resources/icon.png
var iconPNG []byte

var logoBitmap, _ = ui.DecodeBitmap(iconPNG)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx   = user32.NewProc("CallNextHookEx")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")

	hotkeyHook      uintptr
	hotkeyCallback  uintptr
	hotkeyTargetApp *App
)

type keyboardHookData struct {
	VkCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

func hookData(p uintptr) *keyboardHookData {
	return (*keyboardHookData)(*(*unsafe.Pointer)(unsafe.Pointer(&p)))
}

func keyDown(vk uintptr) bool {
	ret, _, _ := procGetAsyncKeyState.Call(vk)
	return int32(uint32(ret))&0x8000 != 0
}

// keyboardHookProc intercepts Ctrl+Win+E when another app already owns the
// RegisterHotKey slot for it. MyGo itself registers first; this is the fallback.
func keyboardHookProc(nCode, wParam, lParam uintptr) uintptr {
	if nCode == 0 && (wParam == 0x0100 || wParam == 0x0104) {
		k := hookData(lParam)
		if k.VkCode == 0x45 && keyDown(0x11) && (keyDown(0x5B) || keyDown(0x5C)) {
			if hotkeyTargetApp != nil {
				hotkeyTargetApp.showWindow()
			}
			return 1
		}
	}
	ret, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return ret
}

func (a *App) installHotkeyHook() bool {
	hotkeyTargetApp = a
	if hotkeyCallback == 0 {
		hotkeyCallback = syscall.NewCallback(keyboardHookProc)
	}
	h, _, _ := procSetWindowsHookEx.Call(13, hotkeyCallback, 0, 0) // WH_KEYBOARD_LL
	if h == 0 {
		return false
	}
	hotkeyHook = h
	return true
}

type favorite struct {
	Label string
	Path  string
	Kind  string
}

// Pane is one independent folder view.
type Pane struct {
	dir      string
	entries  []fsx.Entry
	visible  []fsx.Entry
	stats    fsx.Stats
	query    string
	sortKey  string
	sortAsc  bool
	hidden   bool
	loading  bool
	err      string
	selected int
	list     ui.ListState
	history  []string
	forward  []string
	gen      int
}

func newPane(dir string) *Pane {
	p := &Pane{dir: dir, sortKey: "name", sortAsc: true, selected: -1}
	p.entries, p.stats, _ = fsx.List(dir, false)
	return p
}

func (p *Pane) current() (fsx.Entry, bool) {
	if p.selected < 0 || p.selected >= len(p.visible) {
		return fsx.Entry{}, false
	}
	return p.visible[p.selected], true
}

func (p *Pane) filter() []fsx.Entry {
	q := strings.ToLower(strings.TrimSpace(p.query))
	out := make([]fsx.Entry, 0, len(p.entries))
	for _, e := range p.entries {
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	fsx.Sort(out, p.sortKey, p.sortAsc)
	return out
}

type panePersist struct {
	Dir        string `json:"dir"`
	SortKey    string `json:"sortKey"`
	SortAsc    bool   `json:"sortAsc"`
	ShowHidden bool   `json:"showHidden"`
}

type persisted struct {
	Mode   int           `json:"mode"`
	Split  float64       `json:"split"`
	Active int           `json:"active"`
	Panes  []panePersist `json:"panes"`
}

// App is the window's state. It lives on the UI thread.
type App struct {
	win  *mygo.Window
	tray *mygo.Tray

	panes     []*Pane
	active    int
	mode      int // 2 columns or 2x2 grid
	split     float64
	splitPct  float32
	rowPct    float32
	inspector bool
	query     string

	drives      []fsx.Drive
	favorites   []favorite
	hotkeyErr   string
	hotkeyLabel string
	quitting    bool
}

func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "C:\\"
	}
	a := &App{mode: 2, split: 0.5, splitPct: 50, rowPct: 50, active: 0, hotkeyLabel: "Ctrl+Win+E"}
	for i := 0; i < 4; i++ {
		a.panes = append(a.panes, newPane(home))
	}
	a.drives = fsx.Drives()
	return a
}

func (a *App) currentPane() *Pane {
	return a.panes[a.active]
}

func (a *App) paneCount() int {
	if a.mode == 4 {
		return 4
	}
	return 2
}

func (a *App) setActive(i int) {
	if i < 0 || i >= a.paneCount() || i == a.active {
		return
	}
	a.active = i
	a.query = a.panes[i].query
}

func (a *App) start(capturePath string) {
	a.loadState()
	for _, p := range a.panes {
		if p.dir == "" {
			continue
		}
		if entries, st, err := fsx.List(p.dir, p.hidden); err == nil {
			p.entries, p.stats = entries, st
		}
	}
	a.drives = fsx.Drives()
	a.loadFavorites()
	if capturePath != "" {
		a.panes[0].selected = 2
		a.panes[1].selected = -1
	}
	a.query = a.currentPane().query

	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     "FileSpace233",
		Width:     1380,
		Height:    860,
		MinWidth:  920,
		MinHeight: 560,
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
	a.registerHotkey()

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

func (a *App) registerHotkey() {
	err := mygo.GlobalShortcut.RegisterWith("Ctrl+Super+E", mygo.GlobalShortcutOptions{
		Description: "打开 FileSpace233",
	}, a.showWindow)
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "hotkey Ctrl+Win+E fallback:", err)
	if a.installHotkeyHook() {
		fmt.Fprintln(os.Stderr, "hotkey Ctrl+Win+E: using keyboard hook")
		return
	}
	err2 := mygo.GlobalShortcut.RegisterWith("Ctrl+Alt+E", mygo.GlobalShortcutOptions{
		Description: "打开 FileSpace233",
	}, a.showWindow)
	if err2 == nil {
		a.hotkeyLabel = "Ctrl+Alt+E"
		return
	}
	a.hotkeyErr = err.Error()
	fmt.Fprintln(os.Stderr, "hotkey Ctrl+Win+E:", err, "fallback:", err2)
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
	var s persisted
	if json.Unmarshal(data, &s) != nil {
		return
	}
	if s.Mode == 2 || s.Mode == 4 {
		a.mode = s.Mode
	}
	if s.Split >= 0.25 && s.Split <= 0.75 {
		a.split = s.Split
		a.splitPct = float32(s.Split * 100)
	}
	if s.Active >= 0 && s.Active < a.paneCount() {
		a.active = s.Active
	}
	for i, ps := range s.Panes {
		if i >= len(a.panes) {
			break
		}
		p := a.panes[i]
		if ps.Dir != "" {
			if _, err := os.Stat(ps.Dir); err == nil {
				p.dir = ps.Dir
			}
		}
		if ps.SortKey != "" {
			p.sortKey = ps.SortKey
		}
		p.sortAsc = ps.SortAsc
		p.hidden = ps.ShowHidden
	}
}

func (a *App) save() {
	path := stateFile()
	if path == "" {
		return
	}
	s := persisted{Mode: a.mode, Split: a.split, Active: a.active}
	for _, p := range a.panes {
		s.Panes = append(s.Panes, panePersist{
			Dir: p.dir, SortKey: p.sortKey, SortAsc: p.sortAsc, ShowHidden: p.hidden,
		})
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func (a *App) navigate(p *Pane, dir string, record bool) {
	abs, err := fsx.NormalizeDir(dir)
	if err != nil {
		p.err = err.Error()
		return
	}
	if abs == p.dir && !p.loading {
		return
	}
	p.gen++
	gen := p.gen
	old := p.dir
	hidden := p.hidden
	p.loading = true
	p.err = ""
	go func() {
		entries, st, err := fsx.List(abs, hidden)
		a.win.Update(func() {
			if gen != p.gen {
				return
			}
			p.loading = false
			if err != nil {
				p.err = err.Error()
				return
			}
			if record && old != "" && old != abs {
				p.history = append(p.history, old)
				p.forward = nil
			}
			p.dir = abs
			p.entries = entries
			p.stats = st
			p.selected = -1
			p.list = ui.ListState{}
			p.query = ""
			if a.currentPane() == p {
				a.query = ""
			}
			a.save()
		})
	}()
}

func (a *App) refresh(p *Pane) {
	dir := p.dir
	if dir == "" {
		return
	}
	p.gen++
	gen := p.gen
	hidden := p.hidden
	p.loading = true
	p.err = ""
	go func() {
		entries, st, err := fsx.List(dir, hidden)
		a.win.Update(func() {
			if gen != p.gen {
				return
			}
			p.loading = false
			if err != nil {
				p.err = err.Error()
				return
			}
			p.entries = entries
			p.stats = st
			p.selected = -1
			p.list = ui.ListState{}
		})
	}()
}

func (a *App) back(p *Pane) {
	if len(p.history) == 0 {
		return
	}
	target := p.history[len(p.history)-1]
	p.history = p.history[:len(p.history)-1]
	p.forward = append(p.forward, p.dir)
	a.navigate(p, target, false)
}

func (a *App) forwardDir(p *Pane) {
	if len(p.forward) == 0 {
		return
	}
	target := p.forward[len(p.forward)-1]
	p.forward = p.forward[:len(p.forward)-1]
	p.history = append(p.history, p.dir)
	a.navigate(p, target, false)
}

func (a *App) up(p *Pane) {
	if p.dir == "" {
		return
	}
	a.navigate(p, fsx.Parent(p.dir), true)
}

func (a *App) open(p *Pane, path string) {
	for _, e := range p.entries {
		if e.Path == path {
			if e.IsDir {
				a.navigate(p, e.Path, true)
			} else if err := mygo.Shell.OpenPath(e.Path); err != nil {
				p.err = err.Error()
			}
			return
		}
	}
}

func (a *App) openSelected(p *Pane) {
	if e, ok := p.current(); ok {
		a.open(p, e.Path)
	}
}

func (a *App) copyPath(path string) {
	if err := mygo.Clipboard.Write(transfer.TextData(path)); err != nil {
		if p := a.currentPane(); p != nil {
			p.err = err.Error()
		}
	}
}

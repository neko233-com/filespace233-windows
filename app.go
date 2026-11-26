package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
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

	selection  ui.Selection[string]
	dirty      bool
	reveal     string
	scrollTo   string
	lastAction time.Time
}

func newPane(dir string) *Pane {
	return &Pane{dir: dir, sortKey: "name", sortAsc: true, selected: -1, dirty: true}
}

func (p *Pane) invalidate() {
	p.dirty = true
}

func (p *Pane) refreshVisible() {
	if !p.dirty && p.visible != nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(p.query))
	out := make([]fsx.Entry, 0, len(p.entries))
	for _, e := range p.entries {
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	fsx.Sort(out, p.sortKey, p.sortAsc)
	p.visible = out
	p.dirty = false
}

func (p *Pane) current() (fsx.Entry, bool) {
	if p.selected < 0 || p.selected >= len(p.visible) {
		return fsx.Entry{}, false
	}
	return p.visible[p.selected], true
}

func (p *Pane) resetView(clearSelection bool) {
	p.selected = -1
	p.list = ui.ListState{}
	if clearSelection {
		p.selection.Clear()
	}
	p.invalidate()
}

// chosenEntries returns the rows a file operation acts on: the multi-selection
// in visible order, or the row under the cursor.
func (p *Pane) chosenEntries() []fsx.Entry {
	p.refreshVisible()
	if p.selection.Len() > 0 {
		out := make([]fsx.Entry, 0, p.selection.Len())
		for _, e := range p.visible {
			if p.selection.Has(e.Path) {
				out = append(out, e)
			}
		}
		return out
	}
	if e, ok := p.current(); ok {
		return []fsx.Entry{e}
	}
	return nil
}

func (p *Pane) chosenPaths() []string {
	entries := p.chosenEntries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Path)
	}
	return out
}

func (p *Pane) chosenCount() int { return len(p.chosenEntries()) }

func (p *Pane) hasVisible(path string) bool {
	p.refreshVisible()
	for _, e := range p.visible {
		if samePath(e.Path, path) {
			return true
		}
	}
	return false
}

func (p *Pane) pruneSelection() {
	if p.selection.Len() == 0 {
		return
	}
	p.refreshVisible()
	var gone []string
	for path := range p.selection.All() {
		if !p.hasVisible(path) {
			gone = append(gone, path)
		}
	}
	for _, path := range gone {
		p.selection.Remove(path)
	}
}

func (p *Pane) selectionSnapshot() map[string]bool {
	if p.selection.Len() == 0 {
		return nil
	}
	out := make(map[string]bool, p.selection.Len())
	for path := range p.selection.All() {
		out[path] = true
	}
	return out
}

type panePersist struct {
	Dir        string `json:"dir"`
	SortKey    string `json:"sortKey"`
	SortAsc    bool   `json:"sortAsc"`
	ShowHidden bool   `json:"showHidden"`
}

type persisted struct {
	Mode     int           `json:"mode"`
	Split    float64       `json:"split"`
	RowSplit float64       `json:"rowSplit"`
	Active   int           `json:"active"`
	Panes    []panePersist `json:"panes"`
	Recent   []string      `json:"recent"`
}

type cachedDir struct {
	entries []fsx.Entry
	stats   fsx.Stats
	at      time.Time
}

// App is the window's state. It lives on the UI thread.
type App struct {
	win  *mygo.Window
	tray *mygo.Tray

	panes     []*Pane
	active    int
	focusPane int
	mode      int // 2 columns or 2x2 grid
	split     float64
	splitPct  float32
	rowPct    float32
	rowSlider float64

	inspector bool
	query     string

	drives      []fsx.Drive
	favorites   []favorite
	recent      []string
	hotkeyErr   string
	hotkeyLabel string
	quitting    bool

	cache      map[string]cachedDir
	cacheOrder []string
	cacheCount int

	clipMode  string
	clipPaths []string

	opBusy    bool
	opLabel   string
	opCurrent string
	opDone    int
	opTotal   int
	opFailed  int
	opErr     string

	statusMsg string
	statusErr bool

	savePending bool

	pathEdit  int
	pathDraft string

	promptOpen    bool
	promptTitle   string
	promptValue   string
	promptHint    string
	promptConfirm string
	promptErr     string
	promptAction  func(string) (string, error)

	batchOpen    bool
	batchEntries []fsx.Entry
	batchPattern string
	batchErr     string

	confirmOpen   bool
	confirmTitle  string
	confirmBody   string
	confirmAction func()

	inspectKey    string
	previewPath   string
	previewBitmap *ui.Bitmap
	previewText   string
	previewDim    string
	previewErr    string
	previewLoad   bool

	hashPath  string
	hashValue string
	hashErr   string
	hashBusy  bool
	hashAlgo  string
}

func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "C:\\"
	}
	a := &App{
		mode: 2, split: 0.5, splitPct: 50, rowPct: 50, rowSlider: 0.5, active: 0, focusPane: 0,
		hotkeyLabel: "Ctrl+Win+E", pathEdit: -1, hashAlgo: "SHA-256",
		cache: map[string]cachedDir{},
	}
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
	if i < 0 || i >= a.paneCount() {
		return
	}
	if i == a.active {
		a.focusPane = i
		return
	}
	a.active = i
	a.focusPane = i
	a.query = a.panes[i].query
}

func (a *App) otherPaneIndexes() []int {
	var out []int
	for i := 0; i < a.paneCount(); i++ {
		if i != a.active {
			out = append(out, i)
		}
	}
	return out
}

func (a *App) start(capturePath string, captureQuad, captureInspector bool) {
	a.loadState()
	a.loadInitialPanes()
	a.drives = fsx.Drives()
	a.loadFavorites()
	if captureQuad {
		a.mode = 4
		a.splitPct = 50
		a.rowPct = 50
		a.active = 0
		a.query = a.panes[0].query
	}
	if captureInspector {
		a.inspector = true
		a.active = 0
		a.query = a.panes[0].query
	}
	if capturePath != "" {
		a.panes[0].selected = 2
		a.panes[0].selection.Clear()
		a.panes[1].selected = -1
		a.panes[1].selection.Clear()
	}
	a.query = a.currentPane().query

	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title:     "FileSpace233",
		Width:     1440,
		Height:    900,
		MinWidth:  960,
		MinHeight: 600,
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
	a.registerClipboardWatch()

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

func (a *App) loadInitialPanes() {
	loaded := map[string]cachedDir{}
	for _, p := range a.panes {
		if p.dir == "" {
			continue
		}
		key := listingKey(p.dir, p.hidden)
		if c, ok := loaded[key]; ok {
			p.entries, p.stats = c.entries, c.stats
			p.invalidate()
			continue
		}
		if c, ok := a.cacheGet(p.dir, p.hidden); ok {
			p.entries, p.stats = c.entries, c.stats
			p.invalidate()
			continue
		}
		entries, stats, err := fsx.List(p.dir, p.hidden)
		if err != nil {
			p.err = err.Error()
			continue
		}
		c := cachedDir{entries: entries, stats: stats, at: time.Now()}
		loaded[key] = c
		a.cachePut(p.dir, p.hidden, entries, stats)
		p.entries, p.stats = entries, stats
		p.invalidate()
	}
}

func (a *App) quit() {
	a.quitting = true
	a.save()
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
	if s.RowSplit >= 0.25 && s.RowSplit <= 0.75 {
		a.rowPct = float32(s.RowSplit * 100)
		a.rowSlider = s.RowSplit
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
	for _, dir := range s.Recent {
		if len(a.recent) >= 8 {
			break
		}
		if _, err := os.Stat(dir); err == nil && !a.inRecent(dir) {
			a.recent = append(a.recent, dir)
		}
	}
}

func (a *App) save() {
	path := stateFile()
	if path == "" {
		return
	}
	s := persisted{
		Mode: a.mode, Split: a.split, RowSplit: float64(a.rowPct) / 100,
		Active: a.active, Recent: a.recent,
	}
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

// saveSoon coalesces the dozens of state writes produced while a splitter or
// slider is being dragged into one write.
func (a *App) saveSoon() {
	if a.savePending {
		return
	}
	a.savePending = true
	time.AfterFunc(180*time.Millisecond, func() {
		if a.win != nil {
			a.win.Update(func() {
				a.savePending = false
				a.save()
			})
		}
	})
}

func (a *App) inRecent(dir string) bool {
	for _, r := range a.recent {
		if samePath(r, dir) {
			return true
		}
	}
	return false
}

func (a *App) pushRecent(dir string) {
	if dir == "" {
		return
	}
	var kept []string
	for _, r := range a.recent {
		if !samePath(r, dir) {
			kept = append(kept, r)
		}
	}
	a.recent = append([]string{dir}, kept...)
	if len(a.recent) > 8 {
		a.recent = a.recent[:8]
	}
}

func listingKey(dir string, hidden bool) string {
	return strings.ToLower(filepath.Clean(dir)) + "|" + fmt.Sprintf("%t", hidden)
}

func (a *App) cacheGet(dir string, hidden bool) (cachedDir, bool) {
	c, ok := a.cache[listingKey(dir, hidden)]
	return c, ok
}

func (a *App) cachePut(dir string, hidden bool, entries []fsx.Entry, stats fsx.Stats) {
	key := listingKey(dir, hidden)
	if old, ok := a.cache[key]; ok {
		a.cacheCount -= len(old.entries)
		for i, k := range a.cacheOrder {
			if k == key {
				a.cacheOrder = append(a.cacheOrder[:i], a.cacheOrder[i+1:]...)
				break
			}
		}
	}
	a.cache[key] = cachedDir{entries: entries, stats: stats, at: time.Now()}
	a.cacheOrder = append(a.cacheOrder, key)
	a.cacheCount += len(entries)
	for len(a.cacheOrder) > 64 || a.cacheCount > 200000 {
		oldest := a.cacheOrder[0]
		a.cacheOrder = a.cacheOrder[1:]
		if old, ok := a.cache[oldest]; ok {
			a.cacheCount -= len(old.entries)
			delete(a.cache, oldest)
		}
	}
}

func (a *App) cacheDrop(key string) {
	if old, ok := a.cache[key]; ok {
		a.cacheCount -= len(old.entries)
		delete(a.cache, key)
	}
}

func (a *App) invalidateCache(dir string) {
	prefix := strings.ToLower(filepath.Clean(dir)) + "|"
	var kept []string
	for _, key := range a.cacheOrder {
		if strings.HasPrefix(key, prefix) {
			a.cacheDrop(key)
			continue
		}
		kept = append(kept, key)
	}
	a.cacheOrder = kept
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
	if c, ok := a.cacheGet(abs, hidden); ok {
		p.loading = false
		p.err = ""
		p.dir = abs
		p.entries = c.entries
		p.stats = c.stats
		p.invalidate()
		p.resetView(true)
		p.query = ""
		if record && old != "" && old != abs {
			p.history = append(p.history, old)
			p.forward = nil
		}
		if p == a.currentPane() {
			a.query = ""
		}
		a.pushRecent(abs)
		a.save()
		if time.Since(c.at) > 10*time.Second {
			a.refreshQuiet(p, abs, "")
		}
		_ = gen
		return
	}
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
			p.invalidate()
			p.resetView(true)
			p.query = ""
			if a.currentPane() == p {
				a.query = ""
			}
			a.cachePut(abs, hidden, entries, st)
			a.pushRecent(abs)
			a.save()
		})
	}()
}

func (a *App) refreshQuiet(p *Pane, dir, keepPath string) {
	p.gen++
	gen := p.gen
	hidden := p.hidden
	go func() {
		entries, st, err := fsx.List(dir, hidden)
		if err != nil {
			return
		}
		a.win.Update(func() {
			if gen != p.gen || !samePath(p.dir, dir) {
				return
			}
			p.entries, p.stats = entries, st
			p.invalidate()
			a.cachePut(dir, hidden, entries, st)
			if keepPath != "" {
				p.refreshVisible()
				p.selected = -1
				for i, e := range p.visible {
					if samePath(e.Path, keepPath) {
						p.selected = i
						break
					}
				}
			}
			p.pruneSelection()
		})
	}()
}

func (a *App) refresh(p *Pane) {
	dir := p.dir
	if dir == "" {
		return
	}
	keepPath := ""
	if e, ok := p.current(); ok {
		keepPath = e.Path
	}
	keepSel := p.selectionSnapshot()
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
			p.invalidate()
			p.resetView(true)
			a.cachePut(dir, hidden, entries, st)
			reveal := p.reveal != ""
			if reveal {
				keepPath = p.reveal
				p.reveal = ""
				keepSel = nil
			}
			p.refreshVisible()
			if keepPath != "" || len(keepSel) > 0 {
				for i, e := range p.visible {
					if keepSel != nil && keepSel[e.Path] {
						p.selection.Add(e.Path)
						if p.selected < 0 {
							p.selected = i
						}
					}
					if reveal && samePath(e.Path, keepPath) {
						p.selection.Add(e.Path)
					}
					if samePath(e.Path, keepPath) {
						p.selected = i
						p.scrollTo = e.Path
					}
				}
			}
			p.pruneSelection()
		})
	}()
}

func (a *App) refreshGroup(dirs []string, revealPath string) {
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(dir))
		if seen[key] {
			continue
		}
		seen[key] = true
		var group []*Pane
		for _, p := range a.panes {
			if samePath(p.dir, dir) {
				group = append(group, p)
			}
		}
		if len(group) == 0 {
			continue
		}
		for _, p := range group {
			if revealPath != "" && samePath(filepath.Dir(revealPath), p.dir) {
				p.reveal = revealPath
			}
		}
		a.refreshPaneGroup(dir, group)
	}
}

type refreshSnap struct {
	p        *Pane
	gen      int
	keepPath string
	keepSel  map[string]bool
}

func (a *App) refreshPaneGroup(dir string, group []*Pane) {
	byHidden := map[bool][]refreshSnap{}
	for _, p := range group {
		keepPath := ""
		if e, ok := p.current(); ok {
			keepPath = e.Path
		}
		snap := refreshSnap{p: p, keepPath: keepPath, keepSel: p.selectionSnapshot()}
		p.gen++
		snap.gen = p.gen
		p.loading = true
		p.err = ""
		byHidden[p.hidden] = append(byHidden[p.hidden], snap)
	}
	for hidden, snaps := range byHidden {
		hidden := hidden
		snaps := snaps
		go func() {
			entries, st, err := fsx.List(dir, hidden)
			a.win.Update(func() {
				if err == nil {
					a.cachePut(dir, hidden, entries, st)
				}
				for _, snap := range snaps {
					p := snap.p
					if snap.gen != p.gen || !samePath(p.dir, dir) {
						continue
					}
					p.loading = false
					if err != nil {
						p.err = err.Error()
						continue
					}
					p.entries = entries
					p.stats = st
					p.invalidate()
					p.resetView(true)
					reveal, keepPath, keepSel := false, snap.keepPath, snap.keepSel
					if p.reveal != "" {
						reveal = true
						keepPath = p.reveal
						p.reveal = ""
						keepSel = nil
					}
					p.refreshVisible()
					if keepPath != "" || len(keepSel) > 0 {
						for i, e := range p.visible {
							if keepSel != nil && keepSel[e.Path] {
								p.selection.Add(e.Path)
								if p.selected < 0 {
									p.selected = i
								}
							}
							if reveal && samePath(e.Path, keepPath) {
								p.selection.Add(e.Path)
							}
							if samePath(e.Path, keepPath) {
								p.selected = i
								p.scrollTo = e.Path
							}
						}
					}
					p.pruneSelection()
				}
			})
		}()
	}
}

func (a *App) refreshDirs(dirs ...string) {
	a.refreshGroup(dirs, "")
}

func (a *App) refreshAndReveal(path string) {
	dir := filepath.Dir(path)
	a.invalidateCache(dir)
	a.refreshGroup([]string{dir}, path)
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
		if samePath(e.Path, path) {
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

func (a *App) copyPaths(paths []string) {
	if len(paths) == 0 {
		return
	}
	if err := mygo.Clipboard.Write(transfer.TextData(strings.Join(paths, "\r\n"))); err != nil {
		a.statusMsg = "复制路径失败：" + err.Error()
		a.statusErr = true
	}
}

func (a *App) setQuery(p *Pane, q string) {
	if p.query == q {
		return
	}
	p.query = q
	p.invalidate()
	p.selected = -1
	p.selection.Clear()
	p.list = ui.ListState{}
}

func (a *App) resort(p *Pane) {
	keep := ""
	if e, ok := p.current(); ok {
		keep = e.Path
	}
	p.invalidate()
	p.refreshVisible()
	p.selected = -1
	for i, e := range p.visible {
		if samePath(e.Path, keep) {
			p.selected = i
			break
		}
	}
}

func (a *App) loadClipboardFiles() ([]string, bool) {
	data, err := mygo.Clipboard.Read()
	if err != nil {
		return nil, false
	}
	files, err := data.Files()
	if err != nil || len(files) == 0 {
		return nil, false
	}
	a.clipMode = "copy"
	a.clipPaths = append([]string(nil), files...)
	return files, true
}

func (a *App) reconcileClipboard() {
	if len(a.clipPaths) == 0 {
		return
	}
	files, err := mygo.Clipboard.ReadFiles()
	if err != nil || !samePathSet(a.clipPaths, files) {
		a.clipMode = ""
		a.clipPaths = nil
	}
}

func samePathSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]int, len(a))
	for _, p := range a {
		set[strings.ToLower(filepath.Clean(p))]++
	}
	for _, p := range b {
		key := strings.ToLower(filepath.Clean(p))
		if set[key] == 0 {
			return false
		}
		set[key]--
	}
	return true
}

func (a *App) registerClipboardWatch() {
	mygo.Clipboard.OnChanged(func() {
		a.win.Update(func() { a.reconcileClipboard() })
	})
}

func (a *App) startHash(path string) {
	a.startHashAlgo(a.hashAlgo, path)
}

func hashNew(algo string) hash.Hash {
	switch algo {
	case "MD5":
		return md5.New()
	case "SHA-1":
		return sha1.New()
	case "SHA-512":
		return sha512.New()
	default:
		return sha256.New()
	}
}

func (a *App) startHashAlgo(algo, path string) {
	if path == "" || a.hashBusy {
		return
	}
	if algo == "" {
		algo = "SHA-256"
	}
	a.hashAlgo = algo
	a.hashPath = path
	a.hashValue = ""
	a.hashErr = ""
	a.hashBusy = true
	go func() {
		f, err := os.Open(path)
		if err != nil {
			a.win.Update(func() {
				if a.hashPath != path {
					return
				}
				a.hashBusy = false
				a.hashErr = err.Error()
			})
			return
		}
		defer f.Close()
		h := hashNew(algo)
		if _, err := io.Copy(h, f); err != nil {
			a.win.Update(func() {
				if a.hashPath != path {
					return
				}
				a.hashBusy = false
				a.hashErr = err.Error()
			})
			return
		}
		sum := h.Sum(nil)
		a.win.Update(func() {
			if a.hashPath != path {
				return
			}
			a.hashBusy = false
			a.hashValue = fmt.Sprintf("%x", sum)
		})
	}()
}

func (a *App) openTerminal(dir string) {
	if dir == "" {
		return
	}
	if err := exec.Command("wt.exe", "-d", dir).Start(); err == nil {
		return
	}
	quoted := strings.ReplaceAll(dir, "'", "''")
	cmd := exec.Command("powershell.exe", "-NoExit", "-Command", "Set-Location -LiteralPath '"+quoted+"'")
	if err := cmd.Start(); err != nil {
		a.statusMsg = "无法打开终端：" + err.Error()
		a.statusErr = true
	}
}

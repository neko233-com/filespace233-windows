package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/nav"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

var fileDropOptions = transfer.DropOptions{
	Formats:    []transfer.Format{transfer.FileList, transfer.URIList},
	Operations: transfer.Copy,
}

func (a *App) view(c *ui.Context) {
	applyAppleUI(c)
	tk := theme.TokensOf(c)

	root := ui.Column(c).Fill().Clip().Gradient(appleCanvasTop, appleCanvasEdge, 145)
	root.Children(func() {
		appleBackdrop(c)
		a.header(c, tk)
		ui.Row(c).Grow(1).FillWidth().Children(func() {
			a.sidebar(c, tk)
			a.workspace(c, tk)
			if a.inspector {
				a.inspectorView(c, tk)
			}
		})
		vline(c, tk)
		a.statusBar(c, tk)
	})
	a.shortcuts(c)
	a.renderDialogs(c, tk)
}

func appleBackdrop(c *ui.Context) {
	ui.Box(c).Absolute().Left(-180).Top(-140).Width(820).Height(700).PassThrough().
		RadialGradient(ui.RadialGradient{
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(132, 108, 226, 0.18), At: 0},
				{Color: ui.Transparent, At: 1},
			},
			Center: ui.Point{X: 0.42, Y: 0.42},
			Radius: 520,
		})
	ui.Box(c).Absolute().Right(-260).Bottom(-220).Width(980).Height(820).PassThrough().
		RadialGradient(ui.RadialGradient{
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(74, 148, 222, 0.16), At: 0},
				{Color: ui.Transparent, At: 1},
			},
			Center: ui.Point{X: 0.55, Y: 0.52},
			Radius: 620,
		})
	ui.Box(c).Absolute().RightPercent(18).Top(-180).Width(620).Height(520).PassThrough().
		RadialGradient(ui.RadialGradient{
			Stops: []ui.GradientStop{
				{Color: ui.RGBA(222, 116, 176, 0.10), At: 0},
				{Color: ui.Transparent, At: 1},
			},
			Center: ui.Point{X: 0.5, Y: 0.55},
			Radius: 380,
		})
}

func (a *App) header(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(56).FillWidth().PaddingX(16).Gap(8).AlignItems(ui.Center).
		Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.30), Clarity: 0.12}).
		Children(func() {
			ui.Image(c, logoBitmap).Size(30, 30).Radius(8)
			ui.Column(c).Gap(0).Children(func() {
				ui.Text(c, "FileSpace233").FontSize(15).FontWeight(650).TextColor(tk.Text)
				ui.Text(c, "文件管理器 · Windows").FontSize(9.5).FontWeight(500).
					LetterSpacing(0.4).TextColor(tk.TextTertiary)
			})
			ui.Row(c).Height(22).PaddingX(8).Radius(11).Border(1, tk.BorderSecondary).
				AlignItems(ui.Center).Children(func() {
				ui.Text(c, "v"+version).FontSize(9).TextColor(tk.TextDescription)
			})

			ui.Box(c).Grow(1)

			if len(a.clipPaths) > 0 {
				ui.Row(c).Height(26).PaddingX(9).Gap(5).Radius(13).AlignItems(ui.Center).
					Background(tk.InfoFillTertiary).Border(1, tk.InfoBorder).Children(func() {
					ui.Icon(c, lucide.Copy()).Size(12, 12).TextColor(tk.InfoText)
					ui.Text(c, fmt.Sprintf("剪贴板 %d", len(a.clipPaths))).FontSize(10).TextColor(tk.InfoText)
				})
			}

			a.iconButton(c, tk, lucide.FolderPlus(), "新建文件夹 (Ctrl+Shift+N)", false, a.askNewFolder)
			a.iconButton(c, tk, lucide.ClipboardPaste(), "粘贴 (Ctrl+V)", false, a.paste)
			a.iconButton(c, tk, lucide.Terminal(), "在终端中打开", false, func() {
				a.openTerminal(a.currentPane().dir)
			})
			ui.Box(c).Width(1).Height(22).Background(tk.BorderSecondary)
			a.iconButton(c, tk, lucide.Columns2(), "双栏", a.mode == 2, func() {
				a.mode = 2
				if a.active >= 2 {
					a.active = 0
				}
				a.save()
			})
			a.iconButton(c, tk, lucide.Grid2X2(), "四宫格", a.mode == 4, func() {
				a.mode = 4
				a.save()
			})
			a.iconButton(c, tk, lucide.EyeOff(), "显示隐藏文件", a.currentPane().hidden, func() {
				p := a.currentPane()
				p.hidden = !p.hidden
				a.refresh(p)
				a.save()
			})
			a.iconButton(c, tk, lucide.RotateCw(), "刷新 (F5)", false, func() { a.refresh(a.currentPane()) })
			a.iconButton(c, tk, lucide.PanelRightOpen(), "检查器", a.inspector, func() { a.inspector = !a.inspector })

			search := field.SearchBar(c, &a.query, field.SearchBarProps{
				Placeholder: "搜索当前面板",
				ShortKey:    "mod+k",
				Platform:    display.HotkeyOther,
				Size:        theme.Small,
			}).Width(248).Margin(4, 0, 0, 4)
			if search.Changed() {
				a.setQuery(a.currentPane(), a.query)
			}
		})
}

func (a *App) sidebar(c *ui.Context, tk *theme.Tokens) {
	col := ui.Column(c).Width(228).MinWidth(200).FillHeight().Margin(8, 0, 8, 8).
		Radius(14).Border(1, tk.BorderSecondary).Clip().
		Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.06), Clarity: 0.55})
	col.Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(ui.RGBA(255, 255, 255, 0.55))
		scroll := ui.Scroll(c).Grow(1).PaddingX(10).PaddingY(8)
		scroll.Children(func() {
			inner := ui.Column(c).Gap(2)
			inner.Children(func() {
				eyebrow(c, tk, "位置")
				for _, f := range a.favorites {
					f := f
					a.sideRow(c, tk, sideIcon(f.Kind), f.Label, samePath(a.currentPane().dir, f.Path), f.Path, func() {
						a.navigate(a.currentPane(), f.Path, true)
					})
				}

				ui.Box(c).Height(10)
				eyebrow(c, tk, "此电脑")
				for _, d := range a.drives {
					d := d
					a.driveCard(c, tk, d, samePath(a.currentPane().dir, d.Path))
				}

				if len(a.recent) > 0 {
					ui.Box(c).Height(10)
					eyebrow(c, tk, "最近")
					for _, r := range a.recent {
						if len(a.recent) > 0 && r == "" {
							continue
						}
						r := r
						a.sideRow(c, tk, lucide.History(), filepath.Base(r), samePath(a.currentPane().dir, r), r, func() {
							a.navigate(a.currentPane(), r, true)
						})
					}
				}
			})
		})
		vline(c, tk)
		ui.Row(c).Height(34).FillWidth().PaddingX(12).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, lucide.HardDrive()).Size(12, 12).TextColor(tk.TextQuaternary)
			ui.Text(c, "托盘常驻 · "+a.hotkeyLabel).FontSize(9).TextColor(tk.TextQuaternary).
				Grow(1).SingleLine().PaddingX(6)
			if a.hotkeyErr != "" {
				ui.Text(c, "热键受限").FontSize(9).TextColor(tk.WarningText)
			}
		})
	})
}

func sideIcon(kind string) *ui.SVG {
	switch kind {
	case "home":
		return lucide.House()
	case "desktop":
		return lucide.Monitor()
	case "docs":
		return lucide.FileText()
	case "downloads":
		return lucide.Download()
	case "pictures":
		return lucide.Image()
	case "music":
		return lucide.Music()
	case "videos":
		return lucide.Video()
	default:
		return lucide.Folder()
	}
}

func (a *App) sideRow(c *ui.Context, tk *theme.Tokens, icon *ui.SVG, label string, active bool, tooltip string, onClick func()) {
	row := ui.ButtonBase(c).Height(30).FillWidth().PaddingX(8).Gap(8).Radius(8).
		AlignItems(ui.Center).Tooltip(tooltip)
	if active {
		row.Background(tk.Info)
	}
	if row.Hovered() && !active {
		row.Background(tk.FillQuaternary)
	}
	row.Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(chooseColor(active, tk.PrimaryForeground, tk.TextTertiary))
		ui.Text(c, label).FontSize(11).TextColor(chooseColor(active, tk.PrimaryForeground, tk.TextSecondary)).
			Grow(1).SingleLine()
	})
	if row.Clicked() && onClick != nil {
		onClick()
	}
}

func (a *App) driveCard(c *ui.Context, tk *theme.Tokens, d fsx.Drive, active bool) {
	card := ui.ButtonBase(c).Height(64).FillWidth().PaddingX(10).PaddingY(8).Gap(5).Radius(10).
		Border(1, chooseColor(active, tk.InfoBorder, tk.BorderSecondary)).
		Background(chooseColor(active, tk.InfoFillSecondary, tk.FillQuaternary)).Tooltip(d.Path)
	if card.Hovered() && !active {
		card.Background(tk.FillTertiary)
	}
	card.Children(func() {
		ui.Column(c).FillWidth().FillHeight().Gap(4).Justify(ui.Center).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Icon(c, lucide.HardDrive()).Size(14, 14).TextColor(chooseColor(active, tk.InfoText, tk.TextTertiary))
				ui.Text(c, d.Name).FontSize(12).FontWeight(600).TextColor(tk.Text).Grow(1).SingleLine()
				ui.Text(c, humanSize(int64(d.Free))+" 可用").FontSize(9).TextColor(tk.TextQuaternary).SingleLine()
			})
			used := d.Used()
			total := int64(d.Total)
			pct := d.UsedPercent()
			bar := ui.Row(c).Height(3).FillWidth().Radius(2).Clip().Background(tk.FillTertiary)
			bar.Children(func() {
				left := float32(used)
				right := float32(total - int64(used))
				if left < 1 {
					left = 1
				}
				if right < 1 {
					right = 1
				}
				ui.Box(c).Grow(left).FillHeight().Background(diskColor(tk, pct))
				ui.Box(c).Grow(right).FillHeight().Background(tk.FillTertiary)
			})
			ui.Text(c, fmt.Sprintf("%s / %s · %.0f%%", humanSize(int64(used)), humanSize(total), pct)).
				FontSize(9).TextColor(tk.TextDescription).SingleLine()
		})
	})
	if card.Clicked() {
		a.navigate(a.currentPane(), d.Path, true)
	}
}

func diskColor(tk *theme.Tokens, pct float64) ui.Color {
	switch {
	case pct >= 90:
		return tk.Error
	case pct >= 75:
		return tk.Warning
	default:
		return tk.Info
	}
}

func (a *App) workspace(c *ui.Context, tk *theme.Tokens) {
	grid := ui.Column(c).Grow(1).FillHeight().MinWidth(0).MinHeight(0).Padding(8, 8, 8, 8)
	grid.Children(func() {
		if a.mode == 4 {
			top := func() {
				sp := ui.SplitPercent(c, &a.splitPct,
					func() { a.paneView(c, tk, 0).Fill() },
					func() { a.paneView(c, tk, 1).Fill() },
				).Fill()
				if sp.Changed() {
					a.split = float64(a.splitPct) / 100
					a.saveSoon()
				}
			}
			bottom := func() {
				sp := ui.SplitPercent(c, &a.splitPct,
					func() { a.paneView(c, tk, 2).Fill() },
					func() { a.paneView(c, tk, 3).Fill() },
				).Fill()
				if sp.Changed() {
					a.split = float64(a.splitPct) / 100
					a.saveSoon()
				}
			}
			rows := ui.SplitVerticalPercent(c, &a.rowPct, top, bottom).Grow(1).FillHeight().Clip()
			if rows.Changed() {
				a.rowSlider = float64(a.rowPct) / 100
				a.saveSoon()
			}
			return
		}
		sp := ui.SplitPercent(c, &a.splitPct,
			func() { a.paneView(c, tk, 0).Fill() },
			func() { a.paneView(c, tk, 1).Fill() },
		).Grow(1).FillHeight().Clip()
		if sp.Changed() {
			a.split = float64(a.splitPct) / 100
			a.saveSoon()
		}
	})
}

func (a *App) paneView(c *ui.Context, tk *theme.Tokens, i int) ui.Element {
	c = c.Key(fmt.Sprintf("pane-%d", i))
	p := a.panes[i]
	p.refreshVisible()
	active := i == a.active

	col := ui.Column(c).MinWidth(0).MinHeight(0).Margin(2, 2, 2, 2).
		Radius(12).Clip().
		Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.06), Clarity: 0.55})
	col.OnClick(func() { a.setActive(i) })
	if active {
		col.Border(1, tk.Info)
	} else {
		col.Border(1, tk.BorderSecondary)
	}
	dragOver := false
	if _, ok := ui.DragOver[[]string](col); ok {
		dragOver = true
	}
	if _, ok := ui.DataDragOver(col, fileDropOptions); ok {
		dragOver = true
	}
	if dragOver {
		col.Border(1, tk.Info).Material(glass.Glass{Tint: ui.RGBA(10, 132, 255, 0.10), Clarity: 0.40})
	}

	col.Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(ui.RGBA(255, 255, 255, 0.55))
		a.paneToolbar(c, tk, i, p, active)
		vline(c, tk)
		a.paneListHeader(c, tk, i, p)
		a.paneFileList(c, tk, i, p, active)
	})

	// Internal drag: files land in this pane's folder. Folder rows below take
	// precedence because they are nested drop targets.
	if paths, ok := ui.Drop[[]string](col); ok {
		a.dropPaths(paths, p.dir)
	}
	if drop, ok := ui.DropData(col, fileDropOptions); ok {
		go func() {
			files, err := drop.Data.Files()
			if err == nil && len(files) > 0 {
				a.win.Update(func() { a.importPaths(files, p.dir) })
			}
		}()
	}
	return col
}

func (a *App) paneToolbar(c *ui.Context, tk *theme.Tokens, i int, p *Pane, active bool) {
	compact := a.paneCompact()
	ui.Row(c).Height(44).FillWidth().PaddingX(6).Gap(2).AlignItems(ui.Center).Children(func() {
		a.iconButton(c, tk, lucide.ChevronLeft(), "后退 (Alt+Left)", false, func() { a.back(p) })
		a.iconButton(c, tk, lucide.ChevronRight(), "前进 (Alt+Right)", false, func() { a.forwardDir(p) })
		a.iconButton(c, tk, lucide.ArrowUp(), "上一级 (Backspace)", false, func() { a.up(p) })

		if a.pathEdit == i {
			c.Key("address")
			input := ui.TextInput(c, &a.pathDraft).Height(28).Grow(1).Margin(4, 4, 4, 4).
				FontSize(11).PaddingX(8).Radius(6).Background(tk.BgLayout).Border(1, tk.Info).AutoFocus()
			input.OnTextKey(func(k ui.TextKey) bool {
				if k.Key == ui.KeyEscape {
					a.pathEdit = -1
					return true
				}
				return false
			})
			if input.Submitted() {
				a.navigate(p, a.pathDraft, true)
				a.pathEdit = -1
			}
			a.iconButton(c, tk, lucide.Check(), "前往", true, func() {
				a.navigate(p, a.pathDraft, true)
				a.pathEdit = -1
			})
			a.iconButton(c, tk, lucide.X(), "取消", false, func() { a.pathEdit = -1 })
			return
		}

		title := p.dir
		a.breadcrumb(c, tk, i, p, compact).Grow(1).Clip()
		ui.Row(c).Gap(0).Children(func() {
			a.iconButton(c, tk, lucide.PencilLine(), "编辑路径 (Ctrl+L)", false, func() {
				a.setActive(i)
				a.pathEdit = i
				a.pathDraft = p.dir
			})
			if !compact {
				a.iconButton(c, tk, lucide.ExternalLink(), "在资源管理器中打开", false, func() {
					mygo.Shell.ShowItemInFolder(p.dir)
				})
				a.iconButton(c, tk, lucide.Copy(), "复制路径", false, func() { a.copyPath(p.dir) })
			}
		}).Tooltip(title)
		_ = active
	})
}

func (a *App) breadcrumb(c *ui.Context, tk *theme.Tokens, i int, p *Pane, compact bool) ui.Element {
	items := breadcrumbItems(p.dir, compact)
	var crumbs []nav.BreadcrumbItem
	for _, item := range items {
		item := item
		crumbs = append(crumbs, nav.BreadcrumbItem{
			Title: item.Label,
			OnClick: func() {
				a.setActive(i)
				a.navigate(p, item.Path, true)
			},
		})
	}
	if len(crumbs) > 0 {
		crumbs[len(crumbs)-1].OnClick = nil
	}
	bc := nav.Breadcrumb(c, nav.BreadcrumbProps{Items: crumbs}).FontSize(11).Height(28)
	_ = tk
	return bc
}

type crumb struct {
	Label string
	Path  string
}

func breadcrumbItems(dir string, compact bool) []crumb {
	if dir == "" {
		return nil
	}
	clean := filepath.Clean(dir)
	vol := filepath.VolumeName(clean)
	if vol == "" {
		vol = clean
	}
	var parts []string
	rest := strings.TrimPrefix(clean, vol)
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part != "" {
			parts = append(parts, part)
		}
	}
	var all []crumb
	root := vol
	if root == "" {
		root = string(filepath.Separator)
	}
	all = append(all, crumb{Label: root, Path: root})
	path := root
	for _, part := range parts {
		path = filepath.Join(path, part)
		all = append(all, crumb{Label: part, Path: path})
	}
	if compact && len(all) > 2 {
		return []crumb{all[0], all[len(all)-1]}
	}
	if len(all) > 5 && !compact {
		return append(append([]crumb{all[0], all[1]}, crumb{Label: "…"}), all[len(all)-2:]...)
	}
	return all
}

func (a *App) paneCompact() bool {
	return a.mode == 4 || a.inspector || a.splitPct < 40
}

func (a *App) paneNarrow() bool {
	return a.mode == 2 && !a.inspector && a.splitPct < 32
}

func (a *App) paneListHeader(c *ui.Context, tk *theme.Tokens, i int, p *Pane) {
	narrow := a.paneNarrow()
	compact := a.paneCompact()
	ui.Row(c).Height(30).FillWidth().PaddingX(10).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(16)
		a.sortHeader(c, tk, p, "名称", "name", true)
		if !narrow {
			a.sortHeader(c, tk, p, "大小", "size", false).Width(chooseWidth(compact, 62, 82)).AlignItems(ui.End)
			a.sortHeader(c, tk, p, "修改时间", "time", false).Width(chooseWidth(compact, 82, 108))
			if !compact {
				a.sortHeader(c, tk, p, "类型", "type", false).Width(58)
			}
		}
	})
}

func (a *App) sortHeader(c *ui.Context, tk *theme.Tokens, p *Pane, label, key string, grow bool) ui.Element {
	el := ui.ButtonBase(c).Height(22).PaddingX(4).Radius(4).Gap(2).AlignItems(ui.Center).Tooltip("按" + label + "排序")
	if p.sortKey == key {
		el.TextColor(tk.Text)
	}
	if el.Hovered() {
		el.Background(tk.FillQuaternary)
	}
	if grow {
		el.Grow(1)
	}
	el.Children(func() {
		ui.Text(c, label).FontSize(9).FontWeight(500).LetterSpacing(0.4).
			TextColor(chooseColor(p.sortKey == key, tk.TextSecondary, tk.TextQuaternary))
		if p.sortKey == key {
			if p.sortAsc {
				ui.Icon(c, lucide.ArrowUp()).Size(10, 10).TextColor(tk.InfoText)
			} else {
				ui.Icon(c, lucide.ArrowDown()).Size(10, 10).TextColor(tk.InfoText)
			}
		}
	})
	if el.Clicked() {
		if p.sortKey == key {
			p.sortAsc = !p.sortAsc
		} else {
			p.sortKey = key
			p.sortAsc = true
		}
		a.setActive(a.indexOfPane(p))
		a.resort(p)
		a.save()
	}
	return el
}

func (a *App) indexOfPane(p *Pane) int {
	for i, x := range a.panes {
		if x == p {
			return i
		}
	}
	return a.active
}

func (a *App) paneFileList(c *ui.Context, tk *theme.Tokens, i int, p *Pane, active bool) {
	p.refreshVisible()
	p.list.Key = func(row int) any {
		if row >= 0 && row < len(p.visible) {
			return p.visible[row].Path
		}
		return row
	}
	p.list.KeepOffset = true
	p.list.Selected = &p.selected
	p.list.Selection = &p.selection
	p.list.DragRows = func(rows []int) any {
		var paths []string
		for _, row := range rows {
			if row >= 0 && row < len(p.visible) {
				paths = append(paths, p.visible[row].Path)
			}
		}
		return paths
	}

	p.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyC, func() { a.setActive(i); a.copySelection() })
	p.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyX, func() { a.setActive(i); a.cutSelection() })
	p.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyV, func() { a.setActive(i); a.paste() })
	p.list.Handle.OnShortcut(c, 0, ui.KeyDelete, func() { a.setActive(i); a.askDelete() })
	p.list.Handle.OnShortcut(c, 0, ui.KeyF2, func() { a.setActive(i); a.askRename() })
	p.list.Handle.OnShortcut(c, ui.Cmd|ui.Shift, ui.KeyN, func() { a.setActive(i); a.askNewFolder() })
	p.list.Handle.OnShortcut(c, ui.Cmd|ui.Shift, ui.KeyD, func() { a.setActive(i); a.askDuplicate() })
	p.list.Handle.OnShortcut(c, ui.Cmd|ui.Shift, ui.KeyR, func() { a.setActive(i); a.askBatchRename() })
	p.list.Handle.OnShortcut(c, 0, ui.KeyBackspace, func() { a.setActive(i); a.up(p) })
	p.list.Handle.OnShortcut(c, ui.Alt, ui.KeyLeft, func() { a.setActive(i); a.back(p) })
	p.list.Handle.OnShortcut(c, ui.Alt, ui.KeyRight, func() { a.setActive(i); a.forwardDir(p) })

	if p.loading && len(p.entries) == 0 {
		a.paneEmpty(c, tk, i, "正在读取文件夹…", p.dir)
		return
	}
	if p.err != "" {
		a.paneEmpty(c, tk, i, "无法读取文件夹", p.err)
		return
	}
	if len(p.visible) == 0 {
		msg := "空文件夹"
		sub := "右键可新建文件夹或粘贴"
		if p.query != "" {
			msg = "没有匹配项"
			sub = "换个关键词，或按 Esc 清除搜索"
		}
		a.paneEmpty(c, tk, i, msg, sub)
		return
	}

	menuFor := func(m *ui.Menu) { a.backgroundMenu(m, i, p) }
	dropSrc := []string(nil)
	dropDir := ""

	listEl := ui.List(c.Key("files"), &p.list, len(p.visible)).Grow(1).FillWidth().
		Background(ui.Transparent).ContextMenu(menuFor).Rows(func(row ui.ListRow) {
		e := p.visible[row.Index]
		selected := p.selection.Has(e.Path) || row.Selected()
		if row.Index == p.selected && p.selection.Len() == 0 {
			selected = true
		}
		cut := a.clipMode == "cut" && pathIn(a.clipPaths, e.Path)
		rowEl := ui.Row(row.Context).Height(34).FillWidth().PaddingX(10).Gap(8).Radius(8).
			AlignItems(ui.Center)
		if selected {
			if active {
				rowEl.Background(tk.Info)
			} else {
				rowEl.Background(ui.RGBA(0, 0, 0, 0.06))
			}
		} else if row.Hovered() {
			rowEl.Background(tk.FillTertiary)
		}
		if cut {
			rowEl.Opacity(0.45)
		}
		selectedFg := selected && active
		textColor := chooseColor(selectedFg, tk.PrimaryForeground, tk.Text)
		mutedColor := chooseColor(selectedFg, tk.PrimaryForeground, tk.TextDescription)
		iconColor := chooseColor(selectedFg, tk.PrimaryForeground, entryIconColor(tk, e))

		rowEl.Children(func() {
			ui.Icon(row.Context, entryIcon(e)).Size(15, 15).TextColor(iconColor).Shrink(0).Width(16)
			ui.Text(row.Context, e.Name).FontSize(12).FontWeight(chooseWeight(selected)).
				TextColor(textColor).Grow(1).SingleLine()
			if !a.paneNarrow() {
				size := "—"
				if !e.IsDir {
					size = humanSize(e.Size)
				}
				ui.Text(row.Context, size).FontSize(10).TextColor(mutedColor).
					Width(chooseWidth(a.paneCompact(), 62, 82)).TextAlign(ui.End).SingleLine()
				ui.Text(row.Context, formatEntryTime(e.ModTime)).FontSize(10).TextColor(mutedColor).
					Width(chooseWidth(a.paneCompact(), 82, 108)).SingleLine()
				if !a.paneCompact() {
					ui.Text(row.Context, entryType(e)).FontSize(9).TextColor(mutedColor).
						Width(58).SingleLine()
				}
			}
		})

		rowEl.ContextMenu(func(m *ui.Menu) { a.rowMenu(m, i, p, e) })

		if e.IsDir {
			if _, ok := ui.DragOver[[]string](rowEl); ok {
				rowEl.Background(tk.InfoFillSecondary).Border(1, tk.Info)
			}
			if paths, ok := ui.Drop[[]string](rowEl); ok {
				dropSrc = paths
				dropDir = e.Path
			}
		}
	})
	if dropDir != "" && len(dropSrc) > 0 {
		a.dropPaths(dropSrc, dropDir)
	}
	if p.scrollTo != "" {
		p.refreshVisible()
		for idx, e := range p.visible {
			if samePath(e.Path, p.scrollTo) {
				p.list.ScrollIntoView(idx)
				break
			}
		}
		p.scrollTo = ""
	}

	if a.focusPane == i && p.list.Focus(c) {
		a.focusPane = -1
	}
	if listEl.Submitted() {
		a.setActive(i)
		a.openSelected(p)
	}
}

func (a *App) paneEmpty(c *ui.Context, tk *theme.Tokens, i int, title, sub string) {
	ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Gap(6).
		ContextMenu(func(m *ui.Menu) { a.backgroundMenu(m, i, a.panes[i]) }).Children(func() {
		ui.Icon(c, lucide.Folder()).Size(28, 28).TextColor(tk.TextQuaternary)
		ui.Text(c, title).FontSize(12).TextColor(tk.TextSecondary)
		ui.Text(c, sub).FontSize(10).TextColor(tk.TextQuaternary)
	})
}

func (a *App) rowMenu(m *ui.Menu, i int, p *Pane, e fsx.Entry) {
	count := p.chosenCount()
	paths := p.chosenPaths()
	var action func()
	run := func(fn func()) {
		action = func() {
			a.setActive(i)
			fn()
		}
	}
	if count == 1 {
		if m.Item("打开").Shortcut(0, ui.KeyEnter).Chosen() {
			run(func() { a.open(p, e.Path) })
		}
	}
	if m.Item("在资源管理器中显示").Chosen() {
		run(func() { mygo.Shell.ShowItemInFolder(e.Path) })
	}
	m.Separator()
	label := "复制"
	if count > 1 {
		label = fmt.Sprintf("复制 %d 项", count)
	}
	if m.Item(label).Shortcut(ui.Cmd, ui.KeyC).Chosen() {
		run(a.copySelection)
	}
	label = "剪切"
	if count > 1 {
		label = fmt.Sprintf("剪切 %d 项", count)
	}
	if m.Item(label).Shortcut(ui.Cmd, ui.KeyX).Chosen() {
		run(a.cutSelection)
	}
	m.Submenu("复制到面板", func(sub *ui.Menu) {
		for _, j := range a.otherPaneIndexes() {
			j := j
			if sub.Item(fmt.Sprintf("面板 %d · %s", j+1, filepath.Base(a.panes[j].dir))).Chosen() {
				run(func() { a.copyToPane(j, false) })
			}
		}
	})
	m.Submenu("移动到面板", func(sub *ui.Menu) {
		for _, j := range a.otherPaneIndexes() {
			j := j
			if sub.Item(fmt.Sprintf("面板 %d · %s", j+1, filepath.Base(a.panes[j].dir))).Chosen() {
				run(func() { a.copyToPane(j, true) })
			}
		}
	})
	m.Separator()
	if m.Item("重命名").Shortcut(0, ui.KeyF2).Disabled(count != 1).Chosen() {
		run(a.askRename)
	}
	if m.Item("批量重命名…").Disabled(count < 2).Chosen() {
		run(a.askBatchRename)
	}
	if m.Item("创建副本").Shortcut(ui.Cmd|ui.Shift, ui.KeyD).Chosen() {
		run(a.askDuplicate)
	}
	if m.Item("复制路径").Chosen() {
		run(func() { a.copyPaths(paths) })
	}
	m.Separator()
	if m.Item("计算哈希").Disabled(count != 1 || e.IsDir).Chosen() {
		run(func() {
			a.inspector = true
			a.startHash(e.Path)
		})
	}
	if m.Item("检查器").Chosen() {
		run(func() { a.inspector = true })
	}
	if e.IsDir {
		if m.Item("在终端中打开").Chosen() {
			run(func() { a.openTerminal(e.Path) })
		}
	}
	m.Separator()
	if m.Item("移到回收站").Danger().Shortcut(0, ui.KeyDelete).Chosen() {
		run(a.askDelete)
	}
	if action != nil {
		action()
	}
}

func (a *App) backgroundMenu(m *ui.Menu, i int, p *Pane) {
	var action func()
	run := func(fn func()) {
		action = func() {
			a.setActive(i)
			fn()
		}
	}
	if m.Item("新建文件夹").Shortcut(ui.Cmd|ui.Shift, ui.KeyN).Chosen() {
		run(a.askNewFolder)
	}
	if m.Item("新建文本文件").Chosen() {
		run(a.askNewFile)
	}
	if m.Item("粘贴").Shortcut(ui.Cmd, ui.KeyV).Chosen() {
		run(a.paste)
	}
	m.Separator()
	m.Submenu("同步选择", func(sub *ui.Menu) {
		if sub.Item("两边同名").Chosen() {
			run(func() { a.syncSelect("both") })
		}
		if sub.Item("仅本面板有").Chosen() {
			run(func() { a.syncSelect("left") })
		}
		if sub.Item("仅其他面板有").Chosen() {
			run(func() { a.syncSelect("right") })
		}
		if sub.Item("两边大小不同").Chosen() {
			run(func() { a.syncSelect("size") })
		}
	})
	m.Separator()
	if m.Item("刷新").Shortcut(0, ui.KeyF5).Chosen() {
		run(func() { a.refresh(p) })
	}
	if m.Item("显示隐藏文件").Checked(p.hidden).Chosen() {
		run(func() {
			p.hidden = !p.hidden
			a.refresh(p)
			a.save()
		})
	}
	m.Separator()
	if m.Item("在资源管理器中打开").Chosen() {
		run(func() { mygo.Shell.ShowItemInFolder(p.dir) })
	}
	if m.Item("在终端中打开").Chosen() {
		run(func() { a.openTerminal(p.dir) })
	}
	if action != nil {
		action()
	}
}

func (a *App) iconButton(c *ui.Context, tk *theme.Tokens, icon *ui.SVG, tip string, active bool, onClick func()) ui.Element {
	b := ui.ButtonBase(c).Size(30, 30).Radius(6).AlignItems(ui.Center).Justify(ui.Center).Tooltip(tip)
	if active {
		b.Background(tk.FillTertiary)
	}
	if b.Hovered() {
		b.Background(tk.FillSecondary)
	}
	iconColor := chooseColor(active, tk.InfoText, tk.TextTertiary)
	b.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(iconColor)
	})
	if b.Clicked() && onClick != nil {
		onClick()
	}
	return b
}

func (a *App) inspectorView(c *ui.Context, tk *theme.Tokens) {
	p := a.currentPane()
	a.syncInspector(p)
	entries := p.chosenEntries()

	col := ui.Column(c).Width(320).MinWidth(280).FillHeight().Margin(8, 8, 8, 0).
		Radius(14).Border(1, tk.BorderSecondary).Clip().
		Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.06), Clarity: 0.55})
	col.Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(ui.RGBA(255, 255, 255, 0.55))
		ui.Row(c).Height(46).FillWidth().PaddingX(14).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "检查器").FontSize(12).FontWeight(600).TextColor(tk.Text).Grow(1)
			ui.Text(c, fmt.Sprintf("%d 项", len(entries))).FontSize(9).TextColor(tk.TextQuaternary)
			a.iconButton(c, tk, lucide.X(), "关闭检查器", false, func() { a.inspector = false })
		})
		vline(c, tk)
		body := ui.Scroll(c).Grow(1).Padding(14)
		body.Children(func() {
			if len(entries) == 0 {
				ui.Column(c).Gap(6).PaddingY(40).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, lucide.Search()).Size(24, 24).TextColor(tk.TextQuaternary)
					ui.Text(c, "选择一个项目").FontSize(12).TextColor(tk.TextSecondary)
					ui.Text(c, "预览、元数据与操作会显示在这里").FontSize(10).TextColor(tk.TextQuaternary)
				})
				return
			}
			if len(entries) > 1 {
				a.inspectorMulti(c, tk, entries)
			} else {
				a.inspectorSingle(c, tk, entries[0])
			}
		})
		vline(c, tk)
		a.inspectorActions(c, tk, entries)
	})
}

func (a *App) inspectorMulti(c *ui.Context, tk *theme.Tokens, entries []fsx.Entry) {
	var files, dirs int
	var bytes int64
	for _, e := range entries {
		if e.IsDir {
			dirs++
		} else {
			files++
			bytes += e.Size
		}
	}
	box := ui.Column(c).FillWidth().Gap(8).Padding(12).Radius(8).
		Background(tk.BgLayout).Border(1, tk.BorderSecondary)
	box.Children(func() {
		ui.Icon(c, lucide.Files()).Size(28, 28).TextColor(tk.InfoText)
		ui.Text(c, fmt.Sprintf("已选 %d 项", len(entries))).FontSize(16).FontWeight(600).TextColor(tk.Text)
		ui.Text(c, fmt.Sprintf("%d 文件 · %d 文件夹", files, dirs)).FontSize(10).TextColor(tk.TextDescription)
		if files > 0 {
			ui.Text(c, humanSize(bytes)).FontSize(11).TextColor(tk.TextSecondary)
		}
	})
	ui.Box(c).Height(12)
	eyebrow(c, tk, "选择内容")
	for i, e := range entries {
		if i >= 12 {
			ui.Text(c, fmt.Sprintf("… 另外 %d 项", len(entries)-12)).FontSize(10).TextColor(tk.TextQuaternary)
			break
		}
		ui.Row(c).FillWidth().Height(26).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, entryIcon(e)).Size(13, 13).TextColor(tk.TextTertiary)
			ui.Text(c, e.Name).FontSize(10).TextColor(tk.TextSecondary).Grow(1).SingleLine()
		})
	}
}

func (a *App) inspectorSingle(c *ui.Context, tk *theme.Tokens, e fsx.Entry) {
	a.inspectorPreview(c, tk, e)

	ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, entryIcon(e)).Size(22, 22).TextColor(chooseColor(e.IsDir, tk.InfoText, tk.TextSecondary))
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, e.Name).FontSize(15).FontWeight(600).TextColor(tk.Text).SingleLine()
			ui.Text(c, entryType(e)+" · "+humanSize(e.Size)).FontSize(9).TextColor(tk.TextQuaternary)
		})
	})
	ui.Box(c).Height(12)

	eyebrow(c, tk, "元数据")
	a.metaRow(c, tk, "类型", entryType(e))
	if !e.IsDir {
		a.metaRow(c, tk, "大小", humanSize(e.Size))
	}
	a.metaRow(c, tk, "修改时间", e.ModTime.Format("2006-01-02 15:04:05"))
	if !e.IsDir {
		a.metaRow(c, tk, "扩展名", strings.ToUpper(strings.TrimPrefix(e.Ext, ".")))
	}
	ui.Box(c).Height(10)
	eyebrow(c, tk, "路径")
	ui.Text(c, e.Path).FontSize(10).TextColor(tk.TextSecondary).Selectable().Padding(8).
		FillWidth().Background(tk.BgLayout).Border(1, tk.BorderSecondary).Radius(6)

	if !e.IsDir {
		ui.Box(c).Height(10)
		eyebrow(c, tk, "哈希校验")
		ui.Row(c).Gap(4).Children(func() {
			for _, algo := range []string{"MD5", "SHA-1", "SHA-256", "SHA-512"} {
				algo := algo
				on := a.hashAlgo == algo
				b := ui.ButtonBase(c).Height(20).PaddingX(6).Radius(4).
					Border(1, chooseColor(on, tk.InfoBorder, tk.BorderSecondary)).
					Background(chooseColor(on, tk.InfoFillTertiary, tk.BgLayout))
				if b.Hovered() && !on {
					b.Background(tk.FillQuaternary)
				}
				b.Children(func() {
					ui.Text(c, algo).FontSize(9).TextColor(chooseColor(on, tk.InfoText, tk.TextDescription))
				})
				if b.Clicked() {
					a.startHashAlgo(algo, e.Path)
				}
			}
		})
		switch {
		case a.hashBusy && samePath(a.hashPath, e.Path):
			ui.Text(c, "正在计算…").FontSize(10).TextColor(tk.TextQuaternary)
		case a.hashErr != "" && samePath(a.hashPath, e.Path):
			ui.Text(c, a.hashErr).FontSize(10).TextColor(tk.ErrorText)
		case a.hashValue != "" && samePath(a.hashPath, e.Path):
			ui.Text(c, a.hashValue).FontSize(9).TextColor(tk.TextDescription).Selectable()
		default:
			b := a.textButton(c, tk, "计算 "+a.hashAlgo, false, nil)
			if b.Clicked() {
				a.startHash(e.Path)
			}
		}
	}
}

func (a *App) inspectorPreview(c *ui.Context, tk *theme.Tokens, e fsx.Entry) {
	kind := previewKind(e)
	if kind != "image" && kind != "text" {
		return
	}
	box := ui.Column(c).FillWidth().Height(chooseHeight(kind, 170, 172)).Padding(8).Gap(6).
		Radius(8).Background(tk.BgLayout).Border(1, tk.BorderSecondary)
	box.Children(func() {
		switch {
		case a.previewLoad:
			ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(c, "正在生成预览…").FontSize(10).TextColor(tk.TextQuaternary)
			})
		case a.previewErr != "":
			ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Padding(8).Children(func() {
				ui.Text(c, a.previewErr).FontSize(10).TextColor(tk.TextQuaternary)
			})
		case kind == "image":
			if a.previewBitmap != nil {
				ui.Image(c, a.previewBitmap).Fill().Fit(ui.Contain).Radius(4)
			}
		case kind == "text":
			inner := ui.Scroll(c).Fill()
			inner.Children(func() {
				ui.Text(c, a.previewText).FontSize(10).TextColor(tk.TextSecondary).Selectable()
			})
		}
	})
	if a.previewDim != "" && !a.previewLoad {
		ui.Text(c, a.previewDim).FontSize(9).TextColor(tk.TextQuaternary)
	}
	ui.Box(c).Height(12)
}

func (a *App) metaRow(c *ui.Context, tk *theme.Tokens, label, value string) {
	ui.Row(c).FillWidth().PaddingY(3).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(10).Width(68).TextColor(tk.TextQuaternary)
		ui.Text(c, value).FontSize(10).TextColor(tk.TextSecondary).Grow(1).Selectable()
	})
}

func (a *App) inspectorActions(c *ui.Context, tk *theme.Tokens, entries []fsx.Entry) {
	ui.Row(c).Height(52).FillWidth().PaddingX(12).Gap(6).AlignItems(ui.Center).Children(func() {
		if len(entries) == 0 {
			ui.Text(c, "选择文件后可执行操作").FontSize(10).TextColor(tk.TextQuaternary)
			return
		}
		if len(entries) > 1 {
			b := a.textButton(c, tk, "复制路径", false, nil)
			if b.Clicked() {
				a.copyPaths(a.currentPane().chosenPaths())
			}
			return
		}
		e := entries[0]
		b := a.textButton(c, tk, "打开", true, nil)
		if b.Clicked() {
			a.open(a.currentPane(), e.Path)
		}
		b = a.textButton(c, tk, "定位", false, nil)
		if b.Clicked() {
			mygo.Shell.ShowItemInFolder(e.Path)
		}
		b = a.textButton(c, tk, "复制", false, nil)
		if b.Clicked() {
			a.copyPath(e.Path)
		}
		if e.IsDir {
			a.iconButton(c, tk, lucide.Terminal(), "在终端中打开", false, func() { a.openTerminal(e.Path) })
		}
	})
}

func (a *App) textButton(c *ui.Context, tk *theme.Tokens, label string, primary bool, icon *ui.SVG) ui.Element {
	b := ui.ButtonBase(c).Height(30).PaddingX(10).Gap(5).Radius(6).AlignItems(ui.Center).
		Border(1, chooseColor(primary, tk.Info, tk.BorderSecondary))
	if primary {
		b.Background(tk.Info).TextColor(tk.PrimaryForeground)
	} else {
		b.Background(tk.FillQuaternary).TextColor(tk.TextSecondary)
	}
	if b.Hovered() {
		if primary {
			b.Background(tk.InfoHover)
		} else {
			b.Background(tk.FillSecondary)
		}
	}
	b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(chooseColor(primary, tk.PrimaryForeground, tk.TextTertiary))
		}
		ui.Text(c, label).FontSize(10).FontWeight(chooseWeight(primary)).
			TextColor(chooseColor(primary, tk.PrimaryForeground, tk.TextSecondary))
	})
	return b
}

func (a *App) statusBar(c *ui.Context, tk *theme.Tokens) {
	p := a.currentPane()
	ui.Row(c).Height(36).FillWidth().PaddingX(12).Gap(10).AlignItems(ui.Center).
		Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.22), Clarity: 0.15}).
		Children(func() {
			if a.opBusy {
				ui.Text(c, a.opLabel).FontSize(9).FontWeight(600).TextColor(tk.InfoText)
				ui.Row(c).Width(96).Height(3).Radius(2).Clip().Background(tk.FillTertiary).Children(func() {
					done := float32(a.opDone)
					remaining := float32(maxInt(a.opTotal-a.opDone, 0))
					if done < 1 {
						done = 1
					}
					if remaining < 1 {
						remaining = 1
					}
					ui.Box(c).Grow(done).FillHeight().Background(tk.Info)
					ui.Box(c).Grow(remaining).FillHeight()
				})
				ui.Text(c, fmt.Sprintf("%d/%d · %s", a.opDone, a.opTotal, a.opCurrent)).
					FontSize(9).TextColor(tk.TextDescription).Grow(1).SingleLine()
			} else if a.statusMsg != "" {
				ui.Text(c, a.statusMsg).FontSize(9).TextColor(chooseColor(a.statusErr, tk.ErrorText, tk.TextSecondary)).
					Grow(1).SingleLine()
			} else {
				sel := p.chosenCount()
				txt := p.dir
				if sel > 0 {
					txt = fmt.Sprintf("%s  ·  已选 %d 项", p.dir, sel)
				}
				ui.Text(c, txt).FontSize(9).TextColor(tk.TextQuaternary).Grow(1).SingleLine()
			}

			ui.Text(c, fmt.Sprintf("%d 项 · %d 文件 · %d 文件夹 · %s",
				p.stats.Total, p.stats.Files, p.stats.Dirs, humanSize(p.stats.Bytes))).
				FontSize(9).TextColor(tk.TextQuaternary).SingleLine()

			ui.Box(c).Width(1).Height(18).Background(tk.BorderSecondary)
			ui.Text(c, "分割").FontSize(9).TextColor(tk.TextQuaternary)
			slider := ui.Slider(c, &a.split, 0.25, 0.75).Width(86).Height(16)
			if slider.Changed() {
				a.splitPct = float32(a.split * 100)
				a.saveSoon()
			}
			if a.mode == 4 {
				ui.Text(c, "行高").FontSize(9).TextColor(tk.TextQuaternary)
				rs := ui.Slider(c, &a.rowSlider, 0.25, 0.75).Width(70).Height(16)
				if rs.Changed() {
					a.rowPct = float32(a.rowSlider * 100)

					a.saveSoon()
				}
			}
			ui.Text(c, "Ctrl+Win+E").FontSize(8).TextColor(tk.TextQuaternary).PaddingX(4)
			a.iconButton(c, tk, lucide.PanelRightOpen(), "检查器", a.inspector, func() { a.inspector = !a.inspector })
		})
}

func (a *App) shortcuts(c *ui.Context) {
	c.OnWindowShortcut(0, ui.KeyF5, func() {
		a.drives = fsx.Drives()
		a.refreshGroup([]string{a.currentPane().dir}, "")
	})
	c.OnWindowShortcut(0, ui.KeyEscape, func() {
		if a.pathEdit >= 0 {
			a.pathEdit = -1
			return
		}
		p := a.currentPane()
		if a.query != "" || p.query != "" {
			a.query = ""
			a.setQuery(p, "")
			return
		}
		if p.selection.Len() > 0 {
			p.selection.Clear()
			p.selected = -1
			p.list = ui.ListState{}
		}
	})
	c.OnWindowShortcut(ui.Cmd, ui.KeyL, func() {
		p := a.currentPane()
		a.pathEdit = a.active
		a.pathDraft = p.dir
	})
	c.OnWindowShortcut(ui.Cmd, ui.KeyTab, func() {
		a.setActive((a.active + 1) % a.paneCount())
	})
	for i := 0; i < 4; i++ {
		i := i
		if i >= a.paneCount() {
			continue
		}
		c.OnWindowShortcut(ui.Cmd, ui.Key(int(ui.Key1)+i), func() { a.setActive(i) })
	}
}

func entryIcon(e fsx.Entry) *ui.SVG {
	if e.IsDir {
		return lucide.Folder()
	}
	switch strings.ToLower(e.Ext) {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "svg":
		return lucide.FileImage()
	case "mp4", "mkv", "mov", "avi", "webm":
		return lucide.FileVideo()
	case "mp3", "wav", "flac", "ogg", "m4a":
		return lucide.FileAudio()
	case "zip", "rar", "7z", "tar", "gz", "bz2", "xz":
		return lucide.FileArchive()
	case "go", "ts", "tsx", "js", "jsx", "rs", "cs", "c", "cpp", "h", "py", "java", "kt", "swift":
		return lucide.FileCode()
	case "md", "txt", "log", "json", "yaml", "yml", "toml", "ini":
		return lucide.FileText()
	default:
		return lucide.File()
	}
}

func entryIconColor(tk *theme.Tokens, e fsx.Entry) ui.Color {
	if e.IsDir {
		return tk.InfoText
	}
	switch strings.ToLower(e.Ext) {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "svg":
		return tk.TextSecondary
	case "mp4", "mkv", "mov", "avi", "webm":
		return tk.TextSecondary
	default:
		return tk.TextTertiary
	}
}

func entryType(e fsx.Entry) string {
	if e.IsDir {
		return "文件夹"
	}
	ext := strings.TrimPrefix(strings.ToLower(e.Ext), ".")
	if ext == "" {
		return "文件"
	}
	return strings.ToUpper(ext) + " 文件"
}

func formatEntryTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	if time.Since(t) < 24*time.Hour && time.Since(t) >= 0 {
		return t.Format("15:04")
	}
	if t.Year() == time.Now().Year() {
		return t.Format("01-02 15:04")
	}
	return t.Format("2006-01-02")
}

func humanSize(n int64) string {
	if n < 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit && exp < 3; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func chooseColor(cond bool, yes, no ui.Color) ui.Color {
	if cond {
		return yes
	}
	return no
}

func chooseWeight(bold bool) int {
	if bold {
		return 600
	}
	return 450
}

func chooseWidth(compact bool, a, b float32) float32 {
	if compact {
		return a
	}
	return b
}

func chooseHeight(kind string, image, text float32) float32 {
	if kind == "image" {
		return image
	}
	return text
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func pathIn(paths []string, path string) bool {
	for _, p := range paths {
		if samePath(p, path) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

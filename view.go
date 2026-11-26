package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/nav"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

func (a *App) view(c *ui.Context) {
	appTheme.Apply(c)
	tk := theme.TokensOf(c)

	root := ui.Column(c).Fill().Clip()
	if !c.Vibrancy() {
		root.Background(tk.BgLayout)
	}
	root.Children(func() {
		a.header(c, tk)
		separator(c, tk)
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, tk)
			a.workspace(c, tk)
		})
		separator(c, tk)
		a.statusBar(c, tk)
	})
}

func separator(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary)
}

func vSeparator(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Width(1).FillHeight().Background(tk.BorderSecondary)
}

func (a *App) header(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(58).PaddingX(14).Gap(10).AlignItems(ui.Center).Background(tk.BgContainer).Children(func() {
		ui.Image(c, logoBitmap).Size(30, 30).Radius(8)
		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, "FileSpace233").FontSize(16).FontWeight(700).TextColor(tk.Text)
			ui.Text(c, "文件夹管理器 · Q-Dir 多面板").FontSize(11).TextColor(tk.TextDescription)
		})
		ui.Box(c).Grow(1)

		modeIcon := lucide.Columns2()
		if a.mode == 4 {
			modeIcon = lucide.Grid2X2()
		}
		if action.ActionIcon(c, modeIcon, action.ActionIconProps{
			Title:  "切换 2 / 4 宫格",
			Active: a.mode == 4,
			Size:   theme.Middle,
		}).Clicked() {
			if a.mode == 4 {
				a.mode = 2
				if a.active >= 2 {
					a.active = 0
				}
			} else {
				a.mode = 4
			}
			a.query = a.currentPane().query
			a.save()
		}

		search := field.SearchBar(c, &a.query, field.SearchBarProps{
			Placeholder: "搜索活动面板",
			ShortKey:    "mod+k",
			Size:        theme.Middle,
		}).Width(280)
		if search.Changed() {
			p := a.currentPane()
			p.query = a.query
			p.selected = -1
			p.list = ui.ListState{}
		}

		p := a.currentPane()
		hiddenIcon := lucide.EyeOff()
		if p.hidden {
			hiddenIcon = lucide.Eye()
		}
		if action.ActionIcon(c, hiddenIcon, action.ActionIconProps{
			Title:  "显示活动面板的隐藏文件",
			Active: p.hidden,
			Size:   theme.Middle,
		}).Clicked() {
			p.hidden = !p.hidden
			a.refresh(p)
		}
		if action.ActionIcon(c, lucide.RotateCw(), action.ActionIconProps{
			Title: "刷新活动面板",
			Size:  theme.Middle,
		}).Clicked() {
			a.drives = fsx.Drives()
			a.refresh(p)
		}
	})
}

func (a *App) sidebar(c *ui.Context, tk *theme.Tokens) {
	col := ui.Column(c).Width(238).Padding(10).Gap(6).Background(tk.BgContainer).MinHeight(0)
	col.Children(func() {
		sectionTitle(c, tk, "快速访问")

		items := make([]display.ListItem, 0, len(a.favorites))
		active := ""
		for _, f := range a.favorites {
			key := "fav|" + f.Path
			if samePath(a.currentPane().dir, f.Path) {
				active = key
			}
			items = append(items, display.ListItem{
				Key:   key,
				Label: f.Label,
				Icon:  favoriteIcon(f.Kind),
			})
		}
		if clicked := display.List(c, display.ListProps{Items: items, Active: active, Compact: true}); clicked != "" {
			a.navigate(a.currentPane(), strings.TrimPrefix(clicked, "fav|"), true)
		}

		ui.Box(c).Height(8)
		sectionTitle(c, tk, "此电脑")
		if len(a.drives) == 0 {
			ui.Text(c, "没有可用的驱动器").FontSize(12).TextColor(tk.TextDescription).PaddingX(4)
		}
		for _, d := range a.drives {
			p := a.currentPane()
			isActive := samePath(p.dir, d.Path)
			row := ui.ButtonBase(c).Height(58).PaddingX(6).Radius(6).Justify(ui.Start)
			if isActive {
				row.Background(tk.InfoFillSecondary)
			} else if row.Hovered() {
				row.Background(tk.FillTertiary)
			}
			row.Children(func() {
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, lucide.HardDrive()).Size(16, 16).TextColor(tk.Info)
						ui.Text(c, d.Name).FontSize(12).FontWeight(600).TextColor(tk.Text)
						ui.Box(c).Grow(1)
						ui.Text(c, driveFreeLabel(d)).FontSize(10).TextColor(tk.TextQuaternary)
					})
					used := d.UsedPercent()
					if used < 0 {
						used = 0
					}
					if used > 1 {
						used = 1
					}
					ui.Row(c).Height(4).FillWidth().Radius(2).Background(tk.FillTertiary).Children(func() {
						ui.Box(c).Height(4).Grow(float32(used)).Basis(0).MinWidth(0).Radius(2).Background(driveBarColor(tk, used))
						ui.Box(c).Height(4).Grow(float32(1 - used)).Basis(0).MinWidth(0)
					})
					ui.Text(c, driveUsageLabel(d)).FontSize(10).TextColor(tk.TextDescription)
				})
			})
			if row.Clicked() {
				a.navigate(a.currentPane(), d.Path, true)
			}
		}

		ui.Box(c).Grow(1)
		ui.Text(c, "关闭窗口后仍在托盘常驻").FontSize(10).TextColor(tk.TextQuaternary).PaddingX(4)
	})
	vSeparator(c, tk)
}

func sectionTitle(c *ui.Context, tk *theme.Tokens, s string) {
	ui.Text(c, s).FontSize(11).FontWeight(600).TextColor(tk.TextDescription).PaddingX(4).PaddingY(4)
}

func favoriteIcon(kind string) *ui.SVG {
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
		return lucide.Film()
	default:
		return lucide.Folder()
	}
}

func driveFreeLabel(d fsx.Drive) string {
	if d.Total == 0 {
		return "—"
	}
	return "可用 " + humanSizeU(d.Free)
}

func driveUsageLabel(d fsx.Drive) string {
	if d.Total == 0 {
		return "容量未知"
	}
	return fmt.Sprintf("已用 %s / %s · %.0f%%", humanSizeU(d.Used()), humanSizeU(d.Total), d.UsedPercent()*100)
}

func driveBarColor(tk *theme.Tokens, used float64) ui.Color {
	switch {
	case used >= 0.9:
		return tk.Error
	case used >= 0.75:
		return tk.Warning
	default:
		return tk.Info
	}
}

func (a *App) workspace(c *ui.Context, tk *theme.Tokens) {
	row := ui.Row(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).AlignItems(ui.Stretch).Clip()
	row.Children(func() {
		a.paneGrid(c, tk).Grow(1).Basis(0).MinWidth(0).MinHeight(0)
		if a.inspector {
			vSeparator(c, tk)
			a.inspectorPanel(c, tk)
		}
	})
}

func (a *App) paneGrid(c *ui.Context, tk *theme.Tokens) ui.Element {
	if a.mode == 2 {
		sp := ui.SplitPercent(c, &a.splitPct,
			func() { a.paneView(c, tk, 0).Fill() },
			func() { a.paneView(c, tk, 1).Fill() },
		).Grow(1).FillHeight().Clip()
		if sp.Changed() {
			a.split = float64(a.splitPct) / 100
			a.save()
		}
		return sp
	}

	top := func() {
		sp := ui.SplitPercent(c, &a.splitPct,
			func() { a.paneView(c, tk, 0).Fill() },
			func() { a.paneView(c, tk, 1).Fill() },
		).Fill()
		if sp.Changed() {
			a.split = float64(a.splitPct) / 100
			a.save()
		}
	}
	bottom := func() {
		ui.SplitPercent(c, &a.splitPct,
			func() { a.paneView(c, tk, 2).Fill() },
			func() { a.paneView(c, tk, 3).Fill() },
		).Fill()
	}
	rows := ui.SplitVerticalPercent(c, &a.rowPct, top, bottom).Grow(1).FillHeight().Clip()
	return rows
}

func (a *App) paneView(c *ui.Context, tk *theme.Tokens, i int) ui.Element {
	c = c.Key(fmt.Sprintf("pane-%d", i))
	p := a.panes[i]
	active := i == a.active
	col := ui.Column(c).MinWidth(0).MinHeight(0).Background(tk.BgLayout).Clip()
	if active {
		col.Border(2, tk.Info)
	} else {
		col.Border(1, tk.BorderSecondary)
	}
	col.Children(func() {
		a.paneToolbar(c, tk, i, p)
		separator(c, tk)
		a.paneListHeader(c, tk, i, p)
		a.paneFileList(c, tk, i, p)
	})
	return col
}

func (a *App) paneToolbar(c *ui.Context, tk *theme.Tokens, i int, p *Pane) {
	compact := a.mode == 4 || a.inspector
	ui.Row(c).Height(42).PaddingX(8).Gap(2).AlignItems(ui.Center).Children(func() {
		if action.ActionIcon(c, lucide.ArrowLeft(), action.ActionIconProps{Title: "后退", Size: theme.Small}).Clicked() {
			a.setActive(i)
			a.back(p)
		}
		if action.ActionIcon(c, lucide.ArrowRight(), action.ActionIconProps{Title: "前进", Size: theme.Small}).Clicked() {
			a.setActive(i)
			a.forwardDir(p)
		}
		if action.ActionIcon(c, lucide.ArrowUp(), action.ActionIconProps{Title: "上一级（Backspace）", Size: theme.Small}).Clicked() {
			a.setActive(i)
			a.up(p)
		}
		ui.Box(c).Width(3)
		nav.Breadcrumb(c, nav.BreadcrumbProps{Items: a.crumbs(p, compact), Separator: "›"}).Grow(1).MinWidth(0).Height(28).Clip()
		if action.ActionIcon(c, lucide.ExternalLink(), action.ActionIconProps{Title: "在资源管理器中打开", Size: theme.Small}).Clicked() {
			a.setActive(i)
			mygo.Shell.ShowItemInFolder(p.dir)
		}
		if action.ActionIcon(c, lucide.Copy(), action.ActionIconProps{Title: "复制路径", Size: theme.Small}).Clicked() {
			a.setActive(i)
			a.copyPath(p.dir)
		}
	})
}

func (a *App) crumbs(p *Pane, compact bool) []nav.BreadcrumbItem {
	dir := filepath.Clean(p.dir)
	vol := filepath.VolumeName(dir)
	var items []nav.BreadcrumbItem
	root := dir
	rest := dir
	if vol != "" {
		root = vol + `\`
		items = append(items, nav.BreadcrumbItem{
			Title: vol,
			OnClick: func() {
				a.navigate(p, root, true)
			},
		})
		rest = strings.TrimPrefix(strings.TrimPrefix(dir, vol), `\`)
	}
	if rest != "" && rest != "." {
		accum := root
		segs := strings.Split(rest, `\`)
		for i, seg := range segs {
			if seg == "" {
				continue
			}
			accum = filepath.Join(accum, seg)
			path := accum
			item := nav.BreadcrumbItem{Title: seg}
			if i < len(segs)-1 {
				item.OnClick = func() { a.navigate(p, path, true) }
			}
			items = append(items, item)
		}
	}
	if compact && len(items) > 3 {
		items = []nav.BreadcrumbItem{items[0], items[len(items)-1]}
	}
	if len(items) == 0 {
		items = append(items, nav.BreadcrumbItem{Title: dir})
	}
	return items
}

func (a *App) paneCompact(i int) bool {
	if a.mode == 4 || a.inspector {
		return true
	}
	frac := a.paneFraction(i)
	return frac < 0.40 || frac > 0.60
}

func (a *App) paneNarrow(i int) bool {
	frac := a.paneFraction(i)
	return frac < 0.32 || frac > 0.68
}

func (a *App) paneFraction(i int) float64 {
	frac := float64(a.splitPct) / 100
	if i%2 != 0 {
		frac = 1 - frac
	}
	return frac
}

func (a *App) paneListHeader(c *ui.Context, tk *theme.Tokens, i int, p *Pane) {
	p.visible = p.filter()
	compact := a.paneCompact(i)
	narrow := a.paneNarrow(i)
	ui.Row(c).Height(28).PaddingX(10).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(16).Shrink(0)
		a.sortHeader(c, tk, p, "名称", "name", 0, ui.Start)
		if !narrow {
			a.sortHeader(c, tk, p, "大小", "size", sizeWidth(compact), ui.End)
			a.sortHeader(c, tk, p, "修改", "modified", dateWidth(compact), ui.Start)
		}
		if !compact {
			a.sortHeader(c, tk, p, "类型", "type", 62, ui.Start)
		}
	})
}

func sizeWidth(compact bool) float32 {
	if compact {
		return 62
	}
	return 76
}

func dateWidth(compact bool) float32 {
	if compact {
		return 86
	}
	return 132
}

func (a *App) sortHeader(c *ui.Context, tk *theme.Tokens, p *Pane, label, key string, width float32, align ui.Align) {
	b := ui.ButtonBase(c).Height(24).Radius(4).PaddingX(4).Gap(2).AlignItems(ui.Center)
	if width > 0 {
		b.Width(width)
	} else {
		b.Grow(1).Basis(0)
	}
	if align == ui.End {
		b.Justify(ui.End)
	} else {
		b.Justify(ui.Start)
	}
	if b.Hovered() {
		b.Background(tk.FillTertiary)
	}
	arrow := ""
	if p.sortKey == key {
		if p.sortAsc {
			arrow = " ↑"
		} else {
			arrow = " ↓"
		}
	}
	b.Children(func() {
		ui.Text(c, label+arrow).FontSize(11).FontWeight(600).TextColor(tk.TextDescription)
	})
	if b.Clicked() {
		if p.sortKey == key {
			p.sortAsc = !p.sortAsc
		} else {
			p.sortKey = key
			p.sortAsc = true
		}
		p.selected = -1
		p.list = ui.ListState{}
		a.save()
	}
}

func (a *App) paneFileList(c *ui.Context, tk *theme.Tokens, i int, p *Pane) {
	compact := a.paneCompact(i)
	narrow := a.paneNarrow(i)
	if p.loading {
		ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
			ui.Text(c, "正在读取…").FontSize(12).TextColor(tk.TextDescription)
		})
		return
	}
	if p.err != "" {
		ui.Column(c).Grow(1).Center().Padding(16).Gap(6).Children(func() {
			ui.Icon(c, lucide.CircleAlert()).Size(24, 24).TextColor(tk.Error)
			ui.Text(c, "无法读取文件夹").FontSize(12).FontWeight(600).TextColor(tk.Error)
			ui.Text(c, p.err).FontSize(11).TextColor(tk.TextDescription)
		})
		return
	}
	if len(p.visible) == 0 {
		ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
			ui.Icon(c, lucide.FolderOpen()).Size(28, 28).TextColor(tk.TextQuaternary)
			ui.Text(c, "空文件夹").FontSize(12).TextColor(tk.TextDescription)
			if strings.TrimSpace(p.query) != "" {
				ui.Text(c, "没有匹配项").FontSize(11).TextColor(tk.TextQuaternary)
			}
		})
		return
	}

	var pending func()
	p.list.Key = func(idx int) any { return p.visible[idx].Path }
	p.list.Label = func(idx int) string { return p.visible[idx].Name }
	p.list.Selected = &p.selected

	listEl := ui.List(c.Key("files"), &p.list, len(p.visible)).Grow(1).MinHeight(0).Rows(func(row ui.ListRow) {
		e := p.visible[row.Index]
		selected := row.Selected()
		if selected {
			a.setActive(i)
		}
		bg := ui.Transparent
		if selected {
			if i == a.active {
				bg = tk.InfoFillSecondary
			} else {
				bg = tk.BlueFillTertiary
			}
		} else if row.Hovered() {
			bg = tk.FillTertiary
		}
		fg, sub := tk.Text, tk.TextDescription
		iconColor := entryColor(tk, e)
		if selected {
			fg, sub, iconColor = tk.PrimaryForeground, tk.PrimaryForeground, tk.PrimaryForeground
		}
		rowEl := ui.Row(row.Context).Height(34).PaddingX(10).Gap(8).Radius(5).AlignItems(ui.Center).Background(bg)
		rowEl.Children(func() {
			ui.Icon(row.Context, entryIcon(e)).Size(16, 16).Shrink(0).TextColor(iconColor)
			ui.Text(row.Context, e.Name).Grow(1).Basis(0).SingleLine().FontSize(12).TextColor(fg)
			if !narrow {
				ui.Text(row.Context, sizeLabel(e)).Width(sizeWidth(compact)).Shrink(0).TextAlign(ui.End).FontSize(11).TextColor(sub)
				ui.Text(row.Context, dateLabel(e, compact)).Width(dateWidth(compact)).Shrink(0).FontSize(11).TextColor(sub)
			}
			if !compact {
				ui.Text(row.Context, typeLabel(e)).Width(62).Shrink(0).SingleLine().FontSize(11).TextColor(sub)
			}
		})
		rowEl.ContextMenu(func(m *ui.Menu) {
			openLabel := "打开"
			if e.IsDir {
				openLabel = "进入文件夹"
			}
			if m.Item(openLabel).Chosen() {
				path := e.Path
				idx := i
				pending = func() {
					a.setActive(idx)
					a.open(p, path)
				}
			}
			if m.Item("在资源管理器中显示").Chosen() {
				mygo.Shell.ShowItemInFolder(e.Path)
			}
			if m.Item("复制路径").Chosen() {
				a.copyPath(e.Path)
			}
		})
	})

	if listEl.Submitted() {
		a.setActive(i)
		a.openSelected(p)
	}
	if pending != nil {
		pending()
	}

	listEl.Children(func() {
		if p.loading {
			ui.Column(c).Fill().Center().Children(func() {
				ui.Text(c, "正在读取…").FontSize(11).TextColor(tk.TextDescription)
			})
		}
	})
}

func (a *App) statusBar(c *ui.Context, tk *theme.Tokens) {
	p := a.currentPane()
	ui.Row(c).Height(32).PaddingX(14).Gap(10).AlignItems(ui.Center).Background(tk.BgContainer).Children(func() {
		status := fmt.Sprintf("面板 %d", a.active+1)
		if e, ok := p.current(); ok {
			status += " · 已选 " + e.Name
			if !e.IsDir {
				status += " · " + humanSize(e.Size)
			}
		} else {
			status += " · " + p.dir
		}
		if p.loading {
			status = fmt.Sprintf("面板 %d · 正在读取…", a.active+1)
		}
		ui.Text(c, status).Grow(1).Basis(0).SingleLine().FontSize(11).TextColor(tk.TextDescription)
		ui.Text(c, fmt.Sprintf("%d 项 · %d 文件夹 · %s", p.stats.Total, p.stats.Dirs, humanSize(p.stats.Bytes))).FontSize(11).TextColor(tk.TextQuaternary)
		if a.hotkeyErr == "" {
			ui.Text(c, a.hotkeyLabel).FontSize(10).TextColor(tk.TextQuaternary)
		} else {
			ui.Text(c, "热键冲突").FontSize(10).TextColor(tk.Error)
		}
		ui.Text(c, "分栏").FontSize(10).TextColor(tk.TextQuaternary)
		slider := ui.Slider(c, &a.split, 0.25, 0.75).Width(120).Height(20).Shrink(0)
		if slider.Changed() {
			a.splitPct = float32(a.split * 100)
			a.save()
		}
		if action.ActionIcon(c, lucide.PanelRight(), action.ActionIconProps{
			Title:  "检查器",
			Active: a.inspector,
			Size:   theme.Small,
		}).Clicked() {
			a.inspector = !a.inspector
		}
	})
}

func (a *App) inspectorPanel(c *ui.Context, tk *theme.Tokens) {
	p := a.currentPane()
	col := ui.Column(c).Width(300).MinWidth(260).Padding(14).Gap(10).Background(tk.BgContainer).MinHeight(0)
	col.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "检查器").FontSize(14).FontWeight(700).TextColor(tk.Text).Grow(1)
			if action.ActionIcon(c, lucide.X(), action.ActionIconProps{Title: "关闭检查器", Size: theme.Small}).Clicked() {
				a.inspector = false
			}
		})
		separator(c, tk)

		e, ok := p.current()
		if !ok {
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Icon(c, lucide.PanelRight()).Size(30, 30).TextColor(tk.TextQuaternary)
				ui.Text(c, "选中文件或文件夹后").FontSize(12).TextColor(tk.TextDescription)
				ui.Text(c, "这里显示详细信息").FontSize(12).TextColor(tk.TextDescription)
			})
			return
		}

		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, entryIcon(e)).Size(38, 38).TextColor(entryColor(tk, e))
			ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
				ui.Text(c, e.Name).FontSize(14).FontWeight(600).TextColor(tk.Text)
				ui.Text(c, typeLabel(e)).FontSize(11).TextColor(tk.TextDescription)
			})
		})

		metaRow(c, tk, "类型", typeLabelLong(e))
		metaRow(c, tk, "大小", sizeLabel(e))
		metaRow(c, tk, "修改", e.ModTime.Format("2006-01-02 15:04:05"))
		if e.Ext != "" {
			metaRow(c, tk, "扩展名", strings.ToUpper(e.Ext))
		}
		ui.Text(c, "完整路径").FontSize(10).TextColor(tk.TextQuaternary)
		ui.Text(c, e.Path).FontSize(11).TextColor(tk.TextDescription)

		ui.Box(c).Grow(1)
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			if inspectorButton(c, tk, "打开", e.IsDir) {
				a.open(p, e.Path)
			}
			if inspectorButton(c, tk, "定位", false) {
				mygo.Shell.ShowItemInFolder(e.Path)
			}
			if inspectorButton(c, tk, "复制路径", false) {
				a.copyPath(e.Path)
			}
		})
	})
}

func metaRow(c *ui.Context, tk *theme.Tokens, label, value string) {
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, label).Width(58).Shrink(0).FontSize(11).TextColor(tk.TextQuaternary)
		ui.Text(c, value).Grow(1).Basis(0).FontSize(11).TextColor(tk.TextDescription)
	})
}

func inspectorButton(c *ui.Context, tk *theme.Tokens, label string, primary bool) bool {
	b := ui.ButtonBase(c).Height(28).PaddingX(10).Radius(6).Background(tk.FillTertiary)
	if primary {
		b.Background(tk.InfoFillSecondary)
	}
	if b.Hovered() {
		b.Background(tk.FillSecondary)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(tk.Text)
	})
	return b.Clicked()
}

func typeLabelLong(e fsx.Entry) string {
	if e.IsDir {
		return "文件夹"
	}
	return "文件"
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func entryIcon(e fsx.Entry) *ui.SVG {
	if e.IsDir {
		return lucide.Folder()
	}
	switch e.Ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "svg":
		return lucide.Image()
	case "mp4", "mkv", "avi", "mov", "webm", "wmv":
		return lucide.Film()
	case "mp3", "wav", "flac", "aac", "ogg", "m4a":
		return lucide.Music()
	case "txt", "md", "log", "json", "yaml", "yml", "toml", "xml", "csv", "go", "ts", "js", "rs", "cs", "cpp", "h", "py":
		return lucide.FileText()
	default:
		return lucide.File()
	}
}

func entryColor(tk *theme.Tokens, e fsx.Entry) ui.Color {
	if e.IsDir {
		return tk.Info
	}
	switch e.Ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "svg":
		return tk.Info
	case "mp4", "mkv", "avi", "mov", "webm", "wmv":
		return tk.Warning
	case "mp3", "wav", "flac", "aac", "ogg", "m4a":
		return tk.Error
	default:
		return tk.TextDescription
	}
}

func dateLabel(e fsx.Entry, compact bool) string {
	if compact {
		return e.ModTime.Format("01-02 15:04")
	}
	return e.ModTime.Format("2006-01-02 15:04")
}

func sizeLabel(e fsx.Entry) string {
	if e.IsDir {
		return "—"
	}
	return humanSize(e.Size)
}

func typeLabel(e fsx.Entry) string {
	if e.IsDir {
		return "文件夹"
	}
	if e.Ext == "" {
		return "文件"
	}
	return strings.ToUpper(e.Ext)
}

func humanSize(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	return humanSizeU(uint64(n))
}

func humanSizeU(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	units := "KMGTPE"
	if exp < 0 {
		exp = 0
	}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), units[exp])
}

var appTheme = newAppTheme()

func newAppTheme() *theme.Config {
	light := *theme.Default().Tokens(false)
	dark := *theme.Default().Tokens(true)

	light.Primary, light.PrimaryHover, light.PrimaryActive = light.Info, light.InfoHover, light.InfoActive
	dark.Primary, dark.PrimaryHover, dark.PrimaryActive = dark.Info, dark.InfoHover, dark.InfoActive
	light.PrimaryForeground = ui.RGB(255, 255, 255)
	dark.PrimaryForeground = ui.RGB(255, 255, 255)

	cfg := theme.NewConfig(light, dark)
	cfg.Theme(false).InactiveAccent = light.Info
	cfg.Theme(true).InactiveAccent = dark.Info
	return cfg
}

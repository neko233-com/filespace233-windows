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
	theme.Apply(c)
	tk := theme.TokensOf(c)

	root := ui.Column(c).Fill()
	if !c.Vibrancy() {
		root.Background(tk.BgLayout)
	}
	root.Children(func() {
		a.header(c, tk)
		separator(c, tk)
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, tk)
			a.main(c, tk)
		})
		separator(c, tk)
		a.statusBar(c, tk)
	})
}

func separator(c *ui.Context, tk *theme.Tokens) {
	ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary)
}

func (a *App) header(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(58).PaddingX(14).Gap(10).AlignItems(ui.Center).Background(tk.BgContainer).Children(func() {
		ui.Image(c, logoBitmap).Size(30, 30).Radius(8)
		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, "FileSpace233").FontSize(16).FontWeight(700).TextColor(tk.Text)
			ui.Text(c, "文件夹管理器 · 后台常驻").FontSize(11).TextColor(tk.TextDescription)
		})
		ui.Box(c).Grow(1)
		field.SearchBar(c, &a.query, field.SearchBarProps{
			Placeholder: "搜索当前文件夹",
			ShortKey:    "mod+k",
			Size:        theme.Middle,
		}).Width(300)
		hiddenIcon := lucide.EyeOff()
		if a.showHidden {
			hiddenIcon = lucide.Eye()
		}
		if action.ActionIcon(c, hiddenIcon, action.ActionIconProps{
			Title:  "显示隐藏文件",
			Active: a.showHidden,
			Size:   theme.Middle,
		}).Clicked() {
			a.showHidden = !a.showHidden
			a.refresh()
		}
		if action.ActionIcon(c, lucide.RotateCw(), action.ActionIconProps{
			Title: "刷新",
			Size:  theme.Middle,
		}).Clicked() {
			a.refresh()
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
			if samePath(a.dir, f.Path) {
				active = key
			}
			items = append(items, display.ListItem{
				Key:   key,
				Label: f.Label,
				Icon:  favoriteIcon(f.Kind),
			})
		}
		if clicked := display.List(c, display.ListProps{Items: items, Active: active, Compact: true}); clicked != "" {
			a.navigate(strings.TrimPrefix(clicked, "fav|"), true)
		}

		ui.Box(c).Height(8)
		sectionTitle(c, tk, "此电脑")
		if len(a.drives) == 0 {
			ui.Text(c, "没有可用的驱动器").FontSize(12).TextColor(tk.TextDescription).PaddingX(4)
		}
		driveItems := make([]display.ListItem, 0, len(a.drives))
		driveActive := ""
		for _, d := range a.drives {
			key := "drive|" + d.Path
			if samePath(a.dir, d.Path) {
				driveActive = key
			}
			driveItems = append(driveItems, display.ListItem{
				Key:   key,
				Label: d.Name,
				Icon:  lucide.HardDrive(),
			})
		}
		if clicked := display.List(c, display.ListProps{Items: driveItems, Active: driveActive, Compact: true}); clicked != "" {
			a.navigate(strings.TrimPrefix(clicked, "drive|"), true)
		}

		ui.Box(c).Grow(1)
		ui.Text(c, "关闭窗口后仍在托盘常驻").FontSize(10).TextColor(tk.TextQuaternary).PaddingX(4)
	})
	ui.Box(c).Width(1).FillHeight().Background(tk.BorderSecondary)
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

func (a *App) main(c *ui.Context, tk *theme.Tokens) {
	col := ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Children(func() {
		a.toolbar(c, tk)
		if a.err != "" {
			a.errorRow(c, tk)
		}
		a.listHeader(c, tk)
		a.fileList(c, tk)
	})
	col.MinWidth(0)
}

func (a *App) toolbar(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(50).PaddingX(12).Gap(2).AlignItems(ui.Center).Children(func() {
		if action.ActionIcon(c, lucide.ArrowLeft(), action.ActionIconProps{Title: "后退", Size: theme.Middle}).Clicked() {
			a.back()
		}
		if action.ActionIcon(c, lucide.ArrowRight(), action.ActionIconProps{Title: "前进", Size: theme.Middle}).Clicked() {
			a.forwardDir()
		}
		if action.ActionIcon(c, lucide.ArrowUp(), action.ActionIconProps{Title: "上一级（Backspace）", Size: theme.Middle}).Clicked() {
			a.up()
		}
		ui.Box(c).Width(4)
		nav.Breadcrumb(c, nav.BreadcrumbProps{Items: a.crumbs(), Separator: "›"}).Grow(1).MinWidth(0)
		if action.ActionIcon(c, lucide.ExternalLink(), action.ActionIconProps{Title: "在资源管理器中打开", Size: theme.Middle}).Clicked() {
			mygo.Shell.ShowItemInFolder(a.dir)
		}
		if action.ActionIcon(c, lucide.Copy(), action.ActionIconProps{Title: "复制当前路径", Size: theme.Middle}).Clicked() {
			a.copyPath(a.dir)
		}
	})
}

func (a *App) crumbs() []nav.BreadcrumbItem {
	dir := filepath.Clean(a.dir)
	vol := filepath.VolumeName(dir)
	var items []nav.BreadcrumbItem
	root := dir
	rest := dir
	if vol != "" {
		root = vol + `\`
		items = append(items, nav.BreadcrumbItem{
			Title: vol,
			OnClick: func() {
				a.navigate(root, true)
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
				item.OnClick = func() { a.navigate(path, true) }
			}
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		items = append(items, nav.BreadcrumbItem{Title: dir})
	}
	return items
}

func (a *App) errorRow(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Margin(12, 0, 0, 8).Padding(10, 12).Gap(8).Radius(8).AlignItems(ui.Center).Background(tk.ErrorBg).Children(func() {
		ui.Icon(c, lucide.CircleAlert()).Size(16, 16).TextColor(tk.Error)
		ui.Text(c, a.err).Grow(1).SingleLine().FontSize(12).TextColor(tk.ErrorText)
		if ui.ButtonBase(c).Size(22, 22).Radius(6).Center().Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, lucide.X()).Size(14, 14).TextColor(tk.ErrorText)
		}).Clicked() {
			a.err = ""
		}
	})
}

func (a *App) listHeader(c *ui.Context, tk *theme.Tokens) {
	a.visible = a.filter()
	ui.Row(c).Height(30).PaddingX(12).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(18).Shrink(0)
		a.sortHeader(c, tk, "名称", "name", 0, ui.Start)
		a.sortHeader(c, tk, "大小", "size", 92, ui.End)
		a.sortHeader(c, tk, "修改日期", "modified", 148, ui.Start)
		a.sortHeader(c, tk, "类型", "type", 72, ui.Start)
	})
}

func (a *App) sortHeader(c *ui.Context, tk *theme.Tokens, label, key string, width float32, align ui.Align) {
	b := ui.ButtonBase(c).Height(26).Radius(4).PaddingX(4).Gap(2).AlignItems(ui.Center)
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
	if a.sortKey == key {
		if a.sortAsc {
			arrow = " ↑"
		} else {
			arrow = " ↓"
		}
	}
	b.Children(func() {
		ui.Text(c, label+arrow).FontSize(11).FontWeight(600).TextColor(tk.TextDescription)
	})
	if b.Clicked() {
		if a.sortKey == key {
			a.sortAsc = !a.sortAsc
		} else {
			a.sortKey = key
			a.sortAsc = true
		}
		a.save()
	}
}

func (a *App) fileList(c *ui.Context, tk *theme.Tokens) {
	if len(a.visible) == 0 && !a.loading && a.err == "" {
		ui.Column(c).Grow(1).Center().Gap(10).Children(func() {
			ui.Icon(c, lucide.FolderOpen()).Size(34, 34).TextColor(tk.TextQuaternary)
			ui.Text(c, "这个文件夹是空的").FontSize(14).TextColor(tk.TextDescription)
			if strings.TrimSpace(a.query) != "" {
				ui.Text(c, "没有匹配“"+a.query+"”的项目").FontSize(12).TextColor(tk.TextQuaternary)
			}
		})
		return
	}

	var pending func()
	a.list.Key = func(i int) any { return a.visible[i].Path }
	a.list.Label = func(i int) string { return a.visible[i].Name }
	a.list.Selected = &a.selected

	listEl := ui.List(c, &a.list, len(a.visible)).Grow(1).Rows(func(row ui.ListRow) {
		e := a.visible[row.Index]
		bg := ui.Transparent
		if row.Selected() {
			bg = tk.PrimaryFillTertiary
		} else if row.Hovered() {
			bg = tk.FillTertiary
		}
		rowEl := ui.Row(row.Context).Height(36).PaddingX(12).Gap(10).Radius(6).AlignItems(ui.Center).Background(bg)
		rowEl.Children(func() {
			ui.Icon(row.Context, entryIcon(e)).Size(18, 18).Shrink(0).TextColor(entryColor(tk, e))
			ui.Text(row.Context, e.Name).Grow(1).Basis(0).SingleLine().FontSize(13).TextColor(tk.Text)
			ui.Text(row.Context, sizeLabel(e)).Width(92).Shrink(0).TextAlign(ui.End).FontSize(12).TextColor(tk.TextDescription)
			ui.Text(row.Context, e.ModTime.Format("2006-01-02 15:04")).Width(148).Shrink(0).FontSize(12).TextColor(tk.TextDescription)
			ui.Text(row.Context, typeLabel(e)).Width(72).Shrink(0).SingleLine().FontSize(12).TextColor(tk.TextDescription)
		})
		rowEl.ContextMenu(func(m *ui.Menu) {
			openLabel := "打开"
			if e.IsDir {
				openLabel = "进入文件夹"
			}
			if m.Item(openLabel).Chosen() {
				path := e.Path
				pending = func() { a.open(path) }
			}
			if m.Item("在资源管理器中显示").Chosen() {
				mygo.Shell.ShowItemInFolder(e.Path)
			}
			if m.Item("复制路径").Chosen() {
				a.copyPath(e.Path)
			}
		})
	})

	// A double click, or Enter, on the selected row opens it.
	if listEl.Submitted() {
		a.openSelected()
	}
	if pending != nil {
		pending()
	}

	listEl.Children(func() {
		if a.loading {
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Text(c, "正在读取…").FontSize(13).TextColor(tk.TextDescription)
			})
		}
	})
}

func (a *App) statusBar(c *ui.Context, tk *theme.Tokens) {
	ui.Row(c).Height(30).PaddingX(14).Gap(10).AlignItems(ui.Center).Background(tk.BgContainer).Children(func() {
		status := ""
		if e, ok := a.current(); ok {
			status = "已选：" + e.Name
			if !e.IsDir {
				status += " · " + humanSize(e.Size)
			}
		}
		if a.loading {
			status = "正在读取…"
		}
		if status == "" {
			status = a.dir
		}
		ui.Text(c, status).Grow(1).Basis(0).SingleLine().FontSize(11).TextColor(tk.TextDescription)
		ui.Text(c, fmt.Sprintf("%d 项 · %d 个文件夹 · %d 个文件 · %s", a.stats.Total, a.stats.Dirs, a.stats.Files, humanSize(a.stats.Bytes))).FontSize(11).TextColor(tk.TextQuaternary)
		ui.Text(c, "windows/amd64").FontSize(10).TextColor(tk.TextQuaternary)
	})
}

func (a *App) filter() []fsx.Entry {
	q := strings.ToLower(strings.TrimSpace(a.query))
	out := make([]fsx.Entry, 0, len(a.entries))
	for _, e := range a.entries {
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	fsx.Sort(out, a.sortKey, a.sortAsc)
	return out
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
		return tk.Primary
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
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

package main

import (
	"fmt"
	"strings"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

func (a *App) syncSelect(mode string) {
	if a.paneCount() < 2 {
		return
	}
	other := -1
	for _, i := range a.otherPaneIndexes() {
		other = i
		break
	}
	if other < 0 {
		return
	}
	left, right := a.currentPane(), a.panes[other]
	left.refreshVisible()
	right.refreshVisible()
	nameKey := func(e fsx.Entry) string {
		return strings.ToLower(e.Name)
	}
	leftMap := make(map[string]fsx.Entry, len(left.visible))
	for _, e := range left.visible {
		leftMap[nameKey(e)] = e
	}
	rightMap := make(map[string]fsx.Entry, len(right.visible))
	for _, e := range right.visible {
		rightMap[nameKey(e)] = e
	}

	left.selection.Clear()
	right.selection.Clear()
	left.selected, right.selected = -1, -1
	add := func(p *Pane, e fsx.Entry) {
		p.selection.Add(e.Path)
		if p.selected < 0 {
			for i, x := range p.visible {
				if samePath(x.Path, e.Path) {
					p.selected = i
					break
				}
			}
		}
	}
	switch mode {
	case "both":
		for _, e := range left.visible {
			if r, ok := rightMap[nameKey(e)]; ok && r.IsDir == e.IsDir {
				add(left, e)
				add(right, r)
			}
		}
	case "left":
		for _, e := range left.visible {
			if r, ok := rightMap[nameKey(e)]; !ok || r.IsDir != e.IsDir {
				add(left, e)
			}
		}
	case "right":
		for _, e := range right.visible {
			if l, ok := leftMap[nameKey(e)]; !ok || l.IsDir != e.IsDir {
				add(right, e)
			}
		}
	case "size":
		for _, e := range left.visible {
			if e.IsDir {
				continue
			}
			if r, ok := rightMap[nameKey(e)]; ok && !r.IsDir && r.Size != e.Size {
				add(left, e)
				add(right, r)
			}
		}
	default:
		return
	}
	label := map[string]string{
		"both": "两边同名", "left": "仅本面板", "right": "仅对面板", "size": "大小不同",
	}[mode]
	a.flash(fmt.Sprintf("同步选择：%s · 本面板 %d 项，面板 %d %d 项", label,
		left.selection.Len(), other+1, right.selection.Len()), false)
}

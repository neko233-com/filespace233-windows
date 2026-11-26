package main

import (
	"fmt"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (a *App) renderDialogs(c *ui.Context, tk *theme.Tokens) {
	if a.confirmOpen {
		switch ui.AlertDialog(c, &a.confirmOpen, a.confirmTitle, a.confirmBody, "取消", "移到回收站") {
		case 1:
			action := a.confirmAction
			a.confirmAction = nil
			if action != nil {
				action()
			}
		}
	}
	if a.batchOpen {
		ui.DialogBase(c, &a.batchOpen, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.58))
			panel.Width(520).Padding(20).Radius(14).Border(1, tk.Border).
				Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.94), Clarity: 0.10}).
				Gap(10)

			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "批量重命名").FontSize(15).FontWeight(650).TextColor(tk.Text).Grow(1)
				ui.Text(c, fmt.Sprintf("%d 项", len(a.batchEntries))).FontSize(9).TextColor(tk.TextQuaternary)
			})
			ui.Text(c, "模板支持 {name} 原名（不含扩展名）、{full} 全名、{ext} 扩展名、{n} 序号、{nn} 补零序号、{date} 日期。").
				FontSize(10).TextColor(tk.TextDescription)
			input := ui.TextInput(c, &a.batchPattern).Height(32).FontSize(12).
				PaddingX(10).Radius(8).Background(ui.RGBA(0, 0, 0, 0.04)).Border(1, tk.Border).
				FocusRing(false).AutoFocus()
			if input.Changed() {
				a.batchErr = ""
			}
			if input.Submitted() {
				a.submitBatchRename()
			}

			plan, err := batchRenamePlan(a.batchEntries, a.batchPattern)
			if err != nil || a.batchErr != "" {
				msg := a.batchErr
				if msg == "" && err != nil {
					msg = err.Error()
				}
				if msg == "" {
					msg = "模板无效"
				}
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, lucide.CircleAlert()).Size(12, 12).TextColor(tk.ErrorText)
					ui.Text(c, msg).FontSize(10).TextColor(tk.ErrorText)
				})
			} else {
				ui.Text(c, "预览").FontSize(9).FontWeight(600).LetterSpacing(0.8).TextColor(tk.TextQuaternary)
				for i, p := range plan {
					if i >= 8 {
						ui.Text(c, fmt.Sprintf("… 另外 %d 项", len(plan)-8)).FontSize(9).
							TextColor(tk.TextQuaternary)
						break
					}
					ui.Row(c).FillWidth().Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, p.Old).FontSize(10).TextColor(tk.TextDescription).Width(190).SingleLine()
						ui.Text(c, "→").FontSize(10).TextColor(tk.TextQuaternary)
						ui.Text(c, p.New).FontSize(10).TextColor(tk.Text).Grow(1).SingleLine()
					})
				}
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if a.textButton(c, tk, "取消", false, nil).Clicked() {
					a.batchOpen = false
					a.batchErr = ""
				}
				if a.textButton(c, tk, "重命名", true, nil).Clicked() {
					a.submitBatchRename()
				}
			})
		})
	}
	if a.promptOpen {
		ui.DialogBase(c, &a.promptOpen, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.58))
			panel.Width(430).Padding(20).Radius(14).Border(1, tk.Border).
				Material(glass.Glass{Tint: ui.RGBA(255, 255, 255, 0.94), Clarity: 0.10}).
				Gap(10)

			ui.Text(c, a.promptTitle).FontSize(15).FontWeight(650).TextColor(tk.Text)
			ui.Text(c, a.promptHint).FontSize(10).TextColor(tk.TextDescription)
			input := ui.TextInput(c, &a.promptValue).Height(32).FontSize(12).
				PaddingX(10).Radius(8).Background(ui.RGBA(0, 0, 0, 0.04)).Border(1, tk.Border).
				FocusRing(false).AutoFocus()
			if input.Submitted() {
				a.submitPrompt()
			}
			if a.promptErr != "" {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, lucide.CircleAlert()).Size(12, 12).TextColor(tk.ErrorText)
					ui.Text(c, a.promptErr).FontSize(10).TextColor(tk.ErrorText)
				})
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if a.textButton(c, tk, "取消", false, nil).Clicked() {
					a.promptOpen = false
					a.promptAction = nil
					a.promptErr = ""
				}
				if a.textButton(c, tk, a.promptConfirm, true, nil).Clicked() {
					a.submitPrompt()
				}
			})
		})
	}
}

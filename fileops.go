package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

func (a *App) flash(msg string, isErr bool) {
	a.statusMsg = msg
	a.statusErr = isErr
}

func (a *App) copySelection() {
	p := a.currentPane()
	paths := p.chosenPaths()
	if len(paths) == 0 {
		a.flash("没有选中项", true)
		return
	}
	a.clipMode = "copy"
	a.clipPaths = append([]string(nil), paths...)
	if data, err := transfer.FileData(paths...); err == nil {
		_ = mygo.Clipboard.Write(data)
	}
	a.flash(fmt.Sprintf("已复制 %d 项，Ctrl+V 粘贴", len(paths)), false)
}

func (a *App) cutSelection() {
	p := a.currentPane()
	paths := p.chosenPaths()
	if len(paths) == 0 {
		a.flash("没有选中项", true)
		return
	}
	a.clipMode = "cut"
	a.clipPaths = append([]string(nil), paths...)
	a.flash(fmt.Sprintf("已剪切 %d 项，Ctrl+V 移动", len(paths)), false)
}

func (a *App) paste() {
	target := a.currentPane()
	if target.dir == "" {
		return
	}
	mode := a.clipMode
	paths := append([]string(nil), a.clipPaths...)
	if len(paths) == 0 {
		files, ok := a.loadClipboardFiles()
		if !ok {
			a.flash("剪贴板没有文件", true)
			return
		}
		paths = files
		mode = "copy"
	}
	if mode == "" {
		mode = "copy"
	}
	move := mode == "cut"
	label := "复制"
	if move {
		label = "移动"
	}
	destDir := target.dir
	affected := append([]string{destDir}, parentDirs(paths)...)
	a.transferPaths(label, paths, destDir, move, affected, func(success, failed int) {
		if move && failed == 0 && success > 0 {
			a.clipMode = ""
			a.clipPaths = nil
		}
	})
}

func (a *App) askDelete() {
	entries := a.currentPane().chosenEntries()
	if len(entries) == 0 {
		a.flash("没有选中项", true)
		return
	}
	paths := make([]string, 0, len(entries))
	var names []string
	for i, e := range entries {
		paths = append(paths, e.Path)
		if i < 4 {
			names = append(names, e.Name)
		}
	}
	body := strings.Join(names, "\n")
	if len(entries) > len(names) {
		body += fmt.Sprintf("\n… 另外 %d 项", len(entries)-len(names))
	}
	a.confirmTitle = fmt.Sprintf("将 %d 个项目移到回收站？", len(entries))
	a.confirmBody = body + "\n\n可从回收站还原。"
	a.confirmOpen = true
	a.confirmAction = func() {
		a.deletePaths(paths)
	}
}

func (a *App) deletePaths(paths []string) {
	affected := parentDirs(paths)
	a.runBatch("移到回收站", paths, func(src string) (string, error) {
		return "", mygo.Shell.TrashItem(src)
	}, affected, nil)
}

func (a *App) askRename() {
	entries := a.currentPane().chosenEntries()
	if len(entries) != 1 {
		a.flash("重命名一次只能选择一个项目", true)
		return
	}
	src := entries[0].Path
	a.openPrompt("重命名", filepath.Base(src), "输入新的名称；不会覆盖已有文件", "重命名", func(value string) (string, error) {
		if err := validItemName(value); err != nil {
			return "", err
		}
		dst := filepath.Join(filepath.Dir(src), value)
		if samePath(dst, src) {
			return src, nil
		}
		if fsx.Exists(dst) {
			return "", errors.New("同名项目已存在")
		}
		if err := os.Rename(src, dst); err != nil {
			return "", err
		}
		return dst, nil
	})
}

func (a *App) askNewFolder() {
	dir := a.currentPane().dir
	if dir == "" {
		return
	}
	def := filepath.Base(fsx.UniqueChild(dir, "新建文件夹"))
	a.openPrompt("新建文件夹", def, "在当前面板中创建文件夹", "创建", func(value string) (string, error) {
		if err := validItemName(value); err != nil {
			return "", err
		}
		dst := filepath.Join(dir, value)
		if fsx.Exists(dst) {
			return "", errors.New("同名项目已存在")
		}
		if err := os.Mkdir(dst, 0o755); err != nil {
			return "", err
		}
		return dst, nil
	})
}

func (a *App) askNewFile() {
	dir := a.currentPane().dir
	if dir == "" {
		return
	}
	def := filepath.Base(fsx.UniqueChild(dir, "新建文本文件.txt"))
	a.openPrompt("新建文件", def, "创建一个 UTF-8 文本文件", "创建", func(value string) (string, error) {
		if err := validItemName(value); err != nil {
			return "", err
		}
		dst := filepath.Join(dir, value)
		if fsx.Exists(dst) {
			return "", errors.New("同名项目已存在")
		}
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return "", err
		}
		return dst, f.Close()
	})
}

func (a *App) askDuplicate() {
	entries := a.currentPane().chosenEntries()
	if len(entries) == 0 {
		a.flash("没有选中项", true)
		return
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	affected := parentDirs(paths)
	a.runBatch("创建副本", paths, func(src string) (string, error) {
		dst := fsx.UniqueChild(filepath.Dir(src), fsx.DuplicateName(src))
		return dst, fsx.CopyPath(src, dst)
	}, affected, nil)
}

func (a *App) copyToPane(target int, move bool) {
	if target < 0 || target >= a.paneCount() || target == a.active {
		a.flash("目标面板无效", true)
		return
	}
	src := a.currentPane()
	paths := src.chosenPaths()
	if len(paths) == 0 {
		a.flash("没有选中项", true)
		return
	}
	dst := a.panes[target]
	if dst.dir == "" {
		return
	}
	label := "复制到面板 " + fmt.Sprintf("%d", target+1)
	if move {
		label = "移动到面板 " + fmt.Sprintf("%d", target+1)
	}
	affected := append([]string{dst.dir}, parentDirs(paths)...)
	a.transferPaths(label, paths, dst.dir, move, affected, nil)
}

func (a *App) dropPaths(paths []string, destDir string) {
	if len(paths) == 0 || destDir == "" {
		return
	}
	move := true
	for _, src := range paths {
		if !sameVolume(src, destDir) {
			move = false
			break
		}
	}
	label := "拖放移动"
	if !move {
		label = "拖放复制"
	}
	affected := append([]string{destDir}, parentDirs(paths)...)
	a.transferPaths(label, paths, destDir, move, affected, nil)
}

func (a *App) importPaths(paths []string, destDir string) {
	if len(paths) == 0 || destDir == "" {
		return
	}
	affected := append([]string{destDir}, parentDirs(paths)...)
	a.transferPaths("导入", paths, destDir, false, affected, nil)
}

func (a *App) transferPaths(label string, paths []string, destDir string, move bool, affected []string, onDone func(success, failed int)) {
	fn := func(src string) (string, error) {
		dst, err := transferDestination(src, destDir, !move)
		if err != nil {
			return "", err
		}
		if move {
			return dst, fsx.MovePath(src, dst)
		}
		return dst, fsx.CopyPath(src, dst)
	}
	a.runBatch(label, paths, fn, affected, onDone)
}

func transferDestination(src, destDir string, copyOp bool) (string, error) {
	base := filepath.Base(src)
	if samePath(filepath.Dir(src), destDir) {
		if !copyOp {
			return "", errors.New("项目已在该文件夹中")
		}
		return fsx.UniqueChild(destDir, fsx.DuplicateName(src)), nil
	}
	info, err := os.Stat(src)
	if err == nil && info.IsDir() && pathWithin(destDir, src) {
		return "", errors.New("不能把文件夹放进它自己的子目录")
	}
	dst := filepath.Join(destDir, base)
	if fsx.Exists(dst) {
		return "", errors.New("目标已存在同名项目")
	}
	return dst, nil
}

func (a *App) runBatch(label string, paths []string, fn func(string) (string, error), affected []string, onDone func(success, failed int)) {
	if len(paths) == 0 {
		return
	}
	if a.opBusy {
		a.flash("已有文件操作正在进行", true)
		return
	}
	a.opBusy = true
	a.opLabel = label
	a.opCurrent = filepath.Base(paths[0])
	a.opDone = 0
	a.opTotal = len(paths)
	a.opFailed = 0
	a.opErr = ""
	a.statusMsg = ""
	a.statusErr = false
	go func() {
		success, failed := 0, 0
		firstErr := error(nil)
		firstDest := ""
		for i, src := range paths {
			dst, err := fn(src)
			if err != nil {
				failed++
				if firstErr == nil {
					firstErr = fmt.Errorf("%s：%w", filepath.Base(src), err)
				}
			} else {
				success++
				if firstDest == "" {
					firstDest = dst
				}
			}
			done := i + 1
			current := filepath.Base(src)
			a.win.Update(func() {
				a.opDone = done
				a.opCurrent = current
			})
		}
		a.win.Update(func() {
			a.opBusy = false
			a.opDone = a.opTotal
			a.opErr = ""
			a.opFailed = failed
			if failed > 0 {
				a.opErr = firstErr.Error()
				a.flash(fmt.Sprintf("%s：%d 成功，%d 失败 · %s", label, success, failed, firstErr), true)
			} else {
				a.flash(fmt.Sprintf("%s完成 · %d 项", label, success), false)
			}
			for _, dir := range affected {
				a.invalidateCache(dir)
			}
			a.refreshGroup(affected, firstDest)
			if onDone != nil {
				onDone(success, failed)
			}
		})
	}()
}

func (a *App) openPrompt(title, value, hint, confirm string, action func(string) (string, error)) {
	a.promptOpen = true
	a.promptTitle = title
	a.promptValue = value
	a.promptHint = hint
	a.promptConfirm = confirm
	a.promptErr = ""
	a.promptAction = action
}

func (a *App) submitPrompt() {
	if a.promptAction == nil {
		return
	}
	value := strings.TrimSpace(a.promptValue)
	reveal, err := a.promptAction(value)
	if err != nil {
		a.promptErr = err.Error()
		return
	}
	a.promptOpen = false
	a.promptErr = ""
	a.promptAction = nil
	if reveal != "" {
		a.refreshAndReveal(reveal)
	}
}

func validItemName(name string) error {
	switch {
	case name == "" || name == "." || name == "..":
		return errors.New("名称不能为空")
	case strings.ContainsAny(name, `\/:*?"<>|`):
		return errors.New(`名称不能包含 \ / : * ? " < > |`)
	case len([]rune(name)) > 255:
		return errors.New("名称过长")
	}
	return nil
}

func parentDirs(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Dir(p))
	}
	return out
}

func sameVolume(a, b string) bool {
	va := strings.ToLower(filepath.VolumeName(filepath.Clean(a)))
	vb := strings.ToLower(filepath.VolumeName(filepath.Clean(b)))
	return va != "" && va == vb
}

func pathWithin(candidate, parent string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

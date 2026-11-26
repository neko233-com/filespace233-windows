package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

type renamePlan struct {
	Old string
	New string
	Src string
	Dst string
}

func (a *App) askBatchRename() {
	entries := a.currentPane().chosenEntries()
	if len(entries) == 0 {
		a.flash("没有选中项", true)
		return
	}
	a.batchOpen = true
	a.batchEntries = entries
	a.batchPattern = "{name}"
	a.batchErr = ""
}

func batchRenamePlan(entries []fsx.Entry, pattern string) ([]renamePlan, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, errors.New("模板不能为空")
	}
	now := time.Now()
	width := len(strconv.Itoa(len(entries)))
	if width < 2 {
		width = 2
	}
	out := make([]renamePlan, 0, len(entries))
	seen := map[string]bool{}
	for i, e := range entries {
		ext := filepath.Ext(e.Name)
		stem := strings.TrimSuffix(e.Name, ext)
		if e.IsDir {
			stem = e.Name
			ext = ""
		}
		number := strconv.Itoa(i + 1)
		for len(number) < width {
			number = "0" + number
		}
		name := pattern
		name = strings.ReplaceAll(name, "{name}", stem)
		name = strings.ReplaceAll(name, "{full}", e.Name)
		name = strings.ReplaceAll(name, "{ext}", strings.TrimPrefix(ext, "."))
		name = strings.ReplaceAll(name, "{n}", strconv.Itoa(i+1))
		name = strings.ReplaceAll(name, "{nn}", number)
		name = strings.ReplaceAll(name, "{date}", now.Format("20060102"))
		if strings.Contains(pattern, "{full}") {
			ext = ""
		}
		if ext != "" && !strings.Contains(pattern, "{ext}") {
			name += ext
		}
		if err := validItemName(name); err != nil {
			return nil, fmt.Errorf("第 %d 项：%w", i+1, err)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return nil, fmt.Errorf("生成的新名称重复：%s", name)
		}
		seen[key] = true
		out = append(out, renamePlan{
			Old: e.Name,
			New: name,
			Src: e.Path,
			Dst: filepath.Join(filepath.Dir(e.Path), name),
		})
	}
	for i, p := range out {
		if samePath(p.Dst, p.Src) {
			continue
		}
		if fsx.Exists(p.Dst) {
			return nil, fmt.Errorf("第 %d 项目标已存在：%s", i+1, p.New)
		}
	}
	return out, nil
}

func (a *App) submitBatchRename() {
	plan, err := batchRenamePlan(a.batchEntries, a.batchPattern)
	if err != nil {
		a.batchErr = err.Error()
		return
	}
	paths := make([]string, 0, len(plan))
	bySrc := make(map[string]string, len(plan))
	for _, p := range plan {
		paths = append(paths, p.Src)
		bySrc[p.Src] = p.Dst
	}
	a.batchOpen = false
	a.batchErr = ""
	affected := parentDirs(paths)
	a.runBatch("批量重命名", paths, func(src string) (string, error) {
		dst := bySrc[src]
		if err := os.Rename(src, dst); err != nil {
			return "", err
		}
		return dst, nil
	}, affected, nil)
}

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

var imagePreviewExt = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true,
	"webp": true, "bmp": true, "ico": true,
}

var textPreviewExt = map[string]bool{
	"txt": true, "md": true, "markdown": true, "log": true, "json": true,
	"yaml": true, "yml": true, "toml": true, "ini": true, "xml": true,
	"csv": true, "tsv": true, "go": true, "ts": true, "tsx": true, "js": true,
	"jsx": true, "rs": true, "cs": true, "cpp": true, "cc": true, "c": true,
	"h": true, "hpp": true, "py": true, "java": true, "kt": true, "swift": true,
	"sh": true, "ps1": true, "sql": true, "html": true, "htm": true, "css": true,
	"scss": true, "vue": true, "mod": true, "sum": true, "gitignore": true,
}

func (a *App) syncInspector(p *Pane) {
	entries := p.chosenEntries()
	key := ""
	var only fsx.Entry
	if len(entries) == 1 {
		only = entries[0]
		key = fmt.Sprintf("%s|%d|%d", only.Path, only.Size, only.ModTime.UnixNano())
	} else if len(entries) > 1 {
		key = fmt.Sprintf("multi:%d", len(entries))
	}
	if key == a.inspectKey {
		return
	}
	a.inspectKey = key
	a.resetPreview()
	a.hashPath = ""
	a.hashValue = ""
	a.hashErr = ""
	a.hashBusy = false
	if len(entries) == 1 && !only.IsDir {
		a.startPreview(only)
	}
}

func (a *App) resetPreview() {
	if a.previewBitmap != nil {
		a.previewBitmap.ReleasePixels()
	}
	a.previewPath = ""
	a.previewBitmap = nil
	a.previewText = ""
	a.previewDim = ""
	a.previewErr = ""
	a.previewLoad = false
}

func (a *App) startPreview(e fsx.Entry) {
	a.previewPath = e.Path
	ext := strings.ToLower(e.Ext)
	switch {
	case imagePreviewExt[ext]:
		a.previewLoad = true
		go func() {
			if e.Size > 32<<20 {
				a.win.Update(func() {
					if a.previewPath != e.Path {
						return
					}
					a.previewLoad = false
					a.previewErr = "图像过大，不生成预览（上限 32 MB）"
				})
				return
			}
			data, err := os.ReadFile(e.Path)
			if err != nil {
				a.win.Update(func() {
					if a.previewPath != e.Path {
						return
					}
					a.previewLoad = false
					a.previewErr = err.Error()
				})
				return
			}
			bmp, err := ui.DecodeBitmapScaled(data, 640)
			if err != nil {
				a.win.Update(func() {
					if a.previewPath != e.Path {
						return
					}
					a.previewLoad = false
					a.previewErr = err.Error()
				})
				return
			}
			w, h := ui.ImageSize(bmp)
			dim := fmt.Sprintf("%.0f × %.0f", w, h)
			a.win.Update(func() {
				if a.previewPath != e.Path {
					bmp.ReleasePixels()
					return
				}
				if a.previewBitmap != nil && a.previewBitmap != bmp {
					a.previewBitmap.ReleasePixels()
				}
				a.previewLoad = false
				a.previewErr = ""
				a.previewBitmap = bmp
				a.previewDim = dim
			})
		}()
	case textPreviewExt[ext] || ext == "":
		a.previewLoad = true
		go func() {
			f, err := os.Open(e.Path)
			if err != nil {
				a.win.Update(func() {
					if a.previewPath != e.Path {
						return
					}
					a.previewLoad = false
					a.previewErr = err.Error()
				})
				return
			}
			defer f.Close()
			data, err := io.ReadAll(io.LimitReader(f, 16<<10))
			if err != nil {
				a.win.Update(func() {
					if a.previewPath != e.Path {
						return
					}
					a.previewLoad = false
					a.previewErr = err.Error()
				})
				return
			}
			text, binary := previewText(data)
			a.win.Update(func() {
				if a.previewPath != e.Path {
					return
				}
				a.previewLoad = false
				a.previewErr = ""
				if binary {
					a.previewText = "二进制文件，不显示文本预览"
					a.previewDim = ""
					return
				}
				a.previewText = text
				a.previewDim = humanSize(e.Size) + " · 文本预览（最多 16 KB）"
			})
		}()
	}
}

func previewText(data []byte) (string, bool) {
	if strings.ContainsRune(string(data), 0) {
		return "", true
	}
	s := strings.ToValidUTF8(string(data), "�")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	const maxChars = 4200
	if len([]rune(s)) > maxChars {
		s = string([]rune(s)[:maxChars]) + "\n\n…（预览已截断）"
	}
	return s, false
}

func previewKind(e fsx.Entry) string {
	if e.IsDir {
		return "folder"
	}
	ext := strings.ToLower(strings.TrimPrefix(e.Ext, "."))
	switch {
	case imagePreviewExt[ext]:
		return "image"
	case textPreviewExt[ext] || ext == "":
		return "text"
	default:
		return "file"
	}
}

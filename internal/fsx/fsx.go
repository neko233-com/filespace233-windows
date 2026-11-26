// Package fsx lists folders without any UI dependency.
package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry is one child of a folder.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Ext     string
}

// Stats summarizes a folder's direct children.
type Stats struct {
	Files int
	Dirs  int
	Bytes int64
	Total int
}

// Drive is a Windows volume that is currently reachable.
type Drive struct {
	Name string
	Path string
}

// List reads dir, skipping hidden names when hidden is false.
func List(dir string, hidden bool) ([]Entry, Stats, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, Stats{}, err
	}
	out := make([]Entry, 0, len(items))
	var st Stats
	for _, item := range items {
		name := item.Name()
		if !hidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		entry := Entry{
			Name:    name,
			Path:    filepath.Join(dir, name),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime(),
		}
		if entry.IsDir {
			st.Dirs++
		} else {
			entry.Size = info.Size()
			entry.Ext = strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
			st.Files++
			st.Bytes += entry.Size
		}
		out = append(out, entry)
	}
	st.Total = len(out)
	sortEntries(out, "name", true)
	return out, st, nil
}

// Drives lists the reachable Windows drive letters.
func Drives() []Drive {
	var out []Drive
	for c := 'A'; c <= 'Z'; c++ {
		p := string(c) + ":\\"
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			out = append(out, Drive{Name: string(c) + ":", Path: p})
		}
	}
	return out
}

// Parent returns the parent folder, keeping drive roots in place.
func Parent(dir string) string {
	clean := filepath.Clean(dir)
	parent := filepath.Dir(clean)
	if parent == "." || parent == "" {
		return clean
	}
	return parent
}

// Sort orders entries with folders first, then by key ("name", "size" or
// "modified").
func Sort(entries []Entry, key string, asc bool) {
	sorted := append([]Entry(nil), entries...)
	sortEntries(sorted, key, asc)
	copy(entries, sorted)
}

func sortEntries(entries []Entry, key string, asc bool) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		cmp := 0
		switch key {
		case "size":
			cmp = compareInt64(a.Size, b.Size)
		case "modified":
			cmp = a.ModTime.Compare(b.ModTime)
		case "type":
			cmp = strings.Compare(typeKey(a), typeKey(b))
		}
		if cmp == 0 {
			cmp = naturalCompare(a.Name, b.Name)
		}
		if !asc {
			cmp = -cmp
		}
		return cmp < 0
	})
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// naturalCompare compares names so that "file2" sorts before "file10".
func naturalCompare(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for len(a) > 0 && len(b) > 0 {
		if isDigit(a[0]) && isDigit(b[0]) {
			na, ra := number(a)
			nb, rb := number(b)
			if na != nb {
				return compareInt64(na, nb)
			}
			a, b = ra, rb
			continue
		}
		if a[0] != b[0] {
			if a[0] < b[0] {
				return -1
			}
			return 1
		}
		a, b = a[1:], b[1:]
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	default:
		return 0
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func typeKey(e Entry) string {
	if e.IsDir {
		return "folder"
	}
	if e.Ext == "" {
		return "file"
	}
	return e.Ext
}

func number(s string) (int64, string) {
	var n int64
	i := 0
	for i < len(s) && isDigit(s[i]) {
		n = n*10 + int64(s[i]-'0')
		i++
	}
	return n, s[i:]
}

// NormalizeDir makes a user-supplied path absolute and clean.
func NormalizeDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", abs)
	}
	return abs, nil
}

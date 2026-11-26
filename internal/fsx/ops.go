package fsx

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyPath copies a file, a directory tree or a symbolic link to dst.
// dst must not exist: a file manager asks before overwriting, and refusing
// here keeps a mistaken paste from destroying another file.
func CopyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
		return nil
	}
	if info.IsDir() {
		return copyDir(src, dst, info)
	}
	if !info.Mode().IsRegular() {
		return &os.PathError{Op: "copy", Path: src, Err: errors.New("not a regular file")}
	}
	return copyFile(src, dst, info)
}

func copyDir(src, dst string, info os.FileInfo) error {
	if err := os.Mkdir(dst, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	for _, entry := range entries {
		if err := CopyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			_ = os.RemoveAll(dst)
			return err
		}
	}
	_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	return nil
}

func copyFile(src, dst string, info os.FileInfo) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	buf := make([]byte, 1<<20)
	_, err = io.CopyBuffer(out, in, buf)
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	return nil
}

// MovePath moves src to dst, falling back to copy and remove when the two
// paths are on different volumes, as Windows refuses rename across volumes.
func MovePath(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyPath(src, dst); err != nil {
		return err
	}
	if err := os.RemoveAll(src); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return nil
}

// Exists reports whether path exists, without following a final symlink.
func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// UniqueChild returns a path under dir for name, adding " (2)", " (3)" and
// so on when the name is taken. It never overwrites an existing item.
func UniqueChild(dir, name string) string {
	path := filepath.Join(dir, name)
	if !Exists(path) {
		return path
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; i < 10000; i++ {
		candidate := filepath.Join(dir, stem+" ("+itoa(i)+")"+ext)
		if !Exists(candidate) {
			return candidate
		}
	}
	return filepath.Join(dir, stem+" (copy)"+ext)
}

// DuplicateName returns a name for a copy of path in the same folder, such
// as "report - 副本.pdf". The caller passes it to UniqueChild.
func DuplicateName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext == base {
		ext = ""
	}
	return strings.TrimSuffix(base, ext) + " - 副本" + ext
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

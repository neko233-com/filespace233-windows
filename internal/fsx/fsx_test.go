package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAndStats(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, st, err := List(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2: %+v", len(entries), entries)
	}
	if !entries[0].IsDir || entries[0].Name != "sub" {
		t.Fatalf("folders sort first: %+v", entries)
	}
	if st.Files != 1 || st.Dirs != 1 || st.Bytes != 5 {
		t.Fatalf("stats = %+v", st)
	}

	entries, _, err = List(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("hidden entries = %d, want 3", len(entries))
	}
}

func TestNaturalSort(t *testing.T) {
	names := []string{"file10.txt", "file2.txt", "File1.txt"}
	entries := make([]Entry, len(names))
	for i, n := range names {
		entries[i] = Entry{Name: n}
	}
	Sort(entries, "name", true)
	want := []string{"File1.txt", "file2.txt", "file10.txt"}
	for i := range want {
		if entries[i].Name != want[i] {
			t.Fatalf("sorted = %v, want %v", entries, want)
		}
	}
}

func TestParentAtRoot(t *testing.T) {
	root := "C:\\"
	if got := Parent(root); got != root {
		t.Fatalf("Parent(%q) = %q", root, got)
	}
}

func TestCopyPathTreeAndRefuseOverwrite(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(root, "dst")
	if err := CopyPath(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "nested", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("copied data = %q", data)
	}
	if err := CopyPath(src, dst); err == nil {
		t.Fatal("copy over an existing folder succeeded")
	}
}

func TestUniqueChildAndDuplicateName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes (2).txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := UniqueChild(dir, "notes.txt")
	if want := filepath.Join(dir, "notes (3).txt"); got != want {
		t.Fatalf("UniqueChild = %q, want %q", got, want)
	}
	if name := DuplicateName(filepath.Join(dir, "notes.txt")); name != "notes - 副本.txt" {
		t.Fatalf("DuplicateName = %q", name)
	}
}

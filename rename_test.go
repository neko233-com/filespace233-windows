package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/FileSpace233-windows/internal/fsx"
)

func testEntry(name string, dir bool) fsx.Entry {
	return fsx.Entry{Name: name, Path: filepath.Join("C:\\tmp", name), IsDir: dir}
}

func TestBatchRenamePlanNumbersAndKeepsExt(t *testing.T) {
	plan, err := batchRenamePlan([]fsx.Entry{
		testEntry("alpha.txt", false),
		testEntry("beta.log", false),
	}, "img_{nn}")
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].New != "img_01.txt" || plan[1].New != "img_02.log" {
		t.Fatalf("unexpected names: %q %q", plan[0].New, plan[1].New)
	}
}

func TestBatchRenamePlanExplicitExt(t *testing.T) {
	plan, err := batchRenamePlan([]fsx.Entry{testEntry("alpha.txt", false)}, "{name}.{ext}")
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].New != "alpha.txt" {
		t.Fatalf("unexpected name: %q", plan[0].New)
	}
}

func TestBatchRenamePlanRejectsDuplicate(t *testing.T) {
	_, err := batchRenamePlan([]fsx.Entry{
		testEntry("a.txt", false),
		testEntry("b.txt", false),
	}, "same")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestBatchRenamePlanExistingTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "target.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := fsx.Entry{Name: "source.txt", Path: filepath.Join(dir, "source.txt"), Ext: "txt"}
	if err := os.WriteFile(e.Path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := batchRenamePlan([]fsx.Entry{e}, "target")
	if err == nil {
		t.Fatal("expected existing target error")
	}
}

func TestSyncSelect(t *testing.T) {
	left := newPane("C:\\left")
	left.entries = []fsx.Entry{
		{Name: "same.txt", Path: "C:\\left\\same.txt", Size: 10},
		{Name: "only-left.txt", Path: "C:\\left\\only-left.txt", Size: 20},
	}
	left.invalidate()
	right := newPane("C:\\right")
	right.entries = []fsx.Entry{
		{Name: "same.txt", Path: "C:\\right\\same.txt", Size: 99},
		{Name: "only-right.txt", Path: "C:\\right\\only-right.txt", Size: 30},
	}
	right.invalidate()
	a := &App{mode: 2, active: 0, panes: []*Pane{left, right, newPane("C:\\x"), newPane("C:\\y")}}
	a.syncSelect("both")
	if left.selection.Len() != 1 || right.selection.Len() != 1 {
		t.Fatalf("both: %d %d", left.selection.Len(), right.selection.Len())
	}
	a.syncSelect("left")
	if left.selection.Len() != 1 || right.selection.Len() != 0 {
		t.Fatalf("left: %d %d", left.selection.Len(), right.selection.Len())
	}
	a.syncSelect("size")
	if left.selection.Len() != 1 || right.selection.Len() != 1 {
		t.Fatalf("size: %d %d", left.selection.Len(), right.selection.Len())
	}
}

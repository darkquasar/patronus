package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
)

func cs(diffs ...diff.FileDiff) *diff.ChangeSet {
	return &diff.ChangeSet{Diffs: diffs}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestApplyCreateMakesParentsAndWrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a", "b", "SKILL.md")
	a := &Applier{}
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Create, After: []byte("hello")}))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, p) != "hello" {
		t.Errorf("content = %q", read(t, p))
	}
	if len(res.Applied) != 1 {
		t.Errorf("applied = %d, want 1", len(res.Applied))
	}
}

func TestApplyAppendAndMergeWriteAfter(t *testing.T) {
	dir := t.TempDir()
	ap := filepath.Join(dir, "CLAUDE.md")
	mp := filepath.Join(dir, ".mcp.json")
	a := &Applier{}
	_, err := a.Apply(cs(
		diff.FileDiff{Path: ap, Action: diff.Append, After: []byte("appended")},
		diff.FileDiff{Path: mp, Action: diff.Merge, After: []byte(`{"x":1}`)},
	))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, ap) != "appended" || read(t, mp) != `{"x":1}` {
		t.Errorf("append/merge bytes wrong")
	}
}

func TestApplyDeleteRemovesFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Applier{}
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Delete}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("DELETE must remove the file")
	}
	if len(res.Applied) != 1 {
		t.Errorf("applied = %d, want 1", len(res.Applied))
	}
}

func TestApplyDeleteMissingIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gone.md")
	a := &Applier{}
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Delete}))
	if err != nil {
		t.Fatalf("DELETE of a missing file must succeed: %v", err)
	}
	if len(res.Applied) != 1 {
		t.Errorf("applied = %d, want 1 (idempotent delete still counts)", len(res.Applied))
	}
}

func TestApplyUnappendAndRestoreWriteAfter(t *testing.T) {
	dir := t.TempDir()
	up := filepath.Join(dir, "CLAUDE.md")
	rp := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(up, []byte("with section"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rp, []byte(`{"merged":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Applier{}
	_, err := a.Apply(cs(
		diff.FileDiff{Path: up, Action: diff.Unappend, After: []byte("without section")},
		diff.FileDiff{Path: rp, Action: diff.Restore, After: []byte("{}")},
	))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, up) != "without section" {
		t.Errorf("UNAPPEND bytes wrong: %q", read(t, up))
	}
	if read(t, rp) != "{}" {
		t.Errorf("RESTORE bytes wrong: %q", read(t, rp))
	}
}

func TestApplySkipDoesNothing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	a := &Applier{}
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Skip, After: []byte("nope")}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("SKIP must not create the file")
	}
	if len(res.Skipped) != 1 {
		t.Errorf("skipped = %d, want 1", len(res.Skipped))
	}
}

func TestApplyIdempotentNoTempLeftovers(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "f.md")
	a := &Applier{}
	if _, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Create, After: []byte("v1")})); err != nil {
		t.Fatal(err)
	}
	// No .tmp leftovers in the target dir.
	entries, _ := os.ReadDir(filepath.Dir(p))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestApplyConflictDefaultSkips(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if err := os.WriteFile(p, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Applier{} // no Force, no Conflict fn => skip
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Conflict, Before: []byte("original"), After: []byte("new")}))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, p) != "original" {
		t.Error("conflict must not overwrite by default")
	}
	if len(res.Skipped) != 1 || len(res.Applied) != 0 {
		t.Errorf("expected skipped, got applied=%d skipped=%d", len(res.Applied), len(res.Skipped))
	}
}

func TestApplyConflictForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if err := os.WriteFile(p, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Applier{Force: true}
	if _, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Conflict, After: []byte("new")})); err != nil {
		t.Fatal(err)
	}
	if read(t, p) != "new" {
		t.Errorf("force should overwrite, got %q", read(t, p))
	}
}

func TestApplyConflictPromptOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if err := os.WriteFile(p, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	a := &Applier{Conflict: func(d diff.FileDiff) (Resolution, error) {
		called = true
		return Overwrite, nil
	}}
	if _, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Conflict, After: []byte("new")})); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("conflict fn not invoked")
	}
	if read(t, p) != "new" {
		t.Errorf("prompt-overwrite failed, got %q", read(t, p))
	}
}

func TestApplyPartialOnFailureKeepsPriorWrites(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.md")
	// Force a write failure: make the second op's parent a path under a file.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(blocker, "child", "f.md") // blocker is a file, can't be a dir

	a := &Applier{}
	res, err := a.Apply(cs(
		diff.FileDiff{Path: ok, Action: diff.Create, After: []byte("first")},
		diff.FileDiff{Path: bad, Action: diff.Create, After: []byte("second")},
	))
	if err == nil {
		t.Fatal("expected failure on the second op")
	}
	// First write survived (Terraform-style partial).
	if read(t, ok) != "first" {
		t.Error("prior successful write should survive a later failure")
	}
	if res.Failed == nil || res.Failed.Path != bad {
		t.Errorf("Failed should point at the bad op, got %+v", res.Failed)
	}
	if len(res.Applied) != 1 {
		t.Errorf("applied = %d, want 1 (the op before the failure)", len(res.Applied))
	}
}

func TestApplyIsDirRowsIgnored(t *testing.T) {
	dir := t.TempDir()
	a := &Applier{}
	res, err := a.Apply(cs(diff.FileDiff{Path: filepath.Join(dir, "d"), Action: diff.Create, IsDir: true, After: []byte("x")}))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 0 {
		t.Error("IsDir summary rows must not be written")
	}
}

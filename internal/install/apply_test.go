package install

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	res, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Delete, Before: []byte("x")}))
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
		diff.FileDiff{Path: up, Action: diff.Unappend, Before: []byte("with section"), After: []byte("without section")},
		diff.FileDiff{Path: rp, Action: diff.Restore, Before: []byte(`{"merged":1}`), After: []byte("{}")},
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
	if _, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Conflict, Before: []byte("original"), After: []byte("new")})); err != nil {
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
	if _, err := a.Apply(cs(diff.FileDiff{Path: p, Action: diff.Conflict, Before: []byte("original"), After: []byte("new")})); err != nil {
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

func TestApplyFailsWhenDiskDisagrees(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")

	a := &Applier{}
	// Simulate a filesystem that hands back something other than what we wrote.
	// A real filesystem will not do this, so the branch needs an injected seam or
	// its test is a permanent skip.
	a.readBack = func(p string) ([]byte, error) {
		if p == pathA {
			return []byte("not what we wrote"), nil
		}
		return os.ReadFile(p)
	}

	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: pathA, Action: diff.Create, After: []byte(`{"a":1}`)},
		{Path: pathB, Action: diff.Create, After: []byte(`{"b":2}`)},
	}}

	res, err := a.Apply(cs)
	if err == nil {
		t.Fatal("Apply returned nil error; want a verification failure")
	}
	if res.Failed == nil || res.Failed.Path != pathA {
		t.Fatalf("Failed = %+v, want the op at %s", res.Failed, pathA)
	}
	if len(res.Applied) != 0 {
		t.Fatalf("Applied = %d ops, want 0 (the failing op is not applied)", len(res.Applied))
	}
	if _, statErr := os.Stat(pathB); statErr == nil {
		t.Error("the op after the failure was attempted; it must be left unattempted")
	}
	for _, want := range []string{pathA, "expected", "observed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

func TestApplyRecordsOnDiskBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.json")

	a := &Applier{}
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: path, Action: diff.Create, After: []byte(`{"a":1}`)},
	}}

	res, err := a.Apply(cs)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Applied) != 1 {
		t.Fatalf("Applied = %d, want 1", len(res.Applied))
	}
	on, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res.Applied[0].After, on) {
		t.Fatal("the recorded op's After does not match the bytes on disk")
	}
}

func TestApplyRejectsDirectoryPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kit")
	a := &Applier{Force: true}
	result, err := a.Apply(cs(diff.FileDiff{Path: path, Action: diff.Fetch, Artifact: "kit", Directory: &diff.DirectorySpec{Recipe: "kit"}}))
	if err == nil || !strings.Contains(err.Error(), "package delivery service") {
		t.Fatalf("error = %v", err)
	}
	if len(result.Applied) != 0 || result.Failed == nil {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("directory applier wrote: %v", err)
	}
}

func TestApplyFetchReadBackMismatchDoesNotRecordDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture-binary")
	body := []byte("invented inert bytes")
	d := diff.FileDiff{Artifact: "fixture", Action: diff.Fetch, Path: path, Fetch: &diff.FetchSpec{URL: "https://fixture.invalid/payload", Dest: path, SHA256: sha(body)}}
	app := &Applier{Fetcher: fakeFetcher{bodies: map[string][]byte{d.Fetch.URL: body}}, readBack: func(string) ([]byte, error) { return []byte("concurrent writer"), nil }}
	result, err := app.Apply(cs(d))
	if err == nil || !strings.Contains(err.Error(), "write committed") || !strings.Contains(err.Error(), "ownership uncertain") {
		t.Fatalf("missing uncertain committed-path diagnostic: %v", err)
	}
	if result.Failed == nil || result.Failed.Path != path || len(result.Applied) != 0 || d.Fetch.PlacedSHA256 != "" {
		t.Fatalf("unverified fetch recorded: %+v", result)
	}
	if read(t, path) != string(body) {
		t.Fatal("test did not exercise post-write failure")
	}
}

func TestApplyRejectsEditDuringConflictConfirmation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, []byte("prepared"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &Applier{Conflict: func(diff.FileDiff) (Resolution, error) {
		if err := os.WriteFile(path, []byte("external edit"), 0600); err != nil {
			t.Fatal(err)
		}
		return Overwrite, nil
	}}
	result, err := a.Apply(cs(diff.FileDiff{Path: path, Action: diff.Conflict, Before: []byte("prepared"), After: []byte("proposed")}))
	if err == nil || !strings.Contains(err.Error(), "fresh preview") {
		t.Fatalf("want stale preview error, got %v", err)
	}
	if read(t, path) != "external edit" || len(result.Applied) != 0 || result.Failed == nil {
		t.Fatalf("external edit overwritten: %+v", result)
	}
}

func TestApplyExistenceDriftIsNotEmptyEquality(t *testing.T) {
	for _, tc := range []struct {
		name    string
		before  []byte
		present bool
	}{
		{"appeared empty", nil, true}, {"disappeared empty", []byte{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fixture")
			if tc.present {
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			app := Applier{Force: true}
			result, err := app.Apply(cs(diff.FileDiff{Path: path, Action: diff.Merge, Before: tc.before, After: []byte("proposed")}))
			if err == nil || len(result.Applied) != 0 {
				t.Fatalf("existence drift admitted: %+v %v", result, err)
			}
			_, err = os.Stat(path)
			if (err == nil) != tc.present {
				t.Fatal("changed existence")
			}
		})
	}
}

func TestApplyRefusesRedirectedParent(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	link := filepath.Join(dir, "redirect")
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(link, "fixture")
	result, err := (&Applier{Force: true}).Apply(cs(diff.FileDiff{Path: path, Action: diff.Create, After: []byte("unsafe")}))
	if err == nil || len(result.Applied) != 0 {
		t.Fatalf("redirected write admitted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, "fixture")); !os.IsNotExist(err) {
		t.Fatal("wrote through symlink")
	}
}

type dp05ProgressFunc func([]byte) (int, error)

func (f dp05ProgressFunc) Write(p []byte) (int, error) { return f(p) }

func TestApplySharedPathSkipRechecksVerifiedOwnWrite(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(fmt.Sprint("external-edit=", edit), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fixture.json")
			before := []byte(`{"owned":"old","external":"keep"}`)
			after := []byte(`{"owned":"new","external":"keep"}`)
			if err := os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}
			app := Applier{Progress: dp05ProgressFunc(func(p []byte) (int, error) {
				if edit {
					if err := os.WriteFile(path, []byte("external edit after readback"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return len(p), nil
			})}
			result, err := app.Apply(cs(
				diff.FileDiff{Path: path, Action: diff.Merge, Before: before, After: after},
				diff.FileDiff{Path: path, Action: diff.Skip, Before: before, After: before},
			))
			if len(result.Applied) != 1 {
				t.Fatalf("lost verified partial effect: %+v", result)
			}
			if edit {
				if err == nil || result.Failed == nil || len(result.Skipped) != 0 {
					t.Fatalf("sibling SKIP trusted stale own write: %+v %v", result, err)
				}
				if read(t, path) != "external edit after readback" {
					t.Fatal("clobbered edit")
				}
			} else {
				if err != nil || len(result.Skipped) != 1 {
					t.Fatalf("own write misidentified as drift: %+v %v", result, err)
				}
				if !bytes.Equal(result.Skipped[0].Before, before) {
					t.Fatal("changed sibling ownership prior")
				}
			}
		})
	}
}

type dp05FetchFunc func(context.Context, string) (io.ReadCloser, error)

func (f dp05FetchFunc) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	return f(ctx, url)
}

func TestApplyFetchRejectsEditDuringAcquisition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture-bin")
	before := []byte("previous payload")
	payload := []byte("new inert bytes")
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	app := Applier{Fetcher: dp05FetchFunc(func(context.Context, string) (io.ReadCloser, error) {
		if err := os.WriteFile(path, []byte("external edit during fetch"), 0600); err != nil {
			t.Fatal(err)
		}
		return io.NopCloser(bytes.NewReader(payload)), nil
	})}
	d := diff.FileDiff{Path: path, Action: diff.Fetch, Before: before, Fetch: &diff.FetchSpec{Dest: path, URL: "https://fixture.invalid/data", SHA256: sha(payload)}}
	result, err := app.Apply(cs(d))
	if err == nil || len(result.Applied) != 0 || d.Fetch.PlacedSHA256 != "" {
		t.Fatalf("stale fetch recorded: %+v %v", result, err)
	}
	if read(t, path) != "external edit during fetch" {
		t.Fatal("overwrote editor")
	}
}

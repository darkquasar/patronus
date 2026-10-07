package state_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/state"
)

func sectionReconcileFixture(t *testing.T) ([]state.Item, diff.FileDiff) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	a := diff.FileDiff{Artifact: "first", Tool: "codex", Scope: "global", Version: "1", Path: path, Action: diff.Append, Section: &diff.SectionEdit{Name: "first", Body: []byte("original")}, Before: []byte("User prose\n")}
	a.After = adapter.AppendSection(a.Before, a.Section.Name, a.Section.Body)
	old := state.FromChangeSet([]diff.FileDiff{a}, "original-time")
	b := a
	b.Artifact, b.Version = "second", "2"
	b.Section = &diff.SectionEdit{Name: "second", Body: []byte("new")}
	b.Before = a.After
	b.After = adapter.AppendSection(b.Before, b.Section.Name, b.Section.Body)
	return old, b
}

func TestReconcileVerifiedAppendRefreshesOnlySiblingChecksum(t *testing.T) {
	old, d := sectionReconcileFixture(t)
	before := old[0]
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{d}, Result: install.Result{Applied: []diff.FileDiff{d}}, Now: "new-time"})
	if len(got.Unresolved) != 0 || len(got.Items) != 2 {
		t.Fatalf("append reconciliation: %+v", got)
	}
	want := before
	want.Files = append([]state.FileState(nil), before.Files...)
	want.Files[0].Checksum = got.Items[1].Files[0].Checksum
	if want.Files[0].Checksum == before.Files[0].Checksum || !reflect.DeepEqual(got.Items[0], want) {
		t.Fatalf("sibling baseline/identity/version altered or checksum stale: %+v", got.Items[0])
	}
	if !reflect.DeepEqual(old[0], before) {
		t.Fatal("reconciliation mutated input or baseline")
	}
}

func TestReconcileAppendDoesNotRefreshUnverifiedSiblings(t *testing.T) {
	for _, mode := range []string{"failed", "skipped", "drift", "missing-checksum", "changed-section", "foreign-tool", "foreign-scope", "missing-section"} {
		t.Run(mode, func(t *testing.T) {
			old, d := sectionReconcileFixture(t)
			result := install.Result{Applied: []diff.FileDiff{d}}
			switch mode {
			case "failed":
				result = install.Result{Failed: &d}
			case "skipped":
				d.Action = diff.Skip
				result = install.Result{Skipped: []diff.FileDiff{d}}
			case "drift":
				old[0].Files[0].Checksum = "sha256:unverified"
			case "missing-checksum":
				old[0].Files[0].Checksum = ""
			case "changed-section":
				d.After = adapter.AppendSection(d.After, "first", []byte("unauthorized change"))
				result.Applied[0] = d
			case "foreign-tool":
				old[0].Tool = "pi"
			case "foreign-scope":
				old[0].Scope = "local"
			case "missing-section":
				old[0].Files[0].Section = "absent"
			}
			before := old[0]
			got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{d}, Result: result, Now: "new-time"})
			if !reflect.DeepEqual(got.Items[0], before) {
				t.Fatalf("unverified sibling was refreshed (%s): %+v", mode, got.Items[0])
			}
		})
	}
}

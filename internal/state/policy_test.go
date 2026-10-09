package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/state"
)

func TestOwnershipPolicyStructural(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy state.OwnershipPolicy
		tool   string
		want   bool
	}{
		{"zero codex", state.OwnershipPolicy{}, "codex", true},
		{"zero pi", state.OwnershipPolicy{}, "pi", true},
		{"zero claude", state.OwnershipPolicy{}, "claude", false},
		{"zero opencode", state.OwnershipPolicy{}, "opencode", false},
		{"toy names toy", state.NewOwnershipPolicy("toy"), "toy", true},
		{"toy excludes codex", state.NewOwnershipPolicy("toy"), "codex", false},
		{"empty excludes everything", state.NewOwnershipPolicy(), "codex", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.policy.Structural(tc.tool); got != tc.want {
				t.Fatalf("Structural(%s) = %v want %v", tc.tool, got, tc.want)
			}
		})
	}
	tools := []string{"toy"}
	p := state.NewOwnershipPolicy(tools...)
	tools[0] = "mutated"
	if !p.Structural("toy") {
		t.Fatal("policy shares the caller's slice")
	}
}

func TestLoadWithPolicyValidatesStructuralRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	doc := `{"version":2,"items":[{"artifact":"fixture-item","type":"setting","tool":"toy","scope":"global","files":[{"path":"/fixture/config.json","action":"MERGE","checksum":"sha256:00"}]}]}`
	if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := state.LoadWithPolicy(path, state.OwnershipPolicy{}); err != nil {
		t.Fatalf("legacy policy validated a toy row: %v", err)
	}
	_, err := state.LoadWithPolicy(path, state.NewOwnershipPolicy("toy"))
	if err == nil || err.Error() != "state fixture-item /fixture/config.json: setting: missing structural evidence" {
		t.Fatalf("toy policy did not validate the toy row: %v", err)
	}
}

func TestRefreshSectionChecksumsWithPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	first := adapter.AppendSection([]byte("User prose\n"), "first", []byte("one"))
	both := adapter.AppendSection(first, "second", []byte("two"))
	for _, tc := range []struct {
		name      string
		policy    state.OwnershipPolicy
		refreshed bool
	}{
		{"zero value excludes toy", state.OwnershipPolicy{}, false},
		{"toy policy refreshes toy", state.NewOwnershipPolicy("toy"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := []state.Item{{Artifact: "first", Tool: "toy", Scope: "global", Files: []state.FileState{{Path: path, Action: "APPEND", Section: "first", Checksum: state.Checksum(both)}}}}
			d := diff.FileDiff{Artifact: "second", Tool: "toy", Scope: "global", Path: path, Action: diff.Unappend, Section: &diff.SectionEdit{Name: "second"}, Before: both, After: first}
			if got := state.RefreshSectionChecksumsWithPolicy(items, []diff.FileDiff{d}, tc.policy); got != tc.refreshed {
				t.Fatalf("changed = %v want %v", got, tc.refreshed)
			}
			reconciled := state.Reconcile(state.ReconcileInput{Old: items, Desired: []diff.FileDiff{d}, Result: install.Result{Applied: []diff.FileDiff{d}}, Now: "now", Policy: tc.policy})
			if (reconciled.Items[0].Files[0].Checksum == state.Checksum(first)) != tc.refreshed {
				t.Fatalf("Reconcile ignored Policy: %+v", reconciled.Items[0].Files[0])
			}
		})
	}
}

package state_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// Characterization of the tool allowlists in internal/state at the base
// revision: structural SettingEdit validation on load (state.go:142) and the
// Unappend section-checksum refresh (reconcile.go:192) apply to pi and codex
// rows only; claude and opencode rows keep the legacy shared-lane behaviour.

func TestHarnessLoadStructuralValidationAllowlist(t *testing.T) {
	for _, tc := range []struct {
		tool string
		want string
	}{
		{"codex", "state fixture-item /fixture/config.json: setting: missing structural evidence"},
		{"pi", "state fixture-item /fixture/config.json: setting: missing structural evidence"},
		{"claude", ""},
		{"opencode", ""},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			doc := `{"version":2,"items":[{"artifact":"fixture-item","type":"setting","tool":"` + tc.tool + `","scope":"global","files":[{"path":"/fixture/config.json","action":"MERGE","checksum":"sha256:00"}]}]}`
			if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := state.Load(path)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("Load(%s)\n got: %s\nwant: %s", tc.tool, got, tc.want)
			}
		})
	}
}

func TestHarnessUnappendRefreshAllowlist(t *testing.T) {
	for _, tc := range []struct {
		tool      string
		refreshed bool
	}{
		{"codex", true},
		{"pi", true},
		{"claude", false},
		{"opencode", false},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "AGENTS.md")
			first := adapter.AppendSection([]byte("User prose\n"), "first", []byte("one"))
			both := adapter.AppendSection(first, "second", []byte("two"))
			items := []state.Item{{Artifact: "first", Tool: tc.tool, Scope: "global", Files: []state.FileState{{Path: path, Action: "APPEND", Section: "first", Checksum: harnessSum(both)}}}}
			d := diff.FileDiff{Artifact: "second", Tool: tc.tool, Scope: "global", Path: path, Action: diff.Unappend, Section: &diff.SectionEdit{Name: "second"}, Before: both, After: first}
			changed := state.RefreshSectionChecksums(items, []diff.FileDiff{d})
			if changed != tc.refreshed {
				t.Fatalf("refresh(%s) changed=%v want %v", tc.tool, changed, tc.refreshed)
			}
			want := harnessSum(both)
			if tc.refreshed {
				want = harnessSum(first)
			}
			if items[0].Files[0].Checksum != want {
				t.Fatalf("checksum after refresh(%s) = %s want %s", tc.tool, items[0].Files[0].Checksum, want)
			}
		})
	}
}

func harnessSum(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }

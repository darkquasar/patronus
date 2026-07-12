package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/state"
)

// TestRemoveRevertsV1OrphanPluginMerge proves the v1 orphan cleanup needs NO new
// code: a v1-era plugins.<name> setting recorded as a MERGE FileState (with the
// pre-install bytes in Prior) is reverted by remove.Compute's existing
// Prior-restore path.
func TestRemoveRevertsV1OrphanPluginMerge(t *testing.T) {
	prior := []byte("{\n}\n")
	// The v1 install recorded the post-merge bytes' checksum; the file is unchanged
	// since, so remove restores Prior wholesale (no drift skip).
	current := []byte("{\"plugins\":{}}")
	sum := sha256.Sum256(current)
	items := []state.Item{{
		Artifact: "superpowers", Tool: "claude", Scope: "global",
		Files: []state.FileState{{
			Path: "/tmp/does-not-matter/settings.json", Action: "MERGE",
			Checksum: "sha256:" + hex.EncodeToString(sum[:]), Prior: prior,
		}},
	}}
	read := func(string) ([]byte, bool, error) { return current, true, nil }
	cs, _, err := remove.Compute(items, read)
	if err != nil {
		t.Fatal(err)
	}
	var restored bool
	for _, d := range cs.Diffs {
		if d.Action == diff.Restore || d.Action == diff.Merge {
			restored = true
		}
	}
	if !restored {
		t.Errorf("expected a restore/merge revert of the v1 orphan, got %+v", cs.Diffs)
	}
}

// execRemove executes the remove command with args, returning stdout, stderr, err.
func execRemove(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newRemoveCmd("remove", []string{"revert"})
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func shaState(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// seedLocalInstall writes a fake local install (a CREATEd skill + an APPENDed
// instruction section) into a temp project dir, records it in the local state
// file, chdirs there, and returns the project dir. It mirrors what a real
// `install --local --deploy` would leave behind.
func seedLocalInstall(t *testing.T) (proj string, skillPath, instrPath string, priorInstr []byte) {
	t.Helper()
	proj = t.TempDir()
	t.Chdir(proj)
	// Isolate HOME so any global-scope lookups stay in the sandbox.
	t.Setenv("HOME", t.TempDir())

	// The temp cwd has no artifacts/ + adapters/ above it, so registry selection
	// picks Remote — and remove's plugin path (pluginRemoveDiffs -> scanCatalog)
	// then loads the catalog. Serve an EMPTY registry from memory: scanCatalog
	// degrades to nil on an unavailable catalog (no plugin is seeded here anyway),
	// and the file-revert path this test is about runs untouched. Without this the
	// command reached the LIVE registry over the network — which the deny-all
	// TestMain now catches. Never let a test fetch remote bytes.
	empty := &servingFetcher{bodies: map[string][]byte{}}
	prevReg, prevSrc := registryFetcher, fetcherForCommands
	registryFetcher, fetcherForCommands = empty, empty
	t.Cleanup(func() { registryFetcher, fetcherForCommands = prevReg, prevSrc })

	// CREATEd skill.
	skillPath = filepath.Join(proj, ".claude", "skills", "demo", "SKILL.md")
	skillBody := []byte("# demo skill\n")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, skillBody, 0o644); err != nil {
		t.Fatal(err)
	}

	// APPENDed instruction section into a CLAUDE.md that already had user prose.
	instrPath = filepath.Join(proj, "CLAUDE.md")
	priorInstr = []byte("# My Project\n\nuser's own notes\n")
	withSection := adapter.AppendSection(priorInstr, "demo-instr", []byte("patronus guidance"))
	if err := os.WriteFile(instrPath, withSection, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &state.State{Version: state.Version, Items: []state.Item{{
		Artifact: "demo", ItemVersion: "1.0.0", Tool: "claude", Scope: "local",
		Files: []state.FileState{
			{Path: skillPath, Action: "CREATE", Checksum: shaState(skillBody)},
		},
	}, {
		Artifact: "demo-instr", ItemVersion: "1.0.0", Tool: "claude", Scope: "local",
		Files: []state.FileState{
			{Path: instrPath, Action: "APPEND", Section: "demo-instr", Prior: priorInstr, Checksum: shaState(withSection)},
		},
	}}}
	sp := filepath.Join(proj, ".patronus", "state.json")
	if err := state.Save(sp, s); err != nil {
		t.Fatal(err)
	}
	return proj, skillPath, instrPath, priorInstr
}

func TestRemoveDryRunWritesNothing(t *testing.T) {
	_, skillPath, _, _ := seedLocalInstall(t)
	out, _, err := execRemove(t, "demo", "--local")
	if err != nil {
		t.Fatalf("remove dry-run failed: %v", err)
	}
	if !strings.Contains(out, "DELETE") || !strings.Contains(out, "dry run") {
		t.Errorf("expected a DELETE dry-run plan:\n%s", out)
	}
	if _, err := os.Stat(skillPath); err != nil {
		t.Error("dry run must not delete the file")
	}
}

func TestRemoveDeployRoundTrip(t *testing.T) {
	proj, skillPath, instrPath, priorInstr := seedLocalInstall(t)

	_, _, err := execRemove(t, "demo", "demo-instr", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove --deploy failed: %v", err)
	}

	// CREATEd skill deleted.
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Errorf("skill should be deleted, stat err = %v", err)
	}
	// APPENDed section stripped, user prose intact.
	got, err := os.ReadFile(instrPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, priorInstr) {
		t.Errorf("instruction not restored to prior:\n got %q\nwant %q", got, priorInstr)
	}
	// Both items left state.json.
	s, err := state.Load(filepath.Join(proj, ".patronus", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 0 {
		t.Errorf("state should be empty after removing both items, got %+v", s.Items)
	}
}

func TestRemoveDriftSkipsThenForce(t *testing.T) {
	_, skillPath, _, _ := seedLocalInstall(t)
	// User edits the skill after install → drift.
	if err := os.WriteFile(skillPath, []byte("USER EDITED CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Default: warn + skip, file remains.
	_, errOut, err := execRemove(t, "demo", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove --deploy failed: %v", err)
	}
	if !strings.Contains(errOut, "modified since install") {
		t.Errorf("expected a drift warning on stderr:\n%s", errOut)
	}
	if _, err := os.Stat(skillPath); err != nil {
		t.Error("drifted file must NOT be removed without --force")
	}

	// --force: removes it.
	if _, _, err := execRemove(t, "demo", "--local", "--deploy", "--force"); err != nil {
		t.Fatalf("remove --force failed: %v", err)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("--force should remove the drifted file")
	}
}

func TestRemoveUnknownNameErrors(t *testing.T) {
	seedLocalInstall(t)
	_, _, err := execRemove(t, "does-not-exist", "--local")
	if err == nil {
		t.Fatal("expected an error for an uninstalled name")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should explain it's not installed: %v", err)
	}
}

func TestRemoveMutuallyExclusiveScope(t *testing.T) {
	seedLocalInstall(t)
	_, _, err := execRemove(t, "demo", "--global", "--local")
	if err == nil {
		t.Error("expected error for --global and --local together")
	}
}

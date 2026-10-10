package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/state"
)

// TestOpenCodePointerMoveRetainsOwnership (OP-12): the OpenCode pointer dir moved
// from .config/opencode/patronus/instructions/ to .opencode/patronus/instructions/.
// A legacy install keeps its old body path through `update` (retained, not
// relocated, with the HS-T11b wording); the documented step is explicit removal
// then reinstall, after which no row names the old path.
func TestOpenCodePointerMoveRetainsOwnership(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)
	manifest := func(version string) string {
		return "apiVersion: patronus/v2\nfamily: artifact\ntype: instruction\nmode: pointer\npointer:\n  trigger: \"when the invented trigger fires\"\n  summary: \"Invented pointer summary.\"\nrole: instruction\nname: fix-pointer-oc\ndescription: \"OpenCode only: invented pointer instruction.\"\nversion: " + version + "\nentry: INSTRUCTIONS.md\ntargets: [opencode]\ndefaults:\n  scope: project\n"
	}
	const body = "FULL BODY of the invented pointer\n"
	codexWrite(t, filepath.Join(root, "artifacts/instructions/fix-pointer-oc/patronus.yaml"), []byte(manifest("1.1.0")))
	codexWrite(t, filepath.Join(root, "artifacts/instructions/fix-pointer-oc/INSTRUCTIONS.md"), []byte(body))

	// Seed the legacy install: the body under the OLD pointer dir and a state
	// row at 1.0.0 recording it. The legacy unprefixed AGENTS.md section row is
	// deliberately not seeded here: removing two Patronus sections from one
	// file needs the structural section policy the filled descriptor enables
	// (OC-T4 extends this test with that row); user prose is present so the
	// un-append invariant is exercised.
	oldPath := filepath.Join(root, ".config/opencode/patronus/instructions/fix-pointer-oc.md")
	newPath := filepath.Join(root, ".opencode/patronus/instructions/fix-pointer-oc.md")
	agents := filepath.Join(root, "AGENTS.md")
	codexWrite(t, oldPath, []byte(body))
	codexWrite(t, agents, []byte("# Project\n\nuser prose\n"))
	seed := &state.State{Version: state.Version, Items: []state.Item{{
		Artifact: "fix-pointer-oc", ItemVersion: "1.0.0", Type: "instruction", Tool: "opencode", Scope: "local",
		Files: []state.FileState{{Path: oldPath, Action: "CREATE", Checksum: state.Checksum([]byte(body))}},
	}}}
	statePath := filepath.Join(root, ".patronus/state.json")
	if err := state.Save(statePath, seed); err != nil {
		t.Fatal(err)
	}

	_, _, err := runUpdate(t, "fix-pointer-oc", "--target", "opencode", "--deploy")
	got := harnessErr(err)
	if !strings.Contains(got, "ownership unresolved: fix-pointer-oc (opencode/local) "+oldPath+": obsolete owned path retained; explicit removal required") {
		t.Fatalf("update refusal text: %q", got)
	}
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !harnessOwnsPath(s, "fix-pointer-oc", oldPath) {
		t.Fatalf("old row no longer owns %s: %+v", oldPath, s.Items)
	}
	if raw, err := os.ReadFile(oldPath); err != nil || string(raw) != body {
		t.Fatalf("old body touched by update: %v %q", err, raw)
	}
	if raw, err := os.ReadFile(newPath); err != nil || string(raw) != body {
		t.Fatalf("new body not installed: %v %q", err, raw)
	}
	if raw, err := os.ReadFile(agents); err != nil || !strings.Contains(string(raw), "<!-- patronus:start oc:fix-pointer-oc -->") {
		t.Fatalf("oc: stanza not installed: %v\n%s", err, raw)
	}

	// Explicit removal then reinstall: no row and no file for the old path,
	// user prose survives and the oc: stanza is back.
	if out, errOut, err := execRemove(t, "fix-pointer-oc", "--target", "opencode", "--deploy"); err != nil {
		t.Fatalf("explicit removal: %v\n%s\n%s", err, out, errOut)
	}
	if out, errOut, err := runInstall(t, "fix-pointer-oc", "--target", "opencode", "--local", "--deploy"); err != nil {
		t.Fatalf("reinstall: %v\n%s\n%s", err, out, errOut)
	}
	s, err = state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range s.Items {
		for _, f := range it.Files {
			if f.Path == oldPath {
				t.Fatalf("orphan row for %s after explicit removal: %+v", oldPath, it)
			}
		}
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old body survives explicit removal: %v", err)
	}
	if !harnessOwnsPath(s, "fix-pointer-oc", newPath) {
		t.Fatalf("new path not owned after reinstall: %+v", s.Items)
	}
	raw, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "user prose") || !strings.Contains(string(raw), "<!-- patronus:start oc:fix-pointer-oc -->") {
		t.Fatalf("AGENTS.md after reinstall:\n%s", raw)
	}
}

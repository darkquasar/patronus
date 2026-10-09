package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/state"
)

// HS-T11b: when a sidecar moves from one artifact to a required second artifact
// between versions, update keeps the old row's ownership of the old path and
// reports the exact reconcile refusal (reconcile.go:150-169 wrapped at :55);
// explicit removal of the old artifact is the documented step that leaves no
// orphan row. Pinned before the isolated-writer-supervisor move (HS-T11).
func TestHarnessUpdateMoveRetainsOwnership(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	const helper = "#!/bin/sh\necho invented helper\n"
	manifest := func(name, version, files, requires string) string {
		return "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: " + name + "\ndescription: \"Invented " + name + ".\"\nversion: " + version + "\nentry: SKILL.md\nfiles: [" + files + "]\ntargets: [codex]\ndefaults:\n  scope: global\n" + requires
	}
	codexWrite(t, filepath.Join(root, "artifacts/skills/mover/patronus.yaml"), []byte(manifest("mover", "1.0.0", "\"scripts/helper.sh\"", "")))
	codexWrite(t, filepath.Join(root, "artifacts/skills/mover/SKILL.md"), []byte("---\nname: mover\ndescription: invented mover\n---\nUses {skillDir}/scripts/helper.sh\n"))
	codexWrite(t, filepath.Join(root, "artifacts/skills/mover/scripts/helper.sh"), []byte(helper))
	codexWrite(t, filepath.Join(root, "artifacts/skills/carrier/patronus.yaml"), []byte(manifest("carrier", "1.0.0", "\"scripts/helper.sh\"", "")))
	codexWrite(t, filepath.Join(root, "artifacts/skills/carrier/SKILL.md"), []byte("---\nname: carrier\ndescription: invented carrier\n---\nShips {skillDir}/scripts/helper.sh\n"))
	codexWrite(t, filepath.Join(root, "artifacts/skills/carrier/scripts/helper.sh"), []byte(helper))

	if out, errOut, err := runInstall(t, "mover", "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatalf("install v1: %v\n%s\n%s", err, out, errOut)
	}
	oldPath := filepath.Join(home, ".agents/skills/mover/scripts/helper.sh")
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("v1 sidecar not installed: %v", err)
	}

	// v2 drops the sidecar and requires carrier, which ships its own copy.
	codexWrite(t, filepath.Join(root, "artifacts/skills/mover/patronus.yaml"), []byte(manifest("mover", "2.0.0", "", "requires: [carrier]\n")))
	codexWrite(t, filepath.Join(root, "artifacts/skills/mover/SKILL.md"), []byte("---\nname: mover\ndescription: invented mover\n---\nUses {skillsDir}/carrier/scripts/helper.sh\n"))
	if err := os.RemoveAll(filepath.Join(root, "artifacts/skills/mover/scripts")); err != nil {
		t.Fatal(err)
	}
	_, _, err := runUpdate(t, "mover", "--target", "codex", "--deploy")
	// Observed at base: the reconcile reason is followed by the apply diagnostics
	// suffix ("; verified committed=[...] skipped=[...] failed=none; re-preview
	// current bytes before repair (equality is not adoption authority)").
	got := harnessErr(err)
	if !strings.HasPrefix(got, "ownership unresolved: mover (codex/global) "+oldPath+": obsolete owned path retained; explicit removal required; verified committed=[") || !strings.HasSuffix(got, "failed=none; re-preview current bytes before repair (equality is not adoption authority)") {
		t.Fatalf("update refusal text: %q", got)
	}
	statePath := filepath.Join(home, ".patronus/state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !harnessOwnsPath(s, "mover", oldPath) {
		t.Fatalf("old row no longer owns %s: %+v", oldPath, s.Items)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("old sidecar removed by update: %v", err)
	}
	carrierPath := filepath.Join(home, ".agents/skills/carrier/scripts/helper.sh")
	if raw, err := os.ReadFile(carrierPath); err != nil || string(raw) != helper {
		t.Fatalf("carrier copy not installed: %v %q", err, raw)
	}
	if !harnessOwnsPath(s, "carrier", carrierPath) {
		t.Fatalf("carrier row missing: %+v", s.Items)
	}

	// The documented explicit removal step leaves no row for the old path.
	if out, errOut, err := execRemove(t, "mover", "--target", "codex", "--deploy"); err != nil {
		t.Fatalf("explicit removal: %v\n%s\n%s", err, out, errOut)
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
		t.Fatalf("old sidecar survives explicit removal: %v", err)
	}
	if !harnessOwnsPath(s, "carrier", carrierPath) {
		t.Fatalf("carrier row lost on mover removal: %+v", s.Items)
	}
}

func harnessOwnsPath(s *state.State, artifact, path string) bool {
	for _, it := range s.Items {
		if it.Artifact != artifact {
			continue
		}
		for _, f := range it.Files {
			if f.Path == path {
				return true
			}
		}
	}
	return false
}

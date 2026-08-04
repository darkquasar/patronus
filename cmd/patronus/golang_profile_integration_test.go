package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestComposedAppendRemovesSelectively is the CLASS-A counterpart: the deploy
// mechanics the golang/visual profile tests used to prove, now asserted against
// the FIXTURE catalog, where the delivered binary is bytes this test invented — so
// the FETCH path (download -> verify -> place) actually RUNS, which stubBinary
// never did.
//
// Two instruction artifacts APPEND into ONE CLAUDE.md as distinct fenced sections;
// state records BOTH (the composed-APPEND fix — otherwise remove would leak the
// second); and removing one strips exactly its section while the other survives.
func TestComposedAppendRemovesSelectively(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// The fixture's binaries were FETCHED, verified against their invented pins, and
	// placed — the path stubBinary used to skip entirely.
	raw := mustRead(t, filepath.Join(home, ".patronus", "bin", "fix-bin"))
	if shaHex(raw) != shaHex(fixRawBinary) {
		t.Errorf("placed raw binary does not match the bytes the fixture served")
	}
	archived := mustRead(t, filepath.Join(home, ".patronus", "bin", "fix-archive-bin"))
	if shaHex(archived) != shaHex(fixArchivedBinary) {
		t.Errorf("placed archive binary is not the extracted tarball member")
	}

	// Both instruction artifacts land as distinct fenced sections in ONE CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb := string(mustRead(t, claudeMd))
	for _, want := range []string{"patronus:start fix-instruction", "patronus:start fix-instruction-2"} {
		if !strings.Contains(cb, want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}
	if !strings.Contains(cb, "always fix the fixture") {
		t.Errorf("fix-instruction-2's body did not land:\n%s", cb)
	}

	// State records BOTH contributors, not just the first.
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	for _, want := range []string{"fix-instruction", "fix-instruction-2", "fix-skill"} {
		if !strings.Contains(st, want) {
			t.Errorf("state missing %q:\n%s", want, st)
		}
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Selective remove of the LAST composed CLAUDE.md contributor: its section is
	// stripped while the earlier one survives. (Un-appending a NON-last section is
	// drift-guarded, because later sections shift the recorded Prior — a known
	// composed-APPEND limitation, not exercised here.)
	if _, errOut, err := execRemove(t, "fix-instruction-2", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-instruction-2: %v\n%s", err, errOut)
	}
	cb2 := string(mustRead(t, claudeMd))
	if strings.Contains(cb2, "patronus:start fix-instruction-2") {
		t.Errorf("fix-instruction-2's section should be gone:\n%s", cb2)
	}
	if !strings.Contains(cb2, "patronus:start fix-instruction") {
		t.Errorf("fix-instruction's section should survive selective remove:\n%s", cb2)
	}
	// And the CREATEd skill is untouched by an unrelated item's removal.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")); err != nil {
		t.Errorf("fix-skill should survive an unrelated remove: %v", err)
	}
}

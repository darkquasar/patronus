package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGolangProfileWiresGoStyleOnTopOfCore is a CLASS-B test: it asserts the REAL
// catalog's CONTENTS — that the `golang` profile really extends `core`, really
// appends the distilled Uber-Go-Style instruction, and really inherits core's
// spine and skills. The item names ARE the assertion; renaming them to fixture
// names would produce a green tautology that tests nothing.
//
// It asserts against the PLAN, not the placement, so it never fetches, never
// hashes an upstream digest, and never places a binary. That is what keeps a
// real-catalog test valid: it may read the catalog's SHAPE, never its PINS.
// (`core` wires the gitleaks + tk recipes, whose pins are REAL upstream digests
// that no invented bytes can satisfy. A --deploy here would have to either fetch
// from the network — forbidden — or pre-place a stub, which classifyFetch now
// correctly rejects as tampered. See test-surface-plan.md, Task 9.)
//
// The deploy MECHANICS this file used to cover — composed APPEND into one
// CLAUDE.md, state recording per contributor, selective remove — are asserted
// against the fixture catalog instead (TestComposedAppendRemovesSelectively),
// where the delivered payload is bytes the test itself invented.
func TestGolangProfileWiresGoStyleOnTopOfCore(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)
	withFakeRunner(t)

	out, errOut, err := runInstall(t, "--profile", "golang", "--tool", "claude", "--global")
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, errOut)
	}

	// golang's own overlay AND core's inherited L1 spine both target ONE CLAUDE.md.
	for _, want := range []string{"agents-spine", "agent-rules", "go-style-uber"} {
		if !planHas(out, want, filepath.Join("~", ".claude", "CLAUDE.md")) {
			t.Errorf("golang plan does not APPEND %q into ~/.claude/CLAUDE.md:\n%s", want, out)
		}
	}

	// The inherited core skills + the diagram-explain output-style are planned too.
	for _, tc := range []struct{ item, path string }{
		{"tdd", filepath.Join("~", ".claude", "skills", "tdd")},
		{"codebase-design", filepath.Join("~", ".claude", "skills", "codebase-design")},
		{"diagram-explain", filepath.Join("~", ".claude", "output-styles", "diagram-explain.md")},
	} {
		if !planHas(out, tc.item, tc.path) {
			t.Errorf("golang does not inherit core's %q at %s:\n%s", tc.item, tc.path, out)
		}
	}
}

// planHas reports whether the plan table has a row naming BOTH the item and the
// path it targets. Asserting the pair (not just the name) is what makes this a
// real guarantee rather than a substring coincidence: an item merely MENTIONED
// somewhere in the output would pass a bare Contains.
func planHas(plan, item, path string) bool {
	for _, line := range strings.Split(plan, "\n") {
		if strings.Contains(line, item) && strings.Contains(line, path) {
			return true
		}
	}
	return false
}

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

	if _, errOut, err := runInstall(t, "--profile", "fix-all", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
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
	out, _, err := runInstall(t, "--profile", "fix-all", "--tool", "claude", "--global", "--dry-run")
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

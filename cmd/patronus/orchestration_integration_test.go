package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the §6b acceptance gate for L10 orchestration. Two classes live here:
//
//   - CLASS A (mechanism): the requires CLOSURE — a DIRECT install of an
//     instruction pulls in the binary recipe it documents, dependency-before-
//     dependent — and the APPEND/UNAPPEND round-trip. The item names are arbitrary
//     to those claims, so they are asserted on the FIXTURE, where the delivered
//     binary is bytes the test invented. This is what let cmd/patronus/testdata/tk
//     (47KB of vendored third-party bash, pinned to an upstream digest) be deleted.
//
//   - CLASS B (catalog contents): "the SDD skill really packs its prompt + scripts"
//     and "core's orchestration slot really pins ticket + tk + session-completion".
//     The names ARE those assertions, so they stay real — and both stay off the
//     fetch path (one installs no binary; the other only locks).

// TestRequiresClosureDirectInstall proves the per-item `requires` edge: installing
// ONLY the instruction also installs the binary it documents — the closure is
// honored on the direct install path, not just via a profile. CLASS A: the fixture's
// fix-instruction -> fix-bin edge is the same shape as ticket -> tk, and its pin is
// the sha256 of bytes this test invented, so the FETCH runs for real against bytes
// nothing upstream can drift.
func TestRequiresClosureDirectInstall(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// Dry-run first: the plan for a bare `install fix-instruction` (no profile) must
	// include the fix-bin recipe, pulled in by the requires closure, and announce it.
	out, errOut, err := runInstall(t, "fix-instruction", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run install fix-instruction: %v\n%s", err, errOut)
	}
	if !strings.Contains(errOut, "also installing required item") || !strings.Contains(errOut, "fix-bin") {
		t.Errorf("expected a 'required item(s): fix-bin' notice on stderr:\n%s", errOut)
	}
	// Both the listed instruction and the auto-pulled binary appear in the plan, the
	// latter as an install-only recipe row.
	if !strings.Contains(out, "fix-instruction") {
		t.Errorf("plan missing the instruction:\n%s", out)
	}
	if !strings.Contains(out, "fix-bin") || !strings.Contains(out, "install-only") {
		t.Errorf("plan missing the closure-pulled fix-bin install-only recipe:\n%s", out)
	}

	// Now deploy for real: download -> verify against the invented pin -> place.
	if _, e, err := runInstall(t, "fix-instruction", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install fix-instruction: %v\n%s", err, e)
	}
	cb := string(mustRead(t, filepath.Join(home, ".claude", "CLAUDE.md")))
	if !strings.Contains(cb, "patronus:start fix-instruction") {
		t.Errorf("CLAUDE.md missing the instruction block:\n%s", cb)
	}
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	if !strings.Contains(st, "fix-instruction") {
		t.Errorf("state missing the instruction:\n%s", st)
	}
	// The closure-pulled binary actually landed, executable, and is the bytes served.
	dest := filepath.Join(home, ".patronus", "bin", "fix-bin")
	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("fix-bin not placed by the closure: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("fix-bin mode = %v, want executable", fi.Mode().Perm())
	}
	if shaHex(mustRead(t, dest)) != shaHex(fixRawBinary) {
		t.Errorf("placed fix-bin does not match the bytes the fixture served")
	}

	// Idempotent re-run: the placed binary re-hashes to the pin, so it SKIPs.
	out, _, err = runInstall(t, "fix-instruction", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}
}

// TestOrchestrationSkillsInstall proves the two vendored superpowers orchestration
// skills land as per-tool skills/<name>/SKILL.md, with the SDD skill's aux files
// (prompts + scripts) packed alongside.
//
// CLASS B: "the real SDD skill really ships implementer-prompt.md and
// scripts/review-package" is a statement about the CATALOG's contents — the names
// ARE the assertion, so they stay real. It is safe on the real catalog because it
// installs two ARTIFACTS and no recipe: it never reads a pin, never hashes upstream
// bytes, and never places a binary.
func TestOrchestrationSkillsInstall(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t,
		"subagent-driven-development", "dispatching-parallel-agents",
		"--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install orchestration skills: %v\n%s", err, errOut)
	}

	for _, name := range []string{"subagent-driven-development", "dispatching-parallel-agents"} {
		p := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("skill %q not created at %s: %v", name, p, err)
		}
	}
	// SDD aux files packed: a prompt template and a script helper.
	sddDir := filepath.Join(home, ".claude", "skills", "subagent-driven-development")
	for _, rel := range []string{"implementer-prompt.md", filepath.Join("scripts", "review-package")} {
		if _, err := os.Stat(filepath.Join(sddDir, rel)); err != nil {
			t.Errorf("SDD aux file %q not packed: %v", rel, err)
		}
	}
}

// TestCoreOrchestrationSlotAndLock proves the core profile's orchestration slot
// resolves (ticket + tk via closure + session-completion + the 2 skills) and the
// lock pins the closure — tk is pinned even though no slot names it directly.
//
// CLASS B: "the real core profile really wires ticket, and the closure really pins
// tk" — the names ARE the assertion, so they stay real. It only LOCKS, which
// resolves and pins from the catalog and never fetches a binary, so it stays off
// the fetch path.
func TestCoreOrchestrationSlotAndLock(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "core", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	for _, want := range []string{
		"ticket", "tk", // tk pinned via the requires closure, not a direct slot entry
		"session-completion",
		"subagent-driven-development", "dispatching-parallel-agents",
		"tarballSha256",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q (orchestration slot / closure not pinned):\n%s", want, s)
		}
	}
}

// TestInstructionRemoveRoundTrips proves an instruction round-trips: its CLAUDE.md
// block is APPENDed on install and cleanly UNAPPENDed on remove (the surrounding
// file — here, the patronus-managed boundary markers — is left intact). CLASS A:
// the mechanism, on the fixture, so it never fetches an upstream binary.
func TestInstructionRemoveRoundTrips(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "fix-instruction", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if !strings.Contains(string(mustRead(t, claudeMd)), "patronus:start fix-instruction") {
		t.Fatal("precondition: the instruction block should be present after install")
	}

	if _, e, err := execRemove(t, "fix-instruction", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-instruction: %v\n%s", err, e)
	}
	if strings.Contains(string(mustRead(t, claudeMd)), "patronus:start fix-instruction") {
		t.Errorf("the instruction block should be gone after remove:\n%s", mustRead(t, claudeMd))
	}
}

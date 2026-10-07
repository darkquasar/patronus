package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Shared requires closure, sidecar placement, lock and removal on invented input.

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
	out, errOut, err := runInstall(t, "fix-instruction", "--target", "claude", "--global", "--dry-run")
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
	if _, e, err := runInstall(t, "fix-instruction", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
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
	out, _, err = runInstall(t, "fix-instruction", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}
}

// Entry and declared sidecars travel through build, fetch and install.
func TestOrchestrationSkillsInstall(t *testing.T) {
	f := serveFixtureFrom(t, fixtureSkillBundle(t))
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t,
		"fix-router", "fix-review",
		"--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install orchestration skills: %v\n%s", err, errOut)
	}

	for _, name := range []string{"fix-router", "fix-review"} {
		p := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("skill %q not created at %s: %v", name, p, err)
		}
	}
	// SDD aux files packed: a prompt template and a script helper.
	sddDir := filepath.Join(home, ".claude", "skills", "fix-router")
	for _, rel := range []string{"mode.md", filepath.Join("scripts", "helper")} {
		if _, err := os.Stat(filepath.Join(sddDir, rel)); err != nil {
			t.Errorf("SDD aux file %q not packed: %v", rel, err)
		}
	}
}

// A profile lock includes the instruction dependency even without a layer slot.
func TestCoreOrchestrationSlotAndLock(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "fix-all", "--target", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	for _, want := range []string{
		"fix-instruction", "fix-bin", // tk pinned via the requires closure, not a direct slot entry
		"fix-instruction-2",
		"fix-skill",
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

	if _, e, err := runInstall(t, "fix-instruction", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the §6b acceptance gate for L10 orchestration: the tk recipe + the
// ticket instruction (which `requires: [tk]`) + the two vendored superpowers
// orchestration skills, proven end-to-end against the built catalog. The headline
// proof is the requires CLOSURE: a DIRECT `install ticket` (no profile) pulls in
// the tk binary recipe automatically, dependency-before-dependent.
//
// tk is a `url` (raw) delivery: its sha is verified on download and recomputed
// from the placed file on every run, so it cannot be faked with a dummy stub. The
// offline fetcher serves the real script (see serveBinaries), so these tests
// exercise the true FETCH path.

// TestRequiresClosureDirectInstall proves the per-item `requires` edge: installing
// ONLY the ticket instruction also installs the tk binary it documents — the
// closure is honored on the direct install path, not just via a profile.
func TestRequiresClosureDirectInstall(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	// Dry-run first: the plan for a bare `install ticket` (no profile) must include
	// the tk binary recipe, pulled in by the requires closure, and announce it.
	out, errOut, err := runInstall(t, "ticket", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run install ticket: %v\n%s", err, errOut)
	}
	if !strings.Contains(errOut, "also installing required item") || !strings.Contains(errOut, "tk") {
		t.Errorf("expected a 'required item(s): tk' notice on stderr:\n%s", errOut)
	}
	// Both the listed instruction and the auto-pulled binary appear in the plan,
	// the latter as an install-only recipe row.
	if !strings.Contains(out, "ticket") {
		t.Errorf("plan missing the ticket instruction:\n%s", out)
	}
	if !strings.Contains(out, "tk") || !strings.Contains(out, "install-only") {
		t.Errorf("plan missing the closure-pulled tk install-only recipe:\n%s", out)
	}

	// Now deploy for real. tk is NOT stubbed: a url delivery sha-verifies what it
	// downloads, so the fetcher serves the real script and the FETCH truly runs.
	if _, e, err := runInstall(t, "ticket", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install ticket: %v\n%s", err, e)
	}
	cb := string(mustRead(t, filepath.Join(home, ".claude", "CLAUDE.md")))
	if !strings.Contains(cb, "patronus:start ticket") {
		t.Errorf("CLAUDE.md missing the ticket instruction block:\n%s", cb)
	}
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	if !strings.Contains(st, "ticket") {
		t.Errorf("state missing the ticket instruction:\n%s", st)
	}
	// The closure-pulled binary actually landed, executable.
	fi, err := os.Stat(filepath.Join(home, ".patronus", "bin", "tk"))
	if err != nil {
		t.Fatalf("tk binary not placed by the closure: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("tk mode = %v, want executable", fi.Mode().Perm())
	}

	// Idempotent re-run: the placed tk re-hashes to the pin, so it SKIPs.
	out, _, err = runInstall(t, "ticket", "--tool", "claude", "--global", "--dry-run")
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

// TestTicketRemoveRoundTrips proves the ticket instruction round-trips: its CLAUDE.md
// block is APPENDed on install and cleanly UNAPPENDed on remove (the surrounding
// file — here, the patronus-managed boundary markers — is left intact).
func TestTicketRemoveRoundTrips(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "ticket", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if !strings.Contains(string(mustRead(t, claudeMd)), "patronus:start ticket") {
		t.Fatal("precondition: ticket block should be present after install")
	}

	if _, e, err := execRemove(t, "ticket", "--global", "--deploy"); err != nil {
		t.Fatalf("remove ticket: %v\n%s", err, e)
	}
	if strings.Contains(string(mustRead(t, claudeMd)), "patronus:start ticket") {
		t.Errorf("ticket block should be gone after remove:\n%s", mustRead(t, claudeMd))
	}
}

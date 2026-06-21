package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the §6b acceptance gate for P7.6 (L10 orchestration): the bd recipe +
// the beads instruction (which `requires: [bd]`) + the two vendored superpowers
// orchestration skills, proven end-to-end against the built catalog. The headline
// proof is the requires CLOSURE: a DIRECT `install beads` (no profile) pulls in
// the bd binary recipe automatically, dependency-before-dependent.

// TestRequiresClosureDirectInstall proves the per-item `requires` edge: installing
// ONLY the beads instruction also installs the bd binary it documents — the
// closure is honored on the direct install path, not just via a profile.
func TestRequiresClosureDirectInstall(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	// Dry-run first: the plan for a bare `install beads` (no profile) must include
	// the bd binary recipe, pulled in by the requires closure, and announce it.
	out, errOut, err := runInstall(t, "beads", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run install beads: %v\n%s", err, errOut)
	}
	if !strings.Contains(errOut, "also installing required item") || !strings.Contains(errOut, "bd") {
		t.Errorf("expected a 'required item(s): bd' notice on stderr:\n%s", errOut)
	}
	// Both the listed instruction and the auto-pulled binary appear in the plan,
	// the latter as an install-only recipe row.
	if !strings.Contains(out, "beads") {
		t.Errorf("plan missing the beads instruction:\n%s", out)
	}
	if !strings.Contains(out, "bd") || !strings.Contains(out, "install-only") {
		t.Errorf("plan missing the closure-pulled bd install-only recipe:\n%s", out)
	}

	// Now deploy for real (bd stubbed → its FETCH SKIPs offline) and assert the
	// beads instruction actually lands.
	stubBinary(t, home, "bd")
	if _, e, err := runInstall(t, "beads", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install beads: %v\n%s", err, e)
	}
	cb := string(mustRead(t, filepath.Join(home, ".claude", "CLAUDE.md")))
	if !strings.Contains(cb, "patronus:start beads") {
		t.Errorf("CLAUDE.md missing the beads instruction block:\n%s", cb)
	}
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	if !strings.Contains(st, "beads") {
		t.Errorf("state missing the beads instruction:\n%s", st)
	}

	// Idempotent re-run.
	out, _, err = runInstall(t, "beads", "--tool", "claude", "--global", "--dry-run")
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

// TestCoreOrchestrationSlotAndLock proves the core profile's new orchestration slot
// resolves (beads + bd via closure + the 2 skills) and the lock pins the closure —
// bd is pinned with a content/tarball sha even though no slot names it directly.
func TestCoreOrchestrationSlotAndLock(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "core", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	for _, want := range []string{
		"beads", "bd", // bd pinned via the requires closure, not a direct slot entry
		"subagent-driven-development", "dispatching-parallel-agents",
		"tarballSha256",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q (orchestration slot / closure not pinned):\n%s", want, s)
		}
	}
}

// TestBeadsRemoveRoundTrips proves the beads instruction round-trips: its CLAUDE.md
// block is APPENDed on install and cleanly UNAPPENDed on remove (the surrounding
// file — here, the patronus-managed boundary markers — is left intact).
func TestBeadsRemoveRoundTrips(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	stubBinary(t, home, "bd")

	if _, e, err := runInstall(t, "beads", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if !strings.Contains(string(mustRead(t, claudeMd)), "patronus:start beads") {
		t.Fatal("precondition: beads block should be present after install")
	}

	if _, e, err := execRemove(t, "beads", "--global", "--deploy"); err != nil {
		t.Fatalf("remove beads: %v\n%s", err, e)
	}
	if strings.Contains(string(mustRead(t, claudeMd)), "patronus:start beads") {
		t.Errorf("beads block should be gone after remove:\n%s", mustRead(t, claudeMd))
	}
}

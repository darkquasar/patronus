package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These drive the real `golang` profile end-to-end. golang `extends: core`, so it
// must install the ENTIRE core spine PLUS the distilled Uber-Go-Style instruction
// (go-style-uber) appended into the same CLAUDE.md — proving both `extends:`
// composition and the vendored Go instruction land and remove cleanly.

func TestGolangProfileExtendsCore(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)               // golang inherits core's self-wiring ai-memory
	stubBinary(t, home, "gitleaks") // ...and core's gitleaks recipe FETCH (offline SKIP)

	if _, errOut, err := runInstall(t, "--profile", "golang", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb := string(mustRead(t, claudeMd))
	// The Go instruction AND core's L1 spine both land as distinct sections in one
	// CLAUDE.md (extends appended go-style-uber to core's instructions).
	for _, want := range []string{
		"patronus:start agents-spine",  // from core
		"patronus:start agent-rules",   // from core
		"patronus:start go-style-uber", // golang's own overlay
	} {
		if !strings.Contains(cb, want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}
	// A representative Go rule made it into the appended body (faithful distillation).
	if !strings.Contains(cb, "Start enums at one") {
		t.Errorf("go-style-uber body missing a known rule:\n%s", cb)
	}

	// The inherited core skills + the diagram-explain output-style are present too.
	for _, p := range []string{
		filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md"),
		filepath.Join(home, ".claude", "skills", "codebase-design", "SKILL.md"),
		filepath.Join(home, ".claude", "output-styles", "diagram-explain.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("inherited core item missing at %s: %v", p, err)
		}
	}

	// State records the Go instruction under its own name (so remove strips exactly it).
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	if !strings.Contains(st, "go-style-uber") {
		t.Errorf("state missing go-style-uber:\n%s", st)
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "golang", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Selective remove of the LAST composed CLAUDE.md contributor (session-completion,
	// wired by core's orchestration slot, appended after the instruction sections) →
	// its section is stripped while the earlier sections (go-style-uber, agents-spine)
	// survive. Removing the LAST contributor is what surgical un-append guarantees:
	// un-appending a non-last section is drift-guarded because later sections shift
	// the recorded Prior (a known composed-APPEND limitation, not exercised here).
	if _, errOut, err := execRemove(t, "session-completion", "--global", "--deploy"); err != nil {
		t.Fatalf("remove session-completion: %v\n%s", err, errOut)
	}
	cb2 := string(mustRead(t, claudeMd))
	if strings.Contains(cb2, "patronus:start session-completion") {
		t.Errorf("session-completion section should be gone:\n%s", cb2)
	}
	if !strings.Contains(cb2, "patronus:start go-style-uber") {
		t.Errorf("the go-style-uber section should survive selective remove:\n%s", cb2)
	}
	if !strings.Contains(cb2, "patronus:start agents-spine") {
		t.Errorf("inherited agents-spine should survive selective remove:\n%s", cb2)
	}
}

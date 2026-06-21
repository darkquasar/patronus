package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStrictCoreConsolidated is the P7.5.6 §6b consolidation: it proves the FULLY
// assembled strict `core` end-to-end on Claude — every P7.5 layer present at once,
// the lock pins the new hook/setting items, and remove round-trips each action
// class (CREATE→DELETE, MERGE→strip/restore). The per-layer divergence + offline
// FETCH are covered by the focused tests; this is the "all of it, together" gate.
func TestStrictCoreConsolidated(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks")
	stubBinary(t, home, "bd") // core wires beads -> requires bd (github-release FETCH SKIPs offline)

	// safe-git gives core + git-guardrails; add the tdd-guard enforcement items on
	// top so all four PreToolUse gates (tdd-guard + block-secrets + gitleaks-guard +
	// git-guardrails) coexist — the full strict set is now split across two opt-in
	// profiles (tdd-enforced + safe-git), so the consolidated test composes both.
	if _, e, err := runInstall(t, "--profile", "safe-git", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install safe-git: %v\n%s", err, e)
	}
	if _, e, err := runInstall(t, "tdd-guard", "tdd-guard-hook", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install tdd-guard enforcement: %v\n%s", err, e)
	}

	settings := filepath.Join(home, ".claude", "settings.json")
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings.json: %v", err)
	}

	// All four PreToolUse gates coexist in ONE settings.json array (the compose-fold):
	// tdd-guard + block-secrets + gitleaks-guard + git-guardrails. Plus 2 SessionStart
	// hooks (dispatch keystone activation + work-state reground), 1 UserPromptSubmit
	// (the per-turn skill heartbeat), and the statusLine setting.
	if pre, _ := root["hooks"].(map[string]any)["PreToolUse"].([]any); len(pre) != 4 {
		t.Errorf("want 4 PreToolUse hooks, got %d", len(pre))
	}
	if ss, _ := root["hooks"].(map[string]any)["SessionStart"].([]any); len(ss) != 2 {
		t.Errorf("want 2 SessionStart hooks, got %d", len(ss))
	}
	if ups, _ := root["hooks"].(map[string]any)["UserPromptSubmit"].([]any); len(ups) != 1 {
		t.Errorf("want 1 UserPromptSubmit hook, got %d", len(ups))
	}
	if _, ok := root["statusLine"].(map[string]any); !ok {
		t.Errorf("statusLine setting missing: %v", root["statusLine"])
	}

	// The output-style is a Claude FILE; the bootstrap skill + verification skill landed.
	for _, p := range []string{
		filepath.Join(home, ".claude", "output-styles", "diagram-explain.md"),
		filepath.Join(home, ".claude", "skills", "skills-dispatch", "SKILL.md"),
		filepath.Join(home, ".claude", "skills", "verification-before-completion", "SKILL.md"),
		filepath.Join(home, ".claude", "hooks", "git-guardrails.sh"),
		filepath.Join(home, ".claude", "hooks", "skills-dispatch-activate.sh"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected installed file missing: %s (%v)", p, err)
		}
	}

	// Lock pins the new hook + setting items (not just skills/instructions).
	if _, _, err := runLock(t, "--profile", "safe-git", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	lock := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	// safe-git's lock pins core's strict items + git-guardrails (tdd-guard-hook is an
	// opt-in via tdd-enforced, not in safe-git, so it's not in this lock).
	for _, want := range []string{"git-guardrails", "block-secrets", "gitleaks-guard", "skills-dispatch-activate", "ccusage-statusline"} {
		if !strings.Contains(lock, want) {
			t.Errorf("lock missing strict-gate item %q:\n%s", want, lock)
		}
	}

	// Remove round-trips: a script-bearing guardrail hook's DELETE (script) +
	// settings element strip; its siblings survive.
	if _, e, err := execRemove(t, "git-guardrails", "--global", "--deploy"); err != nil {
		t.Fatalf("remove git-guardrails: %v\n%s", err, e)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "hooks", "git-guardrails.sh")); !os.IsNotExist(err) {
		t.Errorf("git-guardrails script should be deleted on remove, stat err = %v", err)
	}
	root = map[string]any{}
	_ = json.Unmarshal(mustRead(t, settings), &root)
	pre, _ := root["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(pre) != 3 {
		t.Errorf("want 3 PreToolUse hooks after removing git-guardrails, got %d", len(pre))
	}
	// The other guardrails' scripts survive.
	if _, err := os.Stat(filepath.Join(home, ".claude", "hooks", "block-secrets.sh")); err != nil {
		t.Errorf("block-secrets script should survive selective remove: %v", err)
	}
}

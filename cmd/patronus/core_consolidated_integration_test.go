package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/profile"
)

// TestSafeGitWiresTheStrictGates is a CLASS-B test: it asserts the REAL catalog's
// CONTENTS — that the `safe-git` profile (core + git guardrails) really wires the
// strict gate set. The item names ARE the assertion; renaming them to fixture names
// would produce a green tautology, so they stay real.
//
// It asserts against profile.Resolve and the LOCK — both of which read the
// catalog's SHAPE and never its PINS. No install, no fetch, no hash, no binary
// placed. (safe-git extends core, which wires the gitleaks + tk recipes whose pins
// are REAL upstream digests; a --deploy here would have to fetch upstream bytes in
// CI, which is forbidden.)
//
// What it deliberately does NOT assert is the resulting settings.json — how many
// entries land in each hook array. Two reasons: that needs a --deploy, and the plan
// cannot stand in for it because the plan under-reports settings.json MERGEs
// (pat-resg). The compose-FOLD itself is proven on the fixture below, where the
// hooks are ours.
func TestSafeGitWiresTheStrictGates(t *testing.T) {
	cat := realCatalog(t)
	r, err := profile.Resolve(cat, "safe-git", "claude")
	if err != nil {
		t.Fatalf("resolve safe-git: %v", err)
	}
	names := strings.Join(r.Names(), " ")
	for _, want := range []string{
		"git-guardrails", "block-secrets", "gitleaks-guard",
		"skills-dispatch-activate", "ccusage-statusline",
	} {
		if !strings.Contains(names, want) {
			t.Errorf("safe-git should wire the strict gate %q, got: %s", want, names)
		}
	}
}

// TestSafeGitLockPinsTheGateItems is CLASS B: the lock records the hook + setting
// items, not just skills and instructions. `lock` resolves and pins from the
// catalog — it never fetches a binary — so this stays off the fetch path too.
func TestSafeGitLockPinsTheGateItems(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "safe-git", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	lock := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	// (tdd-guard-hook is an opt-in via tdd-enforced, not in safe-git, so it is
	// correctly absent from this lock.)
	for _, want := range []string{"git-guardrails", "block-secrets", "gitleaks-guard", "skills-dispatch-activate", "ccusage-statusline"} {
		if !strings.Contains(lock, want) {
			t.Errorf("lock missing strict-gate item %q:\n%s", want, lock)
		}
	}
}

// TestHooksFoldIntoOneArrayAndRemoveSelectively is the CLASS-A counterpart, on the
// FIXTURE: the deploy MECHANICS the consolidated core test used to prove, now
// asserted with items we invented — so the FETCH path (download -> verify ->
// extract -> place) actually RUNS, which stubBinary skipped entirely.
//
//   - Two hooks on one event fold into ONE settings.json array (the compose-fold).
//   - A script-bearing hook's script is CREATEd on disk.
//   - Remove round-trips: the removed hook's script is DELETEd and its settings
//     element stripped, while its sibling's script and element survive.
func TestHooksFoldIntoOneArrayAndRemoveSelectively(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "--profile", "fix-all", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}

	settings := filepath.Join(home, ".claude", "settings.json")
	if n := preToolUseCount(t, settings); n != 2 {
		t.Errorf("want 2 PreToolUse hooks folded into one array, got %d", n)
	}
	script := filepath.Join(home, ".claude", "hooks", "fix-hook-2.sh")
	if _, err := os.Stat(script); err != nil {
		t.Errorf("the script-bearing hook's script was not created: %v", err)
	}

	// Remove ONE hook: its script is deleted and its settings element stripped…
	if _, e, err := execRemove(t, "fix-hook-2", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-hook-2: %v\n%s", err, e)
	}
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Errorf("fix-hook-2's script should be deleted on remove, stat err = %v", err)
	}
	// …and its sibling survives in the array.
	if n := preToolUseCount(t, settings); n != 1 {
		t.Errorf("want 1 PreToolUse hook after removing fix-hook-2, got %d", n)
	}
}

// preToolUseCount reads a Claude settings.json and returns how many PreToolUse hook
// entries it holds.
func preToolUseCount(t *testing.T, settings string) int {
	t.Helper()
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings.json: %v", err)
	}
	hooks, _ := root["hooks"].(map[string]any)
	pre, _ := hooks["PreToolUse"].([]any)
	return len(pre)
}

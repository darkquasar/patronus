package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

	if _, e, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
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

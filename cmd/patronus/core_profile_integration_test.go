package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// A profile's CONTENTS — which catalog items it names — are reviewed in the PR that
// edits the profile YAML, not asserted here: a test that mirrors catalog item names
// back at the catalog is a tautology (like unit-testing that a package appears in a
// registry). The generic invariant that every profile resolves and is not hollow is
// covered structurally in profile_structure_test.go. What lives here are genuine
// Patronus MECHANISM tests, proven on the FIXTURE catalog where the FETCH/deploy path
// actually runs with items and binaries this suite invents.

// TestScalarSettingRemoveRoundTrips is the CLASS-A counterpart, on the FIXTURE: a
// scalar SETTING merges into settings.json and removes cleanly — the key is gone, and
// the sibling hooks in the same file survive. That is the mechanism ccusage-statusline
// relies on, proven with an item this suite invented, deployed for real.
func TestScalarSettingRemoveRoundTrips(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install profile: %v\n%s", err, e)
	}
	if _, e, err := runInstall(t, "fix-setting", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install fix-setting: %v\n%s", err, e)
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	if !strings.Contains(string(mustRead(t, settings)), "fixtureLine") {
		t.Fatal("the scalar setting was not merged on install")
	}

	if _, e, err := execRemove(t, "fix-setting", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-setting: %v\n%s", err, e)
	}
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings corrupt after remove: %v", err)
	}
	if _, present := root["fixtureLine"]; present {
		t.Errorf("the scalar setting should be gone after remove: %v", root["fixtureLine"])
	}
	// The hooks survive — removing the scalar setting did not clobber them.
	if _, ok := root["hooks"].(map[string]any)["PreToolUse"].([]any); !ok {
		t.Errorf("hooks should survive removing the scalar setting: %v", root["hooks"])
	}
}

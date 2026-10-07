package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These cover the tiered JIT re-grounding hooks: skills-heartbeat (UserPromptSubmit,
// per-turn) and work-state-reground (SessionStart, resume/compaction). Both are
// Claude-only (targets:[claude]) and wired into core flavoured as @claude, so they
// land on Claude and are silently skipped on codex/opencode.
//
// Two classes:
//
//   - CLASS B: "the real `core` profile really wires skills-heartbeat and
//     work-state-reground, @claude-flavoured." The names ARE the assertion, so they
//     stay real — asserted against profile.Resolve, which reads the catalog's SHAPE
//     and never its PINS. (core wires the gitleaks + tk recipes, whose pins are real
//     upstream digests; a --deploy of core cannot be done offline once classifyFetch
//     hashes archive binaries. See test-surface-plan.md, Task 9.)
//
//   - CLASS A: the MECHANISMS — a @claude-flavoured hook lands on claude and is
//     skipped on codex/opencode; hooks on one event compose into one settings array;
//     a placed hook script actually RUNS and enumerates the installed skills. Those
//     are asserted on the FIXTURE, with our own items, deployed for real.

// TestClaudeOnlyHookSkipsOtherTools is CLASS A on the FIXTURE: a @claude-flavoured
// hook registers in Claude's settings.json with its script placed and executable,
// and is silently skipped on codex/opencode — no error, no hook, no script.
func TestClaudeOnlyHookSkipsOtherTools(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		f := fixtureRegistry(t)
		home := withRemoteEnv(t, f)

		if _, e, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
			t.Fatalf("install: %v\n%s", err, e)
		}

		root := map[string]any{}
		if err := json.Unmarshal(mustRead(t, filepath.Join(home, ".claude", "settings.json")), &root); err != nil {
			t.Fatalf("settings.json: %v", err)
		}
		hooks, _ := root["hooks"].(map[string]any)

		// The per-turn hook registers under UserPromptSubmit with NO matcher (it fires
		// every turn) — the shape skills-heartbeat relies on.
		ups, _ := hooks["UserPromptSubmit"].([]any)
		if len(ups) != 1 {
			t.Fatalf("want 1 UserPromptSubmit hook, got %d: %v", len(ups), hooks["UserPromptSubmit"])
		}
		if _, hasMatcher := ups[0].(map[string]any)["matcher"]; hasMatcher {
			t.Errorf("a UserPromptSubmit hook should carry no matcher (it fires every turn): %v", ups[0])
		}

		// The placed script exists and is executable.
		info, err := os.Stat(filepath.Join(home, ".claude", "hooks", "fix-hook-claude.sh"))
		if err != nil {
			t.Fatalf("hook script not placed: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("hook script not executable: %v", info.Mode())
		}
	})

	for _, tool := range []string{"codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			f := fixtureRegistry(t)
			home := withRemoteEnv(t, f)

			if _, e, err := runInstall(t, "--profile", "fix-all", "--target", tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install on %s: %v\n%s", tool, err, e)
			}
			// The claude-only hook's script must NOT be placed: the flavour skipped it.
			if _, err := os.Stat(filepath.Join(home, ".claude", "hooks", "fix-hook-claude.sh")); !os.IsNotExist(err) {
				t.Errorf("%s: the claude-only hook script should not be placed (stat err=%v)", tool, err)
			}
		})
	}
}

// TestPlacedHookScriptRunsAndListsSkills is CLASS A on the FIXTURE: a hook script
// that install PLACED actually runs against the deployed tree and sees the skills
// that were installed beside it. That is the behavior skills-heartbeat depends on
// (it re-asserts the dispatch rule each turn and names what is available).
//
// Asserting it on the fixture is STRONGER than asserting it on core: there, the
// script listing "tdd" only proves tdd exists somewhere; here, the only skill in
// the tree is one this test installed, so the script must genuinely enumerate the
// deployed directory to find it.

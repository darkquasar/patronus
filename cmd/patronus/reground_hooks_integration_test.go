package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runPlacedHook executes a hook script that install placed under
// <home>/.claude/hooks/, with HOME pointed at the test home so the script's own
// ${HOME}/.claude/... lookups resolve to the deployed tree. Returns its stdout.
func runPlacedHook(t *testing.T, home, script string) (string, error) {
	t.Helper()
	p := filepath.Join(home, ".claude", "hooks", script)
	cmd := exec.CommandContext(context.Background(), "bash", p)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	return string(out), err
}

// These cover the tiered JIT re-grounding hooks: skills-heartbeat (UserPromptSubmit,
// per-turn) and work-state-reground (SessionStart, resume/compaction). Both are
// Claude-only (targets:[claude]) and wired into core flavoured as @claude, so they
// land on Claude and are silently skipped on codex/opencode.

// TestRegroundHooksLandOnClaude proves both re-grounding hooks register in
// settings.json under their respective events, with their scripts placed.
func TestRegroundHooksLandOnClaude(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks")
	stubBinary(t, home, "bd")

	if _, e, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install core: %v\n%s", err, e)
	}

	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, filepath.Join(home, ".claude", "settings.json")), &root); err != nil {
		t.Fatalf("settings.json: %v", err)
	}
	hooks, _ := root["hooks"].(map[string]any)

	// The per-turn heartbeat is registered under UserPromptSubmit (no matcher).
	ups, _ := hooks["UserPromptSubmit"].([]any)
	if len(ups) != 1 {
		t.Fatalf("want 1 UserPromptSubmit hook, got %d: %v", len(ups), hooks["UserPromptSubmit"])
	}
	if _, hasMatcher := ups[0].(map[string]any)["matcher"]; hasMatcher {
		t.Errorf("UserPromptSubmit hook should have no matcher (fires every turn): %v", ups[0])
	}

	// Both placed scripts exist and are executable.
	for _, name := range []string{"skills-heartbeat.sh", "work-state-reground.sh"} {
		p := filepath.Join(home, ".claude", "hooks", name)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("re-grounding script %q not placed: %v", name, err)
			continue
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("%s not executable: %v", name, info.Mode())
		}
	}

	// work-state-reground is a SECOND SessionStart group alongside the keystone
	// activation (both on the same event, composed into one array).
	ss, _ := hooks["SessionStart"].([]any)
	if len(ss) != 2 {
		t.Errorf("want 2 SessionStart hooks (dispatch-activate + work-state-reground), got %d", len(ss))
	}
}

// TestRegroundHooksSkipCodexOpencode proves the @claude flavour keeps the
// re-grounding hooks off codex/opencode (which have no wired hook surface), so a
// core install on those tools simply doesn't carry them — no error, no hook.
func TestRegroundHooksSkipCodexOpencode(t *testing.T) {
	for _, tool := range []string{"codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)
			withFakeRunner(t)
			stubBinary(t, home, "gitleaks")
			stubBinary(t, home, "bd")

			if _, e, err := runInstall(t, "--profile", "core", "--tool", tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install core on %s: %v\n%s", tool, err, e)
			}
			// The placed Claude hook scripts must NOT appear under a codex/opencode
			// install — the flavour skipped the hooks entirely.
			for _, name := range []string{"skills-heartbeat.sh", "work-state-reground.sh"} {
				if _, err := os.Stat(filepath.Join(home, ".claude", "hooks", name)); !os.IsNotExist(err) {
					t.Errorf("%s: Claude re-grounding script %q should not be placed (stat err=%v)", tool, name, err)
				}
			}
		})
	}
}

// TestSkillsHeartbeatListsInstalledSkills runs the placed heartbeat script against
// the deployed skills dir and asserts it emits the rule plus the installed skill
// names — the behavior that keeps the agent aware of what's available each turn.
func TestSkillsHeartbeatListsInstalledSkills(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks")
	stubBinary(t, home, "bd")

	if _, e, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install core: %v\n%s", err, e)
	}

	out, err := runPlacedHook(t, home, "skills-heartbeat.sh")
	if err != nil {
		t.Fatalf("running heartbeat script: %v", err)
	}
	var emitted struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &emitted); err != nil {
		t.Fatalf("heartbeat output is not valid JSON: %v\n%s", err, out)
	}
	ctx := emitted.HookSpecificOutput.AdditionalContext
	if !strings.Contains(ctx, "1%") {
		t.Errorf("heartbeat should re-assert the 1%% skill rule: %q", ctx)
	}
	// core installs these skills; the heartbeat should name them.
	for _, want := range []string{"tdd", "grilling", "skills-dispatch"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("heartbeat should list installed skill %q: %q", want, ctx)
		}
	}
}

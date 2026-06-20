package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These drive the real `core` profile (the opinionated default, shipped as of
// P7.2-L2/L4) end-to-end against the built catalog: the L1 instruction spine +
// diagram-explain output-style, the vendored L2 capability skills (superpowers +
// mattpocock subset), and the L4 design-vocabulary skills. They are the §6b
// acceptance gate for this sub-phase — real build → served catalog → temp-dir
// install across all three tools, idempotent re-run, lock + remove round-trip.

// coreSkills are the vendored L2+L4 skill artifacts core wires that land as a
// per-tool skills/<name>/SKILL.md file.
var coreSkills = []string{
	"superpowers-bootstrap", "writing-plans", "executing-plans",
	"grilling", "diagnosing-bugs", "tdd",
	"codebase-design", "domain-modeling",
	"verification-before-completion", // P7.5.2 L8 eval skill
}

// withFakeRunner swaps the self-wiring EXEC runner for a process-free fake (core
// wires memory-ai-memory, whose post-install shells out to the ai-memory binary)
// and restores it on cleanup, keeping the suite offline + process-free.
func withFakeRunner(t *testing.T) {
	t.Helper()
	prev := runnerForCommands
	runnerForCommands = &fakeRunner{}
	t.Cleanup(func() { runnerForCommands = prev })
}

func TestCoreProfileClaude(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks") // core's gitleaks recipe FETCH SKIPs (offline)

	if _, errOut, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// Strict default #1: diagram-explain → a Claude output-styles FILE carrying the
	// keep-coding-instructions frontmatter (ASCII diagrams on by default).
	style := filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")
	sb, err := os.ReadFile(style)
	if err != nil {
		t.Fatalf("output-style not created: %v", err)
	}
	if !strings.Contains(string(sb), "keep-coding-instructions: true") {
		t.Errorf("output-style missing strict frontmatter:\n%s", sb)
	}

	// Strict default #2: the tdd skill is present (test-first discipline ships in
	// core even before the L8 tdd-guard hook ENFORCES it in a later sub-phase).
	for _, name := range coreSkills {
		p := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("skill %q not created at %s: %v", name, p, err)
		}
	}

	// Both L1 instruction artifacts land as distinct fenced sections in ONE CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb := string(mustRead(t, claudeMd))
	for _, want := range []string{"patronus:start agents-spine", "patronus:start agent-rules"} {
		if !strings.Contains(cb, want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}

	// State records every resolved item (instructions, output-style, all skills).
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	for _, want := range append([]string{"agents-spine", "agent-rules", "diagram-explain"}, coreSkills...) {
		if !strings.Contains(st, want) {
			t.Errorf("state missing %q:\n%s", want, st)
		}
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Remove tdd ONLY → its skill dir is gone, the others survive.
	if _, errOut, err := execRemove(t, "tdd", "--global", "--deploy"); err != nil {
		t.Fatalf("remove tdd: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("tdd skill should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "grilling", "SKILL.md")); err != nil {
		t.Errorf("grilling skill should survive selective remove: %v", err)
	}
}

// TestCoreProfileFlavoursDivergeForCodexOpencode proves diagram-explain diverges
// per tool (an AGENTS.md block, not a Claude output-styles file) while the vendored
// skills still land as skills/<name>/SKILL.md under each tool's root.
func TestCoreProfileFlavoursDivergeForCodexOpencode(t *testing.T) {
	for _, tc := range []struct {
		tool, agentsRel, skillsRel string
	}{
		{"codex", filepath.Join(".codex", "AGENTS.md"), filepath.Join(".codex", "skills")},
		{"opencode", filepath.Join(".config", "opencode", "AGENTS.md"), filepath.Join(".config", "opencode", "skills")},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)
			withFakeRunner(t)
			stubBinary(t, home, "gitleaks") // core's gitleaks recipe FETCH SKIPs (offline)

			if _, errOut, err := runInstall(t, "--profile", "core", "--tool", tc.tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install: %v\n%s", err, errOut)
			}
			// No Claude output-styles file for these tools.
			if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")); err == nil {
				t.Errorf("%s must not write a Claude output-styles file", tc.tool)
			}
			// diagram-explain + both instructions all land as AGENTS.md sections.
			body := string(mustRead(t, filepath.Join(home, tc.agentsRel)))
			for _, want := range []string{
				"patronus:start agents-spine",
				"patronus:start agent-rules",
				"patronus:start diagram-explain",
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s AGENTS.md missing %q:\n%s", tc.tool, want, body)
				}
			}
			// The vendored skills land under this tool's skills root.
			for _, name := range coreSkills {
				p := filepath.Join(home, tc.skillsRel, name, "SKILL.md")
				if _, err := os.Stat(p); err != nil {
					t.Errorf("%s skill %q not created at %s: %v", tc.tool, name, p, err)
				}
			}
		})
	}
}

// TestCoreProfileLockPinsEveryItem locks core and asserts the lock pins each
// resolved item per-item (the L1 instructions + every vendored skill), so a
// re-install against the committed lock is reproducible.
func TestCoreProfileLockPinsEveryItem(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "core", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	if !strings.Contains(s, `"version": 2`) {
		t.Errorf("lock should be schema v2:\n%s", s)
	}
	for _, want := range append([]string{"agents-spine", "agent-rules", "diagram-explain", "tarballSha256"}, coreSkills...) {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q:\n%s", want, s)
		}
	}
}

// TestCoreStrictGate is the §6b acceptance for P7.5.2: core's L8 strict gate.
// On Claude the tdd-guard-hook MERGEs a PreToolUse matcher-group into
// settings.json (invoking the `tdd-guard` command), the install-only tdd-guard
// recipe surfaces its npm install as a display-only ADVISORY (never run), the
// verification skill installs, and removing the hook strips exactly its element.
func TestCoreStrictGate(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks") // core's gitleaks recipe FETCH SKIPs (offline)

	out, errOut, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--deploy", "--yes")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// The tdd-guard binary install is surfaced as an ADVISORY (Patronus never runs
	// a global npm install itself), not an executed EXEC.
	if !strings.Contains(out, "ADVISORY") || !strings.Contains(out, "npm install -g tdd-guard") {
		t.Errorf("expected an ADVISORY row for the tdd-guard npm install:\n%s", out)
	}

	// The enforcement hook MERGEd into settings.json under PreToolUse, invoking tdd-guard.
	settings := filepath.Join(home, ".claude", "settings.json")
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings.json unreadable: %v", err)
	}
	// core's L8+L9 PreToolUse hooks all coexist in ONE settings.json array (the
	// compose-fold): tdd-guard-hook + block-secrets + gitleaks-guard + git-guardrails.
	pre, _ := root["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(pre) != 4 {
		t.Fatalf("want 4 coexisting PreToolUse groups (tdd-guard + 3 guardrails), got %d: %v", len(pre), root["hooks"])
	}
	// Find the tdd-guard enforcement group by its command.
	var tdd map[string]any
	for _, g := range pre {
		grp := g.(map[string]any)
		if grp["hooks"].([]any)[0].(map[string]any)["command"] == "tdd-guard" {
			tdd = grp
		}
	}
	if tdd == nil {
		t.Fatalf("tdd-guard enforcement hook not found among the PreToolUse groups: %v", pre)
	}
	if tdd["matcher"] != "Write|Edit|MultiEdit|TodoWrite" {
		t.Errorf("tdd-guard matcher = %v, want the enforcement matcher", tdd["matcher"])
	}

	// The verification skill landed.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "verification-before-completion", "SKILL.md")); err != nil {
		t.Errorf("verification skill not installed: %v", err)
	}

	// Idempotent re-run → SKIP (the hook merge is a no-op the second time).
	reout, _, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reout, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", reout)
	}

	// Removing ONLY the tdd-guard-hook strips exactly its element; the three L9
	// guardrail hooks survive (targeted removal preserves siblings).
	if _, errOut, err := execRemove(t, "tdd-guard-hook", "--global", "--deploy"); err != nil {
		t.Fatalf("remove tdd-guard-hook: %v\n%s", err, errOut)
	}
	root = map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings.json gone/corrupt after remove: %v", err)
	}
	pre, _ = root["hooks"].(map[string]any)["PreToolUse"].([]any)
	if len(pre) != 3 {
		t.Fatalf("want 3 guardrail hooks surviving after removing tdd-guard, got %d: %v", len(pre), pre)
	}
	for _, g := range pre {
		if g.(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"] == "tdd-guard" {
			t.Error("tdd-guard hook should be gone after remove")
		}
	}
}

// TestNoTddGuardOverlayDropsEnforcement proves the relaxation overlay: no-tdd-guard
// installs everything core does EXCEPT the enforcement hook + its binary recipe,
// while KEEPING the tdd skill (test-first as guidance, not a hard block).
func TestNoTddGuardOverlayDropsEnforcement(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks") // no-tdd-guard keeps the gitleaks guardrail

	out, errOut, err := runInstall(t, "--profile", "no-tdd-guard", "--tool", "claude", "--global", "--deploy", "--yes")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// No tdd-guard hook in settings, and no npm advisory (the recipe was subtracted).
	if strings.Contains(out, "npm install -g tdd-guard") {
		t.Errorf("no-tdd-guard should drop the tdd-guard recipe advisory:\n%s", out)
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	if b, err := os.ReadFile(settings); err == nil && strings.Contains(string(b), "tdd-guard") {
		t.Errorf("no-tdd-guard should not register the enforcement hook:\n%s", b)
	}

	// ...but the tdd SKILL (guidance) and verification skill still install.
	for _, skill := range []string{"tdd", "verification-before-completion"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", skill, "SKILL.md")); err != nil {
			t.Errorf("no-tdd-guard should keep the %q skill: %v", skill, err)
		}
	}
}

// TestCoreGuardrails is the §6b acceptance for P7.5.3: core's L9 guardrail set.
// A dry-run (no network) asserts the plan carries the three guardrail hooks
// (block-secrets + gitleaks-guard + git-guardrails, each MERGEd into Claude
// settings.json), the two script-bearing hooks place their helper scripts, and
// the gitleaks recipe contributes a github-release FETCH for the binary. The
// FETCH *download* mechanism is proven offline separately (install/fetch_test.go).
func TestCoreGuardrails(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	out, _, err := runInstall(t, "--profile", "core", "--tool", "claude", "--global", "--dry-run", "--verbose")
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}

	// All three guardrail hooks + the two placed scripts + the gitleaks FETCH show
	// up in the plan.
	for _, want := range []string{
		"block-secrets",
		"gitleaks-guard",
		"git-guardrails",
		"git-guardrails.sh", // a placed hook script (named after the artifact)
		"block-secrets.sh",  // the regex guard's placed script
		"gitleaks",          // the FETCH row for the binary
	} {
		if !strings.Contains(out, want) {
			t.Errorf("core dry-run plan missing %q:\n%s", want, out)
		}
	}
	// The gitleaks binary fetch is a FETCH action.
	if !strings.Contains(out, "FETCH") {
		t.Errorf("expected a FETCH row for the gitleaks binary:\n%s", out)
	}
}

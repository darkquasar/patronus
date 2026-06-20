package main

import (
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

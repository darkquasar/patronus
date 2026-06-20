package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These drive the real `visual` profile (shipped in-repo as of P7.2-L1) end-to-end
// against the built catalog: the vendored L1 instructions (agents-spine, agent-rules)
// and the authored diagram-explain output-style. They prove the output-style flavour
// diverges per tool AND that two instruction artifacts sharing one AGENTS.md/CLAUDE.md
// both land, record state, and remove independently (the composed-APPEND fix).

func TestVisualProfileClaude(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "--profile", "visual", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// diagram-explain → a Claude output-styles FILE (CREATE), carrying the strict
	// keep-coding-instructions frontmatter.
	style := filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")
	sb, err := os.ReadFile(style)
	if err != nil {
		t.Fatalf("output-style not created: %v", err)
	}
	if !strings.Contains(string(sb), "keep-coding-instructions: true") {
		t.Errorf("output-style missing strict frontmatter:\n%s", sb)
	}

	// Both instruction artifacts land as distinct fenced sections in ONE CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb, err := os.ReadFile(claudeMd)
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	for _, want := range []string{"patronus:start agents-spine", "patronus:start agent-rules"} {
		if !strings.Contains(string(cb), want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}

	// State records BOTH instruction artifacts (the composed-APPEND fix), not just
	// the first — otherwise remove would leak the second section.
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	for _, want := range []string{"agents-spine", "agent-rules", "diagram-explain"} {
		if !strings.Contains(st, want) {
			t.Errorf("state missing %q:\n%s", want, st)
		}
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "visual", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Remove agent-rules ONLY → its section is stripped, agents-spine survives.
	if _, errOut, err := execRemove(t, "agent-rules", "--global", "--deploy"); err != nil {
		t.Fatalf("remove agent-rules: %v\n%s", err, errOut)
	}
	cb2 := string(mustRead(t, claudeMd))
	if strings.Contains(cb2, "patronus:start agent-rules") {
		t.Errorf("agent-rules section should be gone:\n%s", cb2)
	}
	if !strings.Contains(cb2, "patronus:start agents-spine") {
		t.Errorf("agents-spine section should survive selective remove:\n%s", cb2)
	}
}

func TestVisualProfileOutputStyleDivergesForCodexOpencode(t *testing.T) {
	for _, tc := range []struct {
		tool, agentsRel string
	}{
		{"codex", filepath.Join(".codex", "AGENTS.md")},
		{"opencode", filepath.Join(".config", "opencode", "AGENTS.md")},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t, "--profile", "visual", "--tool", tc.tool, "--global", "--deploy", "--yes"); err != nil {
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
		})
	}
}

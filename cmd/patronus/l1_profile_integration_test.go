package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic profile placement, state and selective removal across adapters.
func TestVisualProfileClaude(t *testing.T) {
	root := fixtureCatalog(t)
	codexWrite(t, filepath.Join(root, "profiles/fix-visual.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nname: fix-visual\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  instructions: [fix-instruction-global, fix-instruction-2]\n  capabilities: [fix-skill, fix-skill-project]\n"))
	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "--profile", "fix-visual", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// fix-instruction-2 is an instruction, so nothing lands under output-styles/ and
	// nothing needs selecting: the body appends into CLAUDE.md, live from install.
	if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "fix-instruction-2.md")); err == nil {
		t.Error("fix-instruction-2 must not write a Claude output-styles file")
	}

	// Both instructions land as fenced sections in CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb, err := os.ReadFile(claudeMd)
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	for _, want := range []string{"patronus:start fix-instruction-global", "patronus:start fix-instruction-2"} {
		if !strings.Contains(string(cb), want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}

	// The design-discipline skills land as standalone SKILL.md files (CREATE), not
	// as CLAUDE.md sections — dispatch, not inlining.
	for _, name := range []string{"fix-skill", "fix-skill-project"} {
		skill := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(skill); err != nil {
			t.Errorf("skill %q not created at %s: %v", name, skill, err)
		}
	}

	// State records the spine, the two skills, and the output-style.
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	for _, want := range []string{"fix-instruction-global", "fix-skill", "fix-skill-project", "fix-instruction-2"} {
		if !strings.Contains(st, want) {
			t.Errorf("state missing %q:\n%s", want, st)
		}
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "fix-visual", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Remove the fix-skill skill ONLY → its dir is gone, fix-instruction-global's CLAUDE.md
	// section and the sibling skill both survive.
	if _, errOut, err := execRemove(t, "fix-skill", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-skill: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")); err == nil {
		t.Errorf("fix-skill skill should be gone after remove")
	}
	cb2 := string(mustRead(t, claudeMd))
	if !strings.Contains(cb2, "patronus:start fix-instruction-global") {
		t.Errorf("fix-instruction-global section should survive an unrelated remove:\n%s", cb2)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "fix-skill-project", "SKILL.md")); err != nil {
		t.Errorf("fix-skill-project skill should survive the sibling's remove: %v", err)
	}
}

// The regression guard for the retype: codex and opencode route BOTH instruction
// and output-style to the same AGENTS.md appendSection target, so the change from
// one type to the other must be invisible here. Both instructions land as fenced
// AGENTS.md sections, and nothing appears under Claude's output-styles/.
//
// The output-style type's per-target divergence (CREATE on Claude, APPEND
// elsewhere) is still proven end-to-end, against the `smoke-style` fixture in
// outputstyle_integration_test.go. It is not proven here any more, because no
// real catalog artifact carries that type.
func TestVisualProfileAppendsDiagramExplainForCodexOpencode(t *testing.T) {
	for _, tc := range []struct {
		tool, agentsRel string
	}{
		{"codex", filepath.Join(".codex", "AGENTS.md")},
		{"opencode", filepath.Join(".config", "opencode", "AGENTS.md")},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			root := fixtureCatalog(t)
			codexWrite(t, filepath.Join(root, "profiles/fix-visual.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nname: fix-visual\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  instructions: [fix-instruction-global, fix-instruction-2]\n  capabilities: [fix-skill, fix-skill-project]\n"))
			f := serveFixtureFrom(t, root)
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t, "--profile", "fix-visual", "--target", tc.tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install: %v\n%s", err, errOut)
			}
			// No Claude output-styles file for these tools.
			if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "fix-instruction-2.md")); err == nil {
				t.Errorf("%s must not write a Claude output-styles file", tc.tool)
			}
			// The fix-instruction-2 and fix-instruction-global instructions both land as AGENTS.md
			// sections. The design-discipline skills are CREATEd under skills/, not
			// appended here.
			body := string(mustRead(t, filepath.Join(home, tc.agentsRel)))
			for _, want := range []string{
				"patronus:start fix-instruction-global",
				"patronus:start fix-instruction-2",
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s AGENTS.md missing %q:\n%s", tc.tool, want, body)
				}
			}
		})
	}
}

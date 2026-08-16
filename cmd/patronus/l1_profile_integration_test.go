package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These drive the real `visual` profile (shipped in-repo as of P7.2-L1) end-to-end
// against the built catalog: the vendored L1 spine (agents-spine), the authored
// diagram-explain instruction, and the design-discipline skills (ddd-distilled,
// refactoring-distilled). They prove that the two instructions append identically
// on every target, and that an APPENDed instruction section is recorded in state
// and removed cleanly.
//
// (The composed-APPEND fix — TWO instructions sharing one CLAUDE.md, removed
// independently — is proven against the fixture catalog by
// TestComposedAppendRemovesSelectively, where the payload is invented bytes. This
// real-catalog test no longer carries that proof: `visual`'s only CLAUDE.md APPEND
// on claude is agents-spine, since the design-discipline guidance is skills now.)
//
// CLASS B, and it STAYS deployed on the real catalog. The item names ARE the
// assertion ("the visual profile really ships agents-spine, the two design skills
// and diagram-explain"), so renaming them to fixture names would produce a green
// tautology — exactly what test-surface-plan.md exists to prevent.
//
// It is safe here where the other real-catalog profile tests were not, because
// `visual` wires NO recipe: it is three artifacts and nothing else. Verified, not
// assumed — a full `--profile visual --deploy` makes 5 in-memory fetcher hits
// (index + sha sidecar + 3 artifact tarballs) and ZERO binary fetches. It therefore
// never reads a recipe PIN, never hashes upstream bytes, and cannot be broken by
// classifyFetch hashing archive binaries.
//
// If `visual` ever gains a recipe, this test must move off --deploy (see how
// golang/hardened/core_consolidated were split) — a real-catalog test may read the
// catalog's SHAPE, never its PINS.

func TestVisualProfileClaude(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "--profile", "visual", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// diagram-explain is an instruction, so nothing lands under output-styles/ and
	// nothing needs selecting: the body appends into CLAUDE.md, live from install.
	if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")); err == nil {
		t.Error("diagram-explain must not write a Claude output-styles file")
	}

	// Both instructions land as fenced sections in CLAUDE.md.
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb, err := os.ReadFile(claudeMd)
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	for _, want := range []string{"patronus:start agents-spine", "patronus:start diagram-explain"} {
		if !strings.Contains(string(cb), want) {
			t.Errorf("CLAUDE.md missing %q:\n%s", want, cb)
		}
	}

	// The design-discipline skills land as standalone SKILL.md files (CREATE), not
	// as CLAUDE.md sections — dispatch, not inlining.
	for _, name := range []string{"ddd-distilled", "refactoring-distilled"} {
		skill := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(skill); err != nil {
			t.Errorf("skill %q not created at %s: %v", name, skill, err)
		}
	}

	// State records the spine, the two skills, and the output-style.
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	for _, want := range []string{"agents-spine", "ddd-distilled", "refactoring-distilled", "diagram-explain"} {
		if !strings.Contains(st, want) {
			t.Errorf("state missing %q:\n%s", want, st)
		}
	}

	// Idempotent re-run.
	out, _, err := runInstall(t, "--profile", "visual", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// Remove the ddd-distilled skill ONLY → its dir is gone, agents-spine's CLAUDE.md
	// section and the sibling skill both survive.
	if _, errOut, err := execRemove(t, "ddd-distilled", "--global", "--deploy"); err != nil {
		t.Fatalf("remove ddd-distilled: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "ddd-distilled", "SKILL.md")); err == nil {
		t.Errorf("ddd-distilled skill should be gone after remove")
	}
	cb2 := string(mustRead(t, claudeMd))
	if !strings.Contains(cb2, "patronus:start agents-spine") {
		t.Errorf("agents-spine section should survive an unrelated remove:\n%s", cb2)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "refactoring-distilled", "SKILL.md")); err != nil {
		t.Errorf("refactoring-distilled skill should survive the sibling's remove: %v", err)
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
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t, "--profile", "visual", "--target", tc.tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install: %v\n%s", err, errOut)
			}
			// No Claude output-styles file for these tools.
			if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")); err == nil {
				t.Errorf("%s must not write a Claude output-styles file", tc.tool)
			}
			// The diagram-explain and agents-spine instructions both land as AGENTS.md
			// sections. The design-discipline skills are CREATEd under skills/, not
			// appended here.
			body := string(mustRead(t, filepath.Join(home, tc.agentsRel)))
			for _, want := range []string{
				"patronus:start agents-spine",
				"patronus:start diagram-explain",
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s AGENTS.md missing %q:\n%s", tc.tool, want, body)
				}
			}
		})
	}
}

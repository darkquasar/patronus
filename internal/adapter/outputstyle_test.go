package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func outputStyleArtifact() *manifest.Artifact {
	return &manifest.Artifact{
		Meta:  manifest.Meta{Family: manifest.FamilyArtifact, Name: "diagram-explain", Role: manifest.RoleInstruction},
		Type:  manifest.TypeOutputStyle,
		Entry: "STYLE.md",
	}
}

func writeStyle(t *testing.T, body string) string {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "STYLE.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return src
}

// On Claude the output-style is a standalone CREATE under output-styles/, body
// passed through verbatim (frontmatter included).
func TestTransformOutputStyleClaudeCreates(t *testing.T) {
	body := "---\nkeep-coding-instructions: true\n---\nDraw ASCII diagrams."
	src := writeStyle(t, body)
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))

	diffs, err := eng.Transform(outputStyleArtifact(), loadAdapter(t, "claude"), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff, got %d", len(diffs))
	}
	d := diffs[0]
	want := filepath.Join(home, ".claude", "output-styles", "diagram-explain.md")
	if d.Path != want {
		t.Errorf("path = %q, want %q", d.Path, want)
	}
	if d.Action != diff.Create {
		t.Errorf("action = %s, want CREATE", d.Action)
	}
	if string(d.After) != body {
		t.Errorf("body not passed through verbatim: %q", d.After)
	}
	if d.Type != "output-style" || d.Tool != "claude" || d.Scope != "global" || d.Role != "instruction" {
		t.Errorf("metadata wrong: %+v", d)
	}
}

func TestTransformOutputStyleClaudeProjectPath(t *testing.T) {
	src := writeStyle(t, "body")
	home, proj := t.TempDir(), t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, proj))

	diffs, err := eng.Transform(outputStyleArtifact(), loadAdapter(t, "claude"), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(proj, ".claude", "output-styles", "diagram-explain.md")
	if diffs[0].Path != want {
		t.Errorf("project path = %q, want %q", diffs[0].Path, want)
	}
}

// On Codex and OpenCode the output-style has no native surface, so it APPENDs a
// fenced section into AGENTS.md — idempotent, prose-preserving.
func TestTransformOutputStyleAppendsAgentsMd(t *testing.T) {
	for _, tool := range []string{"codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			src := writeStyle(t, "Draw ASCII diagrams.")
			home, proj := t.TempDir(), t.TempDir()
			eng := New(toolpath.New(testEnv(home), home, proj))

			// Seed AGENTS.md with the user's own prose to prove it survives.
			agents := filepath.Join(proj, "AGENTS.md")
			if err := os.WriteFile(agents, []byte("# My rules\n\nkeep it tidy\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			read := func(p string) ([]byte, bool, error) {
				b, err := os.ReadFile(p)
				if os.IsNotExist(err) {
					return nil, false, nil
				}
				if err != nil {
					return nil, false, err
				}
				return b, true, nil
			}

			diffs, err := eng.Transform(outputStyleArtifact(), loadAdapter(t, tool), "local", src, read)
			if err != nil {
				t.Fatal(err)
			}
			if len(diffs) != 1 {
				t.Fatalf("want 1 diff, got %d", len(diffs))
			}
			d := diffs[0]
			if d.Path != agents {
				t.Errorf("path = %q, want %q", d.Path, agents)
			}
			if d.Action != diff.Append {
				t.Errorf("action = %s, want APPEND", d.Action)
			}
			s := string(d.After)
			if !strings.Contains(s, "<!-- patronus:start diagram-explain -->") {
				t.Errorf("missing fenced section:\n%s", s)
			}
			if !strings.Contains(s, "# My rules") || !strings.Contains(s, "keep it tidy") {
				t.Errorf("user prose not preserved:\n%s", s)
			}
			if d.Section == nil || d.Section.Name != "diagram-explain" {
				t.Errorf("section edit not recorded: %+v", d.Section)
			}

			// Idempotent: re-applying onto the produced output yields no change.
			read2 := func(string) ([]byte, bool, error) { return d.After, true, nil }
			again, err := eng.Transform(outputStyleArtifact(), loadAdapter(t, tool), "local", src, read2)
			if err != nil {
				t.Fatal(err)
			}
			if string(again[0].After) != s {
				t.Errorf("not idempotent:\n once: %q\ntwice: %q", s, again[0].After)
			}
		})
	}
}

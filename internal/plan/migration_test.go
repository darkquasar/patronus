package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

func codexMigrationInput(t *testing.T) (CodexMigrationInput, string, string) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	write := func(path, body string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	skill := "---\nname: fixture-cx\ndescription: invented\n---\n"
	gsrc := filepath.Join(home, ".codex/skills/fixture-cx/SKILL.md")
	lsrc := filepath.Join(project, ".codex/skills/fixture-cx/SKILL.md")
	write(gsrc, skill)
	write(lsrc, skill)
	row := func(scope, path string) *state.State {
		items := state.FromChangeSet([]diff.FileDiff{{Path: path, Action: diff.Create, After: []byte(skill), Artifact: "fixture-cx", Type: "skill", Tool: "codex", Scope: scope}}, "t")
		return &state.State{Version: state.Version, Items: items}
	}
	in := CodexMigrationInput{
		Scope:  CodexMigrationScope{Scope: "local", Legacy: []string{filepath.Join(project, ".codex/skills")}, Dest: filepath.Join(project, ".agents/skills")},
		States: map[string]*state.State{"global": row("global", gsrc), "local": row("local", lsrc)},
		Roots:  scan.CodexSkillRoots(home, project, filepath.Join(home, ".codex")),
	}
	return in, home, project
}

// Both scopes are inspected: a verified owned same-name skill in the other
// scope is admitted, while structured, foreign or ambiguous rows refuse.
func TestCodexSkillMigrationPlanIdentityAndBothScopes(t *testing.T) {
	in, _, project := codexMigrationInput(t)
	moves, err := CodexSkillMigration(in)
	if err != nil || len(moves) != 1 || len(moves[0].Writes) != 1 || len(moves[0].Retire) != 1 {
		t.Fatalf("verified both-scope plan: %v %+v", err, moves)
	}
	if got := moves[0].Final.Files[0].Path; got != filepath.Join(project, ".agents/skills/fixture-cx/SKILL.md") {
		t.Fatalf("final ownership %s", got)
	}
	if len(moves[0].Staged.Files) != 2 {
		t.Fatalf("staged ownership must keep old and new rows: %+v", moves[0].Staged.Files)
	}
	for name, mutate := range map[string]func(in *CodexMigrationInput){
		"structured-row":   func(in *CodexMigrationInput) { in.States["local"].Items[0].Files[0].Action = "MERGE" },
		"missing-checksum": func(in *CodexMigrationInput) { in.States["local"].Items[0].Files[0].Checksum = "" },
		"duplicate-row": func(in *CodexMigrationInput) {
			s := in.States["local"]
			s.Items = append(s.Items, s.Items[0])
		},
		"outside-skill-dir": func(in *CodexMigrationInput) {
			it := &in.States["local"].Items[0]
			it.Files = append(it.Files, state.FileState{Path: filepath.Join(project, ".codex/config.toml"), Action: "CREATE", Checksum: "sha256:00"})
		},
		"foreign-tool": func(in *CodexMigrationInput) { in.States["local"].Items[0].Tool = "pi" },
		"other-scope-edited": func(in *CodexMigrationInput) {
			in.States["global"].Items[0].Files[0].Checksum = "sha256:00"
		},
		"name-mismatch": func(in *CodexMigrationInput) {
			path := in.States["local"].Items[0].Files[0].Path
			raw := []byte("---\nname: other-cx\ndescription: invented\n---\n")
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			in.States["local"].Items[0].Files[0].Checksum = codexSum(raw)
		},
	} {
		t.Run(name, func(t *testing.T) {
			in, _, _ := codexMigrationInput(t)
			project = filepath.Dir(filepath.Dir(in.Scope.Dest))
			mutate(&in)
			if moves, err := CodexSkillMigration(in); err == nil || !strings.Contains(err.Error(), "preserved") && !strings.Contains(err.Error(), "foreign") {
				t.Fatalf("refusal expected: %v %+v", err, moves)
			}
		})
	}
}

package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envOf(pairs map[string]string) EnvLookup {
	return func(k string) (string, bool) {
		v, ok := pairs[k]
		return v, ok
	}
}

// TestOpenCodeSkillRootsHonourEnv: the global base follows OPENCODE_CONFIG_DIR,
// then XDG_CONFIG_HOME/opencode, then ~/.config/opencode; the project roots are
// the four OpenCode reads; no root is legacy; both Claude roots are recursive.
func TestOpenCodeSkillRootsHonourEnv(t *testing.T) {
	home, project := "/invented/home", "/invented/project"
	for _, tc := range []struct {
		name string
		env  map[string]string
		base string
	}{
		{"OPENCODE_CONFIG_DIR", map[string]string{"OPENCODE_CONFIG_DIR": "/invented/cfg", "XDG_CONFIG_HOME": "/invented/xdg"}, "/invented/cfg"},
		{"XDG_CONFIG_HOME", map[string]string{"XDG_CONFIG_HOME": "/invented/xdg"}, "/invented/xdg/opencode"},
		{"neither", nil, "/invented/home/.config/opencode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roots := OpenCodeSkillRoots(home, project, envOf(tc.env))
			var paths []string
			for _, r := range roots {
				paths = append(paths, r.Path)
				if r.Legacy || r.DuplicatePolicy != RefuseDifferentBytesVerifiedException || r.Name == nil || r.ConflictMessage == nil {
					t.Fatalf("root %s rules: %+v", r.Path, r)
				}
				if claude := strings.HasSuffix(r.Path, "/.claude/skills"); claude != r.Recursive {
					t.Fatalf("root %s recursive = %v", r.Path, r.Recursive)
				}
			}
			want := []string{
				project + "/.opencode/skills", project + "/.opencode/skill", project + "/.claude/skills", project + "/.agents/skills",
				tc.base + "/skills", tc.base + "/skill", home + "/.claude/skills", home + "/.agents/skills",
			}
			if strings.Join(paths, ",") != strings.Join(want, ",") {
				t.Fatalf("roots\n got: %v\nwant: %v", paths, want)
			}
		})
	}
}

// TestOpenCodeClaudeRootsAreRecursive: a skill nested under ~/.claude/skills
// (synced/<id>/<name>/SKILL.md) is discovered; an identical copy visible from
// two roots is one resource (same bytes); differing bytes under the same name
// are refused with the OpenCode wording.
func TestOpenCodeClaudeRootsAreRecursive(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const demo = "---\nname: demo\ndescription: invented\n---\nbody\n"
	write(".claude/skills/synced/abc123/demo/SKILL.md", demo)
	roots := OpenCodeSkillRoots(home, project, envOf(nil))
	found, err := DiscoverSkills(roots, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "demo" || !strings.HasSuffix(found[0].Path, "synced/abc123/demo/SKILL.md") {
		t.Fatalf("nested discovery: %+v", found)
	}
	write(".config/opencode/skills/demo/SKILL.md", demo)
	if found, err = DiscoverSkills(roots, nil); err != nil || len(found) != 2 {
		t.Fatalf("identical bytes twice must be admitted: %v %+v", err, found)
	}
	write(".config/opencode/skills/demo/SKILL.md", "---\nname: demo\ndescription: different\n---\nother\n")
	_, err = DiscoverSkills(roots, nil)
	if err == nil || !strings.HasPrefix(err.Error(), `opencode skill name "demo" conflicts at `) || !strings.HasSuffix(err.Error(), "; remove one copy or rename") {
		t.Fatalf("conflict wording: %v", err)
	}
}

// TestSkillDirName: the neutral name == directory rule carries no target
// wording and names the mismatch.
func TestSkillDirName(t *testing.T) {
	if name, err := SkillDirName([]byte("---\nname: demo\ndescription: x\n---\n"), "/r/demo/SKILL.md"); err != nil || name != "demo" {
		t.Fatalf("match: %q %v", name, err)
	}
	_, err := SkillDirName([]byte("---\nname: other\ndescription: x\n---\n"), "/r/demo/SKILL.md")
	if err == nil || err.Error() != `skill /r/demo/SKILL.md: frontmatter name "other" must equal directory "demo"` {
		t.Fatalf("mismatch: %v", err)
	}
}

// TestEnvSnapshotNeverRecordsConfigContent: the snapshot records the
// OPENCODE_CONFIG path and only the presence of OPENCODE_CONFIG_CONTENT.
func TestEnvSnapshotNeverRecordsConfigContent(t *testing.T) {
	snap := envSnapshot(envOf(map[string]string{"OPENCODE_CONFIG": "/invented/override.json", "OPENCODE_CONFIG_CONTENT": `{"provider":{"x":{"options":{"apiKey":"fixture-not-a-secret"}}}}`}))
	if snap.OpencodeConfig != "/invented/override.json" || !snap.OpencodeConfigContentSet {
		t.Fatalf("snapshot: %+v", snap)
	}
	if strings.Contains(fmt.Sprintf("%+v", snap), "fixture-not-a-secret") {
		t.Fatalf("snapshot carries the inline content: %+v", snap)
	}
	if snap := envSnapshot(envOf(map[string]string{"OPENCODE_CONFIG_CONTENT": ""})); snap.OpencodeConfigContentSet {
		t.Fatal("empty content counted as set")
	}
}

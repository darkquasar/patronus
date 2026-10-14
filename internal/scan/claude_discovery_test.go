package scan

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeSkillRootsDiscovery(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	roots := ClaudeSkillRoots(home, project)
	for i, want := range []string{filepath.Join(project, ".claude/skills"), filepath.Join(home, ".claude/skills")} {
		if roots[i].Path != want || roots[i].Legacy || !roots[i].Recursive {
			t.Fatalf("Claude root: %+v", roots[i])
		}
	}
	if found, err := DiscoverSkills(roots, nil); err != nil || len(found) != 0 {
		t.Fatalf("absent roots: %+v %v", found, err)
	}
	for i, name := range []string{"local-cl", "global-cl"} {
		writeResource(t, filepath.Join(roots[i].Path, "synced/id", name, "SKILL.md"), "---\nname: "+name+"\ndescription: invented\n---\nbody\n")
	}
	found, err := DiscoverSkills(roots, nil)
	if err != nil || len(found) != 2 || found[0].Name != "local-cl" || found[1].Name != "global-cl" {
		t.Fatalf("scoped nested discovery: %+v %v", found, err)
	}
	writeResource(t, filepath.Join(roots[0].Path, "wrong-cl/SKILL.md"), "---\nname: other-cl\n---\n")
	if _, err := DiscoverSkills(roots, nil); err == nil || !strings.Contains(err.Error(), "must equal directory") {
		t.Fatalf("mismatched identity: %v", err)
	}
}

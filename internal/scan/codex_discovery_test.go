package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexPreflightDiscoveryInspectsNativeAndLegacyRoots(t *testing.T) {
	home, project, config := t.TempDir(), t.TempDir(), t.TempDir()
	roots := CodexSkillRoots(home, project, config)
	for _, path := range []string{filepath.Join(home, ".codex/skills"), filepath.Join(home, ".agents/skills"), filepath.Join(config, "skills"), filepath.Join(project, ".agents/skills"), filepath.Join(project, ".codex/skills")} {
		found := false
		for _, r := range roots {
			found = found || r.Path == path
		}
		if !found {
			t.Fatal("uninspected candidate", path)
		}
	}
	root := filepath.Join(project, ".agents/skills")
	file := filepath.Join(root, "fixture-cx/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("---\nname: wrong-cx\ndescription: invented\n---\nfixture\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverCodexSkills(roots); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("malformed identity admitted: %v", err)
	}
}

func TestCodexPreflightDiscoveryRefusesIncompatibleNamesAndSymlinks(t *testing.T) {
	for _, mode := range []string{"incompatible", "symlink", "sidecar-symlink", "bad-name"} {
		t.Run(mode, func(t *testing.T) {
			home, project, config := t.TempDir(), t.TempDir(), t.TempDir()
			if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
				t.Fatal(err)
			}
			first := filepath.Join(home, ".agents/skills/fixture-cx/SKILL.md")
			second := filepath.Join(project, ".agents/skills/fixture-cx/SKILL.md")
			for _, path := range []string{first, second} {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("---\nname: fixture-cx\ndescription: invented\n---\nfixture\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := DiscoverCodexSkills(CodexSkillRoots(home, project, config)); err != nil {
				t.Fatal("identical native copies incompatible", err)
			}
			switch mode {
			case "incompatible":
				if err := os.WriteFile(second, []byte("---\nname: fixture-cx\ndescription: invented\n---\ndifferent\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "bad-name":
				if err := os.WriteFile(second, []byte("---\nname: Bad_Name\ndescription: invented\n---\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(second); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(first, second); err != nil {
					t.Fatal(err)
				}
			case "sidecar-symlink":
				if err := os.Symlink(first, filepath.Join(filepath.Dir(second), "helper.md")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := DiscoverCodexSkills(CodexSkillRoots(home, project, config)); err == nil {
				t.Fatal("unsafe candidate admitted", mode)
			}
		})
	}
}

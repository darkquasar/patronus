package adapter

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func testEnv(home string) toolpath.EnvLookup {
	return func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}
}

// claudeAdapter loads the real claude adapter from the repo.
func claudeAdapter(t *testing.T) *manifest.Adapter {
	t.Helper()
	ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", "claude.yaml"))
	if err != nil {
		t.Fatalf("load claude adapter: %v", err)
	}
	return ad
}

func noExisting(string) ([]byte, bool, error) { return nil, false, nil }

func TestTransformSkillPassthrough(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("SKILL BODY"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Kind: manifest.KindSkill, Name: "team-research", Role: manifest.RoleCapability, Entry: "SKILL.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff, got %d", len(diffs))
	}
	d := diffs[0]
	wantPath := filepath.Join(home, ".claude", "skills", "team-research", "SKILL.md")
	if d.Path != wantPath {
		t.Errorf("path = %q, want %q", d.Path, wantPath)
	}
	if string(d.After) != "SKILL BODY" {
		t.Errorf("content = %q, want verbatim passthrough", d.After)
	}
	if d.Action != diff.Create || d.Tool != "claude" || d.Scope != "global" || d.Role != "capability" {
		t.Errorf("metadata wrong: %+v", d)
	}
}

func TestTransformSkillWithFilesDir(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), "index")
	mustWrite(t, filepath.Join(src, "patterns", "pattern-001.md"), "p1")
	mustWrite(t, filepath.Join(src, "patterns", "pattern-002.md"), "p2")
	mustWrite(t, filepath.Join(src, "patterns", "nested", "deep.md"), "deep")

	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Kind: manifest.KindSkill, Name: "pattern-cloudflare", Role: manifest.RolePattern, Entry: "SKILL.md", Files: []string{"patterns/"}}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	// 1 SKILL.md + 3 pattern files (incl. nested).
	if len(diffs) != 4 {
		t.Fatalf("want 4 diffs, got %d: %+v", len(diffs), paths(diffs))
	}
	base := filepath.Join(home, ".claude", "skills", "pattern-cloudflare")
	wantPaths := []string{
		filepath.Join(base, "SKILL.md"),
		filepath.Join(base, "patterns", "pattern-001.md"),
		filepath.Join(base, "patterns", "pattern-002.md"),
		filepath.Join(base, "patterns", "nested", "deep.md"),
	}
	got := paths(diffs)
	sort.Strings(got)
	sort.Strings(wantPaths)
	for i := range wantPaths {
		if got[i] != wantPaths[i] {
			t.Errorf("path[%d] = %q, want %q", i, got[i], wantPaths[i])
		}
	}
}

func TestTransformSkillProjectScope(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), "x")
	proj := t.TempDir()
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, proj))
	art := &manifest.Artifact{Kind: manifest.KindSkill, Name: "s", Entry: "SKILL.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(proj, ".claude", "skills", "s", "SKILL.md")
	if diffs[0].Path != want {
		t.Errorf("project path = %q, want %q", diffs[0].Path, want)
	}
}

func TestTransformSkillMissingEntryErrors(t *testing.T) {
	src := t.TempDir() // no SKILL.md written
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Kind: manifest.KindSkill, Name: "s", Entry: "SKILL.md"}
	if _, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting); err == nil {
		t.Error("expected error for missing entry file")
	}
}

func TestTransformUnsupportedKind(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Kind: manifest.KindHook, Name: "h"}
	if _, err := eng.Transform(art, claudeAdapter(t), "global", t.TempDir(), noExisting); err == nil {
		t.Error("expected error for Hook kind (no transform)")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func paths(diffs []diff.FileDiff) []string {
	out := make([]string, len(diffs))
	for i, d := range diffs {
		out[i] = d.Path
	}
	return out
}

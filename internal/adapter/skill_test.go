package adapter

import (
	"bytes"
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
	return loadAdapter(t, "claude")
}

func noExisting(string) ([]byte, bool, error) { return nil, false, nil }

func TestTransformSkillPassthrough(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("SKILL BODY"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "team-research", Role: manifest.RoleCapability}, Type: manifest.TypeSkill, Entry: "SKILL.md"}

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
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "pattern-cloudflare", Role: manifest.RoleContext}, Type: manifest.TypeSkill, Entry: "SKILL.md", Files: []string{"patterns/"}}

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
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s"}, Type: manifest.TypeSkill, Entry: "SKILL.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(proj, ".claude", "skills", "s", "SKILL.md")
	if diffs[0].Path != want {
		t.Errorf("project path = %q, want %q", diffs[0].Path, want)
	}
}

// tokenBody exercises both tokens plus an unknown one that must survive.
const tokenBody = "run {skillDir}/scripts/verify.sh and {skillsDir}/sibling/serve.sh with {skilDir} and ${VAR}\n"

// TestTransformSkillTokensPerAdapter: at project scope the tokens resolve to a
// path relative to the project, spelled in each adapter's own layout.
func TestTransformSkillTokensPerAdapter(t *testing.T) {
	tests := []struct {
		tool      string
		skillsDir string
	}{
		{tool: "claude", skillsDir: filepath.Join(".claude", "skills")},
		{tool: "codex", skillsDir: filepath.Join(".agents", "skills")},
		{tool: "opencode", skillsDir: filepath.Join(".opencode", "skills")},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			src := t.TempDir()
			mustWrite(t, filepath.Join(src, "SKILL.md"), tokenBody)
			mustWrite(t, filepath.Join(src, "scripts", "run.sh"), tokenBody)

			home, proj := t.TempDir(), t.TempDir()
			eng := New(toolpath.New(testEnv(home), home, proj))
			art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "teach"}, Type: manifest.TypeSkill, Entry: "SKILL.md", Files: []string{"scripts/"}}

			diffs, err := eng.Transform(art, loadAdapter(t, tt.tool), "local", src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			want := "run " + filepath.Join(tt.skillsDir, "teach") + "/scripts/verify.sh and " +
				tt.skillsDir + "/sibling/serve.sh with {skilDir} and ${VAR}\n"
			// Both the entry body and the files: tree substitute.
			if len(diffs) != 2 {
				t.Fatalf("want 2 diffs, got %d: %v", len(diffs), paths(diffs))
			}
			for _, d := range diffs {
				if string(d.After) != want {
					t.Errorf("%s content = %q, want %q", d.Path, d.After, want)
				}
			}
		})
	}
}

// TestTransformSkillTokensGlobalScopeAbsolute: at global scope there is no
// meaningful relative form, so the absolute install path is used.
func TestTransformSkillTokensGlobalScopeAbsolute(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), "at {skillDir}\n")

	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "teach"}, Type: manifest.TypeSkill, Entry: "SKILL.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := "at " + filepath.Join(home, ".claude", "skills", "teach") + "\n"
	if string(diffs[0].After) != want {
		t.Errorf("content = %q, want %q", diffs[0].After, want)
	}
}

// TestTransformSkillNoTokensByteIdentical: a body carrying no token is
// unchanged, so the substitution is inert for every existing skill.
func TestTransformSkillNoTokensByteIdentical(t *testing.T) {
	const body = "plain body with {\"json\": 1} and f\"{value}\"\n"
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), body)

	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s"}, Type: manifest.TypeSkill, Entry: "SKILL.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if string(diffs[0].After) != body {
		t.Errorf("content = %q, want byte-identical %q", diffs[0].After, body)
	}
}

// TestTransformSkillNonUTF8FileUntouched: a binary asset under files: is copied
// unchanged, by construction rather than by luck.
func TestTransformSkillNonUTF8FileUntouched(t *testing.T) {
	binary := []byte{0xff, 0xfe, '{', 's', 'k', 'i', 'l', 'l', 'D', 'i', 'r', '}', 0x00}
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), "x")
	if err := os.MkdirAll(filepath.Join(src, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "assets", "logo.bin"), binary, 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s"}, Type: manifest.TypeSkill, Entry: "SKILL.md", Files: []string{"assets/"}}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diffs {
		if filepath.Base(d.Path) == "logo.bin" && !bytes.Equal(d.After, binary) {
			t.Errorf("binary asset = %v, want untouched %v", d.After, binary)
		}
	}
}

// TestTransformSkillTokensIdempotent: transforming twice yields identical bytes,
// so a re-install is a no-op rather than a compounding rewrite.
func TestTransformSkillTokensIdempotent(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "SKILL.md"), tokenBody)

	home, proj := t.TempDir(), t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, proj))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s"}, Type: manifest.TypeSkill, Entry: "SKILL.md"}

	first, err := eng.Transform(art, claudeAdapter(t), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	second, err := eng.Transform(art, claudeAdapter(t), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first[0].After, second[0].After) {
		t.Errorf("not idempotent: %q then %q", first[0].After, second[0].After)
	}
}

func TestTransformSkillMissingEntryErrors(t *testing.T) {
	src := t.TempDir() // no SKILL.md written
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s"}, Type: manifest.TypeSkill, Entry: "SKILL.md"}
	if _, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting); err == nil {
		t.Error("expected error for missing entry file")
	}
}

func TestTransformUnsupportedKind(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "h"}, Type: manifest.TypeHook}
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

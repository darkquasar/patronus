package lock

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	l := &Lock{
		Version:   Version,
		Profile:   "cloudflare",
		Generated: "2026-06-07T00:00:00Z",
		Entries: []Entry{
			{Name: "team-research", Source: "registry", Version: "1.0.0", SHA256: "sha256:abc", Slot: "capabilities", Kind: "artifact"},
		},
	}
	path := filepath.Join(t.TempDir(), "patronus.lock")
	if err := Save(path, l); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, l) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, l)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "absent.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != Version || len(got.Entries) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestSaveDeterministic(t *testing.T) {
	l := &Lock{Version: Version, Profile: "p", Generated: "fixed", Entries: []Entry{
		{Name: "b", Source: "registry", SHA256: "sha256:2"},
		{Name: "a", Source: "registry", SHA256: "sha256:1"},
	}}
	dir := t.TempDir()
	p1 := filepath.Join(dir, "1.lock")
	p2 := filepath.Join(dir, "2.lock")
	if err := Save(p1, l); err != nil {
		t.Fatal(err)
	}
	if err := Save(p2, l); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(p1)
	b2, _ := os.ReadFile(p2)
	if string(b1) != string(b2) {
		t.Fatal("save is not deterministic")
	}
}

func TestFromResolvedSortsAndProvenance(t *testing.T) {
	// Build a fake catalog with on-disk artifact content so hashing has inputs.
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "team-research")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# body")

	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{
			Manifest: &manifest.Artifact{Name: "team-research", Version: "1.0.0", Entry: "SKILL.md"},
			Source:   registry.Source{LocalDir: skillDir},
		}},
		Recipes: []registry.RecipeEntry{{
			Manifest: &manifest.Recipe{Name: "memory-ai-memory", Capability: "memory"},
		}},
	}
	r := &profile.Resolved{
		Profile: &manifest.Profile{Name: "p"},
		Items: []profile.ResolvedItem{
			{Name: "memory-ai-memory", Slot: "memory", Kind: profile.KindRecipe, Source: "registry"},
			{Name: "team-research", Slot: "capabilities", Kind: profile.KindArtifact, Source: "registry"},
		},
	}
	l, err := FromResolved(cat, r, "2026-06-07T00:00:00Z", "")
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by name: memory-ai-memory before team-research.
	if l.Entries[0].Name != "memory-ai-memory" || l.Entries[1].Name != "team-research" {
		t.Fatalf("entries not sorted: %+v", l.Entries)
	}
	for _, e := range l.Entries {
		if e.Source != "registry" {
			t.Errorf("%s: source %q", e.Name, e.Source)
		}
		if e.SHA256 == "" {
			t.Errorf("%s: empty sha256", e.Name)
		}
	}
	if l.Entries[1].Version != "1.0.0" {
		t.Errorf("artifact version = %q", l.Entries[1].Version)
	}
}

func TestHashArtifactStableAndSensitive(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "s")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "original")
	mustWrite(t, filepath.Join(skillDir, "patterns", "p1.md"), "pat")

	entry := registry.ArtifactEntry{
		Manifest: &manifest.Artifact{Name: "s", Version: "1.0.0", Entry: "SKILL.md", Files: []string{"patterns"}},
		Source:   registry.Source{LocalDir: skillDir},
	}
	h1, err := hashArtifact(entry)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := hashArtifact(entry)
	if h1 != h2 {
		t.Fatal("hashArtifact not stable")
	}

	// Changing a byte in a files: member changes the digest.
	mustWrite(t, filepath.Join(skillDir, "patterns", "p1.md"), "changed")
	h3, _ := hashArtifact(entry)
	if h3 == h1 {
		t.Fatal("hashArtifact insensitive to content change")
	}
}

func TestHashRecipeStableAndSensitive(t *testing.T) {
	e1 := registry.RecipeEntry{Manifest: &manifest.Recipe{Name: "r", Capability: "memory", Summary: "a"}}
	h1, err := hashRecipe(e1)
	if err != nil {
		t.Fatal(err)
	}
	if h2, _ := hashRecipe(e1); h1 != h2 {
		t.Fatal("hashRecipe not stable")
	}
	e2 := registry.RecipeEntry{Manifest: &manifest.Recipe{Name: "r", Capability: "memory", Summary: "b"}}
	if h3, _ := hashRecipe(e2); h3 == h1 {
		t.Fatal("hashRecipe insensitive to manifest change")
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

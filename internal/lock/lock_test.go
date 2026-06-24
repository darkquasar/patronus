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
			Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "team-research", Version: "1.0.0"}, Entry: "SKILL.md"},
			// LocalDir drives the content-fold hash; TarballURL/SHA256 mirror a
			// remote-resolved entry so the lock pins the tarball bytes too.
			Source: registry.Source{LocalDir: skillDir, TarballURL: "https://x/catalog/team-research/1.0.0/team-research-1.0.0.tar.gz", SHA256: "sha256:tarbytes"},
		}},
		Recipes: []registry.RecipeEntry{{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "memory-ai-memory", Role: "memory"}},
		}},
	}
	r := &profile.Resolved{
		Profile: &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}},
		Items: []profile.ResolvedItem{
			{Name: "memory-ai-memory", Slot: "memory", Family: manifest.FamilyRecipe, Source: "registry"},
			{Name: "team-research", Slot: "capabilities", Family: manifest.FamilyArtifact, Source: "registry"},
		},
	}
	l, err := FromResolved(cat, r, "2026-06-07T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if l.Version != 2 {
		t.Errorf("lock version = %d, want 2", l.Version)
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
	art := l.Entries[1]
	if art.Version != "1.0.0" {
		t.Errorf("artifact version = %q", art.Version)
	}
	if art.TarballSha256 != "sha256:tarbytes" {
		t.Errorf("artifact tarballSha256 = %q, want sha256:tarbytes", art.TarballSha256)
	}
}

func TestHashArtifactStableAndSensitive(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "s")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "original")
	mustWrite(t, filepath.Join(skillDir, "patterns", "p1.md"), "pat")

	entry := registry.ArtifactEntry{
		Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "s", Version: "1.0.0"}, Entry: "SKILL.md", Files: []string{"patterns"}},
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
	e1 := registry.RecipeEntry{Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "r", Role: "memory"}, Summary: "a"}}
	h1, err := hashRecipe(e1)
	if err != nil {
		t.Fatal(err)
	}
	if h2, _ := hashRecipe(e1); h1 != h2 {
		t.Fatal("hashRecipe not stable")
	}
	e2 := registry.RecipeEntry{Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "r", Role: "memory"}, Summary: "b"}}
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

func TestLockRoundTripsPluginEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.lock")
	in := &Lock{Version: Version, Entries: []Entry{{
		Name: "superpowers", Source: "registry", Version: "2.1.0",
		SHA256: "sha256:deadbeef", Kind: "plugin",
	}}}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out.Entries) != 1 || out.Entries[0].Kind != "plugin" {
		t.Fatalf("entries = %+v, want one kind=plugin", out.Entries)
	}
	if out.Entries[0].Name != "superpowers" {
		t.Errorf("name = %s, want superpowers", out.Entries[0].Name)
	}
}

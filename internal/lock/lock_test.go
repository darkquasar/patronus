package lock

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func TestEntryStatusRoundTrips(t *testing.T) {
	l := &Lock{Version: Version, Entries: []Entry{
		{Name: "superpowers", Kind: "plugin", Source: "registry", SHA256: "sha256:x", Status: StatusUnverified},
	}}
	dir := t.TempDir()
	p := dir + "/patronus.lock"
	if err := Save(p, l); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Status != StatusUnverified {
		t.Errorf("status = %q, want %q", got.Entries[0].Status, StatusUnverified)
	}
}

func TestFromResolvedWritesPluginUnverified(t *testing.T) {
	cat := &registry.Catalog{
		Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
			Meta: manifest.Meta{Family: manifest.FamilyPlugin, Name: "superpowers", Version: "2.1.0"},
		}}},
	}
	res := &profile.Resolved{
		Profile: &manifest.Profile{Meta: manifest.Meta{Name: "p"}},
		Items: []profile.ResolvedItem{
			{Name: "superpowers", Family: manifest.FamilyPlugin, Source: "registry"},
		},
	}
	l, err := FromResolved(cat, res, "2026-06-26T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	var e *Entry
	for i := range l.Entries {
		if l.Entries[i].Name == "superpowers" {
			e = &l.Entries[i]
		}
	}
	if e == nil {
		t.Fatalf("no superpowers entry: %+v", l.Entries)
	}
	if e.Status != StatusUnverified {
		t.Errorf("plugin status = %q, want %q", e.Status, StatusUnverified)
	}
	if e.Kind != "plugin" || e.Version != "2.1.0" {
		t.Errorf("entry = %+v, want kind=plugin version=2.1.0", *e)
	}
}

func TestArtifactEntryOmitsStatus(t *testing.T) {
	l := &Lock{Version: Version, Entries: []Entry{{Name: "a", Kind: "artifact", SHA256: "sha256:y"}}}
	dir := t.TempDir()
	p := dir + "/patronus.lock"
	if err := Save(p, l); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "\"status\"") {
		t.Errorf("artifact entry must omit status (omitempty): %s", data)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	l := &Lock{
		Version:   Version,
		Profile:   "cloudflare",
		Generated: "2026-06-07T00:00:00Z",
		Entries: []Entry{
			{Name: "research-team", Source: "registry", Version: "1.0.0", SHA256: "sha256:abc", Slot: "capabilities", Kind: "artifact"},
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
	skillDir := filepath.Join(dir, "research-team")
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# body")

	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{
			Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "research-team", Version: "1.0.0"}, Entry: "SKILL.md"},
			// LocalDir drives the content-fold hash; TarballURL/SHA256 mirror a
			// remote-resolved entry so the lock pins the tarball bytes too.
			Source: registry.Source{LocalDir: skillDir, TarballURL: "https://x/catalog/research-team/1.0.0/research-team-1.0.0.tar.gz", SHA256: "sha256:tarbytes"},
		}},
		Recipes: []registry.RecipeEntry{{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "memory-ai-memory", Role: "memory"}},
		}},
	}
	r := &profile.Resolved{
		Profile: &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}},
		Items: []profile.ResolvedItem{
			{Name: "memory-ai-memory", Slot: "memory", Family: manifest.FamilyRecipe, Source: "registry"},
			{Name: "research-team", Slot: "capabilities", Family: manifest.FamilyArtifact, Source: "registry"},
		},
	}
	l, err := FromResolved(cat, r, "2026-06-07T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if l.Version != 2 {
		t.Errorf("lock version = %d, want 2", l.Version)
	}
	// Sorted by name: memory-ai-memory before research-team.
	if l.Entries[0].Name != "memory-ai-memory" || l.Entries[1].Name != "research-team" {
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

func TestRecipeVersionPinned(t *testing.T) {
	recipe := &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "kit", Version: "1.2.3"}}
	cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: recipe}}}
	resolved := &profile.Resolved{Profile: &manifest.Profile{}, Items: []profile.ResolvedItem{{Name: "kit", Family: manifest.FamilyRecipe}}}
	l, err := FromResolved(cat, resolved, "")
	if err != nil {
		t.Fatal(err)
	}
	if l.Entries[0].Version != "1.2.3" {
		t.Fatalf("recipe version = %q", l.Entries[0].Version)
	}
}

func directoryLockJSON() string {
	return `{"version":2,"entries":[{"name":"kit","kind":"recipe","version":"1.0.0","source":"registry","sha256":"sha256:` + strings.Repeat("0", 64) + `","delivery":{"via":"fetch","unpack":"directory","package":{"name":"payload","version":"2.0.0"},"assets":[{"os":"darwin","arch":"arm64","url":"https://example.test/kit.tar.gz","sha256":"` + strings.Repeat("0", 64) + `","archive":"tar.gz"}]}}]}`
}

func TestDirectoryLockRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "patronus.lock")
	mustWrite(t, path, directoryLockJSON())
	l, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(path, l); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"unpack": "directory"`) {
		t.Fatal("directory pin lost")
	}
}

func TestDirectoryLockRejectsMalformedPins(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"future schema", `"version":2`, `"version":3`},
		{"negative schema", `"version":2`, `"version":-1`},
		{"wrong kind", `"kind":"recipe"`, `"kind":"artifact"`},
		{"bad recipe name", `"name":"kit"`, `"name":"../kit"`},
		{"bad recipe version", `"version":"1.0.0"`, `"version":"01.0.0"`},
		{"bad package version", `"version":"2.0.0"`, `"version":"next"`},
		{"bad pin", "https://", "http://"},
		{"unknown unpack", `"unpack":"directory"`, `"unpack":"future"`},
		{"file metadata", `"unpack":"directory",`, ``},
		{"short digest", strings.Repeat("0", 64), "abc"},
		{"bad asset digest", `"sha256":"` + strings.Repeat("0", 64), `"sha256":"abc`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "patronus.lock")
			mustWrite(t, path, strings.Replace(directoryLockJSON(), tc.old, tc.replacement, 1))
			if _, err := Load(path); err == nil {
				t.Fatal("invalid directory lock accepted")
			}
		})
	}
}

func TestDirectoryLockCopiesResolvedDelivery(t *testing.T) {
	d := &manifest.Delivery{Via: manifest.ViaFetch, Unpack: "directory", Package: &manifest.PackageIdentity{Name: "payload", Version: "2.0.0"}, Assets: []manifest.Asset{
		{OS: "darwin", Arch: "arm64", URL: "https://example.test/darwin.tar.gz", SHA256: strings.Repeat("a", 64), Archive: "tar.gz"},
		{OS: "linux", Arch: "amd64", URL: "https://example.test/linux.tar.gz", SHA256: "sha256:" + strings.Repeat("b", 64), Archive: "tar.gz"},
	}}
	r := &manifest.Recipe{Meta: manifest.Meta{APIVersion: "patronus/v3", Family: manifest.FamilyRecipe, Role: manifest.RoleSandbox, Name: "kit", Version: "1.0.0"}, Delivery: d}
	cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: r}}}
	resolved := &profile.Resolved{Profile: &manifest.Profile{}, Items: []profile.ResolvedItem{{Name: "kit", Family: manifest.FamilyRecipe}}}
	l, err := FromResolved(cat, resolved, "")
	if err != nil {
		t.Fatal(err)
	}
	pinned := l.Entries[0].Delivery
	if !reflect.DeepEqual(pinned, d) {
		t.Fatalf("delivery changed during locking: %#v", pinned)
	}
	d.Package.Version = "3.0.0"
	d.Assets[0].URL = "https://example.test/changed.tar.gz"
	if pinned.Package.Version != "2.0.0" || pinned.Assets[0].URL != "https://example.test/darwin.tar.gz" {
		t.Fatal("catalog mutation changed lock pins")
	}
	path := filepath.Join(t.TempDir(), "patronus.lock")
	if err := Save(path, l); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, l) {
		t.Fatal("directory lock changed in round trip")
	}
}

func TestRecipeVersionLegacyDeliveryOmitted(t *testing.T) {
	r := &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "legacy", Version: "1.0.0"}, Delivery: &manifest.Delivery{Via: manifest.ViaFetch}}
	cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: r}}}
	resolved := &profile.Resolved{Profile: &manifest.Profile{}, Items: []profile.ResolvedItem{{Name: "legacy", Family: manifest.FamilyRecipe}}}
	l, err := FromResolved(cat, resolved, "")
	if err != nil {
		t.Fatal(err)
	}
	if l.Entries[0].Delivery != nil {
		t.Fatal("legacy lock gained delivery pin")
	}
}

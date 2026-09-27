package registry

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func sampleIndex() *Index {
	return &Index{
		SchemaVersion: IndexSchemaVersion,
		Generated:     "2026-06-08T00:00:00Z",
		Artifacts: []IndexArtifact{{
			Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "research-team", Version: "1.0.0"}, Type: manifest.TypeSkill},
			Tarball:  Tarball{URL: "https://x/catalog/research-team/1.0.0/research-team-1.0.0.tar.gz", SHA256: "sha256:abc"},
		}},
		Recipes: []IndexRecipe{{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "memory-ai-memory", Role: "memory"}},
		}},
		Profiles: []IndexProfile{{
			Manifest: &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "cloudflare"}},
		}},
	}
}

func TestIndexMarshalLoadRoundTrip(t *testing.T) {
	ix := sampleIndex()
	data, err := ix.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Artifacts) != 1 ||
		got.Artifacts[0].Manifest.Name != "research-team" ||
		got.Artifacts[0].Tarball.SHA256 != "sha256:abc" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestToCatalogSetsRemoteSource(t *testing.T) {
	cat := sampleIndex().ToCatalog()
	if len(cat.Artifacts) != 1 || len(cat.Recipes) != 1 || len(cat.Profiles) != 1 {
		t.Fatalf("catalog shape: %+v", cat)
	}
	src := cat.Artifacts[0].Source
	if src.TarballURL == "" || src.SHA256 == "" || src.LocalDir != "" {
		t.Fatalf("artifact source should be remote, got %+v", src)
	}
}

func TestLoadIndexRejectsNewerSchema(t *testing.T) {
	data := []byte(`{"schemaVersion": 999, "artifacts": []}`)
	if _, err := LoadIndex(data); err == nil {
		t.Fatal("expected rejection of newer schema")
	}
}

func TestMarshalDeterministic(t *testing.T) {
	ix := sampleIndex()
	a, _ := ix.Marshal()
	b, _ := ix.Marshal()
	if string(a) != string(b) {
		t.Fatal("Marshal not deterministic")
	}
}

func TestIndexToCatalogPlugins(t *testing.T) {
	ix := &Index{
		Plugins: []IndexPlugin{
			{Manifest: &manifest.Plugin{Meta: manifest.Meta{Name: "superpowers", Family: manifest.FamilyPlugin}}},
		},
	}
	cat := ix.ToCatalog()
	if len(cat.Plugins) != 1 {
		t.Fatalf("plugins = %d, want 1", len(cat.Plugins))
	}
	if cat.Plugins[0].Manifest.Name != "superpowers" {
		t.Errorf("name = %s, want superpowers", cat.Plugins[0].Manifest.Name)
	}
}

func TestIndexDirectorySchemaAndValidation(t *testing.T) {
	valid := `{"schemaVersion":2,"recipes":[{"manifest":{"apiVersion":"patronus/v3","family":"recipe","role":"sandbox","name":"kit","version":"1.0.0","deliver":{"via":"fetch","unpack":"directory","package":{"name":"payload","version":"2.0.0"},"assets":[{"os":"darwin","arch":"arm64","url":"https://example.test/kit.tar.gz","sha256":"` + strings.Repeat("0", 64) + `","archive":"tar.gz"}]}}}]}`
	ix, err := LoadIndex([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	data, err := ix.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"unpack": "directory"`) {
		t.Fatal("directory fields lost")
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"schema 1", `"schemaVersion":2`, `"schemaVersion":1`},
		{"missing schema", `"schemaVersion":2,`, ``},
		{"future schema", `"schemaVersion":2`, `"schemaVersion":3`},
		{"negative schema", `"schemaVersion":2`, `"schemaVersion":-1`},
		{"v2 directory", "patronus/v3", "patronus/v2"},
		{"future recipe", "patronus/v3", "patronus/v4"},
		{"invalid pin", "https://", "http://"},
		{"invalid name", `"name":"kit"`, `"name":"../kit"`},
		{"no directory", `"unpack":"directory",`, ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := LoadIndex([]byte(strings.Replace(valid, tc.old, tc.replacement, 1))); err == nil {
				t.Fatal("invalid directory index accepted")
			}
		})
	}
}

func TestIndexMissingVersionRemainsLegacy(t *testing.T) {
	ix, err := LoadIndex([]byte(`{"recipes":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if ix.SchemaVersion != 1 {
		t.Fatalf("legacy schema = %d", ix.SchemaVersion)
	}
}

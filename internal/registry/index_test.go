package registry

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func sampleIndex() *Index {
	return &Index{
		SchemaVersion: IndexSchemaVersion,
		Generated:     "2026-06-08T00:00:00Z",
		Artifacts: []IndexArtifact{{
			Manifest: &manifest.Artifact{Name: "team-research", Version: "1.0.0", Kind: "Skill"},
			Tarball:  Tarball{URL: "https://x/catalog/team-research/1.0.0/team-research-1.0.0.tar.gz", SHA256: "sha256:abc"},
		}},
		Recipes: []IndexRecipe{{
			Manifest: &manifest.Recipe{Name: "memory-ai-memory", Capability: "memory"},
		}},
		Profiles: []IndexProfile{{
			Manifest: &manifest.Profile{Name: "cloudflare"},
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
		got.Artifacts[0].Manifest.Name != "team-research" ||
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

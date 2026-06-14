package profile

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// fakeCatalog builds an in-memory catalog from artifact names, recipe names, and
// profiles — no filesystem, so the resolver is tested in pure isolation.
func fakeCatalog(artifacts, recipes []string, profiles ...*manifest.Profile) *registry.Catalog {
	cat := &registry.Catalog{}
	for _, n := range artifacts {
		cat.Artifacts = append(cat.Artifacts, registry.ArtifactEntry{
			Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: n}, Type: manifest.TypeSkill},
		})
	}
	for _, n := range recipes {
		cat.Recipes = append(cat.Recipes, registry.RecipeEntry{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: n, Role: "memory"}},
		})
	}
	for _, p := range profiles {
		cat.Profiles = append(cat.Profiles, registry.ProfileEntry{Manifest: p})
	}
	return cat
}

func TestResolveDispatchesArtifactsAndRecipes(t *testing.T) {
	cat := fakeCatalog(
		[]string{"team-research", "pattern-cloudflare"},
		[]string{"memory-ai-memory"},
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"team-research"},
			Context:      manifest.StringList{"pattern-cloudflare"},
			Memory:       "memory-ai-memory",
		}},
	)
	r, err := Resolve(cat, "p")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]manifest.Family{
		"team-research":      manifest.FamilyArtifact,
		"pattern-cloudflare": manifest.FamilyArtifact,
		"memory-ai-memory":   manifest.FamilyRecipe,
	}
	if len(r.Items) != len(want) {
		t.Fatalf("got %d items, want %d: %+v", len(r.Items), len(want), r.Items)
	}
	for _, it := range r.Items {
		if want[it.Name] != it.Family {
			t.Errorf("%s: family %v, want %v", it.Name, it.Family, want[it.Name])
		}
		if it.Source != "registry" {
			t.Errorf("%s: source %q, want registry", it.Name, it.Source)
		}
	}
}

func TestResolveSlotAndAuthorOrder(t *testing.T) {
	cat := fakeCatalog(
		[]string{"a", "b", "ctx"},
		[]string{"mem"},
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"a", "b"},
			Context:      manifest.StringList{"ctx"},
			Memory:       "mem",
		}},
	)
	r, _ := Resolve(cat, "p")
	got := r.Names()
	// Fixed §1A order: capabilities (a,b) before context (ctx) before memory (mem).
	want := []string{"a", "b", "ctx", "mem"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestResolveStubWarns(t *testing.T) {
	cat := fakeCatalog([]string{"a"}, nil,
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Status: "stub", Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"a"},
		}},
	)
	r, _ := Resolve(cat, "p")
	if !hasWarning(r.Warnings, "stub") {
		t.Fatalf("expected stub warning, got %v", r.Warnings)
	}
}

func TestResolveUnknownNameWarnsAndSkips(t *testing.T) {
	cat := fakeCatalog([]string{"a"}, nil,
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"a", "ghost"},
		}},
	)
	r, _ := Resolve(cat, "p")
	if len(r.Items) != 1 || r.Items[0].Name != "a" {
		t.Fatalf("ghost should be dropped, got %+v", r.Items)
	}
	if !hasWarning(r.Warnings, "ghost") {
		t.Fatalf("expected unresolved warning, got %v", r.Warnings)
	}
}

func TestResolveDedup(t *testing.T) {
	cat := fakeCatalog([]string{"a"}, nil,
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"a"},
			Context:      manifest.StringList{"a"}, // same name in two slots
		}},
	)
	r, _ := Resolve(cat, "p")
	if len(r.Items) != 1 {
		t.Fatalf("expected dedup to 1, got %+v", r.Items)
	}
}

func TestResolveExtendsAppendListsReplaceScalars(t *testing.T) {
	parent := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "base"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"team-research", "team-implement"},
		Context:      manifest.StringList{"pattern-go"},
		Memory:       "memory-ai-memory",
	}}
	child := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "derived"}, Extends: "base", Layers: manifest.ProfileLayers{
		Context: manifest.StringList{"pattern-cloudflare"}, // appends to parent's context
		Memory:  "memory-engram",                           // replaces parent's scalar
	}}
	cat := fakeCatalog(
		[]string{"team-research", "team-implement", "pattern-go", "pattern-cloudflare"},
		[]string{"memory-ai-memory", "memory-engram"},
		parent, child,
	)
	r, err := Resolve(cat, "derived")
	if err != nil {
		t.Fatal(err)
	}
	got := r.Names()
	// capabilities inherited; context = parent then child (append); memory replaced.
	want := []string{"team-research", "team-implement", "pattern-go", "pattern-cloudflare", "memory-engram"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extends result = %v, want %v", got, want)
	}
}

func TestResolveExtendsDedupsAcrossInheritance(t *testing.T) {
	parent := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "base"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"team-research"},
	}}
	child := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "derived"}, Extends: "base", Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"team-research", "team-implement"}, // restates inherited
	}}
	cat := fakeCatalog([]string{"team-research", "team-implement"}, nil, parent, child)
	r, _ := Resolve(cat, "derived")
	want := []string{"team-research", "team-implement"}
	if strings.Join(r.Names(), ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", r.Names(), want)
	}
}

func TestResolveExtendsCycle(t *testing.T) {
	a := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "a"}, Extends: "b"}
	b := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "b"}, Extends: "a"}
	cat := fakeCatalog(nil, nil, a, b)
	if _, err := Resolve(cat, "a"); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("want cycle error, got %v", err)
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	cat := fakeCatalog(nil, nil)
	if _, err := Resolve(cat, "nope"); err == nil {
		t.Fatal("want error for unknown profile")
	}
}

func hasWarning(ws []string, sub string) bool {
	for _, w := range ws {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}

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
		[]string{"demo-cap", "demo-ctx2"},
		[]string{"demo-mem"},
		&manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"demo-cap"},
			Context:      manifest.StringList{"demo-ctx2"},
			Memory:       "demo-mem",
		}},
	)
	r, err := Resolve(cat, "p", "all")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]manifest.Family{
		"demo-cap":  manifest.FamilyArtifact,
		"demo-ctx2": manifest.FamilyArtifact,
		"demo-mem":  manifest.FamilyRecipe,
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
	r, _ := Resolve(cat, "p", "all")
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
	r, _ := Resolve(cat, "p", "all")
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
	r, _ := Resolve(cat, "p", "all")
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
	r, _ := Resolve(cat, "p", "all")
	if len(r.Items) != 1 {
		t.Fatalf("expected dedup to 1, got %+v", r.Items)
	}
}

func TestResolveExtendsAppendListsReplaceScalars(t *testing.T) {
	parent := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "base"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"demo-cap", "demo-cap2"},
		Context:      manifest.StringList{"demo-ctx"},
		Memory:       "demo-mem",
	}}
	child := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "derived"}, Extends: "base", Layers: manifest.ProfileLayers{
		Context: manifest.StringList{"demo-ctx2"}, // appends to parent's context
		Memory:  "demo-mem2",                      // replaces parent's scalar
	}}
	cat := fakeCatalog(
		[]string{"demo-cap", "demo-cap2", "demo-ctx", "demo-ctx2"},
		[]string{"demo-mem", "demo-mem2"},
		parent, child,
	)
	r, err := Resolve(cat, "derived", "all")
	if err != nil {
		t.Fatal(err)
	}
	got := r.Names()
	// capabilities inherited; context = parent then child (append); memory replaced.
	want := []string{"demo-cap", "demo-cap2", "demo-ctx", "demo-ctx2", "demo-mem2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("extends result = %v, want %v", got, want)
	}
}

func TestResolveExtendsDedupsAcrossInheritance(t *testing.T) {
	parent := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "base"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"demo-cap"},
	}}
	child := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "derived"}, Extends: "base", Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"demo-cap", "demo-cap2"}, // restates inherited
	}}
	cat := fakeCatalog([]string{"demo-cap", "demo-cap2"}, nil, parent, child)
	r, _ := Resolve(cat, "derived", "all")
	want := []string{"demo-cap", "demo-cap2"}
	if strings.Join(r.Names(), ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", r.Names(), want)
	}
}

func TestResolveExtendsCycle(t *testing.T) {
	a := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "a"}, Extends: "b"}
	b := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "b"}, Extends: "a"}
	cat := fakeCatalog(nil, nil, a, b)
	if _, err := Resolve(cat, "a", "all"); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("want cycle error, got %v", err)
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	cat := fakeCatalog(nil, nil)
	if _, err := Resolve(cat, "nope", "all"); err == nil {
		t.Fatal("want error for unknown profile")
	}
}

// flavourProfile wires a slot with a bare name plus per-tool flavours, so each
// tool resolves to exactly one of the flavoured items (plus the bare base).
func flavourProfile() *registry.Catalog {
	p := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"base-cap"},
		Guardrails:   manifest.StringList{"sandbox-native@claude", "sandbox-native@codex", "sandbox-runtime@opencode"},
	}}
	return fakeCatalog([]string{"base-cap", "sandbox-native", "sandbox-runtime"}, nil, p)
}

func TestResolveFlavourSelectsMatchingTool(t *testing.T) {
	cat := flavourProfile()

	// claude: bare base + sandbox-native (the @claude flavour); NOT sandbox-runtime.
	r, err := Resolve(cat, "p", "claude")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Names(), ","); got != "base-cap,sandbox-native" {
		t.Errorf("claude flavour = %q, want base-cap,sandbox-native", got)
	}

	// opencode: bare base + sandbox-runtime; NOT sandbox-native.
	r, _ = Resolve(cat, "p", "opencode")
	if got := strings.Join(r.Names(), ","); got != "base-cap,sandbox-runtime" {
		t.Errorf("opencode flavour = %q, want base-cap,sandbox-runtime", got)
	}
}

func TestResolveAllToolDropsFlavours(t *testing.T) {
	cat := flavourProfile()
	// The tool-agnostic baseline (lock default) keeps only bare names.
	r, _ := Resolve(cat, "p", "all")
	if got := strings.Join(r.Names(), ","); got != "base-cap" {
		t.Errorf("all-tool baseline = %q, want base-cap only (flavours dropped)", got)
	}
	// "" behaves like "all".
	r, _ = Resolve(cat, "p", "")
	if got := strings.Join(r.Names(), ","); got != "base-cap" {
		t.Errorf("empty-tool baseline = %q, want base-cap only", got)
	}
}

func TestResolveFlavourDedupsBareAndFlavoured(t *testing.T) {
	// A bare name and its @tool flavour resolve to the same base — install once.
	p := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"x", "x@claude"},
	}}
	cat := fakeCatalog([]string{"x"}, nil, p)
	r, _ := Resolve(cat, "p", "claude")
	if len(r.Items) != 1 || r.Items[0].Name != "x" {
		t.Fatalf("bare + @claude should dedup to one base 'x', got %+v", r.Items)
	}
}

func TestResolveNonFlavourAtIsNotSplit(t *testing.T) {
	// An @ that isn't a known-tool suffix stays part of the base name, so it falls
	// through to the existing warn-and-skip (catalog miss) — never silently dropped.
	p := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"}, Layers: manifest.ProfileLayers{
		Capabilities: manifest.StringList{"user@example.com"},
	}}
	cat := fakeCatalog([]string{"a"}, nil, p)
	r, _ := Resolve(cat, "p", "claude")
	if len(r.Items) != 0 {
		t.Fatalf("non-flavour @ should not resolve, got %+v", r.Items)
	}
	if !hasWarning(r.Warnings, "user@example.com") {
		t.Fatalf("expected catalog-miss warning naming the full base, got %v", r.Warnings)
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

// TestResolvePullsRequiresClosure proves a profile that lists only the dependent
// item (an instruction) also resolves the item it `requires` (a binary recipe),
// ordered dependency-before-dependent — the ticket/tk pairing. The required
// recipe is pulled even though no profile slot names it.
func TestResolvePullsRequiresClosure(t *testing.T) {
	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{
			Manifest: &manifest.Artifact{
				Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "demo-instr", Requires: []string{"demo-bin"}},
				Type: manifest.TypeInstruction,
			},
		}},
		Recipes: []registry.RecipeEntry{{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "demo-bin", Role: "orchestration"}},
		}},
		Profiles: []registry.ProfileEntry{{Manifest: &manifest.Profile{
			Meta:   manifest.Meta{Family: manifest.FamilyProfile, Name: "p"},
			Layers: manifest.ProfileLayers{Orchestration: manifest.StringList{"demo-instr"}},
		}}},
	}
	r, err := Resolve(cat, "p", "all")
	if err != nil {
		t.Fatal(err)
	}
	names := r.Names()
	if len(names) != 2 || names[0] != "demo-bin" || names[1] != "demo-instr" {
		t.Fatalf("resolved %v, want [demo-bin demo-instr] (dep before dependent)", names)
	}
	// demo-bin inherits the dependent's slot for provenance and is classified as a recipe.
	for _, it := range r.Items {
		if it.Name == "demo-bin" {
			if it.Family != manifest.FamilyRecipe {
				t.Errorf("demo-bin family = %v, want recipe", it.Family)
			}
			if it.Slot != "orchestration" {
				t.Errorf("demo-bin slot = %q, want orchestration (inherited from dependent)", it.Slot)
			}
		}
	}
}

// TestResolveWithoutBlocksRequiresPullback proves `without` wins over the closure:
// subtracting the required item keeps it out even though a listed item requires it.
func TestResolveWithoutBlocksRequiresPullback(t *testing.T) {
	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{
			Manifest: &manifest.Artifact{
				Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "demo-instr", Requires: []string{"demo-bin"}},
				Type: manifest.TypeInstruction,
			},
		}},
		Recipes: []registry.RecipeEntry{{
			Manifest: &manifest.Recipe{Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "demo-bin", Role: "orchestration"}},
		}},
		Profiles: []registry.ProfileEntry{{Manifest: &manifest.Profile{
			Meta:    manifest.Meta{Family: manifest.FamilyProfile, Name: "p"},
			Without: manifest.StringList{"demo-bin"},
			Layers:  manifest.ProfileLayers{Orchestration: manifest.StringList{"demo-instr"}},
		}}},
	}
	r, err := Resolve(cat, "p", "all")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range r.Items {
		if it.Name == "demo-bin" {
			t.Fatalf("demo-bin resolved despite `without: [demo-bin]`: %v", r.Names())
		}
	}
}

package profile

import (
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// A Codex-target profile resolves through the same shared resolver as every
// other harness: explicit (no extends), registry provenance, flavour filtering
// and no warnings. Invented data only; the shipped core-profile-cx closure is
// checked by the public CLI lock cases in the catalog gate, not here.
func TestCodexProfileResolvesThroughSharedResolver(t *testing.T) {
	p := &manifest.Profile{
		Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "invented-profile-cx", Version: "3.2.1"},
		Layers: manifest.ProfileLayers{
			Capabilities: manifest.StringList{"invented-plan-cx", "invented-guard@codex", "invented-guard-pi@pi"},
			Tools:        manifest.StringList{"invented-relay"},
		},
	}
	cat := fakeCatalog([]string{"invented-plan-cx", "invented-review-cx", "invented-guard", "invented-guard-pi"}, []string{"invented-relay"}, p)
	cat.Artifacts[0].Manifest.Requires = []string{"invented-review-cx"}
	resolved, err := Resolve(cat, p.Name, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Profile.Extends != "" || resolved.Profile.Version != p.Version {
		t.Fatalf("profile identity not preserved: %+v", resolved.Profile)
	}
	if resolved.Target != "codex" || len(resolved.Warnings) != 0 {
		t.Fatalf("unresolved Codex profile: %+v", resolved)
	}
	want := []string{"invented-review-cx", "invented-plan-cx", "invented-guard", "invented-relay"}
	if got := resolved.Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("codex selection = %v, want %v", got, want)
	}
	for _, item := range resolved.Items {
		if item.Source != "registry" {
			t.Errorf("foreign provenance: %+v", item)
		}
	}
}

func TestCodexCoreProfileFixtureUsesRequiresClosure(t *testing.T) {
	p := &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "invented-profile-cx"}, Layers: manifest.ProfileLayers{Capabilities: manifest.StringList{"invented-execute-cx"}}}
	cat := fakeCatalog([]string{"invented-execute-cx", "invented-review-cx"}, nil, p)
	cat.Artifacts[0].Manifest.Requires = []string{"invented-review-cx"}
	resolved, err := Resolve(cat, p.Name, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolved.Names(), []string{"invented-review-cx", "invented-execute-cx"}) || len(resolved.Warnings) != 0 {
		t.Fatalf("shared requires closure: %+v", resolved)
	}
}

// Recipes in context/tools/guardrails layers of a Codex-target profile stay
// together and dispatch as recipes, not artifacts. Pi's generic EXEC admission
// restriction is not a blanket doctrine requiring every harness to split its
// runtime. Invented catalog; shipped membership is not asserted here.
func TestCodexCoreProfileKeepsAuditedRecipes(t *testing.T) {
	p := &manifest.Profile{
		Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "invented-profile-cx"},
		Layers: manifest.ProfileLayers{
			Context:    manifest.StringList{"invented-context", "invented-docs-mcp"},
			Tools:      manifest.StringList{"invented-pattern-cx", "invented-forge-mcp"},
			Guardrails: manifest.StringList{"invented-guard-cx", "invented-scanner"},
		},
	}
	recipes := []string{"invented-docs-mcp", "invented-forge-mcp", "invented-scanner"}
	cat := fakeCatalog([]string{"invented-context", "invented-pattern-cx", "invented-guard-cx"}, recipes, p)
	resolved, err := Resolve(cat, p.Name, "codex")
	if err != nil || len(resolved.Warnings) != 0 {
		t.Fatalf("recipe membership: %+v %v", resolved, err)
	}
	for _, name := range recipes {
		found := false
		for _, item := range resolved.Items {
			if item.Name == name {
				found = true
				if item.Family != manifest.FamilyRecipe {
					t.Errorf("recipe dispatched as artifact: %+v", item)
				}
			}
		}
		if !found {
			t.Errorf("recipe lost from cx profile: %s", name)
		}
	}
}

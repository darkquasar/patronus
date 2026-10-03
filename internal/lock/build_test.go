package lock

import (
	"encoding/json"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func TestFromResolvedBaselineBytes(t *testing.T) {
	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{Manifest: &manifest.Artifact{Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyArtifact, Name: "sample-skill", Version: "1.2.3", Role: manifest.RoleCapability}, Type: manifest.TypeSkill, Entry: "SKILL.md"}}},
		Profiles:  []registry.ProfileEntry{{Manifest: &manifest.Profile{Meta: manifest.Meta{Name: "sample-profile"}, Layers: manifest.ProfileLayers{Capabilities: manifest.StringList{"sample-skill"}}}}},
	}
	for _, target := range []string{"", "all", "claude", "codex", "opencode"} {
		t.Run(target, func(t *testing.T) {
			r, err := profile.Resolve(cat, "sample-profile", target)
			if err != nil {
				t.Fatal(err)
			}
			l, err := FromResolved(cat, r, "2026-01-02T03:04:05Z")
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.MarshalIndent(l, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != baselineResolvedLock {
				t.Fatalf("non-Pi baseline bytes changed:\n%s", data)
			}
		})
	}
}

// Captured from FromResolved at a51052be before DP-01 production changes.
const baselineResolvedLock = `{
  "version": 2,
  "profile": "sample-profile",
  "generated": "2026-01-02T03:04:05Z",
  "entries": [
    {
      "name": "sample-skill",
      "source": "registry",
      "version": "1.2.3",
      "sha256": "sha256:03d96d2a2b1c357e0ee1f8e08123d59abadfa1abdb37daea7ad426cd99e82c75",
      "slot": "capabilities",
      "kind": "artifact"
    }
  ]
}`

func TestFromResolvedPiTarget(t *testing.T) {
	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{Manifest: &manifest.Artifact{Meta: manifest.Meta{Name: "sample-pi", Version: "1.0.0"}}}},
		Profiles:  []registry.ProfileEntry{{Manifest: &manifest.Profile{Meta: manifest.Meta{Name: "sample-profile"}, Layers: manifest.ProfileLayers{Capabilities: manifest.StringList{"sample-pi@pi"}}}}},
	}
	r, err := profile.Resolve(cat, "sample-profile", "pi")
	if err != nil {
		t.Fatal(err)
	}
	l, err := FromResolved(cat, r, "fixed")
	if err != nil {
		t.Fatal(err)
	}
	if l.Version != 3 || l.Target != "pi" || len(l.Entries) != 1 || l.Entries[0].Name != "sample-pi" {
		t.Fatalf("Pi lock = %+v", l)
	}
	data, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	// Frozen baseline reader's version guard, not unknown-field behavior. The
	// actual baseline-binary check remains a separately qualified operational test.
	var baseline struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if baseline.Version >= 1 && baseline.Version <= 2 {
		t.Fatal("baseline v2 guard accepted Pi lock")
	}
}

func TestFromResolvedDirectoryEligibility(t *testing.T) {
	for _, role := range []manifest.Role{manifest.RoleSandbox, manifest.RoleOrchestration, manifest.RoleTools} {
		t.Run(string(role), func(t *testing.T) {
			var fixture Lock
			if err := json.Unmarshal([]byte(directoryLockJSON()), &fixture); err != nil {
				t.Fatal(err)
			}
			rec := &manifest.Recipe{Meta: manifest.Meta{APIVersion: "patronus/v3", Family: manifest.FamilyRecipe, Name: "kit", Version: "1.0.0", Role: role}, Delivery: fixture.Entries[0].Delivery}
			cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: rec}}}
			r := &profile.Resolved{Target: "pi", Profile: &manifest.Profile{}, Items: []profile.ResolvedItem{{Name: "kit", Family: manifest.FamilyRecipe}}}
			l, err := FromResolved(cat, r, "fixed")
			if err != nil {
				t.Fatal(err)
			}
			if l.Entries[0].Delivery == nil {
				t.Fatal("missing delivery pin")
			}
			for _, tc := range []struct {
				name   string
				mutate func(*manifest.Recipe)
			}{
				{"role", func(r *manifest.Recipe) { r.Role = manifest.RoleMemory }},
				{"local", func(r *manifest.Recipe) { r.Scope = &manifest.RecipeScope{Marker: ".sample"} }},
				{"wiring", func(r *manifest.Recipe) { r.Wire.Tools = []string{"pi"} }},
				{"manifest version", func(r *manifest.Recipe) { r.APIVersion = "patronus/v2" }},
				{"identity", func(r *manifest.Recipe) { r.Version = "next" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					invalid := *rec
					tc.mutate(&invalid)
					cat.Recipes[0].Manifest = &invalid
					if _, err := FromResolved(cat, r, "fixed"); err == nil {
						t.Fatal("invalid recipe locked")
					}
				})
			}
		})
	}
}

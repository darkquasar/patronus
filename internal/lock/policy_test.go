package lock

import (
	"encoding/json"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func policyCatalog() *registry.Catalog {
	return &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{Manifest: &manifest.Artifact{Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyArtifact, Name: "fixture-any", Version: "1.0.0", Role: manifest.RoleCapability}, Type: manifest.TypeSkill, Entry: "SKILL.md"}}},
		Profiles:  []registry.ProfileEntry{{Manifest: &manifest.Profile{Meta: manifest.Meta{Name: "fixture-profile"}, Layers: manifest.ProfileLayers{Capabilities: manifest.StringList{"fixture-any"}}}}},
	}
}

func TestTargetPolicyTargeted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy TargetPolicy
		target string
		want   bool
	}{
		{"zero pi", TargetPolicy{}, "pi", true},
		{"zero codex", TargetPolicy{}, "codex", false},
		{"zero toy", TargetPolicy{}, "toy", false},
		{"toy names toy", NewTargetPolicy("toy"), "toy", true},
		{"toy excludes pi", NewTargetPolicy("toy"), "pi", false},
		{"empty excludes pi", NewTargetPolicy(), "pi", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.policy.Targeted(tc.target); got != tc.want {
				t.Fatalf("Targeted(%s) = %v want %v", tc.target, got, tc.want)
			}
		})
	}
}

func TestFromResolvedWithPolicyStampsTargetVersion(t *testing.T) {
	cat := policyCatalog()
	for _, tc := range []struct {
		name       string
		target     string
		policy     TargetPolicy
		wantVer    int
		wantTarget string
	}{
		{"toy under toy policy", "toy", NewTargetPolicy("toy"), TargetVersion, "toy"},
		{"toy under zero policy", "toy", TargetPolicy{}, Version, ""},
		{"pi under zero policy", "pi", TargetPolicy{}, TargetVersion, "pi"},
		{"pi under pi,toy policy", "pi", NewTargetPolicy("pi", "toy"), TargetVersion, "pi"},
		{"codex under zero policy", "codex", TargetPolicy{}, Version, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := profile.Resolve(cat, "fixture-profile", tc.target)
			if err != nil {
				t.Fatal(err)
			}
			l, err := FromResolvedWithPolicy(cat, r, "fixed", tc.policy)
			if err != nil {
				t.Fatal(err)
			}
			if l.Version != tc.wantVer || l.Target != tc.wantTarget {
				t.Fatalf("version=%d target=%q want %d/%q", l.Version, l.Target, tc.wantVer, tc.wantTarget)
			}
		})
	}
}

// Byte identity: FromResolvedWithPolicy under a codex-stamping policy yields the
// same bytes as FromResolved plus today's Codex assignment in the lock command.
func TestFromResolvedWithPolicyMatchesCodexStamp(t *testing.T) {
	cat := policyCatalog()
	r, err := profile.Resolve(cat, "fixture-profile", "codex")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := FromResolved(cat, r, "fixed")
	if err != nil {
		t.Fatal(err)
	}
	legacy.Version, legacy.Target = TargetVersion, "codex"
	stamped, err := FromResolvedWithPolicy(cat, r, "fixed", NewTargetPolicy("codex", "pi"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.MarshalIndent(legacy, "", "  ")
	b, _ := json.MarshalIndent(stamped, "", "  ")
	if string(a) != string(b) {
		t.Fatalf("bytes differ\n%s\n%s", a, b)
	}
}

// A native recipe still requires the Pi lock target under a policy that
// stamps another target.
func TestFromResolvedWithPolicyNativeRequiresPi(t *testing.T) {
	cat := policyCatalog()
	cat.Recipes = []registry.RecipeEntry{{Manifest: &manifest.Recipe{Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyRecipe, Name: "fixture-native", Version: "1.0.0", Role: manifest.RoleTools}, Delivery: &manifest.Delivery{Via: manifest.ViaPackageManager, Install: []manifest.InstallCandidate{{Manager: "pi", Ref: "npm:@invented/plugin@1.2.3"}}}}}}
	cat.Profiles[0].Manifest.Layers.Capabilities = manifest.StringList{"fixture-any", "fixture-native"}
	r, err := profile.Resolve(cat, "fixture-profile", "toy")
	if err != nil {
		t.Fatal(err)
	}
	_, err = FromResolvedWithPolicy(cat, r, "fixed", NewTargetPolicy("toy"))
	if err == nil || err.Error() != "native recipe requires Pi lock target" {
		t.Fatalf("native recipe under toy: %v", err)
	}
}

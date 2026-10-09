package lock

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

// Characterization of build.go:33 at the base revision: FromResolved stamps the
// target-bearing schema and Target for pi only. Codex gets its stamp from the
// lock command (cmd/patronus/lock.go:116), never from this package; every other
// target yields the baseline schema with an empty Target.
func TestHarnessFromResolvedTargetStamping(t *testing.T) {
	cat := &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{{Manifest: &manifest.Artifact{Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyArtifact, Name: "fixture-any", Version: "1.0.0", Role: manifest.RoleCapability}, Type: manifest.TypeSkill, Entry: "SKILL.md"}}},
		Profiles:  []registry.ProfileEntry{{Manifest: &manifest.Profile{Meta: manifest.Meta{Name: "fixture-profile"}, Layers: manifest.ProfileLayers{Capabilities: manifest.StringList{"fixture-any"}}}}},
	}
	for _, tc := range []struct {
		target     string
		wantVer    int
		wantTarget string
	}{
		{"", Version, ""},
		{"all", Version, ""},
		{"claude", Version, ""},
		{"codex", Version, ""},
		{"opencode", Version, ""},
		{"pi", TargetVersion, "pi"},
	} {
		t.Run("target="+tc.target, func(t *testing.T) {
			r, err := profile.Resolve(cat, "fixture-profile", tc.target)
			if err != nil {
				t.Fatal(err)
			}
			l, err := FromResolved(cat, r, "fixed")
			if err != nil {
				t.Fatal(err)
			}
			if l.Version != tc.wantVer || l.Target != tc.wantTarget {
				t.Fatalf("FromResolved(%q) version=%d target=%q want %d/%q", tc.target, l.Version, l.Target, tc.wantVer, tc.wantTarget)
			}
		})
	}
}

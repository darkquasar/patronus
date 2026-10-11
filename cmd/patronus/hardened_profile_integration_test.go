package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

// The sandbox layer resolves per-target flavours from invented profile data.
func TestHardenedProfileSandboxFlavourDiverges(t *testing.T) {
	cat := fixtureApplicationCatalog(t)
	cat.Artifacts = append(cat.Artifacts, registry.ArtifactEntry{Manifest: &manifest.Artifact{Meta: manifest.Meta{Name: "fix-sandbox-alt"}, Type: manifest.TypeSetting, Targets: []string{"opencode"}}})
	cat.Profiles = append(cat.Profiles, registry.ProfileEntry{Manifest: &manifest.Profile{Meta: manifest.Meta{Name: "fix-sandbox"}, Layers: manifest.ProfileLayers{Sandbox: manifest.StringList{"fix-skill-claude@claude", "fix-skill-codex@codex", "fix-sandbox-alt@opencode"}}}})
	for _, tc := range []struct{ tool, want string }{{"claude", "fix-skill-claude"}, {"codex", "fix-skill-codex"}, {"opencode", "fix-sandbox-alt"}} {
		t.Run(tc.tool, func(t *testing.T) {
			r, err := profile.Resolve(cat, "fix-sandbox", tc.tool)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, it := range r.Items {
				if it.Slot == "sandbox" {
					got = append(got, it.Name)
				}
			}
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("sandbox = %v, want [%s]", got, tc.want)
			}
		})
	}
}

// TestFixtureHookFoldsIntoSettings is the CLASS-A counterpart, on the FIXTURE: a
// hook artifact folds into the tool's settings.json, and its `requires:` edge pulls
// the binary it invokes into the closure — the gitleaks-guard -> gitleaks shape,
// proven with bytes this test invented, so download -> verify -> extract -> place
// actually RUNS (the path stubBinary skipped entirely).
func TestFixtureHookFoldsIntoSettings(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "fix-hook", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	settings := string(mustRead(t, filepath.Join(home, ".claude", "settings.json")))
	if !strings.Contains(settings, "PreToolUse") || !strings.Contains(settings, "fix-archive-bin --check") {
		t.Errorf("fix-hook should fold a PreToolUse entry into settings.json:\n%s", settings)
	}
	// requires: [fix-archive-bin] pulled the binary into the closure and PLACED it.
	placed, err := os.ReadFile(filepath.Join(home, ".patronus", "bin", "fix-archive-bin"))
	if err != nil {
		t.Fatalf("the hook's required binary was not placed: %v", err)
	}
	if shaHex(placed) != shaHex(fixArchivedBinary) {
		t.Errorf("placed binary is not the tarball's extracted member")
	}
}

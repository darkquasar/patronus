package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
)

func sampleCatalog() *registry.Catalog {
	return &registry.Catalog{
		Artifacts: []registry.ArtifactEntry{
			{Manifest: &manifest.Artifact{
				Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "team-research", Role: manifest.RoleCapability, Description: "research skill"},
				Type: manifest.TypeSkill, Targets: []string{"claude", "codex"},
			}},
			{Manifest: &manifest.Artifact{
				Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "pattern-cloudflare", Role: manifest.RoleContext, Description: "cf patterns"},
				Type: manifest.TypeSkill, Targets: []string{"claude"},
			}},
		},
		Recipes: []registry.RecipeEntry{
			{Manifest: &manifest.Recipe{
				Meta:    manifest.Meta{Family: manifest.FamilyRecipe, Name: "github", Role: manifest.RoleTools},
				Summary: "hosted MCP", Wire: manifest.Wire{Mode: manifest.WireModeMcp, Mcp: &manifest.WireMcp{Transport: "http", URL: "https://x"}},
			}},
			{Manifest: &manifest.Recipe{
				Meta:     manifest.Meta{Family: manifest.FamilyRecipe, Name: "memory-engram", Role: manifest.RoleMemory},
				Summary:  "engram",
				Delivery: &manifest.Delivery{Source: manifest.SourceGithubRelease},
				Wire:     manifest.Wire{Mode: manifest.WireModeMcp, Mcp: &manifest.WireMcp{Transport: "stdio", Command: "x"}},
			}},
		},
		Profiles: []registry.ProfileEntry{
			{Manifest: &manifest.Profile{
				Meta:    manifest.Meta{Family: manifest.FamilyProfile, Name: "cloudflare", Role: manifest.RoleLifecycle},
				Summary: "cf env", Status: "stub",
				Layers: manifest.ProfileLayers{
					Capabilities: manifest.StringList{"team-research"},
					Context:      manifest.StringList{"pattern-cloudflare"},
					Memory:       "memory-engram",
				},
			}},
		},
	}
}

// TestPrintArtifactsV2Columns: the artifact list shows TYPE + ROLE (the v2
// one-axis-per-column layout), not the old KIND. A pattern skill reads
// type=skill, role=context — proving the un-leak.
func TestPrintArtifactsV2Columns(t *testing.T) {
	var buf bytes.Buffer
	PrintCatalog(&buf, sampleCatalog(), CatalogView{Artifacts: true})
	out := buf.String()

	for _, want := range []string{"TYPE", "ROLE", "team-research", "skill", "capability", "pattern-cloudflare", "context"} {
		if !strings.Contains(out, want) {
			t.Errorf("artifact list missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "KIND") {
		t.Errorf("artifact list still shows the old KIND column:\n%s", out)
	}
	// The compact default table omits the description column.
	if strings.Contains(out, "DESCRIPTION") || strings.Contains(out, "research skill") {
		t.Errorf("default artifact table should omit the description:\n%s", out)
	}
}

// TestPrintArtifactsDescriptionBlocks: --description switches to a block list with
// each artifact's FULL description, separated by `---`.
func TestPrintArtifactsDescriptionBlocks(t *testing.T) {
	var buf bytes.Buffer
	PrintCatalog(&buf, sampleCatalog(), CatalogView{Artifacts: true, Description: true})
	out := buf.String()

	for _, want := range []string{"---", "team-research", "research skill", "pattern-cloudflare", "cf patterns", "description:", "targets:"} {
		if !strings.Contains(out, want) {
			t.Errorf("description block view missing %q\n%s", want, out)
		}
	}
	// Two artifacts => at least 3 rule lines (top, between, bottom).
	if c := strings.Count(out, "---"); c < 3 {
		t.Errorf("expected >=3 `---` rules for 2 artifacts, got %d\n%s", c, out)
	}
}

// TestPrintSingleArtifact: --artifact <name> shows just that one item's block;
// an unknown name reports clearly.
func TestPrintSingleArtifact(t *testing.T) {
	var found bytes.Buffer
	PrintCatalog(&found, sampleCatalog(), CatalogView{Artifacts: true, Artifact: "pattern-cloudflare"})
	fo := found.String()
	if !strings.Contains(fo, "pattern-cloudflare") || !strings.Contains(fo, "cf patterns") {
		t.Errorf("single-artifact view missing the item:\n%s", fo)
	}
	if strings.Contains(fo, "team-research") {
		t.Errorf("single-artifact view should not list other items:\n%s", fo)
	}

	var missing bytes.Buffer
	PrintCatalog(&missing, sampleCatalog(), CatalogView{Artifacts: true, Artifact: "does-not-exist"})
	if !strings.Contains(missing.String(), `no artifact named "does-not-exist"`) {
		t.Errorf("expected a not-found message:\n%s", missing.String())
	}
}

// TestPrintRecipesShowsShapeAndRole: recipes list TYPE (= computed Shape) + ROLE,
// not the old CAPABILITY column.
func TestPrintRecipesShowsShapeAndRole(t *testing.T) {
	var buf bytes.Buffer
	PrintCatalog(&buf, sampleCatalog(), CatalogView{Recipes: true})
	out := buf.String()

	for _, want := range []string{"TYPE", "ROLE", "github", "wire-only", "tools", "memory-engram", "fetch+wire", "memory"} {
		if !strings.Contains(out, want) {
			t.Errorf("recipe list missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "CAPABILITY") {
		t.Errorf("recipe list still shows the old CAPABILITY column:\n%s", out)
	}
}

func TestPrintProfilesAndLayers(t *testing.T) {
	// Compact view: NAME/STATUS/SUMMARY.
	var compact bytes.Buffer
	PrintCatalog(&compact, sampleCatalog(), CatalogView{Profiles: true})
	if !strings.Contains(compact.String(), "cloudflare") || !strings.Contains(compact.String(), "stub") {
		t.Errorf("profile compact view:\n%s", compact.String())
	}

	// Expanded layers view: shows the per-layer selections.
	var layers bytes.Buffer
	PrintCatalog(&layers, sampleCatalog(), CatalogView{Profiles: true, Layers: true})
	out := layers.String()
	for _, want := range []string{"cloudflare", "capabilities", "team-research", "context", "pattern-cloudflare", "memory", "memory-engram"} {
		if !strings.Contains(out, want) {
			t.Errorf("profile layers view missing %q\n%s", want, out)
		}
	}
}

func TestPrintCatalogEmptySections(t *testing.T) {
	var buf bytes.Buffer
	PrintCatalog(&buf, &registry.Catalog{}, CatalogView{Artifacts: true, Recipes: true, Profiles: true})
	if c := strings.Count(buf.String(), "(none)"); c != 3 {
		t.Errorf("expected 3 (none) sections, got %d:\n%s", c, buf.String())
	}
}

func TestJSONRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"k": "v"`) {
		t.Errorf("JSON output: %s", buf.String())
	}
}

func TestPrintInventory(t *testing.T) {
	inv := &scan.Inventory{
		ProjectDir: "/proj",
		Home:       "/home/u",
		Env:        scan.EnvSnapshot{CodexHome: "/custom/codex"},
		Tools: []scan.ToolStatus{
			{
				Tool:   "claude",
				Global: &scan.Detection{Scope: scan.ScopeGlobal, Detected: true, MatchedPaths: []string{"/home/u/.claude"}},
				Local:  &scan.Detection{Scope: scan.ScopeLocal, Detected: false},
			},
		},
	}
	var buf bytes.Buffer
	PrintInventory(&buf, inv)
	out := buf.String()
	for _, want := range []string{"/proj", "/home/u", "CODEX_HOME=/custom/codex", "claude", "yes", "no"} {
		if !strings.Contains(out, want) {
			t.Errorf("inventory missing %q\n%s", want, out)
		}
	}
}

func TestTruncateAndJoinList(t *testing.T) {
	if got := truncate("abcdef", 4); got != "abc…" {
		t.Errorf("truncate = %q, want abc…", got)
	}
	if got := truncate("ab", 10); got != "ab" {
		t.Errorf("truncate short = %q", got)
	}
	if got := joinList(nil); got != "-" {
		t.Errorf("joinList(nil) = %q, want -", got)
	}
	if got := joinList([]string{"a", "b"}); got != "a, b" {
		t.Errorf("joinList = %q, want 'a, b'", got)
	}
}

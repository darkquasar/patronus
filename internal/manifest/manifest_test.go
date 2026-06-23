package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadArtifactV2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.yaml")
	content := `apiVersion: patronus/v2
family: artifact
type: skill
role: context
name: demo
description: A demo skill.
version: 1.0.0
entry: SKILL.md
targets: [claude]
defaults:
  scope: project
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadArtifact(path)
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}
	if a.Family != FamilyArtifact {
		t.Errorf("family = %s, want artifact", a.Family)
	}
	if a.Type != TypeSkill {
		t.Errorf("type = %s, want skill", a.Type)
	}
	if a.Role != RoleContext {
		t.Errorf("role = %s, want context", a.Role)
	}
	// Meta fields promote through the embed and Header() exposes them.
	if a.Header().Name != "demo" {
		t.Errorf("Header().Name = %s, want demo", a.Header().Name)
	}
}

func TestLoadArtifactRejectsBadType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.yaml")
	content := `apiVersion: patronus/v2
family: artifact
type: recipe
name: x
description: y
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArtifact(path); err == nil {
		t.Fatal("expected error for invalid artifact type, got nil")
	}
}

func TestLoadArtifactRejectsWrongFamily(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.yaml")
	content := `apiVersion: patronus/v2
family: recipe
type: skill
name: x
description: y
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArtifact(path); err == nil {
		t.Fatal("expected error for non-artifact family, got nil")
	}
}

func TestRecipeShape(t *testing.T) {
	cases := []struct {
		name string
		r    Recipe
		want RecipeShape
	}{
		{"wire-only", Recipe{Wire: Wire{Mode: WireModeMcp}}, ShapeWireOnly},
		{"fetch+wire", Recipe{Delivery: &Delivery{}, Wire: Wire{Mode: WireModeMcp}}, ShapeFetchWire},
		{"fetch+run", Recipe{Delivery: &Delivery{}, Wire: Wire{Mode: WireModeRun}}, ShapeFetchRun},
		{"fetch+self", Recipe{Delivery: &Delivery{}, Wire: Wire{Mode: WireModeSelf}}, ShapeFetchRun},
		{"install-only", Recipe{Delivery: &Delivery{Source: SourceNpm}, Wire: Wire{}}, ShapeInstall},
	}
	for _, tc := range cases {
		if got := tc.r.Shape(); got != tc.want {
			t.Errorf("%s: Shape() = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestValidateRecipeWireMode(t *testing.T) {
	base := func() *Recipe {
		return &Recipe{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyRecipe, Role: RoleTools, Name: "x"},
			Wire: Wire{Mode: WireModeMcp, Mcp: &WireMcp{Transport: "http", URL: "https://example"}},
		}
	}
	if err := validateRecipe(base()); err != nil {
		t.Fatalf("valid mcp recipe rejected: %v", err)
	}

	// mcp mode without an mcp block is invalid.
	r := base()
	r.Wire.Mcp = nil
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for mcp mode without mcp block")
	}

	// run mode without run commands is invalid.
	r = base()
	r.Wire = Wire{Mode: WireModeRun}
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for run mode without run commands")
	}

	// self mode with run commands is valid.
	r = base()
	r.Wire = Wire{Mode: WireModeSelf, Run: []string{"installer --apply"}}
	if err := validateRecipe(r); err != nil {
		t.Errorf("valid self recipe rejected: %v", err)
	}

	// bad delivery source is invalid.
	r = base()
	r.Delivery = &Delivery{Source: "ftp"}
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for invalid deliver.source")
	}

	// install-only: empty wire.mode is valid WITH a deliver block.
	r = base()
	r.Wire = Wire{}
	r.Delivery = &Delivery{Source: SourceNpm, Ref: "tdd-guard"}
	if err := validateRecipe(r); err != nil {
		t.Errorf("install-only recipe (empty mode + deliver) rejected: %v", err)
	}

	// ...but empty wire.mode WITHOUT a deliver block does nothing — invalid.
	r = base()
	r.Wire = Wire{}
	r.Delivery = nil
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for a recipe that neither wires nor delivers")
	}
}

func TestDeliveryInstallCommand(t *testing.T) {
	cases := []struct {
		name   string
		d      Delivery
		recipe string
		want   string
	}{
		{"npm with ref", Delivery{Source: SourceNpm, Ref: "tdd-guard"}, "tdd-guard", "npm install -g tdd-guard"},
		{"npm defaults ref to recipe name", Delivery{Source: SourceNpm}, "ccusage", "npm install -g ccusage"},
		{"cargo", Delivery{Source: SourceCargo, Ref: "ripgrep"}, "rg", "cargo install ripgrep"},
		{"non-PM source has no install command", Delivery{Source: SourceGithubRelease}, "engram", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.InstallCommand(tc.recipe); got != tc.want {
				t.Errorf("InstallCommand = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInstallableHeader(t *testing.T) {
	var _ Installable = (*Artifact)(nil)
	var _ Installable = (*Recipe)(nil)
	var _ Installable = (*Profile)(nil)
}

func TestFamilyPluginConstant(t *testing.T) {
	if FamilyPlugin != "plugin" {
		t.Errorf("FamilyPlugin = %q, want \"plugin\"", FamilyPlugin)
	}
}

func TestStringListScalarOrSequence(t *testing.T) {
	type wrap struct {
		Items StringList `yaml:"items"`
	}
	var scalar wrap
	if err := yaml.Unmarshal([]byte("items: solo\n"), &scalar); err != nil {
		t.Fatal(err)
	}
	if len(scalar.Items) != 1 || scalar.Items[0] != "solo" {
		t.Errorf("scalar => %v, want [solo]", scalar.Items)
	}

	var seq wrap
	if err := yaml.Unmarshal([]byte("items: [a, b]\n"), &seq); err != nil {
		t.Fatal(err)
	}
	if len(seq.Items) != 2 || seq.Items[0] != "a" || seq.Items[1] != "b" {
		t.Errorf("seq => %v, want [a b]", seq.Items)
	}
}

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

func TestShape(t *testing.T) {
	tests := []struct {
		name string
		rec  Recipe
		want RecipeShape
	}{
		{"wire-only merge", Recipe{Wire: Wire{Method: WireMerge, Actor: ActorPatronus, Mcp: &WireMcp{}}}, ShapeWireOnly},
		{"fetch+wire", Recipe{Delivery: &Delivery{}, Wire: Wire{Method: WireMerge, Actor: ActorPatronus, Mcp: &WireMcp{}}}, ShapeFetchWire},
		{"fetch+run patronus", Recipe{Delivery: &Delivery{}, Wire: Wire{Method: WireExec, Actor: ActorPatronus, Run: []string{"x"}}}, ShapeFetchRun},
		{"fetch+run external", Recipe{Delivery: &Delivery{}, Wire: Wire{Method: WireExec, Actor: ActorExternal, Run: []string{"x"}}}, ShapeFetchRun},
		{"install-only", Recipe{Delivery: &Delivery{}, Wire: Wire{Method: WireNone}}, ShapeInstall},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rec.Shape(); got != tt.want {
				t.Errorf("Shape() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateRecipeWireMethod(t *testing.T) {
	base := func() *Recipe {
		return &Recipe{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyRecipe, Role: RoleTools, Name: "x", Version: "1.0.0"},
			Wire: Wire{Method: WireMerge, Actor: ActorPatronus, Mcp: &WireMcp{Transport: "http", URL: "https://example"}},
		}
	}
	if err := validateRecipe(base()); err != nil {
		t.Fatalf("valid merge recipe rejected: %v", err)
	}

	// merge method without an mcp block is invalid.
	r := base()
	r.Wire.Mcp = nil
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for merge method without mcp block")
	}

	// merge method with a non-patronus actor is invalid.
	r = base()
	r.Wire.Actor = ActorExternal
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for merge method with actor: external")
	}

	// exec method without run commands is invalid.
	r = base()
	r.Wire = Wire{Method: WireExec, Actor: ActorPatronus}
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for exec method without run commands")
	}

	// exec method without an actor is invalid.
	r = base()
	r.Wire = Wire{Method: WireExec, Run: []string{"installer --apply"}}
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for exec method without an actor")
	}

	// exec×external with run commands is valid (self-wiring).
	r = base()
	r.Wire = Wire{Method: WireExec, Actor: ActorExternal, Run: []string{"installer --apply"}}
	if err := validateRecipe(r); err != nil {
		t.Errorf("valid exec×external recipe rejected: %v", err)
	}

	// bad delivery via is invalid.
	r = base()
	r.Delivery = &Delivery{Via: "ftp"}
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for invalid deliver.via")
	}

	// install-only: empty wire.method is valid WITH a deliver block.
	r = base()
	r.Wire = Wire{}
	r.Delivery = &Delivery{Via: ViaPackageManager, Install: []InstallCandidate{{Manager: PMNpm, Ref: "tdd-guard"}}}
	if err := validateRecipe(r); err != nil {
		t.Errorf("install-only recipe (empty method + deliver) rejected: %v", err)
	}

	// ...but empty wire.method WITHOUT a deliver block does nothing — invalid.
	r = base()
	r.Wire = Wire{}
	r.Delivery = nil
	if err := validateRecipe(r); err == nil {
		t.Error("expected error for a recipe that neither wires nor delivers")
	}
}

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name      string
		candidate InstallCandidate
		recipe    string
		want      string
	}{
		{"npm with ref", InstallCandidate{Manager: PMNpm, Ref: "tdd-guard"}, "tdd-guard", "npm install -g tdd-guard"},
		{"npm defaults ref", InstallCandidate{Manager: PMNpm}, "ccusage", "npm install -g ccusage"},
		{"cargo", InstallCandidate{Manager: PMCargo, Ref: "ripgrep"}, "rg", "cargo install ripgrep"},
		{"uv", InstallCandidate{Manager: PMUv, Ref: "graphifyy"}, "graphify", "uv tool install graphifyy"},
		// A PEP 508 requirement string carries an extra + pin in one ref — this is how
		// graphify pulls its [mcp] extra (without it the MCP entrypoint can't import mcp).
		{"uv with extra and pin", InstallCandidate{Manager: PMUv, Ref: "graphifyy[mcp]==0.9.31"}, "graphify", "uv tool install graphifyy[mcp]==0.9.31"},
		{"manager without a template has no command", InstallCandidate{Manager: PMAur, Ref: "x"}, "x", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.candidate.InstallCommand(tt.recipe); got != tt.want {
				t.Errorf("InstallCommand = %q, want %q", got, tt.want)
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

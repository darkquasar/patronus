package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestShapeMatrix exhausts the deliver × wire shape matrix (§4c) so a future
// change to Shape() or the enums can't silently reclassify a recipe.
func TestShapeMatrix(t *testing.T) {
	cases := []struct {
		name     string
		delivery *Delivery
		mode     WireMode
		want     RecipeShape
	}{
		{"nil-delivery+mcp", nil, WireModeMcp, ShapeWireOnly},
		{"nil-delivery+run", nil, WireModeRun, ShapeWireOnly},   // no delivery wins
		{"nil-delivery+self", nil, WireModeSelf, ShapeWireOnly}, // no delivery wins
		{"delivery+mcp", &Delivery{Source: SourceGithubRelease}, WireModeMcp, ShapeFetchWire},
		{"delivery+run", &Delivery{Source: SourceScript}, WireModeRun, ShapeFetchRun},
		{"delivery+self", &Delivery{Source: SourceDocker}, WireModeSelf, ShapeFetchRun},
		{"delivery+no-wire", &Delivery{Source: SourceNpm}, "", ShapeInstall}, // install-only
		{"nil-delivery+no-wire", nil, "", ShapeWireOnly},                     // no delivery wins (degenerate)
	}
	for _, tc := range cases {
		r := &Recipe{Delivery: tc.delivery, Wire: Wire{Mode: tc.mode}}
		if got := r.Shape(); got != tc.want {
			t.Errorf("%s: Shape() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestValidateArtifact covers every accept/reject branch of the artifact rules.
func TestValidateArtifact(t *testing.T) {
	base := func() *Artifact {
		return &Artifact{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyArtifact, Role: RoleCapability, Name: "x", Description: "d"},
			Type: TypeSkill,
		}
	}
	cases := []struct {
		name    string
		mutate  func(*Artifact)
		wantErr bool
	}{
		{"valid", func(*Artifact) {}, false},
		{"valid-no-role", func(a *Artifact) { a.Role = "" }, false}, // role optional on artifacts
		{"bad-apiversion", func(a *Artifact) { a.APIVersion = "patronus/v1" }, true},
		{"wrong-family", func(a *Artifact) { a.Family = FamilyRecipe }, true},
		{"empty-family", func(a *Artifact) { a.Family = "" }, true},
		{"bad-type", func(a *Artifact) { a.Type = "widget" }, true},
		{"empty-type", func(a *Artifact) { a.Type = "" }, true},
		{"missing-name", func(a *Artifact) { a.Name = "" }, true},
		{"missing-description", func(a *Artifact) { a.Description = "" }, true},
		{"every-valid-type-skill", func(a *Artifact) { a.Type = TypeSkill }, false},
		{"every-valid-type-agent", func(a *Artifact) { a.Type = TypeAgent }, false},
		{"every-valid-type-command", func(a *Artifact) { a.Type = TypeCommand }, false},
		{"every-valid-type-hook", func(a *Artifact) {
			a.Type = TypeHook
			a.Hook = &HookSpec{Event: "PreToolUse", Command: "true"}
		}, false},
		{"hook-missing-block", func(a *Artifact) { a.Type = TypeHook }, true},
		{"hook-missing-command", func(a *Artifact) {
			a.Type = TypeHook
			a.Hook = &HookSpec{Event: "PreToolUse"}
		}, true},
		{"every-valid-type-instruction", func(a *Artifact) { a.Type = TypeInstruction }, false},
		{"every-valid-type-output-style", func(a *Artifact) { a.Type = TypeOutputStyle }, false},
		{"attribution-complete", func(a *Artifact) {
			a.Attribution = &Attribution{Upstream: "github.com/x/y", License: "MIT", Copyright: "Copyright (c) 2026 X"}
		}, false},
		{"attribution-missing-copyright", func(a *Artifact) {
			a.Attribution = &Attribution{Upstream: "github.com/x/y", License: "MIT"}
		}, true},
		{"attribution-missing-upstream", func(a *Artifact) {
			a.Attribution = &Attribution{License: "MIT", Copyright: "Copyright (c) 2026 X"}
		}, true},
	}
	for _, tc := range cases {
		a := base()
		tc.mutate(a)
		err := a.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr = %v", tc.name, err, tc.wantErr)
		}
	}
}

// TestValidateRecipeRules covers the recipe accept/reject branches beyond the
// wire-mode cases in manifest_test.go: family, role, delivery source.
func TestValidateRecipeRules(t *testing.T) {
	base := func() *Recipe {
		return &Recipe{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyRecipe, Role: RoleTools, Name: "r"},
			Wire: Wire{Mode: WireModeMcp, Mcp: &WireMcp{Transport: "http", URL: "https://x"}},
		}
	}
	cases := []struct {
		name    string
		mutate  func(*Recipe)
		wantErr bool
	}{
		{"valid", func(*Recipe) {}, false},
		{"bad-apiversion", func(r *Recipe) { r.APIVersion = "patronus/v1" }, true},
		{"wrong-family", func(r *Recipe) { r.Family = FamilyArtifact }, true},
		{"missing-role", func(r *Recipe) { r.Role = "" }, true},
		{"missing-name", func(r *Recipe) { r.Name = "" }, true},
		{"bad-wire-mode", func(r *Recipe) { r.Wire.Mode = "teleport" }, true},
		{"empty-wire-mode", func(r *Recipe) { r.Wire.Mode = "" }, true},
		{"valid-delivery-source", func(r *Recipe) { r.Delivery = &Delivery{Source: SourceGithubRelease} }, false},
		{"bad-delivery-source", func(r *Recipe) { r.Delivery = &Delivery{Source: "ftp"} }, true},
		{"every-valid-source-docker", func(r *Recipe) { r.Delivery = &Delivery{Source: SourceDocker} }, false},
		{"every-valid-source-cargo", func(r *Recipe) { r.Delivery = &Delivery{Source: SourceCargo} }, false},
		{"every-valid-source-script", func(r *Recipe) { r.Delivery = &Delivery{Source: SourceScript} }, false},
		// url is the one source with a required shape: it now has a fetcher, and a
		// url delivery with no url+sha256 would be unfetchable. The other sources
		// are still names in the closed set with no fields behind them, so a bare
		// deliver block remains valid for them.
		{"bare-source-url-rejected", func(r *Recipe) { r.Delivery = &Delivery{Source: SourceURL} }, true},
		{"every-valid-source-url", func(r *Recipe) {
			r.Delivery = &Delivery{Source: SourceURL, URL: "https://x/tk", SHA256: "abc"}
		}, false},
	}
	for _, tc := range cases {
		r := base()
		tc.mutate(r)
		err := validateRecipe(r)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr = %v", tc.name, err, tc.wantErr)
		}
	}
}

// TestLoadRecipeFromDisk exercises LoadRecipe end to end (parse + validate) and
// the run-mode path that the catalog doesn't yet use, so the run branch has a
// guard before a feature relies on it.
func TestLoadRecipeFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.yaml")
	content := `apiVersion: patronus/v2
family: recipe
name: scripted
role: tools
deliver:
  source: script
wire:
  mode: run
  run:
    - "curl -sSf https://example/install.sh | sh"
  tools: [claude]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRecipe(path)
	if err != nil {
		t.Fatalf("LoadRecipe: %v", err)
	}
	if r.Shape() != ShapeFetchRun {
		t.Errorf("Shape() = %q, want fetch+run", r.Shape())
	}
	if r.Wire.Mode != WireModeRun || len(r.Wire.Run) != 1 {
		t.Errorf("wire = %+v", r.Wire)
	}
	if r.Header().Family != FamilyRecipe {
		t.Errorf("Header().Family = %q", r.Header().Family)
	}
}

// TestLoadProfileFromDisk exercises LoadProfile (parse + family check) and the
// Header() accessor on a profile.
func TestLoadProfileFromDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	content := `apiVersion: patronus/v2
family: profile
role: lifecycle
name: demo
layers:
  capabilities: [team-research]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.Header().Family != FamilyProfile {
		t.Errorf("Header().Family = %q, want profile", p.Header().Family)
	}
	if len(p.Layers.Capabilities) != 1 {
		t.Errorf("layers.capabilities = %v", p.Layers.Capabilities)
	}

	// A non-profile family must be rejected.
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("apiVersion: patronus/v2\nfamily: artifact\nname: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProfile(bad); err == nil {
		t.Error("LoadProfile accepted family: artifact")
	}
}

// TestDecodeArtifactAndRecipe covers the Decode* byte-seam (used for https:
// sourced manifests) including a rejection.
func TestDecodeArtifactAndRecipe(t *testing.T) {
	a, err := DecodeArtifact([]byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nname: s\ndescription: d\n"))
	if err != nil {
		t.Fatalf("DecodeArtifact: %v", err)
	}
	if a.Type != TypeSkill {
		t.Errorf("type = %q", a.Type)
	}
	if _, err := DecodeArtifact([]byte("apiVersion: patronus/v2\nfamily: artifact\ntype: bogus\nname: s\ndescription: d\n")); err == nil {
		t.Error("DecodeArtifact accepted bogus type")
	}

	r, err := DecodeRecipe([]byte("apiVersion: patronus/v2\nfamily: recipe\nname: r\nrole: tools\nwire:\n  mode: mcp\n  mcp:\n    transport: http\n    url: https://x\n"))
	if err != nil {
		t.Fatalf("DecodeRecipe: %v", err)
	}
	if r.Shape() != ShapeWireOnly {
		t.Errorf("Shape() = %q, want wire-only", r.Shape())
	}
	if _, err := DecodeRecipe([]byte("apiVersion: patronus/v2\nfamily: recipe\nname: r\nrole: tools\nwire:\n  mode: mcp\n")); err == nil {
		t.Error("DecodeRecipe accepted mcp mode with no mcp block")
	}
}

// TestResolveAsset covers host match, no-match, and empty-assets branches.
func TestResolveAsset(t *testing.T) {
	d := &Delivery{Assets: []Asset{
		{OS: "linux", Arch: "amd64", URL: "u1", SHA256: "s1"},
		{OS: "darwin", Arch: "arm64", URL: "u2", SHA256: "s2"},
	}}
	got, err := d.ResolveAsset("darwin", "arm64")
	if err != nil {
		t.Fatalf("ResolveAsset match: %v", err)
	}
	if got.URL != "u2" {
		t.Errorf("URL = %q, want u2", got.URL)
	}
	if _, err := d.ResolveAsset("windows", "arm64"); err == nil {
		t.Error("expected error for unpinned host")
	}
	empty := &Delivery{}
	if _, err := empty.ResolveAsset("linux", "amd64"); err == nil {
		t.Error("expected error for no pinned assets")
	}
}

func TestResolveURLReturnsPinnedArtifact(t *testing.T) {
	d := &Delivery{
		Source:    SourceURL,
		URL:       "https://example.test/tk",
		SHA256:    "408f2c113ecc3bc071507593a78386f1b4cc743be6491c9e9f2627efd4d9902b",
		Platforms: []string{"linux", "darwin"},
	}

	got, err := d.ResolveURL("darwin")
	if err != nil {
		t.Fatalf("ResolveURL(darwin) error = %v, want nil", err)
	}
	want := &PinnedURL{
		URL:    "https://example.test/tk",
		SHA256: "408f2c113ecc3bc071507593a78386f1b4cc743be6491c9e9f2627efd4d9902b",
	}
	if *got != *want {
		t.Errorf("ResolveURL(darwin) = %+v, want %+v", *got, *want)
	}
}

// TestResolveURLPlatformGate is its own function rather than a row in the test
// above: it exercises the error path, which is different logic and not merely
// different data.
func TestResolveURLPlatformGate(t *testing.T) {
	tests := []struct {
		name      string
		platforms []string
		goos      string
		wantErr   bool
	}{
		{"listed host resolves", []string{"linux", "darwin"}, "darwin", false},
		{"unlisted host errors", []string{"linux", "darwin"}, "windows", true},
		{"empty platforms is unrestricted", nil, "windows", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Delivery{
				Source:    SourceURL,
				URL:       "https://example.test/tk",
				SHA256:    "abc",
				Platforms: tt.platforms,
			}
			_, err := d.ResolveURL(tt.goos)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveURL(%s) error = %v, wantErr %v", tt.goos, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURLDelivery(t *testing.T) {
	tests := []struct {
		name    string
		d       Delivery
		wantErr bool
	}{
		{"valid", Delivery{Source: SourceURL, URL: "https://x/tk", SHA256: "abc"}, false},
		{"missing url", Delivery{Source: SourceURL, SHA256: "abc"}, true},
		{"missing sha256", Delivery{Source: SourceURL, URL: "https://x/tk"}, true},
		{"assets not allowed", Delivery{
			Source: SourceURL, URL: "https://x/tk", SHA256: "abc",
			Assets: []Asset{{OS: "linux", Arch: "amd64"}},
		}, true},
		{"unknown source rejected", Delivery{Source: DeliverySource("bogus")}, true},
		{"github-release unaffected", Delivery{Source: SourceGithubRelease}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDelivery(&tt.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDelivery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
		method   WireMethod
		actor    WireActor
		want     RecipeShape
	}{
		{"nil-delivery+merge", nil, WireMerge, ActorPatronus, ShapeWireOnly},
		{"nil-delivery+exec-patronus", nil, WireExec, ActorPatronus, ShapeWireOnly}, // no delivery wins
		{"nil-delivery+exec-external", nil, WireExec, ActorExternal, ShapeWireOnly}, // no delivery wins
		{"delivery+merge", &Delivery{Via: ViaFetch}, WireMerge, ActorPatronus, ShapeFetchWire},
		{"delivery+exec-patronus", &Delivery{Via: ViaScript}, WireExec, ActorPatronus, ShapeFetchRun},
		{"delivery+exec-external", &Delivery{Via: ViaDocker}, WireExec, ActorExternal, ShapeFetchRun},
		{"delivery+no-wire", &Delivery{Via: ViaPackageManager}, WireNone, "", ShapeInstall}, // install-only
		{"nil-delivery+no-wire", nil, WireNone, "", ShapeWireOnly},                          // no delivery wins (degenerate)
	}
	for _, tc := range cases {
		r := &Recipe{Delivery: tc.delivery, Wire: Wire{Method: tc.method, Actor: tc.actor}}
		if got := r.Shape(); got != tc.want {
			t.Errorf("%s: Shape() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestValidateArtifact covers every accept/reject branch of the artifact rules.
func TestValidateArtifact(t *testing.T) {
	base := func() *Artifact {
		return &Artifact{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyArtifact, Role: RoleCapability, Name: "x", Description: "d", Version: "1.0.0"},
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
		{"missing-version", func(a *Artifact) { a.Version = "" }, true}, // version is required schema-wide (ADR-0004)
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
		{"hook-gate-intent-valid", func(a *Artifact) {
			a.Type = TypeHook
			a.Hook = &HookSpec{Event: "PreToolUse", Command: "x", Intent: HookGate}
		}, false},
		{"hook-bad-intent", func(a *Artifact) {
			a.Type = TypeHook
			a.Hook = &HookSpec{Event: "PreToolUse", Command: "x", Intent: "bogus"}
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
// wire-method cases in manifest_test.go: family, role, delivery via.
func TestValidateRecipeRules(t *testing.T) {
	base := func() *Recipe {
		return &Recipe{
			Meta: Meta{APIVersion: APIVersion, Family: FamilyRecipe, Role: RoleTools, Name: "r", Version: "1.0.0"},
			Wire: Wire{Method: WireMerge, Actor: ActorPatronus, Mcp: &WireMcp{Transport: "http", URL: "https://x"}},
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
		{"missing-version", func(r *Recipe) { r.Version = "" }, true}, // version is required schema-wide (ADR-0004)
		{"bad-wire-method", func(r *Recipe) { r.Wire.Method = "teleport" }, true},
		{"empty-wire-method-no-deliver", func(r *Recipe) { r.Wire = Wire{} }, true},
		{"valid-delivery-via-fetch", func(r *Recipe) { r.Delivery = &Delivery{Via: ViaFetch} }, false},
		{"bad-delivery-via", func(r *Recipe) { r.Delivery = &Delivery{Via: "ftp"} }, true},
		{"every-valid-via-docker", func(r *Recipe) { r.Delivery = &Delivery{Via: ViaDocker} }, false},
		{"every-valid-via-script", func(r *Recipe) { r.Delivery = &Delivery{Via: ViaScript} }, false},
		{"via-package-manager-needs-candidate", func(r *Recipe) { r.Delivery = &Delivery{Via: ViaPackageManager} }, true},
		{"via-package-manager-with-candidate", func(r *Recipe) {
			r.Delivery = &Delivery{Via: ViaPackageManager, Install: []InstallCandidate{{Manager: PMNpm}}}
		}, false},
		// A url fetch needs url+sha256: a url with no sha256 is unfetchable and
		// rejected; a full url+sha256 fetch is valid.
		{"url-fetch-without-sha-rejected", func(r *Recipe) { r.Delivery = &Delivery{Via: ViaFetch, URL: "https://x/tk"} }, true},
		{"url-fetch-with-sha-valid", func(r *Recipe) {
			r.Delivery = &Delivery{Via: ViaFetch, URL: "https://x/tk", SHA256: "abc"}
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
version: 1.0.0
deliver:
  via: script
wire:
  method: exec
  actor: patronus
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
	if r.Wire.Method != WireExec || len(r.Wire.Run) != 1 {
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
version: 1.0.0
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

	// A profile with no version: must be rejected — version is required schema-wide
	// (ADR-0004), enforced in the shared validateMeta for every family.
	noVer := filepath.Join(dir, "noversion.yaml")
	if err := os.WriteFile(noVer, []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: nv\nlayers:\n  capabilities: [x]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProfile(noVer); err == nil {
		t.Error("LoadProfile accepted a profile with no version")
	}
}

// TestDecodeArtifactAndRecipe covers the Decode* byte-seam (used for https:
// sourced manifests) including a rejection.
func TestDecodeArtifactAndRecipe(t *testing.T) {
	a, err := DecodeArtifact([]byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nname: s\ndescription: d\nversion: 1.0.0\n"))
	if err != nil {
		t.Fatalf("DecodeArtifact: %v", err)
	}
	if a.Type != TypeSkill {
		t.Errorf("type = %q", a.Type)
	}
	if _, err := DecodeArtifact([]byte("apiVersion: patronus/v2\nfamily: artifact\ntype: bogus\nname: s\ndescription: d\n")); err == nil {
		t.Error("DecodeArtifact accepted bogus type")
	}

	r, err := DecodeRecipe([]byte("apiVersion: patronus/v2\nfamily: recipe\nname: r\nrole: tools\nversion: 1.0.0\nwire:\n  method: merge\n  actor: patronus\n  mcp:\n    transport: http\n    url: https://x\n"))
	if err != nil {
		t.Fatalf("DecodeRecipe: %v", err)
	}
	if r.Shape() != ShapeWireOnly {
		t.Errorf("Shape() = %q, want wire-only", r.Shape())
	}
	if _, err := DecodeRecipe([]byte("apiVersion: patronus/v2\nfamily: recipe\nname: r\nrole: tools\nwire:\n  method: merge\n  actor: patronus\n")); err == nil {
		t.Error("DecodeRecipe accepted merge method with no mcp block")
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
		Via:       ViaFetch,
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
				Via:       ViaFetch,
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

func TestValidateFetchDelivery(t *testing.T) {
	tests := []struct {
		name    string
		d       Delivery
		wantErr bool
	}{
		{"url+sha valid", Delivery{Via: ViaFetch, URL: "https://x/tk", SHA256: "abc"}, false},
		// A fetch with neither url nor assets is a stub whose upstream isn't pinned
		// yet (the sandbox recipe) — valid, emits no FETCH until assets land.
		{"bare fetch is a stub", Delivery{Via: ViaFetch}, false},
		{"url without sha256 rejected", Delivery{Via: ViaFetch, URL: "https://x/tk"}, true},
		{"url and assets both rejected", Delivery{
			Via: ViaFetch, URL: "https://x/tk", SHA256: "abc",
			Assets: []Asset{{OS: "linux", Arch: "amd64"}},
		}, true},
		{"asset-matrix fetch valid", Delivery{Via: ViaFetch, Assets: []Asset{{OS: "linux", Arch: "amd64"}}}, false},
		{"unknown via rejected", Delivery{Via: DeliverVia("bogus")}, true},
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

package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

func directoryRecipe() *manifest.Recipe {
	return &manifest.Recipe{Meta: manifest.Meta{APIVersion: "patronus/v3", Family: manifest.FamilyRecipe, Name: "pi-sandbox", Version: "1.0.0", Role: manifest.RoleSandbox}, Delivery: &manifest.Delivery{Via: manifest.ViaFetch, Unpack: "directory", Package: &manifest.PackageIdentity{Name: "pi-sandbox", Version: "2.0.0"}, Assets: []manifest.Asset{{OS: "linux", Arch: "amd64", URL: "https://example.test/kit.tar.gz", SHA256: strings.Repeat("a", 64), Archive: "tar.gz"}}}}
}

func TestDirectoryComputeHostIntent(t *testing.T) {
	for _, target := range []string{"", "all", "claude"} {
		t.Run(target, func(t *testing.T) {
			res, home, _ := testEnv(t)
			home, err := filepath.EvalSymlinks(home)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := Compute(Request{Recipe: directoryRecipe(), Resolver: res, Tool: target, GOOS: "linux", GOARCH: "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 {
				t.Fatalf("rows = %#v", rows)
			}
			d := rows[0]
			if d.Directory == nil || d.Fetch != nil || d.Action != diff.Fetch || d.Tool != TargetAgnostic || d.Scope != "global" || d.Type != "install-only" {
				t.Fatalf("row = %#v", d)
			}
			if d.Path != filepath.Join(home, ".patronus", "packages", "pi-sandbox") {
				t.Fatal(d.Path)
			}
			if _, err := os.Stat(filepath.Join(home, ".patronus")); !os.IsNotExist(err) {
				t.Fatalf("planning wrote files: %v", err)
			}
		})
	}
}

func TestDirectoryComputeRejectsLocalAndUnsupported(t *testing.T) {
	for _, tc := range []struct{ name, scope, os, arch string }{{"local", "local", "linux", "amd64"}, {"unsupported", "", "darwin", "arm64"}} {
		t.Run(tc.name, func(t *testing.T) {
			res, _, _ := testEnv(t)
			if _, err := Compute(Request{Recipe: directoryRecipe(), Resolver: res, Scope: tc.scope, GOOS: tc.os, GOARCH: tc.arch}); err == nil {
				t.Fatal("expected directory planning error")
			}
		})
	}
}

func TestDirectoryComputeAdmittedRoles(t *testing.T) {
	for _, role := range []manifest.Role{manifest.RoleSandbox, manifest.RoleOrchestration, manifest.RoleTools} {
		t.Run(string(role), func(t *testing.T) {
			rec := directoryRecipe()
			rec.Name, rec.Role = "sample-package", role
			res, _, _ := testEnv(t)
			rows, err := Compute(Request{Recipe: rec, Resolver: res, Tool: "pi", GOOS: "linux", GOARCH: "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0].Directory == nil || rows[0].Role != string(role) || rows[0].Tool != TargetAgnostic || rows[0].Scope != "global" {
				t.Fatalf("rows = %+v", rows)
			}
			rec.Role = manifest.RoleMemory
			if _, err := Compute(Request{Recipe: rec, Resolver: res, Tool: "pi", GOOS: "linux", GOARCH: "amd64"}); err == nil {
				t.Fatal("invalid role planned")
			}
		})
	}
}

func TestDirectoryComputeRejectsRestoredCorruption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*manifest.Recipe)
	}{
		{"role", func(r *manifest.Recipe) { r.Role = manifest.RoleCapability }},
		{"local", func(r *manifest.Recipe) { r.Scope = &manifest.RecipeScope{Marker: ".sample"} }},
		{"wire", func(r *manifest.Recipe) { r.Wire.Tools = []string{"pi"} }},
		{"digest", func(r *manifest.Recipe) { r.Delivery.Assets[0].SHA256 = "bad" }},
		{"version", func(r *manifest.Recipe) { r.APIVersion = "patronus/v2" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := directoryRecipe()
			rec.Name, rec.Role = "sample-package", manifest.RoleOrchestration
			tc.mutate(rec)
			res, home, _ := testEnv(t)
			if _, err := Compute(Request{Recipe: rec, Resolver: res, Tool: "pi", GOOS: "linux", GOARCH: "amd64"}); err == nil {
				t.Fatal("corrupt recipe planned")
			}
			if _, err := os.Stat(filepath.Join(home, ".patronus")); !os.IsNotExist(err) {
				t.Fatalf("planning wrote files: %v", err)
			}
		})
	}
}

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

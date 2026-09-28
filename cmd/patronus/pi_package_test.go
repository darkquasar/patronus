package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestPiPackageInstallWithoutSbx(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := manifest.LoadRecipe(filepath.Join(root, "recipes/pi-sandbox.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	built, err := buildPackage(root, "pi-sandbox", outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 {
		t.Fatalf("built %d platforms", len(built))
	}
	asset := rec.Delivery.Assets[0]
	if asset.URL != registry.DefaultRegistryURL+"/"+built[0].Key || "sha256:"+strings.TrimPrefix(asset.SHA256, "sha256:") != built[0].SHA256 {
		t.Fatal("recipe pin differs from actual package build")
	}
	f := newDirectoryFixture(t)
	f.fetcher.bodies[asset.URL] = mustRead(t, filepath.Join(outDir, built[0].Key))
	lookups := 0
	directoryLookPath = func(name string) (string, error) {
		lookups++
		if name != "sbx" {
			t.Fatalf("unexpected executable lookup %q", name)
		}
		return "", os.ErrNotExist
	}
	res := toolpath.New(func(k string) (string, bool) { return f.home, k == "HOME" }, f.home, f.root)
	// Exercise the real Darwin payload on Linux through the existing platform
	// selection seam, then pass that plan through the actual CLI deploy boundary.
	diffs, err := recipe.Compute(recipe.Request{Recipe: rec, Resolver: res, GOOS: "darwin", GOARCH: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Directory == nil || diffs[0].Exec != nil {
		t.Fatalf("unexpected plan: %+v", diffs)
	}
	cmd := newInstallCmd()
	cmd.SetContext(context.Background())
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	runner := &fakeRunner{}
	if err := runDeployWith(cmd, &diff.ChangeSet{Diffs: diffs}, res, deployOptions{home: f.home, projectDir: f.root}, runner); err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 0 || f.fetcher.calls != 1 || lookups != 1 {
		t.Fatalf("runner=%v fetches=%d lookups=%d", runner.ran, f.fetcher.calls, lookups)
	}
	if !strings.Contains(output.String(), "Package installed; install sbx before using it.") {
		t.Fatal(output.String())
	}
	receipt, err := packagestate.Load(f.home, "pi-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if receipt == nil || len(receipt.Files) != 5 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	for _, name := range []string{"spec.yaml", "README.md", "LICENSE", "NOTICE"} {
		got := mustRead(t, filepath.Join(f.home, ".patronus/packages/pi-sandbox", name))
		want := mustRead(t, filepath.Join(root, "packages/pi-sandbox", name))
		if !bytes.Equal(got, want) {
			t.Errorf("installed %s differs", name)
		}
	}
}

func TestPiPackageUnsupportedPlatform(t *testing.T) {
	rec, err := manifest.LoadRecipe("../../recipes/pi-sandbox.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ os, arch string }{{"linux", "arm64"}, {"darwin", "amd64"}} {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			if _, err := recipe.Compute(recipe.Request{Recipe: rec, GOOS: tt.os, GOARCH: tt.arch}); err == nil {
				t.Fatal("unsupported host accepted")
			}
		})
	}
}

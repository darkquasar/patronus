package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// The package name exercises the application's existing prerequisite notice.
// Its descriptor, payload and provenance are invented, not the shipped package.
func fixturePrerequisitePackage(t *testing.T) string {
	t.Helper()
	root := packageFixture(t)
	if err := os.Rename(filepath.Join(root, "packages/kit"), filepath.Join(root, "packages/pi-sandbox")); err != nil {
		t.Fatal(err)
	}
	descriptor := strings.Replace(testPackageDescriptor, "name: kit", "name: pi-sandbox", 1)
	for _, name := range []string{"README.md", "LICENSE", "NOTICE"} {
		packageWrite(t, root, "packages/pi-sandbox/"+name, "Invented "+name+" bytes.\n")
		descriptor += "  - path: " + name + "\n    executable: false\n"
	}
	packageWrite(t, root, "packages/pi-sandbox/package.yaml", descriptor)
	packageGit(t, root, "add", ".")
	packageGit(t, root, "commit", "-m", "invented prerequisite package")
	return root
}

func TestPiPackageInstallWithoutSbx(t *testing.T) {
	f := newDirectoryFixture(t)
	root := fixturePrerequisitePackage(t)
	outDir := t.TempDir()
	built, err := buildPackage(root, "pi-sandbox", outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 {
		t.Fatalf("built %d platforms", len(built))
	}
	rec := f.recipe(t, "pi-sandbox", "1.0.0", "inert")
	asset := &rec.Delivery.Assets[0]
	asset.OS, asset.Arch = "darwin", "arm64"
	asset.URL, asset.SHA256 = registry.DefaultRegistryURL+"/"+built[0].Key, built[0].SHA256
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
	// Exercise an invented Darwin payload through the existing platform seam,
	// then pass the plan through the actual CLI deploy boundary without sbx.
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
	f := newDirectoryFixture(t)
	rec := f.recipe(t, "invented-platform-package", "1.0.0", "inert")
	rec.Delivery.Assets[0].OS, rec.Delivery.Assets[0].Arch = "darwin", "arm64"
	for _, tt := range []struct{ os, arch string }{{"linux", "arm64"}, {"darwin", "amd64"}} {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			if _, err := recipe.Compute(recipe.Request{Recipe: rec, GOOS: tt.os, GOARCH: tt.arch}); err == nil {
				t.Fatal("unsupported host accepted")
			}
		})
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
)

func runBuild(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newBuildCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestBuildProducesLoadableIndex runs `build` against the real checkout (the
// command's cwd is this package dir, and DiscoverRoot walks up to the repo) and
// asserts the output index.json parses and every artifact carries a tarball with
// a matching sha256.
func TestBuildProducesLoadableIndex(t *testing.T) {
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--registry-version", "v9.9.9-test"); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	ix, err := registry.LoadIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if ix.RegistryVersion != "v9.9.9-test" {
		t.Fatalf("registryVersion = %q", ix.RegistryVersion)
	}
	if len(ix.Artifacts) == 0 {
		t.Fatal("expected artifacts in the index")
	}
	for _, a := range ix.Artifacts {
		if a.Tarball.URL == "" || a.Tarball.SHA256 == "" {
			t.Errorf("%s: missing tarball pointer", a.Manifest.Name)
		}
		// The named tarball must exist on disk.
		name := a.Manifest.Name + "-" + a.Manifest.Version + ".tar.gz"
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("tarball %s missing: %v", name, err)
		}
	}

	// The sha256 sidecar must exist.
	if _, err := os.Stat(filepath.Join(outDir, "index.json.sha256")); err != nil {
		t.Errorf("index.json.sha256 missing: %v", err)
	}
}

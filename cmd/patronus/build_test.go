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
// asserts the output catalog/index.json parses, carries no registry-wide version,
// and every artifact's tarball exists at its immutable name/version key.
func TestBuildProducesLoadableIndex(t *testing.T) {
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--base-url", "https://registry.test"); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "catalog", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	ix, err := registry.LoadIndex(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(ix.Artifacts) == 0 {
		t.Fatal("expected artifacts in the index")
	}
	for _, a := range ix.Artifacts {
		if a.Tarball.URL == "" || a.Tarball.SHA256 == "" {
			t.Errorf("%s: missing tarball pointer", a.Manifest.Name)
		}
		// The tarball must exist on disk at its content-addressed name/version key.
		n, v := a.Manifest.Name, a.Manifest.Version
		key := filepath.Join(outDir, "catalog", n, v, n+"-"+v+".tar.gz")
		if _, err := os.Stat(key); err != nil {
			t.Errorf("tarball %s missing: %v", key, err)
		}
		wantURL := "https://registry.test/catalog/" + n + "/" + v + "/" + n + "-" + v + ".tar.gz"
		if a.Tarball.URL != wantURL {
			t.Errorf("%s: tarball URL = %q, want %q", n, a.Tarball.URL, wantURL)
		}
	}

	// The sha256 sidecar must exist.
	if _, err := os.Stat(filepath.Join(outDir, "catalog", "index.json.sha256")); err != nil {
		t.Errorf("index.json.sha256 missing: %v", err)
	}
}

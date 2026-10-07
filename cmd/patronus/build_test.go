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

// Build packaging and immutable-key behavior use invented input. The catalog
// gate checks the same structural contract against shipped metadata separately.
func TestBuildProducesLoadableIndex(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatal(err)
	}
	ix, err := registry.LoadIndex(mustRead(t, filepath.Join(outDir, "catalog", "index.json")))
	if err != nil {
		t.Fatal(err)
	}
	if len(ix.Artifacts) == 0 {
		t.Fatal("expected fixture artifacts")
	}
	for _, a := range ix.Artifacts {
		n, v := a.Manifest.Name, a.Manifest.Version
		key := filepath.Join(outDir, "catalog", n, v, n+"-"+v+".tar.gz")
		if _, err := os.Stat(key); err != nil {
			t.Errorf("missing tarball: %v", err)
		}
		want := testRegistryBase + "/catalog/" + n + "/" + v + "/" + n + "-" + v + ".tar.gz"
		if a.Tarball.URL != want || a.Tarball.SHA256 != shaOf(mustRead(t, key)) {
			t.Errorf("incorrect immutable pointer: %+v", a)
		}
	}
	raw := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	if string(mustRead(t, filepath.Join(outDir, "catalog", "index.json.sha256"))) != shaOf(raw)+"\n" {
		t.Fatal("incorrect index digest")
	}
}

func TestFixtureRegistryRejectsUnprovidedBinary(t *testing.T) {
	f := builtRegistry(t)
	if _, err := f.Fetch(t.Context(), "https://unprovided.invalid/binary"); !os.IsNotExist(err) {
		t.Fatalf("unprovided fetch accepted: %v", err)
	}
}

func TestBuildLegacyCatalogUsesSchemaOne(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	out := t.TempDir()
	if _, err := runBuild(t, "--out", out); err != nil {
		t.Fatal(err)
	}
	ix, err := registry.LoadIndex(mustRead(t, filepath.Join(out, "catalog", "index.json")))
	if err != nil {
		t.Fatal(err)
	}
	if ix.SchemaVersion != 1 {
		t.Fatalf("legacy writer schema = %d, want 1", ix.SchemaVersion)
	}
}

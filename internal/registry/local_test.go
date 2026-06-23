package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeManifest creates dir/name with content.
func writeManifest(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// scaffoldRepo lays down a minimal repo root (artifacts/ + adapters/) under a
// temp dir and returns the root.
func scaffoldRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "adapters"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestDiscoverRootFindsArtifactsPlusAdapters(t *testing.T) {
	root := scaffoldRepo(t)
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverRoot(nested)
	if err != nil {
		t.Fatalf("DiscoverRoot: %v", err)
	}
	// macOS /var -> /private/var symlinks; compare resolved paths.
	gotR, _ := filepath.EvalSymlinks(got)
	rootR, _ := filepath.EvalSymlinks(root)
	if gotR != rootR {
		t.Errorf("DiscoverRoot = %q, want %q", gotR, rootR)
	}
}

func TestDiscoverRootViaGoMod(t *testing.T) {
	// A dir with no artifacts/+adapters/ but a go.mod declaring the module.
	root := t.TempDir()
	writeManifest(t, root, "go.mod", "module github.com/darkquasar/patronus\n\ngo 1.25\n")
	got, err := DiscoverRoot(root)
	if err != nil {
		t.Fatalf("DiscoverRoot via go.mod: %v", err)
	}
	gotR, _ := filepath.EvalSymlinks(got)
	rootR, _ := filepath.EvalSymlinks(root)
	if gotR != rootR {
		t.Errorf("DiscoverRoot = %q, want %q", gotR, rootR)
	}
}

func TestDiscoverRootNotFound(t *testing.T) {
	// A bare temp dir with no markers and no patronus go.mod above it.
	dir := t.TempDir()
	writeManifest(t, dir, "go.mod", "module example.com/other\n")
	if _, err := DiscoverRoot(dir); err == nil {
		t.Error("expected error when not inside a patronus repo")
	}
}

func TestCatalogLoadsScaffoldedItems(t *testing.T) {
	root := scaffoldRepo(t)
	writeManifest(t, filepath.Join(root, "artifacts", "skills", "demo"), "patronus.yaml",
		"apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: demo\ndescription: d\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n")
	writeManifest(t, filepath.Join(root, "recipes"), "rec.yaml",
		"apiVersion: patronus/v2\nfamily: recipe\nname: rec\nrole: tools\nwire:\n  mode: mcp\n  mcp:\n    transport: http\n    url: https://x\n")
	writeManifest(t, filepath.Join(root, "profiles"), "prof.yaml",
		"apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: prof\nlayers:\n  capabilities: [demo]\n")

	cat, err := NewLocalRegistry(root).Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(cat.Artifacts) != 1 || len(cat.Recipes) != 1 || len(cat.Profiles) != 1 {
		t.Fatalf("counts: a=%d r=%d p=%d", len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
	}
	if NewLocalRegistry(root).Root() != root {
		t.Error("Root() mismatch")
	}
}

func TestCatalogRejectsNameCollision(t *testing.T) {
	root := scaffoldRepo(t)
	// Same name "dup" used by an artifact AND a recipe -> name-uniqueness error.
	writeManifest(t, filepath.Join(root, "artifacts", "skills", "dup"), "patronus.yaml",
		"apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nname: dup\ndescription: d\ntargets: [claude]\ndefaults:\n  scope: project\n")
	writeManifest(t, filepath.Join(root, "recipes"), "dup.yaml",
		"apiVersion: patronus/v2\nfamily: recipe\nname: dup\nrole: tools\nwire:\n  mode: mcp\n  mcp:\n    transport: http\n    url: https://x\n")

	if _, err := NewLocalRegistry(root).Catalog(context.Background()); err == nil {
		t.Error("expected name-collision error across artifact and recipe")
	}
}

func TestCatalogPropagatesBadManifest(t *testing.T) {
	root := scaffoldRepo(t)
	// An artifact with an invalid family must fail the whole catalog load (fail
	// loud, never ship a half-loaded catalog).
	writeManifest(t, filepath.Join(root, "artifacts", "skills", "broken"), "patronus.yaml",
		"apiVersion: patronus/v2\nfamily: recipe\ntype: skill\nname: broken\ndescription: d\n")
	if _, err := NewLocalRegistry(root).Catalog(context.Background()); err == nil {
		t.Error("expected catalog load to fail on a mis-declared manifest")
	}
}

func TestLocalRegistryLoadsPlugins(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `apiVersion: patronus/v2
family: plugin
role: lifecycle
name: superpowers
description: d
version: 2.1.0
sources:
  claude-code:
    kind: marketplace
    marketplace: claude-plugins-official
    plugin: superpowers
    ref: v2.1.0
targets: [claude]
defaults:
  scope: user
`
	if err := os.WriteFile(filepath.Join(root, "plugins", "superpowers.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := NewLocalRegistry(root).Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(cat.Plugins) != 1 {
		t.Fatalf("plugins = %d, want 1", len(cat.Plugins))
	}
	if cat.Plugins[0].Manifest.Name != "superpowers" {
		t.Errorf("name = %s, want superpowers", cat.Plugins[0].Manifest.Name)
	}
	if cat.Plugins[0].Source.LocalDir != filepath.Join(root, "plugins") {
		t.Errorf("source dir = %s, want %s", cat.Plugins[0].Source.LocalDir, filepath.Join(root, "plugins"))
	}
}

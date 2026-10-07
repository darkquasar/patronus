package registry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// repoRoot walks up from the test's working directory to the directory holding
// the built-in adapters. Actual adapters are application contracts; the shipped
// artifacts/ and profiles/ catalog trees are deliberately not required here.
// Real-catalog structure is checked by tools/catalog-check and the public CLI
// catalog gate, not by application tests.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if isDir(filepath.Join(dir, "adapters")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found (no adapters/ above cwd)")
		}
		dir = parent
	}
}

// inventedOntologyCatalog lays down a small invented catalog that carries one
// recipe of every shape, an attributed artifact and a lifecycle profile.
func inventedOntologyCatalog(t *testing.T) string {
	t.Helper()
	root := scaffoldRepo(t)
	writeManifest(t, filepath.Join(root, "artifacts", "skills", "invented-guide"), "patronus.yaml",
		"apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: invented-guide\nversion: 1.2.3\ndescription: invented fixture\nentry: SKILL.md\n"+
			"attribution: {upstream: example.invalid/upstream, license: MIT, copyright: Fixture}\n")
	writeManifest(t, filepath.Join(root, "artifacts", "instructions", "invented-rule"), "patronus.yaml",
		"apiVersion: patronus/v2\nfamily: artifact\ntype: instruction\nrole: instruction\nname: invented-rule\nversion: 0.1.0\ndescription: invented fixture\nentry: RULE.md\nrequires: [invented-guide]\n")
	base := "apiVersion: patronus/v2\nfamily: recipe\nrole: tools\nversion: 2.3.4\n"
	merge := "wire: {method: merge, actor: patronus, mcp: {transport: stdio, command: invented}}\n"
	delivery := "deliver: {via: script}\n"
	for name, body := range map[string]string{
		"invented-wire":    merge,
		"invented-install": delivery,
		"invented-fetch":   delivery + merge,
		"invented-run":     delivery + "wire: {method: exec, actor: external, run: [invented]}\n",
	} {
		writeManifest(t, filepath.Join(root, "recipes"), name+".yaml", base+"name: "+name+"\n"+body)
	}
	writeManifest(t, filepath.Join(root, "profiles"), "invented-profile.yaml",
		"apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented-profile\nversion: 1.0.0\nlayers:\n  instructions: [invented-rule]\n  tools: [invented-wire]\n")
	return root
}

// TestInventedCatalogLoadsAndMatchesOntology proves the local loaders and the
// catalog-policy helpers agree on family/type/role, SemVer, attribution and the
// deliver×wire Shape, using invented data only. Shipped-catalog membership,
// attribution coverage and per-shape availability are catalog-gate concerns.
func TestInventedCatalogLoadsAndMatchesOntology(t *testing.T) {
	cat, err := NewLocalRegistry(inventedOntologyCatalog(t)).Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading invented catalog: %v", err)
	}
	if len(cat.Artifacts) != 2 || len(cat.Recipes) != 4 || len(cat.Profiles) != 1 {
		t.Fatalf("counts: a=%d r=%d p=%d", len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
	}
	for _, entry := range cat.Artifacts {
		if err := entry.Manifest.Validate(); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
		if err := cp00ValidateMeta(entry.Manifest.Header()); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
		if entry.Manifest.Name == "invented-guide" {
			at := entry.Manifest.Attribution
			if at == nil || at.Upstream == "" || at.License == "" || at.Copyright == "" {
				t.Errorf("attribution not loaded: %+v", at)
			}
		}
	}
	if deps, ok := cat.Deps("invented-rule"); !ok || len(deps) != 1 || deps[0] != "invented-guide" {
		t.Errorf("requires edge not loaded: %v %v", deps, ok)
	}
	shapeSeen := map[manifest.RecipeShape]bool{}
	for _, entry := range cat.Recipes {
		if err := cp00ValidateRecipe(entry.Manifest); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
		shapeSeen[entry.Manifest.Shape()] = true
	}
	for _, sh := range []manifest.RecipeShape{
		manifest.ShapeWireOnly, manifest.ShapeFetchWire,
		manifest.ShapeFetchRun, manifest.ShapeInstall,
	} {
		if !shapeSeen[sh] {
			t.Errorf("loader lost recipe shape %q", sh)
		}
	}
	for _, entry := range cat.Profiles {
		if err := cp00ValidateMeta(entry.Manifest.Header()); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
	}
}

// TestRealAdaptersLoad discovers every shipped adapter, including newly added
// tools. Built-in adapters are application contracts, not catalog content.
// Unsupported surfaces may be omitted; declared surfaces must be usable.
func TestRealAdaptersLoad(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(repoRoot(t), "adapters", "*.yaml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("discover adapters: paths=%v, err=%v", paths, err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			ad, err := manifest.LoadAdapter(path)
			if err != nil {
				t.Fatal(err)
			}
			if ad.Tool != strings.TrimSuffix(filepath.Base(path), ".yaml") {
				t.Errorf("adapter tool %q disagrees with filename", ad.Tool)
			}
			if err := cp00ValidateAdapter(ad); err != nil {
				t.Fatal(err)
			}
		})
	}
}

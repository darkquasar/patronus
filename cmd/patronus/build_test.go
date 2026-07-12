package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
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

	// --- the VALIDITY GATE (Class C) ------------------------------------------
	// This is the ONE test that reads the real catalog for its own sake. It reads the
	// catalog's SHAPE — never its PINS: it asserts a digest is well-FORMED, and never
	// what that digest IS. It must never fetch, hash upstream bytes, or install.
	//
	// It has to exist. build-registry.yml is paths:-gated on artifacts/**|recipes/**,
	// so it does NOT run on a Go-only PR — and every other test has now been unbound
	// from the catalog's contents. Without this gate, a typo in a recipe, a profile
	// slot naming a deleted artifact, or a requires: edge pointing at nothing would
	// ship GREEN.
	root, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := registry.NewLocalRegistry(root).Catalog(context.Background())
	if err != nil {
		t.Fatalf("the real catalog does not load: %v", err)
	}

	// Every item name the catalog knows (artifacts + recipes; profiles are excluded
	// because they are not `requires` targets).
	known := map[string]bool{}
	for _, n := range cat.ItemNames() {
		known[n] = true
	}

	// 1. Every pinned sha256 is well-formed: 64 lowercase hex chars. The VALUE is
	//    never asserted — that would re-couple this test to upstream.
	for _, e := range cat.Recipes {
		r := e.Manifest
		if r.Delivery == nil {
			continue // wire-only recipe (remote MCP): nothing to pin
		}
		if r.Delivery.SHA256 != "" {
			assertWellFormedSHA(t, "recipe "+r.Name, r.Delivery.SHA256)
		}
		for _, a := range r.Delivery.Assets {
			assertWellFormedSHA(t, "recipe "+r.Name+" asset "+a.OS+"/"+a.Arch, a.SHA256)
		}
	}

	// 2. Every requires: target resolves to a real item.
	//
	// `build` ALREADY rejects a dangling edge ("requires: dangling edge(s)"), so the
	// runBuild above is itself the gate for this — verified: pointing ticket's
	// requires: at a non-existent item fails the build, not these lines. They are kept
	// as a belt-and-braces check on the catalog as LOADED (a future build that stopped
	// validating would still be caught here), and they are cheap.
	for _, e := range cat.Artifacts {
		a := e.Manifest
		for _, req := range a.Requires {
			if !known[req] {
				t.Errorf("artifact %s requires %q, which is not in the catalog", a.Name, req)
			}
		}
	}
	for _, e := range cat.Recipes {
		r := e.Manifest
		for _, req := range r.Requires {
			if !known[req] {
				t.Errorf("recipe %s requires %q, which is not in the catalog", r.Name, req)
			}
		}
	}

	// 3. Every profile slot names an item that exists (stripping any @tool flavour).
	//    Unlike a dangling requires:, build does NOT reject this — verified: renaming a
	//    core slot to `grilling-typo` builds green and is caught only here.
	for _, e := range cat.Profiles {
		p := e.Manifest
		for _, slot := range profileSlots(p) {
			name := slot
			if i := strings.IndexByte(name, '@'); i >= 0 {
				name = name[:i]
			}
			if !known[name] {
				t.Errorf("profile %s names %q, which is not in the catalog", p.Name, slot)
			}
		}
	}
}

// assertWellFormedSHA checks a pinned digest is 64 lowercase hex chars. It asserts the
// digest's FORM, never its VALUE: asserting a value would re-bind the suite to an
// upstream release, which is the coupling this whole test surface exists to remove.
func assertWellFormedSHA(t *testing.T, who, sum string) {
	t.Helper()
	if len(sum) != 64 {
		t.Errorf("%s: sha256 %q is %d chars, want 64", who, sum, len(sum))
		return
	}
	for _, c := range sum {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("%s: sha256 %q is not lowercase hex", who, sum)
			return
		}
	}
}

// profileSlots flattens every layer slot of a profile into one list, so the validity
// gate can check that each names a real item.
//
// ⚠️ ProfileLayers has TEN slots and Memory is a lone SCALAR (one memory recipe per
// environment) — see internal/manifest/profile.go. A slot omitted here is a layer the
// gate SILENTLY STOPS COVERING. If ProfileLayers gains a slot, add it here.
func profileSlots(p *manifest.Profile) []string {
	l := p.Layers
	var out []string
	out = append(out, l.Instructions...)
	out = append(out, l.Capabilities...)
	out = append(out, l.Context...)
	out = append(out, l.Tools...)
	out = append(out, l.Sandbox...)
	out = append(out, l.Observability...)
	out = append(out, l.Eval...)
	out = append(out, l.Guardrails...)
	out = append(out, l.Orchestration...)
	if l.Memory != "" {
		out = append(out, l.Memory)
	}
	return out
}

// TestProfileSlotsCoversEveryLayer guards the guard: profileSlots is a hand-written
// flattening of manifest.ProfileLayers, so a NEW layer slot would be silently dropped
// from the validity gate — the gate would keep passing while quietly covering less.
// This fails the moment ProfileLayers gains a field, forcing profileSlots to be updated.
func TestProfileSlotsCoversEveryLayer(t *testing.T) {
	// Fill every slot with a distinct sentinel, then assert profileSlots returns all
	// of them. Reflection drives the fill so a new field cannot be forgotten here either.
	var layers manifest.ProfileLayers
	v := reflect.ValueOf(&layers).Elem()
	want := map[string]bool{}
	for i := range v.NumField() {
		name := "slot-" + v.Type().Field(i).Name
		want[name] = true
		switch f := v.Field(i); f.Kind() {
		case reflect.Slice:
			f.Set(reflect.ValueOf(manifest.StringList{name}))
		case reflect.String:
			f.SetString(name)
		default:
			t.Fatalf("ProfileLayers.%s has unhandled kind %s — teach this test (and profileSlots) about it",
				v.Type().Field(i).Name, f.Kind())
		}
	}

	got := map[string]bool{}
	for _, s := range profileSlots(&manifest.Profile{Layers: layers}) {
		got[s] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("profileSlots drops %s — the validity gate silently stops covering that layer. "+
				"Add it to profileSlots.", name)
		}
	}
}

// TestNoRealCatalogTestCanFetchABinary is the STRUCTURAL guard for the rule this whole
// test surface rests on: a real-catalog test may read the catalog's SHAPE, never its
// PINS.
//
// builtRegistry serves the real catalog's index and artifact tarballs, and NOTHING
// else — no binaries. So a real-catalog test that tried to install a recipe-delivered
// binary would FETCH, miss in the served bodies, and fail. That is already true by
// construction; this test makes it CHECKABLE rather than a convention someone can
// quietly break by adding one --deploy of a binary-bearing profile.
//
// It asserts the property at its source: the fetcher builtRegistry hands out carries
// no URL that any recipe pins. If someone teaches builtRegistry to serve a binary
// again — which is how the vendored testdata/tk crept in the first time — this fails.
func TestNoRealCatalogTestCanFetchABinary(t *testing.T) {
	f := builtRegistry(t)

	root, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := registry.NewLocalRegistry(root).Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range cat.Recipes {
		r := e.Manifest
		if r.Delivery == nil {
			continue
		}
		pinned := []string{}
		if r.Delivery.URL != "" {
			pinned = append(pinned, r.Delivery.URL)
		}
		for _, a := range r.Delivery.Assets {
			pinned = append(pinned, a.URL)
		}
		for _, url := range pinned {
			if _, served := f.bodies[url]; served {
				t.Errorf("builtRegistry serves %s's binary at %s.\n"+
					"A real-catalog test must never fetch or hash an UPSTREAM digest: those bytes are a "+
					"third party's, and `go test` runs on every PR — including fork PRs — before human "+
					"review. Serve invented bytes from the fixture catalog instead (see fixtureRegistry).",
					r.Name, url)
			}
		}
	}
}

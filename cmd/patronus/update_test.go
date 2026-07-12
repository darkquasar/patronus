package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
)

// TestUpdateInstalledItemFollowsNewerVersion drives the real commands end-to-end:
// install a fixture item at its catalog baseline (state records it), mutate the
// served index to advertise a newer version, then `update <item> --deploy`
// re-installs the newer version and state records it. A second update reports
// up-to-date.
//
// CLASS A (mechanism): the item's identity is irrelevant to "update follows the
// newer version", so it is the fixture's. It keeps the explicit build+serve form
// because it MUTATES the built index — and preserves the ordering rule: the build
// runs while cwd is the fixture root, BEFORE withRemoteEnv chdirs into a dir where
// DiscoverRoot fails by design.
func TestUpdateInstalledItemFollowsNewerVersion(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixRawURL] = fixRawBinary
	f.bodies[fixArchiveURL] = fixArchiveTarGz(t)
	home := withRemoteEnv(t, f)

	// Baseline = whatever the fixture catalog actually advertises (read, not
	// hardcoded). The "newer" version is a fixed, obviously-synthetic value the test
	// fabricates — update compares versions by string equality, so any distinct value
	// is "newer".
	const newerVer = "99.0.0"
	baseVer := catalogItemVersion(t, outDir, "fix-skill")
	if baseVer == newerVer {
		t.Fatalf("baseline version unexpectedly equals the synthetic %q", newerVer)
	}

	// Install the baseline at the global scope.
	if _, _, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v", err)
	}
	statePath := filepath.Join(home, ".patronus", "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Find("fix-skill", "claude", "global")
	if len(got) != 1 || got[0].ItemVersion != baseVer {
		t.Fatalf("expected recorded version %s, got %+v", baseVer, got)
	}

	// Mutate the served index to advertise fix-skill@<newerVer>, serving a new
	// tarball at its own immutable key (the baseline tarball stays served too).
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	newTgz := mustTarGz(t, map[string][]byte{
		"patronus.yaml": []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fix-skill\ndescription: d\nversion: " + newerVer + "\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n"),
		"SKILL.md":      []byte("# fix-skill v" + newerVer + " body"),
	})
	newURL := testRegistryBase + "/catalog/fix-skill/" + newerVer + "/fix-skill-" + newerVer + ".tar.gz"
	for i := range ix.Artifacts {
		if ix.Artifacts[i].Manifest.Name == "fix-skill" {
			ix.Artifacts[i].Manifest.Version = newerVer
			ix.Artifacts[i].Tarball = registry.Tarball{URL: newURL, SHA256: shaOf(newTgz)}
		}
	}
	mutated, _ := ix.Marshal()
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")
	f.bodies[newURL] = newTgz

	// update <name> --deploy: refreshes the cache, sees base -> newer, re-installs.
	out, _, err := runUpdate(t, "fix-skill", "--deploy")
	if err != nil {
		t.Fatalf("update --deploy: %v", err)
	}
	if !strings.Contains(out, baseVer+" -> "+newerVer) {
		t.Errorf("expected update to report %s -> %s:\n%s", baseVer, newerVer, out)
	}

	// State now records the newer version.
	s2, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	got2 := s2.Find("fix-skill", "claude", "global")
	if len(got2) != 1 || got2[0].ItemVersion != newerVer {
		t.Fatalf("expected recorded version %s after update, got %+v", newerVer, got2)
	}

	// A second update is a no-op: up to date.
	out2, _, err := runUpdate(t, "fix-skill", "--deploy")
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if !strings.Contains(out2, "up to date") {
		t.Errorf("expected 'up to date' on the second update:\n%s", out2)
	}
}

// TestUpdateUnknownItemErrors proves updating a name that isn't installed fails
// clearly rather than silently doing nothing.
func TestUpdateUnknownItemErrors(t *testing.T) {
	f := fixtureRegistry(t)
	withRemoteEnv(t, f)
	if _, _, err := runUpdate(t, "not-installed-anywhere"); err == nil {
		t.Error("expected an error updating an uninstalled item")
	}
}

// TestUpdateNoArgsRefreshesCache proves the classic no-args cache refresh still
// works after the command grew a second job.
func TestUpdateNoArgsRefreshesCache(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)
	out, _, err := runUpdate(t)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(out, "updated registry cache") {
		t.Errorf("expected cache-refresh message:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(home, ".patronus", "cache", "index-*.json"))
	if len(matches) == 0 {
		t.Error("no cache index written")
	}
}

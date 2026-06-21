package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
)

// TestUpdateInstalledItemFollowsNewerVersion drives the real commands end-to-end:
// install team-research@1.0.1 to the global scope (state records 1.0.1), mutate the
// served index to advertise 1.1.0, then `update team-research --deploy` re-installs
// the newer version and state records 1.1.0. A second update reports up-to-date.
func TestUpdateInstalledItemFollowsNewerVersion(t *testing.T) {
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build: %v", err)
	}
	f := serveTree(t, outDir)
	home := withRemoteEnv(t, f)

	// Install v1.0.1 at the global scope.
	if _, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v", err)
	}
	statePath := filepath.Join(home, ".patronus", "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Find("team-research", "claude", "global")
	if len(got) != 1 || got[0].ItemVersion != "1.0.1" {
		t.Fatalf("expected recorded version 1.0.1, got %+v", got)
	}

	// Mutate the served index to advertise team-research@1.1.0, serving a 1.1.0
	// tarball at its own immutable key (the 1.0.1 tarball stays served too).
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	newTgz := mustTarGz(t, map[string][]byte{
		"patronus.yaml": []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: team-research\ndescription: d\nversion: 1.1.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n"),
		"SKILL.md":      []byte("# team-research v1.1.0 body"),
	})
	newURL := testRegistryBase + "/catalog/team-research/1.1.0/team-research-1.1.0.tar.gz"
	for i := range ix.Artifacts {
		if ix.Artifacts[i].Manifest.Name == "team-research" {
			ix.Artifacts[i].Manifest.Version = "1.1.0"
			ix.Artifacts[i].Tarball = registry.Tarball{URL: newURL, SHA256: shaOf(newTgz)}
		}
	}
	mutated, _ := ix.Marshal()
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")
	f.bodies[newURL] = newTgz

	// update <name> --deploy: refreshes the cache, sees 1.0.1 -> 1.1.0, re-installs.
	out, _, err := runUpdate(t, "team-research", "--deploy")
	if err != nil {
		t.Fatalf("update --deploy: %v", err)
	}
	if !strings.Contains(out, "1.0.1 -> 1.1.0") {
		t.Errorf("expected update to report 1.0.1 -> 1.1.0:\n%s", out)
	}

	// State now records 1.1.0.
	s2, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	got2 := s2.Find("team-research", "claude", "global")
	if len(got2) != 1 || got2[0].ItemVersion != "1.1.0" {
		t.Fatalf("expected recorded version 1.1.0 after update, got %+v", got2)
	}

	// A second update is a no-op: up to date.
	out2, _, err := runUpdate(t, "team-research", "--deploy")
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
	f := builtRegistry(t)
	withRemoteEnv(t, f)
	if _, _, err := runUpdate(t, "not-installed-anywhere"); err == nil {
		t.Error("expected an error updating an uninstalled item")
	}
}

// TestUpdateNoArgsRefreshesCache proves the classic no-args cache refresh still
// works after the command grew a second job.
func TestUpdateNoArgsRefreshesCache(t *testing.T) {
	f := builtRegistry(t)
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

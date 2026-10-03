package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
	if _, _, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
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

// TestUpdateFollowsNewerRecipeVersion proves a RECIPE updates by the SAME uniform
// version compare as an artifact (ADR-0004): install at the baseline (state records
// its version), advertise a newer recipe version in the served index, and
// `update <recipe> --deploy` reports base -> newer and re-records the version. No
// isRecipe special-case, no "unversioned — refreshing" message. Uses the fixture's
// fetch+wire recipe (fix-mcp-bin) — a mechanism test on a fixture item, not a real
// catalog name.
func TestUpdateFollowsNewerRecipeVersion(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	home := withRemoteEnv(t, f)

	// Baseline = whatever the fixture recipe advertises (read, not hardcoded).
	const newerVer = "99.0.0"
	baseVer := catalogRecipeVersion(t, outDir, "fix-mcp-bin")
	if baseVer == "" || baseVer == newerVer {
		t.Fatalf("unexpected baseline recipe version %q", baseVer)
	}

	// Install the fetch+wire recipe on claude/global (records recipe state rows).
	if _, e, err := runInstall(t, "fix-mcp-bin", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	statePath := filepath.Join(home, ".patronus", "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Find("fix-mcp-bin", "claude", "global"); len(got) != 1 || got[0].ItemVersion != baseVer {
		t.Fatalf("expected recorded recipe version %s, got %+v", baseVer, got)
	}

	// Mutate the served index to advertise fix-mcp-bin@<newerVer>. A recipe rides
	// inline in the index (no tarball), so editing its Manifest is the whole change.
	// The newer version also changes the wiring (an extra MCP arg) so the reinstall
	// produces a real (non-idempotent) MERGE — a pure metadata bump with byte-identical
	// wiring would SKIP, exactly as it does for an artifact whose content is unchanged.
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ix.Recipes {
		if ix.Recipes[i].Manifest.Name == "fix-mcp-bin" {
			ix.Recipes[i].Manifest.Version = newerVer
			ix.Recipes[i].Manifest.Wire.Mcp.Args = []string{"mcp", "--v2"}
		}
	}
	mutated, _ := ix.Marshal()
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")

	// update <recipe> --deploy: the uniform version-compare arm reports base -> newer.
	out, e, err := runUpdate(t, "fix-mcp-bin", "--deploy")
	if err != nil {
		t.Fatalf("update recipe: %v\n%s", err, e)
	}
	if !strings.Contains(out, baseVer+" -> "+newerVer) {
		t.Errorf("expected recipe update to report %s -> %s:\n%s", baseVer, newerVer, out)
	}
	if strings.Contains(out, "unversioned") {
		t.Errorf("recipe took the removed unversioned path:\n%s", out)
	}

	// State now records the newer recipe version.
	s2, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Find("fix-mcp-bin", "claude", "global"); len(got) != 1 || got[0].ItemVersion != newerVer {
		t.Fatalf("expected recorded recipe version %s after update, got %+v", newerVer, got)
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

// TestUpdateProfileRefreshesMembers proves `update <profile>` (model A: profile-as-
// expansion) re-resolves the profile against the fresh catalog to its member names
// and updates each installed member — the profile leaves NO state row of its own, so
// this is the only way it can be updated. Install the fixture profile (records member
// rows, among them fix-skill on claude/global), then advertise a newer fix-skill in
// the served index and `update fix-all --deploy`: the member moves base -> newer.
func TestUpdateProfileRefreshesMembers(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixRawURL] = fixRawBinary
	f.bodies[fixArchiveURL] = fixArchiveTarGz(t)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	home := withRemoteEnv(t, f)

	// Baseline = whatever the fixture advertises for the member we will bump.
	const newerVer = "99.0.0"
	baseVer := catalogItemVersion(t, outDir, "fix-skill")
	if baseVer == newerVer {
		t.Fatalf("baseline version unexpectedly equals the synthetic %q", newerVer)
	}

	// Install the PROFILE on claude/global: state records its members (fix-skill among
	// them), not the profile itself.
	if _, e, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install profile: %v\n%s", err, e)
	}
	statePath := filepath.Join(home, ".patronus", "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Find("fix-skill", "claude", "global"); len(got) != 1 || got[0].ItemVersion != baseVer {
		t.Fatalf("expected recorded member version %s, got %+v", baseVer, got)
	}

	// Advertise fix-skill@<newerVer> with a new tarball at its own immutable key, and
	// change its body so the reinstall is a real (non-idempotent) write.
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

	// update <profile> --deploy: expands fix-all to its members and refreshes each; the
	// bumped member reports base -> newer.
	out, e, err := runUpdate(t, "fix-all", "--deploy")
	if err != nil {
		t.Fatalf("update profile: %v\n%s", err, e)
	}
	if !strings.Contains(out, baseVer+" -> "+newerVer) {
		t.Errorf("expected a member refresh line %s -> %s, got:\n%s", baseVer, newerVer, out)
	}
}

// TestUpdateLocalRegistryDoesNotClaimACacheWrite covers BOTH local branches of
// resolveRegistry. Neither has a cache to write, so neither may say it wrote one.
func TestUpdateLocalRegistryDoesNotClaimACacheWrite(t *testing.T) {
	assertLocalWording := func(t *testing.T, out, root string) {
		t.Helper()
		if strings.Contains(out, "updated registry cache") {
			t.Errorf("claimed a cache write against a local registry:\n%s", out)
		}
		if !strings.Contains(out, "read local registry checkout") {
			t.Errorf("did not report the local checkout it read:\n%s", out)
		}
		if !strings.Contains(out, root) {
			t.Errorf("message does not name the checkout %s:\n%s", root, out)
		}
	}

	t.Run("in-checkout fallthrough", func(t *testing.T) {
		root := fixtureCatalog(t)
		t.Chdir(root)
		out, _, err := runUpdate(t)
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		assertLocalWording(t, out, root)
	})

	// --local-registry is a BOOL: it forces registry.DiscoverRoot(wd) rather than
	// taking a path, so the flag can only select the checkout we are standing in.
	// The subtest still exercises a distinct branch (registry.go:63 vs the :70
	// fallthrough) — it just cannot be driven from outside a checkout.
	t.Run("explicit --local-registry", func(t *testing.T) {
		root := fixtureCatalog(t)
		t.Chdir(root)
		out, _, err := runUpdate(t, "--local-registry")
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		assertLocalWording(t, out, root)
	})
}

func TestUpdateReconcilesTwoFileSkillSkipAndObsoleteSidecar(t *testing.T) {
	for _, tool := range []string{"pi", "claude", "codex"} {
		for _, edited := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/edited=%t", tool, edited), func(t *testing.T) {
				f := dp02Setup(t)
				src := filepath.Join(f.root, "artifacts", "fixture-skill")
				manifest := func(version string, sidecar bool) string {
					files := ""
					if sidecar {
						files = "files: [support]\n"
					}
					return fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fixture-skill\ndescription: Invented fixture\nversion: %s\nentry: SKILL.md\ntargets: [%s]\n%s", version, tool, files)
				}
				body := "---\nname: fixture-skill\ndescription: Fixture\n---\nMain fixture\n"
				dp02File(t, filepath.Join(src, "patronus.yaml"), manifest("1", true))
				dp02File(t, filepath.Join(src, "SKILL.md"), body)
				dp02File(t, filepath.Join(src, "support", "note.txt"), "sidecar one")
				if _, _, err := runInstall(t, "fixture-skill", "--target", tool, "--global", "--deploy", "--yes"); err != nil {
					t.Fatal(err)
				}
				sp := filepath.Join(f.home, ".patronus/state.json")
				old, err := state.Load(sp)
				if err != nil {
					t.Fatal(err)
				}
				rows := old.Find("fixture-skill", tool, "global")
				if len(rows) != 1 || len(rows[0].Files) != 2 {
					t.Fatalf("initial state: %+v", rows)
				}
				sidecar := ""
				for _, file := range rows[0].Files {
					if filepath.Base(file.Path) == "note.txt" {
						sidecar = file.Path
					}
				}
				if sidecar == "" {
					t.Fatal("missing sidecar")
				}
				// Update only the sidecar; SKILL.md is a verified owned SKIP.
				dp02File(t, filepath.Join(src, "patronus.yaml"), manifest("2", true))
				dp02File(t, filepath.Join(src, "support", "note.txt"), "sidecar two")
				if _, _, err := runUpdate(t, dp06UpdateArgs(tool, "fixture-skill")...); err != nil {
					t.Fatal(err)
				}
				updated, err := state.Load(sp)
				if err != nil {
					t.Fatal(err)
				}
				rows = updated.Find("fixture-skill", tool, "global")
				if len(rows[0].Files) != 2 || rows[0].ItemVersion != "2" {
					t.Fatalf("SKIP ownership lost: %+v", rows)
				}
				if edited {
					dp02File(t, sidecar, "user-edited obsolete sidecar")
				}
				// Omission alone grants no delete authority. Keep both paths and v2 until
				// explicit removal resolves the obsolete contribution.
				dp02File(t, filepath.Join(src, "patronus.yaml"), manifest("3", false))
				if _, _, err := runUpdate(t, dp06UpdateArgs(tool, "fixture-skill")...); err == nil || !strings.Contains(err.Error(), sidecar) {
					t.Fatalf("obsolete path claimed resolved: %v", err)
				}
				updated, err = state.Load(sp)
				if err != nil {
					t.Fatal(err)
				}
				rows = updated.Find("fixture-skill", tool, "global")
				if len(rows[0].Files) != 2 || rows[0].ItemVersion != "2" {
					t.Fatalf("obsolete ownership/version lost: %+v", rows)
				}
				_, _, removeErr := execRemove(t, "fixture-skill", "--target", tool, "--global", "--deploy")
				if !edited && removeErr != nil {
					t.Fatal(removeErr)
				}
				_, statErr := os.Stat(sidecar)
				if edited && statErr != nil {
					t.Fatal("edited sidecar deleted")
				}
				if !edited && !os.IsNotExist(statErr) {
					t.Fatal("unchanged obsolete sidecar not removed")
				}
			})
		}
	}
}

func TestUpdateSettingWithEqualExternalDependency(t *testing.T) {
	for _, tool := range []string{"pi", "claude"} {
		for _, externalOnInitial := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/external-on-initial=%t", tool, externalOnInitial), func(t *testing.T) {
				f := dp02Setup(t)
				root := filepath.Join(f.home, ".claude")
				if tool == "pi" {
					root = filepath.Join(f.home, ".pi/agent")
				}
				path := filepath.Join(root, "settings.json")
				dp02File(t, path, `{"external":true,"user":"keep"}`)
				base := "apiVersion: patronus/v2\nfamily: artifact\ntype: setting\nrole: capability\ndescription: Invented fixture\ntargets: [" + tool + "]\ndefaults:\n  scope: global\n"
				dp02File(t, filepath.Join(f.root, "artifacts/fixture-external/patronus.yaml"), base+"name: fixture-external\nversion: 1\nsetting:\n  path: external\n  value: true\n")
				statePath := filepath.Join(f.home, ".patronus/state.json")
				for _, version := range []string{"1", "2"} {
					requires := "requires: [fixture-external]\n"
					if version == "1" && !externalOnInitial {
						requires = ""
					}
					dp02File(t, filepath.Join(f.root, "artifacts/fixture-owned/patronus.yaml"), base+"name: fixture-owned\nversion: "+version+"\n"+requires+"setting:\n  path: managed\n  value: managed-"+version+"\n")
					var err error
					if version == "1" {
						_, _, err = runInstall(t, "fixture-owned", "--target", tool, "--global", "--deploy", "--yes")
					} else {
						beforeConfig, beforeState := mustRead(t, path), mustRead(t, statePath)
						_, _, err = runUpdate(t, dp06UpdateArgs(tool, "fixture-owned")...)
						if tool == "pi" {
							if err == nil || !strings.Contains(err.Error(), "dependency-incomplete") {
								t.Fatalf("absent external prerequisite admitted: %v", err)
							}
							s, loadErr := state.Load(statePath)
							if loadErr != nil {
								t.Fatal(loadErr)
							}
							rows := s.Find("fixture-owned", tool, "global")
							if len(rows) != 1 || rows[0].ItemVersion != "1" || rows[0].Files[0].Setting.PriorPresent {
								t.Fatalf("refused update advanced state: %+v", s.Items)
							}
							if !bytes.Equal(beforeConfig, mustRead(t, path)) || !bytes.Equal(beforeState, mustRead(t, statePath)) {
								t.Fatal("refused update changed bytes")
							}
							// A separately selected install remains allowed: equal
							// external settings do not acquire ownership.
							_, _, err = runInstall(t, "fixture-owned", "--target", "pi", "--global", "--deploy", "--yes")
						}
					}
					if err != nil {
						t.Fatalf("version %s: %v", version, err)
					}
					s, err := state.Load(statePath)
					if err != nil {
						t.Fatal(err)
					}
					owned := s.Find("fixture-owned", tool, "global")
					if len(s.Find("fixture-external", tool, "global")) != 0 || len(owned) != 1 || owned[0].ItemVersion != version || len(owned[0].Files) != 1 || owned[0].Files[0].Setting.PriorPresent {
						t.Fatalf("external adopted or owned baseline/version lost: %+v", s.Items)
					}
					var doc map[string]any
					if err := json.Unmarshal(mustRead(t, path), &doc); err != nil {
						t.Fatal(err)
					}
					if doc["external"] != true || doc["user"] != "keep" || doc["managed"] != "managed-"+version {
						t.Fatalf("wrong config after version %s: %+v", version, doc)
					}
				}
				beforeConfig, beforeState := mustRead(t, path), mustRead(t, statePath)
				_, _, _ = execRemove(t, "fixture-external", "--target", tool, "--global", "--deploy")
				if !bytes.Equal(beforeConfig, mustRead(t, path)) || !bytes.Equal(beforeState, mustRead(t, statePath)) {
					t.Fatal("unmanaged removal changed config/state")
				}
				if _, _, err := execRemove(t, "fixture-owned", "--target", tool, "--global", "--deploy"); err != nil {
					t.Fatal(err)
				}
				var doc map[string]any
				if err := json.Unmarshal(mustRead(t, path), &doc); err != nil {
					t.Fatal(err)
				}
				if _, present := doc["managed"]; present || doc["external"] != true || doc["user"] != "keep" {
					t.Fatalf("inverse changed external/user values: %+v", doc)
				}
			})
		}
	}
}

func TestUpdatePiSettingKeepsOriginalBaselineThroughNoopReinstall(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".pi/agent/settings.json")
	dp02File(t, path, `{"user":"keep"}`)
	source := filepath.Join(f.root, "artifacts", "fixture-setting", "patronus.yaml")
	for _, version := range []string{"1", "2", "3"} {
		dp02File(t, source, fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\ntype: setting\nrole: capability\nname: fixture-setting\ndescription: Invented fixture\nversion: %s\ntargets: [pi]\ndefaults:\n  scope: global\nsetting:\n  path: fixture\n  value: installed-%s\n", version, version))
		var err error
		if version == "1" {
			_, _, err = runInstall(t, "fixture-setting", "--target", "pi", "--global", "--deploy", "--yes")
		} else {
			_, _, err = runUpdate(t, "fixture-setting", "--global", "--deploy")
		}
		if err != nil {
			t.Fatal(err)
		}
		s, err := state.Load(filepath.Join(f.home, ".patronus/state.json"))
		if err != nil {
			t.Fatal(err)
		}
		rows := s.Find("fixture-setting", "pi", "global")
		if len(rows) != 1 || rows[0].ItemVersion != version || rows[0].Files[0].Setting.PriorPresent {
			t.Fatalf("baseline/version changed incorrectly: %+v", rows)
		}
	}
	if _, _, err := runInstall(t, "fixture-setting", "--target", "pi", "--global", "--deploy", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execRemove(t, "fixture-setting", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	got := string(mustRead(t, path))
	if strings.Contains(got, "fixture") || !strings.Contains(got, `"user": "keep"`) {
		t.Fatalf("inverse did not restore absent baseline: %s", got)
	}
}

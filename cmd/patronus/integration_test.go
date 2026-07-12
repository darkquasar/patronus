package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/registry"
)

// These integration tests drive the REAL cobra commands (list/install/lock/update)
// end-to-end against a remote R2-style registry that is built by `patronus build`
// and then served from memory by a fakeFetcher — no network, no checkout. They
// prove each command actually does what it claims: that a remote install
// fetches+unpacks an item's source and transforms it for the target tool, that
// lock pins per-item provenance, that update refreshes the cache, that a profile
// install follows the LOCKED item version (not the index's latest), and that the
// installed bytes land where the adapter says they should.

const testRegistryBase = "https://registry.test"

// servingFetcher serves canned bytes per URL and counts hits so a test can assert
// a warm cache performs zero network.
type servingFetcher struct {
	bodies map[string][]byte
	hits   int
}

func (f *servingFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	f.hits++
	b, ok := f.bodies[url]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// builtRegistry builds the REAL catalog and serves its index + artifact tarballs.
// It serves NO binaries — a real-catalog test may read the catalog's SHAPE, never
// its PINS, so it must never fetch or hash an upstream digest. Its callers are the
// CLASS-B contents tests (core_profile, orchestration, core_consolidated), which
// assert against profile resolution, the lock, or the plan and never deploy a
// binary. Every mechanism (Class-A) test uses fixtureRegistry instead.
//
// A test that reaches this and then tries to install a binary will FETCH, miss in
// the served bodies, and fail loudly — which is correct: the bytes it wants are a
// third party's, and they must never enter CI.
func builtRegistry(t *testing.T) *servingFetcher {
	t.Helper()
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build registry: %v", err)
	}
	return serveTree(t, outDir)
}

// serveTree maps an on-disk R2-layout tree (<dir>/catalog/...) onto a fetcher
// keyed by the testRegistryBase URLs the index points at.
func serveTree(t *testing.T, outDir string) *servingFetcher {
	t.Helper()
	bodies := map[string][]byte{}
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	sha := mustRead(t, filepath.Join(outDir, "catalog", "index.json.sha256"))
	bodies[testRegistryBase+"/catalog/index.json"] = idx
	bodies[testRegistryBase+"/catalog/index.json.sha256"] = sha

	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range ix.Artifacts {
		n, v := a.Manifest.Name, a.Manifest.Version
		key := filepath.Join(outDir, "catalog", n, v, n+"-"+v+".tar.gz")
		bodies[a.Tarball.URL] = mustRead(t, key)
	}
	// serveTree serves the CATALOG. Binaries are the caller's business:
	// fixtureRegistry adds the fixture's invented ones; a real-catalog test must
	// not install a binary at all (it may read the catalog's SHAPE, never its PINS).
	return &servingFetcher{bodies: bodies}
}

// withRemoteEnv points the commands at a temp HOME (for the cache), a temp cwd
// OUTSIDE any checkout (so registry selection picks Remote), and the test registry
// base URL, then swaps in the given fetcher for BOTH the source seam and the
// registry seam. It restores everything on cleanup.
func withRemoteEnv(t *testing.T, f *servingFetcher) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATRONUS_REGISTRY_URL", testRegistryBase)
	// Pin the per-tool config-dir env vars INTO the temp HOME so an install can't
	// escape the sandbox via an inherited override. OpenCode resolves its global
	// config from XDG_CONFIG_HOME/OPENCODE_CONFIG_DIR and Codex from CODEX_HOME
	// (see internal/toolpath/resolver.go); on a host where any of these is already
	// set (e.g. CI runners set XDG_CONFIG_HOME=/home/runner/.config) the writes
	// would land outside HOME and the `~/.config/opencode/...` assertions would
	// miss them. Setting XDG_CONFIG_HOME to <home>/.config keeps the resolved path
	// identical to the ~/.config fallback the tests assert against.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(home, ".config", "opencode"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	// A cwd with no artifacts/ + adapters/ above it → DiscoverRoot fails → Remote.
	work := t.TempDir()
	t.Chdir(work)

	prevSrc, prevReg, prevDep := fetcherForCommands, registryFetcher, fetcherForDeploy
	fetcherForCommands = f
	registryFetcher = f
	fetcherForDeploy = f // FETCH downloads on --deploy serve from memory too (no network)
	t.Cleanup(func() {
		fetcherForCommands, registryFetcher, fetcherForDeploy = prevSrc, prevReg, prevDep
	})
	return home
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	b, err := archive.CreateTarGz(files)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func shaOf(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// catalogItemVersion reads a built registry's index and returns the advertised
// version of an artifact by name. Tests that install a REAL catalog item read its
// version from here instead of hardcoding a literal, so a future version bump of
// that item never breaks the test (the recurring breakage that motivated this).
func catalogItemVersion(t *testing.T, outDir, name string) string {
	t.Helper()
	ix, err := registry.LoadIndex(mustRead(t, filepath.Join(outDir, "catalog", "index.json")))
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	for i := range ix.Artifacts {
		if ix.Artifacts[i].Manifest.Name == name {
			return ix.Artifacts[i].Manifest.Version
		}
	}
	t.Fatalf("artifact %q not in built catalog index", name)
	return ""
}

func runList(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newListCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func runUpdate(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newUpdateCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// TestRemoteListBrowsesWithoutFetchingContent proves `list` shows the catalog from
// the index alone — it must NOT download any artifact tarball just to list.
func TestRemoteListBrowsesWithoutFetchingContent(t *testing.T) {
	f := fixtureRegistry(t)
	withRemoteEnv(t, f)

	out, _, err := runList(t)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, want := range []string{"fix-instruction", "fix-skill", "fix-hook", "fix-all"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
	// list fetched the index (+ its sha sidecar) but NO tarball: the only network
	// hits should be index + sidecar (2), proving content is never pulled to browse.
	if f.hits > 2 {
		t.Errorf("list hit the network %d times; expected only index + sha (no content)", f.hits)
	}
}

// TestRemoteInstallMaterializesAndTransforms proves a remote `install --dry-run`
// fetches+unpacks one item's source and plans the per-tool transform — the full
// remote→materialize→adapter path.
func TestRemoteInstallMaterializesAndTransforms(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	out, _, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("remote install: %v", err)
	}
	// The plan must target the Claude skill layout for the materialized artifact.
	for _, want := range []string{"fix-skill", "SKILL.md", "CREATE"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan missing %q:\n%s", want, out)
		}
	}
	// The source was materialized into the cache (patronus.yaml on disk). Glob the
	// versioned dir rather than hardcoding the version, so an item version bump
	// never breaks this — the test asserts materialization, not a specific version.
	// (fix-skill-1.* rather than fix-skill-*: the latter would also match the
	// fix-skill-claude/-codex flavour items.)
	matches, _ := filepath.Glob(filepath.Join(home, ".patronus", "cache", "items", "fix-skill-1.*", "patronus.yaml"))
	if len(matches) == 0 {
		t.Errorf("artifact source not materialized (no fix-skill-1.* in cache)")
	}
}

// TestRemoteInstallDeployWritesFiles proves a remote `install --deploy` actually
// writes the transformed artifact to disk under the (temp) global scope.
func TestRemoteInstallDeployWritesFiles(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	_, _, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--deploy", "--yes")
	if err != nil {
		t.Fatalf("remote deploy: %v", err)
	}
	skill := filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("expected installed skill at %s: %v", skill, err)
	}
	// Re-running is idempotent (SKIP), proving the same change model end-to-end.
	out, _, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}
}

// TestRemoteUpdateRefreshesCache proves `update` writes the cache so a subsequent
// `list` runs offline (zero further network).
func TestRemoteUpdateRefreshesCache(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}
	// The cache file is keyed on a hash of the base URL, so assert by glob.
	matches, _ := filepath.Glob(filepath.Join(home, ".patronus", "cache", "index-*.json"))
	if len(matches) == 0 {
		t.Fatalf("update did not write the cache")
	}
	hitsAfterUpdate := f.hits
	if _, _, err := runList(t); err != nil {
		t.Fatal(err)
	}
	if f.hits != hitsAfterUpdate {
		t.Errorf("list after update hit the network (%d → %d); cache should be warm", hitsAfterUpdate, f.hits)
	}
}

// TestRemoteLockPinsPerItemProvenance proves `lock --profile` against a remote
// registry writes a v2 lock that pins each item PER-ITEM (source + version +
// sha256 + tarballSha256) and carries NO registry-wide version.
func TestRemoteLockPinsPerItemProvenance(t *testing.T) {
	f := fixtureRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "fix-all"); err != nil {
		t.Fatalf("remote lock: %v", err)
	}
	// The lock lands in the cwd (the temp work dir).
	wd, _ := os.Getwd()
	data := mustRead(t, filepath.Join(wd, "patronus.lock"))
	s := string(data)
	if strings.Contains(s, "registryVersion") {
		t.Errorf("lock must not carry a registry-wide version:\n%s", s)
	}
	if !strings.Contains(s, `"version": 2`) {
		t.Errorf("lock should be schema v2:\n%s", s)
	}
	for _, want := range []string{`"source": "registry"`, `"tarballSha256"`, "fix-instruction", "fix-skill"} {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q:\n%s", want, s)
		}
	}
}

// TestProfileInstallFollowsPerItemLock is THE CRUX test: it proves per-item
// reality-follows-lock. After locking the cloudflare profile (// the catalog baseline), the served registry is mutated to advertise a NEWER
// synthetic newer version (both tarballs remain served, mirroring R2's
// immutable keys). A profile install must then fetch the LOCKED baseline — not the
// index's newer latest — proving the lock, not the moving index, drives
// reproducibility.
func TestProfileInstallFollowsPerItemLock(t *testing.T) {
	// This test MUTATES the built index, so it keeps the explicit build+serve form
	// rather than calling fixtureRegistry. Same ordering rule: build while cwd is
	// the fixture root, BEFORE withRemoteEnv chdirs into a dir where DiscoverRoot
	// fails by design.
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

	// Baseline = the fixture catalog's actual fix-skill version (read, not
	// hardcoded). The synthetic "newer" version is a fixed value the test fabricates.
	const newerVer = "99.0.0"
	baseVer := catalogItemVersion(t, outDir, "fix-skill")

	// Lock the profile → pins fix-skill@<baseVer> + its tarball sha.
	if _, _, err := runLock(t, "--profile", "fix-all"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	if !strings.Contains(string(mustRead(t, filepath.Join(wd, "patronus.lock"))), `"version": "`+baseVer+`"`) {
		t.Fatalf("expected fix-skill %s pinned in the lock", baseVer)
	}

	// Mutate the served index to advertise fix-skill@<newerVer> (a new, newer item)
	// while STILL serving the immutable baseline tarball. Also serve a newer tarball
	// at its own immutable key.
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	newTgz := mustTarGz(t, map[string][]byte{
		"patronus.yaml": []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fix-skill\ndescription: d\nversion: " + newerVer + "\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n"),
		"SKILL.md":      []byte("# v" + newerVer + " body — should NOT be installed"),
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

	// Refresh the cache so the client sees the mutated (newer) index.
	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Install the profile against the committed lock → must materialize the baseline.
	if _, errOut, err := runInstall(t, "--profile", "fix-all", "--tool", "claude", "--global", "--dry-run"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "cache", "items", "fix-skill-"+baseVer, "patronus.yaml")); err != nil {
		t.Errorf("lock should pin %s, but it was not materialized: %v", baseVer, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "cache", "items", "fix-skill-"+newerVer, "patronus.yaml")); err == nil {
		t.Errorf("install fetched the index's newer %s instead of the locked %s", newerVer, baseVer)
	}
}

// TestGitSourceInstallEndToEnd proves `install git:<...>#<item>` runs the real
// command end-to-end: fetch the host source archive, unpack, select the item,
// materialize it, and plan the per-tool transform — all through the install path,
// no checkout, no network.
func TestGitSourceInstallEndToEnd(t *testing.T) {
	// Build a GitHub-style source archive holding one artifact dir.
	members := map[string]string{
		"kit-v2/my-pattern/patronus.yaml": "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: context\nname: my-pattern\ndescription: d\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n",
		"kit-v2/my-pattern/SKILL.md":      "# my pattern",
	}
	files := map[string][]byte{}
	for k, v := range members {
		files[k] = []byte(v)
	}
	tgz, err := archive.CreateTarGz(files)
	if err != nil {
		t.Fatal(err)
	}
	gitURL := "https://github.com/me/kit/archive/v2.tar.gz"
	f := &servingFetcher{bodies: map[string][]byte{gitURL: tgz}}
	withRemoteEnv(t, f)

	out, errOut, err := runInstall(t, "git:github.com/me/kit@v2#my-pattern", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("git install: %v\n%s", err, errOut)
	}
	for _, want := range []string{"my-pattern", "SKILL.md", "CREATE"} {
		if !strings.Contains(out, want) {
			t.Errorf("git-sourced plan missing %q:\n%s", want, out)
		}
	}
}

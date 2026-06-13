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

// builtRegistry runs `patronus build` against the real checkout into the R2 layout
// at testRegistryBase, then serves catalog/index.json (+ .sha256) and every item
// tarball at the URLs the index records.
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
	// A cwd with no artifacts/ + adapters/ above it → DiscoverRoot fails → Remote.
	work := t.TempDir()
	t.Chdir(work)

	prevSrc, prevReg := fetcherForCommands, registryFetcher
	fetcherForCommands = f
	registryFetcher = f
	t.Cleanup(func() { fetcherForCommands, registryFetcher = prevSrc, prevReg })
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
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	out, _, err := runList(t)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, want := range []string{"team-research", "pattern-cloudflare", "memory-ai-memory", "cloudflare"} {
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
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	out, _, err := runInstall(t, "pattern-cloudflare", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("remote install: %v", err)
	}
	// The plan must target the Claude skill layout for the materialized artifact.
	for _, want := range []string{"pattern-cloudflare", "SKILL.md", "CREATE"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan missing %q:\n%s", want, out)
		}
	}
	// The source was materialized into the cache (patronus.yaml on disk).
	matzd := filepath.Join(home, ".patronus", "cache", "items", "pattern-cloudflare-1.0.0", "patronus.yaml")
	if _, err := os.Stat(matzd); err != nil {
		t.Errorf("artifact source not materialized: %v", err)
	}
}

// TestRemoteInstallDeployWritesFiles proves a remote `install --deploy` actually
// writes the transformed artifact to disk under the (temp) global scope.
func TestRemoteInstallDeployWritesFiles(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	_, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--deploy", "--yes")
	if err != nil {
		t.Fatalf("remote deploy: %v", err)
	}
	skill := filepath.Join(home, ".claude", "skills", "team-research", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("expected installed skill at %s: %v", skill, err)
	}
	// Re-running is idempotent (SKIP), proving the same change model end-to-end.
	out, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--dry-run")
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
	f := builtRegistry(t)
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
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "cloudflare"); err != nil {
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
	for _, want := range []string{`"source": "registry"`, `"tarballSha256"`, "pattern-cloudflare", "memory-ai-memory"} {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q:\n%s", want, s)
		}
	}
}

// TestProfileInstallFollowsPerItemLock is THE CRUX test: it proves per-item
// reality-follows-lock. After locking the cloudflare profile (pinning
// pattern-cloudflare@1.0.0), the served registry is mutated to advertise a NEWER
// pattern-cloudflare@1.1.0 (both versions' tarballs remain served, mirroring R2's
// immutable keys). A profile install must then fetch the LOCKED 1.0.0 — not the
// index's newer 1.1.0 latest — proving the lock, not the moving index, drives
// reproducibility.
func TestProfileInstallFollowsPerItemLock(t *testing.T) {
	// Build the baseline registry (pattern-cloudflare@1.0.0) and serve it.
	outDir := t.TempDir()
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build: %v", err)
	}
	f := serveTree(t, outDir)
	home := withRemoteEnv(t, f)

	// Lock the profile → pins pattern-cloudflare@1.0.0 + its tarball sha.
	if _, _, err := runLock(t, "--profile", "cloudflare"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	if !strings.Contains(string(mustRead(t, filepath.Join(wd, "patronus.lock"))), `"version": "1.0.0"`) {
		t.Fatal("expected pattern-cloudflare 1.0.0 pinned in the lock")
	}

	// Mutate the served index to advertise pattern-cloudflare@1.1.0 (a new, newer
	// item) while STILL serving the immutable 1.0.0 tarball. Also serve a 1.1.0
	// tarball at its own immutable key.
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	newTgz := mustTarGz(t, map[string][]byte{
		"patronus.yaml": []byte("apiVersion: patronus/v1\nkind: Skill\nrole: pattern\nname: pattern-cloudflare\ndescription: d\nversion: 1.1.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n"),
		"SKILL.md":      []byte("# v1.1.0 body — should NOT be installed"),
	})
	newURL := testRegistryBase + "/catalog/pattern-cloudflare/1.1.0/pattern-cloudflare-1.1.0.tar.gz"
	for i := range ix.Artifacts {
		if ix.Artifacts[i].Manifest.Name == "pattern-cloudflare" {
			ix.Artifacts[i].Manifest.Version = "1.1.0"
			ix.Artifacts[i].Tarball = registry.Tarball{URL: newURL, SHA256: shaOf(newTgz)}
		}
	}
	mutated, _ := ix.Marshal()
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")
	f.bodies[newURL] = newTgz

	// Refresh the cache so the client sees the mutated (1.1.0) index.
	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Install the profile against the committed lock → must materialize 1.0.0.
	if _, errOut, err := runInstall(t, "--profile", "cloudflare", "--tool", "claude", "--global", "--dry-run"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "cache", "items", "pattern-cloudflare-1.0.0", "patronus.yaml")); err != nil {
		t.Errorf("lock should pin 1.0.0, but it was not materialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "cache", "items", "pattern-cloudflare-1.1.0", "patronus.yaml")); err == nil {
		t.Error("install fetched the index's newer 1.1.0 instead of the locked 1.0.0")
	}
}

// TestGitSourceInstallEndToEnd proves `install git:<...>#<item>` runs the real
// command end-to-end: fetch the host source archive, unpack, select the item,
// materialize it, and plan the per-tool transform — all through the install path,
// no checkout, no network.
func TestGitSourceInstallEndToEnd(t *testing.T) {
	// Build a GitHub-style source archive holding one artifact dir.
	members := map[string]string{
		"kit-v2/my-pattern/patronus.yaml": "apiVersion: patronus/v1\nkind: Skill\nrole: pattern\nname: my-pattern\ndescription: d\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n",
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

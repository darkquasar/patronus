package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/registry"
)

// These integration tests drive the REAL cobra commands (list/install/lock/update)
// end-to-end against a remote registry that is built by `patronus build` and then
// served from memory by a fakeFetcher — no network, no checkout. They prove each
// command actually does what it claims: that a remote install fetches+unpacks an
// item's source and transforms it for the target tool, that lock pins the registry
// tag and per-item provenance, that update refreshes the cache, and that the
// installed bytes land where the adapter says they should.

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

// builtRegistry runs `patronus build` against the real checkout, then maps every
// produced file (index.json, its .sha256, and each tarball) to the GitHub Release
// download URL the index points at, returning a fetcher that serves them. tag is
// the registry version the URLs are built for.
func builtRegistry(t *testing.T, tag string) *servingFetcher {
	t.Helper()
	outDir := t.TempDir()
	base := "https://github.com/darkquasar/patronus/releases/download/" + tag
	if _, err := runBuild(t, "--out", outDir, "--registry-version", tag, "--base-url", base); err != nil {
		t.Fatalf("build registry: %v", err)
	}

	bodies := map[string][]byte{}
	// The index is fetched from the "latest" URL by an unpinned RemoteRegistry and
	// from the "<tag>" URL by a pinned one; serve both so either policy works.
	idx := mustRead(t, filepath.Join(outDir, "index.json"))
	sha := mustRead(t, filepath.Join(outDir, "index.json.sha256"))
	for _, u := range []string{
		"https://github.com/darkquasar/patronus/releases/latest/download/index.json",
		base + "/index.json",
	} {
		bodies[u] = idx
		bodies[u+".sha256"] = sha
	}
	// Each artifact tarball is served at the URL the index recorded.
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range ix.Artifacts {
		name := a.Manifest.Name + "-" + a.Manifest.Version + ".tar.gz"
		bodies[a.Tarball.URL] = mustRead(t, filepath.Join(outDir, name))
	}
	return &servingFetcher{bodies: bodies}
}

// withRemoteEnv points the commands at a temp HOME (for the cache) and a temp cwd
// OUTSIDE any checkout (so registry selection picks Remote), and swaps in the
// given fetcher. It restores everything on cleanup.
func withRemoteEnv(t *testing.T, f *servingFetcher) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	// A cwd with no artifacts/ + adapters/ above it → DiscoverRoot fails → Remote.
	work := t.TempDir()
	t.Chdir(work)

	prev := fetcherForCommands
	fetcherForCommands = f
	t.Cleanup(func() { fetcherForCommands = prev })
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
	f := builtRegistry(t, "v1.2.3")
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
	f := builtRegistry(t, "v1.2.3")
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
	f := builtRegistry(t, "v1.2.3")
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
	f := builtRegistry(t, "v1.2.3")
	home := withRemoteEnv(t, f)

	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "cache", "index-latest.json")); err != nil {
		t.Fatalf("update did not write the cache: %v", err)
	}
	hitsAfterUpdate := f.hits
	if _, _, err := runList(t); err != nil {
		t.Fatal(err)
	}
	if f.hits != hitsAfterUpdate {
		t.Errorf("list after update hit the network (%d → %d); cache should be warm", hitsAfterUpdate, f.hits)
	}
}

// TestRemoteLockPinsRegistryAndProvenance proves `lock --profile` against a remote
// registry records the registry tag and per-item source provenance.
func TestRemoteLockPinsRegistryAndProvenance(t *testing.T) {
	f := builtRegistry(t, "v1.2.3")
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "cloudflare"); err != nil {
		t.Fatalf("remote lock: %v", err)
	}
	// The lock lands in the cwd (the temp work dir).
	wd, _ := os.Getwd()
	data := mustRead(t, filepath.Join(wd, "patronus.lock"))
	s := string(data)
	if !strings.Contains(s, `"registryVersion": "v1.2.3"`) {
		t.Errorf("lock missing registry pin:\n%s", s)
	}
	if !strings.Contains(s, `"source": "registry"`) {
		t.Errorf("lock missing source provenance:\n%s", s)
	}
	if !strings.Contains(s, "pattern-cloudflare") || !strings.Contains(s, "memory-ai-memory") {
		t.Errorf("lock missing resolved items:\n%s", s)
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

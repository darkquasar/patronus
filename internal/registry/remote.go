package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/install"
)

// Fetcher downloads the bytes at a URL. It mirrors recipe.Fetcher so the cmd
// layer can inject recipe.HTTPFetcher and tests inject a fake — declared here to
// avoid a registry→recipe import cycle.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// DefaultOwner / DefaultRepo are the official registry's GitHub coordinates.
const (
	DefaultOwner = "darkquasar"
	DefaultRepo  = "patronus"
)

// RemoteRegistry reads a published index.json (one per GitHub Release) and serves
// it behind the Registry interface, so the planner/profile-resolver are unchanged
// from the local case. It caches the index under CacheDir and materializes an
// item's portable source on demand (Materialize), at which point the entry is
// indistinguishable from a local-checkout entry.
//
// Refresh policy is apt-style EXPLICIT-ONLY (DESIGN §6, Phase-6 decision): a warm
// cache is never auto-refreshed by Catalog; only a cold cache bootstraps a single
// fetch, and `patronus update` (Refresh) forces a re-fetch. This keeps day-to-day
// commands offline and free of per-invocation network round-trips.
type RemoteRegistry struct {
	Fetcher  Fetcher
	CacheDir string // ~/.patronus/cache
	Owner    string // "" => DefaultOwner
	Repo     string // "" => DefaultRepo
	Version  string // "" => latest; else a pinned release tag
	BaseURL  string // "" => GitHub Releases; override for a fork/mirror
	Warnf    func(string, ...any)

	// resolvedTag is the concrete registry tag the last-loaded index reported. For
	// an unpinned ("latest") registry this is how a caller learns which tag it
	// actually resolved to (e.g. to record in patronus.lock).
	resolvedTag string
}

// ResolvedVersion returns the concrete registry tag of the most recently loaded
// index — the requested Version when pinned, or the tag "latest" actually
// resolved to once Catalog/Refresh has run.
func (r *RemoteRegistry) ResolvedVersion() string {
	if r.Version != "" {
		return r.Version
	}
	return r.resolvedTag
}

// NewRemoteRegistry builds a registry for the given release version ("" = latest)
// using the official GitHub coordinates and the given cache dir.
func NewRemoteRegistry(f Fetcher, cacheDir, version string) *RemoteRegistry {
	return &RemoteRegistry{Fetcher: f, CacheDir: cacheDir, Version: version}
}

func (r *RemoteRegistry) owner() string {
	if r.Owner != "" {
		return r.Owner
	}
	return DefaultOwner
}

func (r *RemoteRegistry) repo() string {
	if r.Repo != "" {
		return r.Repo
	}
	return DefaultRepo
}

// indexURL returns the stable download URL for this version's index.json. A
// pinned tag uses releases/download/<tag>/; latest uses releases/latest/download/
// (no GitHub API token required).
func (r *RemoteRegistry) indexURL() string {
	if r.BaseURL != "" {
		return strings.TrimRight(r.BaseURL, "/") + "/index.json"
	}
	if r.Version == "" {
		return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/index.json", r.owner(), r.repo())
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/index.json", r.owner(), r.repo(), r.Version)
}

// cacheKey is the on-disk filename for this version's cached index. Pinned tags
// are immutable so they cache forever; the unpinned "latest" caches under a
// single file refreshed only by bootstrap or update.
func (r *RemoteRegistry) cacheKey() string {
	v := r.Version
	if v == "" {
		v = "latest"
	}
	// Sanitize a tag into a safe filename.
	v = strings.ReplaceAll(v, "/", "_")
	return filepath.Join(r.CacheDir, "index-"+v+".json")
}

// Catalog returns the catalog for this version. A warm cache is read with ZERO
// network; a cold cache bootstrap-fetches once and caches it (so the next call is
// offline). On a fetch failure with no cache, the error is returned.
func (r *RemoteRegistry) Catalog(ctx context.Context) (*Catalog, error) {
	path := r.cacheKey()
	if data, err := os.ReadFile(path); err == nil {
		ix, err := LoadIndex(data)
		if err != nil {
			return nil, err
		}
		r.resolvedTag = ix.RegistryVersion
		return ix.ToCatalog(), nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Cold cache → bootstrap-fetch once.
	ix, err := r.fetchIndex(ctx)
	if err != nil {
		return nil, err
	}
	r.resolvedTag = ix.RegistryVersion
	return ix.ToCatalog(), nil
}

// Refresh force-fetches the index and overwrites the cache, regardless of whether
// a warm cache exists. This is the explicit-refresh path `patronus update` calls.
// On a network failure, if a cache already exists it is kept and a warning is
// emitted (offline-tolerant); otherwise the error is returned.
func (r *RemoteRegistry) Refresh(ctx context.Context) (*Catalog, error) {
	ix, err := r.fetchIndex(ctx)
	if err != nil {
		if data, rerr := os.ReadFile(r.cacheKey()); rerr == nil {
			r.warnf("registry refresh failed (%v); keeping cached index", err)
			cached, lerr := LoadIndex(data)
			if lerr != nil {
				return nil, lerr
			}
			r.resolvedTag = cached.RegistryVersion
			return cached.ToCatalog(), nil
		}
		return nil, err
	}
	r.resolvedTag = ix.RegistryVersion
	return ix.ToCatalog(), nil
}

// fetchIndex downloads index.json, optionally verifies it against the published
// index.json.sha256 sibling (TLS-trust fallback when that asset is absent), and
// writes it to the cache.
func (r *RemoteRegistry) fetchIndex(ctx context.Context) (*Index, error) {
	if r.Fetcher == nil {
		return nil, fmt.Errorf("registry: no fetcher configured")
	}
	url := r.indexURL()
	body, err := r.Fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("registry: fetch %s: %w", url, err)
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("registry: read index: %w", err)
	}

	// Best-effort integrity check against the .sha256 sibling. A 404/absent
	// sidecar falls back to TLS trust; a present-but-mismatched sidecar is fatal.
	if want, ok := r.fetchIndexSHA(ctx); ok {
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != want {
			return nil, fmt.Errorf("registry: index sha256 mismatch (got %s, want %s)", hex.EncodeToString(got[:]), want)
		}
	}

	ix, err := LoadIndex(data)
	if err != nil {
		return nil, err
	}
	if err := install.WriteFileAtomic(r.cacheKey(), data, 0o644); err != nil {
		return nil, err
	}
	return ix, nil
}

// fetchIndexSHA fetches the index.json.sha256 sidecar; ok is false when it is
// absent/unreachable (TLS-trust fallback).
func (r *RemoteRegistry) fetchIndexSHA(ctx context.Context) (sum string, ok bool) {
	body, err := r.Fetcher.Fetch(ctx, r.indexURL()+".sha256")
	if err != nil {
		return "", false
	}
	b, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(b))
	s = strings.TrimPrefix(s, "sha256:")
	// A sidecar may be "<hex>  filename"; take the first field.
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return "", false
	}
	return strings.ToLower(s), true
}

// Materialize ensures an artifact entry's portable source is unpacked on disk and
// sets e.Source.LocalDir to it, so the existing plan.Compute/adapter path runs
// against it unchanged. Idempotent: a previously-materialized item is reused
// without a second fetch. A sha256 mismatch writes nothing.
func (r *RemoteRegistry) Materialize(ctx context.Context, e *ArtifactEntry) (string, error) {
	if e.Source.LocalDir != "" {
		return e.Source.LocalDir, nil
	}
	if e.Source.TarballURL == "" {
		return "", fmt.Errorf("registry: artifact %q has no tarball to materialize", e.Manifest.Name)
	}
	dir := filepath.Join(r.CacheDir, "items", e.Manifest.Name+"-"+e.Manifest.Version)
	if _, err := os.Stat(filepath.Join(dir, "patronus.yaml")); err == nil {
		e.Source.LocalDir = dir // cache hit
		return dir, nil
	}

	if r.Fetcher == nil {
		return "", fmt.Errorf("registry: no fetcher configured")
	}
	body, err := r.Fetcher.Fetch(ctx, e.Source.TarballURL)
	if err != nil {
		return "", fmt.Errorf("registry: fetch %s: %w", e.Source.TarballURL, err)
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return "", fmt.Errorf("registry: read tarball: %w", err)
	}
	if err := verifyTarballSHA(data, e.Source.SHA256); err != nil {
		return "", fmt.Errorf("registry: %q: %w", e.Manifest.Name, err)
	}

	files, err := archive.Extract(bytes.NewReader(data), archive.FormatTarGz)
	if err != nil {
		return "", fmt.Errorf("registry: extract %q: %w", e.Manifest.Name, err)
	}
	for name, content := range files {
		if err := install.WriteFileAtomic(filepath.Join(dir, filepath.FromSlash(name)), content, 0o644); err != nil {
			return "", err
		}
	}
	e.Source.LocalDir = dir
	return dir, nil
}

// verifyTarballSHA checks data against a pinned "sha256:"-prefixed digest. An
// empty pin is an error (a remote item must be pinned).
func verifyTarballSHA(data []byte, wantHex string) error {
	want := strings.TrimPrefix(wantHex, "sha256:")
	if want == "" {
		return fmt.Errorf("no pinned sha256")
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != strings.ToLower(want) {
		return fmt.Errorf("sha256 mismatch (got %s, want %s)", hex.EncodeToString(got[:]), strings.ToLower(want))
	}
	return nil
}

func (r *RemoteRegistry) warnf(format string, args ...any) {
	if r.Warnf != nil {
		r.Warnf(format, args...)
	}
}

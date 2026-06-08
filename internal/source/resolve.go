package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// Fetcher downloads the bytes at a URL — mirrors recipe.Fetcher, declared locally
// to avoid an import cycle. The caller injects recipe.HTTPFetcher; tests inject a
// fake.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// Resolved is the outcome of resolving a non-registry Ref into a catalog entry.
// Exactly one of Artifact/Recipe is non-nil. ResolvedRef records the concrete
// version a (possibly mutable) reference pinned to, for the lock's provenance.
type Resolved struct {
	Artifact    *registry.ArtifactEntry
	Recipe      *registry.RecipeEntry
	ResolvedRef string
}

// Resolver fetches out-of-tree sourced references (git:/https:). Registry/file
// refs need neither Fetcher nor cache, so the zero-value Resolver still handles
// them; git:/https: require a Fetcher and a writable CacheDir.
type Resolver struct {
	Fetcher  Fetcher
	CacheDir string // ~/.patronus/cache/sources
}

// Resolve turns a parsed Ref into a catalog entry the planner/profile resolver
// dispatches exactly like an in-tree item. Registry → (nil, nil); File → local
// load; Git/HTTPS → fetch+materialize into the cache.
func (rs *Resolver) Resolve(ctx context.Context, ref *Ref) (*Resolved, error) {
	switch ref.Scheme {
	case Registry:
		// (nil, nil) is the documented contract: a registry ref has no out-of-tree
		// entry — the caller already holds it in the main catalog.
		return nil, nil //nolint:nilnil
	case File:
		return resolveFile(ref)
	case Git:
		return rs.resolveGit(ctx, ref)
	case HTTPS:
		return rs.resolveHTTPS(ctx, ref)
	default:
		return nil, fmt.Errorf("unsupported source scheme %q", ref.Scheme)
	}
}

// Resolve is the back-compat free function for registry/file references. A
// git:/https: ref reaching it errors clearly (it needs a Resolver with a Fetcher).
func Resolve(ref *Ref) (*Resolved, error) {
	switch ref.Scheme {
	case Registry:
		return nil, nil //nolint:nilnil // documented: registry refs have no out-of-tree entry
	case File:
		return resolveFile(ref)
	default:
		return nil, fmt.Errorf("%s: sources need a Resolver with a fetcher", ref.Scheme)
	}
}

// resolveFile reads a local artifact directory or recipe manifest into a catalog
// entry (unchanged from Phase 5).
func resolveFile(ref *Ref) (*Resolved, error) {
	info, err := os.Stat(ref.Path)
	if err != nil {
		return nil, fmt.Errorf("file source %q: %w", ref.Path, err)
	}
	if info.IsDir() {
		mPath := filepath.Join(ref.Path, "patronus.yaml")
		if _, err := os.Stat(mPath); err != nil {
			return nil, fmt.Errorf("file source %q is a directory but has no patronus.yaml", ref.Path)
		}
		m, err := manifest.LoadArtifact(mPath)
		if err != nil {
			return nil, err
		}
		return &Resolved{
			Artifact:    &registry.ArtifactEntry{Manifest: m, Source: registry.Source{LocalDir: ref.Path}},
			ResolvedRef: ref.Raw,
		}, nil
	}
	ext := filepath.Ext(ref.Path)
	if ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("file source %q must be a recipe .yaml manifest or an artifact directory", ref.Path)
	}
	m, err := manifest.LoadRecipe(ref.Path)
	if err != nil {
		return nil, err
	}
	return &Resolved{
		Recipe:      &registry.RecipeEntry{Manifest: m, Source: registry.Source{LocalDir: filepath.Dir(ref.Path)}},
		ResolvedRef: ref.Raw,
	}, nil
}

// resolveGit fetches a repo's source archive over the host tarball API
// (https://<host>/<owner>/<repo>/archive/<ref>.tar.gz), unpacks it, selects the
// #item, materializes it into the cache, and returns an entry with LocalDir set —
// so it dispatches like a file: item. No git binary, no extra deps.
func (rs *Resolver) resolveGit(ctx context.Context, ref *Ref) (*Resolved, error) {
	if rs.Fetcher == nil {
		return nil, fmt.Errorf("git: source needs a fetcher")
	}
	gitRef := ref.GitRef
	if gitRef == "" {
		gitRef = "HEAD"
	}
	archiveURL := fmt.Sprintf("https://%s/%s/%s/archive/%s.tar.gz", ref.Host, ref.Owner, ref.Repo, gitRef)
	files, err := rs.fetchArchive(ctx, archiveURL)
	if err != nil {
		return nil, fmt.Errorf("git: %w", err)
	}

	// Host tarballs nest everything under a top-level <repo>-<ref>/ prefix; strip
	// the first path segment so member paths are repo-relative.
	stripped := stripTopDir(files)

	dest := filepath.Join(rs.CacheDir, "git", ref.Host, ref.Owner, ref.Repo, sanitize(gitRef))
	resolvedRef := ref.Raw // mutable-branch pinning to a commit SHA is a Phase-6+ refinement

	// Locate the selected item.
	if ref.Item == "" {
		// Repo is a single item: a root patronus.yaml (artifact) or a single recipe.
		if _, ok := stripped["patronus.yaml"]; ok {
			return materializeArtifactDir(stripped, "", filepath.Join(dest, ref.Repo), resolvedRef)
		}
		return nil, fmt.Errorf("git: no #item given and repo has no root patronus.yaml")
	}

	// #item names an artifact subdir (has patronus.yaml) or a recipe yaml.
	itemDir := path.Clean(ref.Item)
	if _, ok := stripped[itemDir+"/patronus.yaml"]; ok {
		return materializeArtifactDir(subtree(stripped, itemDir), "", filepath.Join(dest, itemDir), resolvedRef)
	}
	for _, cand := range []string{itemDir + ".yaml", "recipes/" + itemDir + ".yaml", itemDir} {
		if data, ok := stripped[cand]; ok && strings.HasSuffix(cand, ".yaml") {
			m, err := manifest.DecodeRecipe(data)
			if err != nil {
				return nil, fmt.Errorf("git: recipe %q: %w", ref.Item, err)
			}
			recDir := filepath.Join(dest, filepath.Dir(cand))
			if err := install.WriteFileAtomic(filepath.Join(recDir, path.Base(cand)), data, 0o644); err != nil {
				return nil, err
			}
			return &Resolved{
				Recipe:      &registry.RecipeEntry{Manifest: m, Source: registry.Source{LocalDir: recDir}},
				ResolvedRef: resolvedRef,
			}, nil
		}
	}
	return nil, fmt.Errorf("git: item %q not found in %s/%s/%s@%s", ref.Item, ref.Host, ref.Owner, ref.Repo, gitRef)
}

// resolveHTTPS fetches a manifest yaml directly. A recipe manifest is returned as
// is (its delivery.assets are absolute upstream URLs handled by the recipe FETCH
// engine). An artifact manifest's relative entry/files: are fetched against the
// manifest base URL and materialized. Directory files: members can't be listed
// over plain HTTPS, so artifact files: must name individual files (use git: for
// dir-based content).
func (rs *Resolver) resolveHTTPS(ctx context.Context, ref *Ref) (*Resolved, error) {
	if rs.Fetcher == nil {
		return nil, fmt.Errorf("https: source needs a fetcher")
	}
	data, err := rs.fetchBytes(ctx, ref.Raw)
	if err != nil {
		return nil, fmt.Errorf("https: %w", err)
	}

	// Try recipe first (it has a fixed kind: Recipe), else artifact.
	if rec, rerr := manifest.DecodeRecipe(data); rerr == nil {
		dest := rs.httpsDest(ref.Raw)
		if err := install.WriteFileAtomic(filepath.Join(dest, path.Base(ref.Raw)), data, 0o644); err != nil {
			return nil, err
		}
		return &Resolved{
			Recipe:      &registry.RecipeEntry{Manifest: rec, Source: registry.Source{LocalDir: dest}},
			ResolvedRef: ref.Raw,
		}, nil
	}
	art, aerr := manifest.DecodeArtifact(data)
	if aerr != nil {
		return nil, fmt.Errorf("https: manifest is neither a valid recipe nor artifact: %w", aerr)
	}

	base := baseURL(ref.Raw)
	dest := rs.httpsDest(ref.Raw)
	if err := install.WriteFileAtomic(filepath.Join(dest, "patronus.yaml"), data, 0o644); err != nil {
		return nil, err
	}
	if art.Entry != "" {
		body, err := rs.fetchBytes(ctx, base+"/"+art.Entry)
		if err != nil {
			return nil, fmt.Errorf("https: entry %q: %w", art.Entry, err)
		}
		if err := install.WriteFileAtomic(filepath.Join(dest, filepath.FromSlash(art.Entry)), body, 0o644); err != nil {
			return nil, err
		}
	}
	for _, f := range art.Files {
		if strings.HasSuffix(f, "/") {
			return nil, fmt.Errorf("https: artifact files: %q is a directory — not listable over HTTPS; name individual files or use a git: source", f)
		}
		body, err := rs.fetchBytes(ctx, base+"/"+f)
		if err != nil {
			return nil, fmt.Errorf("https: file %q: %w", f, err)
		}
		if err := install.WriteFileAtomic(filepath.Join(dest, filepath.FromSlash(f)), body, 0o644); err != nil {
			return nil, err
		}
	}
	return &Resolved{
		Artifact:    &registry.ArtifactEntry{Manifest: art, Source: registry.Source{LocalDir: dest}},
		ResolvedRef: ref.Raw,
	}, nil
}

// --- helpers ---

func (rs *Resolver) fetchBytes(ctx context.Context, url string) ([]byte, error) {
	body, err := rs.Fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

func (rs *Resolver) fetchArchive(ctx context.Context, url string) (map[string][]byte, error) {
	data, err := rs.fetchBytes(ctx, url)
	if err != nil {
		return nil, err
	}
	return archive.Extract(bytes.NewReader(data), archive.FormatTarGz)
}

func (rs *Resolver) httpsDest(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	u, _ := url.Parse(rawURL)
	host := "https"
	if u != nil {
		host = u.Host
	}
	return filepath.Join(rs.CacheDir, "https", host, hex.EncodeToString(sum[:])[:16])
}

// materializeArtifactDir writes a repo-relative subtree (already rooted at the
// artifact dir, with patronus.yaml at its top) to dest and returns an entry.
func materializeArtifactDir(files map[string][]byte, _ string, dest, resolvedRef string) (*Resolved, error) {
	mb, ok := files["patronus.yaml"]
	if !ok {
		return nil, fmt.Errorf("git: artifact has no patronus.yaml")
	}
	m, err := manifest.DecodeArtifact(mb)
	if err != nil {
		return nil, err
	}
	for name, content := range files {
		if err := install.WriteFileAtomic(filepath.Join(dest, filepath.FromSlash(name)), content, 0o644); err != nil {
			return nil, err
		}
	}
	return &Resolved{
		Artifact:    &registry.ArtifactEntry{Manifest: m, Source: registry.Source{LocalDir: dest}},
		ResolvedRef: resolvedRef,
	}, nil
}

// stripTopDir removes the leading "<dir>/" path segment that host source archives
// wrap every member in.
func stripTopDir(files map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(files))
	for name, data := range files {
		if i := strings.IndexByte(name, '/'); i >= 0 {
			out[name[i+1:]] = data
		}
	}
	return out
}

// subtree returns the members under dir/, re-keyed relative to dir.
func subtree(files map[string][]byte, dir string) map[string][]byte {
	prefix := dir + "/"
	out := map[string][]byte{}
	for name, data := range files {
		if strings.HasPrefix(name, prefix) {
			out[name[len(prefix):]] = data
		}
	}
	return out
}

func baseURL(rawURL string) string {
	if i := strings.LastIndexByte(rawURL, '/'); i >= 0 {
		return rawURL[:i]
	}
	return rawURL
}

func sanitize(ref string) string {
	return strings.NewReplacer("/", "_", "\\", "_").Replace(ref)
}

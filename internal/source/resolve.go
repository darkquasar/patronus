package source

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// Resolved is the outcome of resolving a non-registry Ref into a catalog entry.
// Exactly one of Artifact/Recipe is non-nil. A Registry Ref resolves to neither
// (the caller already has the official catalog), so Resolve returns a nil
// *Resolved with a nil error for it — see Resolve's doc.
type Resolved struct {
	Artifact *registry.ArtifactEntry
	Recipe   *registry.RecipeEntry
}

// Resolve turns a parsed Ref into a catalog entry the planner/profile resolver
// can dispatch exactly like an in-tree item. The returned entry is meant to be
// merged into the working catalog before lookup.
//
//   - Registry → (nil, nil): no out-of-tree entry; the caller's main catalog
//     already contains the item under Ref.Name.
//   - File → load the manifest + record its location (the existing FETCH/adapter
//     engines read the body/files from Source.LocalDir).
//   - Git / HTTPS → a clear "Phase 6" error; the grammar is accepted (Parse
//     succeeded) but fetching is not implemented until the remote registry.
func Resolve(ref *Ref) (*Resolved, error) {
	switch ref.Scheme {
	case Registry:
		return nil, nil

	case File:
		return resolveFile(ref)

	case Git, HTTPS:
		return nil, fmt.Errorf("%s: sources are accepted but not yet fetched (Phase 6); use a registry name or file: for now", ref.Scheme)

	default:
		return nil, fmt.Errorf("unsupported source scheme %q", ref.Scheme)
	}
}

// resolveFile reads a local artifact directory or recipe manifest into a catalog
// entry. An artifact is a directory containing patronus.yaml; a recipe is a
// *.yaml manifest file. Source.LocalDir mirrors the LocalRegistry convention:
// the artifact's directory, or (for a recipe) the directory the manifest lives in.
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
		return &Resolved{Artifact: &registry.ArtifactEntry{
			Manifest: m,
			Source:   registry.Source{LocalDir: ref.Path},
		}}, nil
	}

	// A file: treat it as a recipe manifest.
	ext := filepath.Ext(ref.Path)
	if ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("file source %q must be a recipe .yaml manifest or an artifact directory", ref.Path)
	}
	m, err := manifest.LoadRecipe(ref.Path)
	if err != nil {
		return nil, err
	}
	return &Resolved{Recipe: &registry.RecipeEntry{
		Manifest: m,
		Source:   registry.Source{LocalDir: filepath.Dir(ref.Path)},
	}}, nil
}

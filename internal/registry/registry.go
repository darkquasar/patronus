// Package registry abstracts "give me the catalog of installable items." Phase 1
// ships a LocalRegistry that reads on-disk manifests from a Patronus repo; a
// RemoteRegistry (GitHub Releases index.json) slots in behind the same interface
// in a later phase with no caller changes.
package registry

import (
	"context"

	"github.com/darkquasar/patronus/internal/manifest"
)

// Registry yields the full catalog of installable items.
type Registry interface {
	Catalog(ctx context.Context) (*Catalog, error)
}

// Catalog is the resolved set of artifacts, recipes, and profiles.
type Catalog struct {
	Artifacts []ArtifactEntry `json:"artifacts"`
	Recipes   []RecipeEntry   `json:"recipes"`
	Profiles  []ProfileEntry  `json:"profiles"`
	Plugins   []PluginEntry   `json:"plugins"`
}

// Source records where an entry came from. LocalDir is populated by the local
// registry; TarballURL/SHA256 are reserved for the remote registry.
type Source struct {
	LocalDir   string `json:"localDir,omitempty"`
	TarballURL string `json:"tarballUrl,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
}

// ArtifactEntry wraps an artifact manifest with its source.
type ArtifactEntry struct {
	Manifest *manifest.Artifact `json:"manifest"`
	Source   Source             `json:"source"`
}

// RecipeEntry wraps a recipe manifest with its source.
type RecipeEntry struct {
	Manifest *manifest.Recipe `json:"manifest"`
	Source   Source           `json:"source"`
}

// ProfileEntry wraps a profile manifest with its source.
type ProfileEntry struct {
	Manifest *manifest.Profile `json:"manifest"`
	Source   Source            `json:"source"`
}

// PluginEntry is a plugin manifest plus where it came from.
type PluginEntry struct {
	Manifest *manifest.Plugin `json:"manifest"`
	Source   Source           `json:"source"`
}

// ItemNames returns every installable item's name (artifacts + recipes). Profiles
// are excluded: they are not `requires` targets (grouping is their job, not a
// dependency edge). Order is artifacts-then-recipes, each already sorted by the
// catalog loader.
func (c *Catalog) ItemNames() []string {
	out := make([]string, 0, len(c.Artifacts)+len(c.Recipes))
	for i := range c.Artifacts {
		out = append(out, c.Artifacts[i].Manifest.Name)
	}
	for i := range c.Recipes {
		out = append(out, c.Recipes[i].Manifest.Name)
	}
	return out
}

// Deps adapts the catalog to the requires.Deps lookup: it reports an item's
// direct `requires` edges and whether the name is a known artifact or recipe.
// Profiles are intentionally not lookupable — they cannot be a requires target
// (an unknown name there is a dangling edge, which Validate rejects). This is the
// one seam the requires package needs over a catalog.
func (c *Catalog) Deps(name string) ([]string, bool) {
	for i := range c.Artifacts {
		if c.Artifacts[i].Manifest.Name == name {
			return c.Artifacts[i].Manifest.Requires, true
		}
	}
	for i := range c.Recipes {
		if c.Recipes[i].Manifest.Name == name {
			return c.Recipes[i].Manifest.Requires, true
		}
	}
	return nil, false
}

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

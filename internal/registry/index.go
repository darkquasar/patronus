package registry

import (
	"encoding/json"
	"fmt"

	"github.com/darkquasar/patronus/internal/manifest"
)

// IndexSchemaVersion is bumped when the published index shape changes so an older
// binary can refuse a newer index rather than mis-parse it.
const IndexSchemaVersion = 1

// Index is the published, metadata-ONLY catalog — ONE per GitHub Release. It
// embeds each item's FULL manifest inline so `list`, profile resolution, and
// lock generation need only this one document; content (an artifact's portable
// source) is fetched lazily, only at install time, via the per-item Tarball
// pointer. This is what lets a user browse the catalog without downloading any
// artifact bodies.
//
// Serialized as deterministic stdlib JSON (same family as state.json /
// patronus.lock), so it is git-diffable and its sha256 is stable.
type Index struct {
	SchemaVersion   int             `json:"schemaVersion"`
	RegistryVersion string          `json:"registryVersion"` // the Release tag, e.g. "v0.6.0"
	Generated       string          `json:"generated"`       // RFC3339, set at build time
	Artifacts       []IndexArtifact `json:"artifacts"`
	Recipes         []IndexRecipe   `json:"recipes"`
	Profiles        []IndexProfile  `json:"profiles"`
}

// Tarball points at an artifact's portable-source tarball, published as a Release
// asset. The binary fetches+verifies it at install time and the existing adapter
// engine transforms it per-tool locally (so the registry ships source, not
// pre-baked per-tool output).
type Tarball struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"` // "sha256:" + hex over the tarball bytes
}

// IndexArtifact is an artifact's inline manifest plus its source tarball.
type IndexArtifact struct {
	Manifest *manifest.Artifact `json:"manifest"`
	Tarball  Tarball            `json:"tarball"`
}

// IndexRecipe carries only the manifest: a recipe is self-describing and its
// delivery.assets already pin upstream binaries (fetched by the recipe engine),
// so the registry ships no recipe content tarball.
type IndexRecipe struct {
	Manifest *manifest.Recipe `json:"manifest"`
}

// IndexProfile carries only the manifest: a profile just references other items
// by name and has no content of its own.
type IndexProfile struct {
	Manifest *manifest.Profile `json:"manifest"`
}

// LoadIndex parses index.json bytes, rejecting a schema version this binary does
// not understand.
func LoadIndex(data []byte) (*Index, error) {
	var ix Index
	if err := json.Unmarshal(data, &ix); err != nil {
		return nil, fmt.Errorf("registry: parse index: %w", err)
	}
	if ix.SchemaVersion == 0 {
		ix.SchemaVersion = IndexSchemaVersion
	}
	if ix.SchemaVersion > IndexSchemaVersion {
		return nil, fmt.Errorf("registry: index schema v%d is newer than this binary supports (v%d); upgrade patronus", ix.SchemaVersion, IndexSchemaVersion)
	}
	return &ix, nil
}

// Marshal renders the index as deterministic, indented JSON with a trailing
// newline (matching the state.json / patronus.lock writers).
func (ix *Index) Marshal() ([]byte, error) {
	out, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// ToCatalog converts the index into the same *Catalog shape LocalRegistry
// returns, so every downstream consumer (list, profile.Resolve, plan.Compute,
// lock) is unchanged. Artifact entries carry Source{TarballURL, SHA256} (the
// fields reserved for the remote registry); LocalDir stays empty until the
// caller materializes the tarball.
func (ix *Index) ToCatalog() *Catalog {
	cat := &Catalog{}
	for _, a := range ix.Artifacts {
		cat.Artifacts = append(cat.Artifacts, ArtifactEntry{
			Manifest: a.Manifest,
			Source:   Source{TarballURL: a.Tarball.URL, SHA256: a.Tarball.SHA256},
		})
	}
	for _, r := range ix.Recipes {
		cat.Recipes = append(cat.Recipes, RecipeEntry{Manifest: r.Manifest})
	}
	for _, p := range ix.Profiles {
		cat.Profiles = append(cat.Profiles, ProfileEntry{Manifest: p.Manifest})
	}
	return cat
}

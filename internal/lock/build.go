package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

// FromResolved builds a deterministic Lock from a resolved profile. now is the
// caller's RFC3339 timestamp (the only non-deterministic input; the package takes
// no clock). registryVersion is the registry release tag the catalog came from
// ("" for a local checkout). Entries are sorted by name so re-locking an unchanged
// profile yields byte-identical output modulo the timestamp.
func FromResolved(cat *registry.Catalog, r *profile.Resolved, now, registryVersion string) (*Lock, error) {
	l := &Lock{
		Version:         Version,
		Profile:         r.Profile.Name,
		RegistryVersion: registryVersion,
		Generated:       now,
	}
	for _, it := range r.Items {
		e := Entry{
			Name:   it.Name,
			Source: it.Source,
			Slot:   it.Slot,
			Kind:   it.Kind.String(),
		}
		sum, version, err := hashItem(cat, it)
		if err != nil {
			return nil, fmt.Errorf("hashing %q: %w", it.Name, err)
		}
		e.SHA256 = sum
		e.Version = version
		l.Entries = append(l.Entries, e)
	}
	sort.Slice(l.Entries, func(i, j int) bool { return l.Entries[i].Name < l.Entries[j].Name })
	return l, nil
}

// hashItem computes the integrity sha256 and version for a resolved item by
// looking it up in the catalog and hashing its manifest + content.
func hashItem(cat *registry.Catalog, it profile.ResolvedItem) (sum, version string, err error) {
	switch it.Kind {
	case profile.KindArtifact:
		entry := findArtifact(cat, it.Name)
		if entry == nil {
			return "", "", fmt.Errorf("artifact not in catalog")
		}
		sum, err = hashArtifact(*entry)
		return sum, entry.Manifest.Version, err
	case profile.KindRecipe:
		entry := findRecipe(cat, it.Name)
		if entry == nil {
			return "", "", fmt.Errorf("recipe not in catalog")
		}
		sum, err = hashRecipe(*entry)
		// Recipes carry no version field today; the manifest (and its pinned asset
		// SHAs) is the reproducibility anchor. Left empty, forward-compatible.
		return sum, "", err
	default:
		return "", "", fmt.Errorf("unknown item kind %v", it.Kind)
	}
}

// hashArtifact hashes the artifact's manifest (canonically re-marshalled to YAML
// so the digest depends on the structured manifest, not on incidental file
// formatting) folded with its entry body and every file under its files: dirs,
// walked in sorted path order. Each contribution is path-prefixed so two
// different layouts can't collide.
func hashArtifact(entry registry.ArtifactEntry) (string, error) {
	h := sha256.New()

	mb, err := yaml.Marshal(entry.Manifest)
	if err != nil {
		return "", err
	}
	writeChunk(h, "manifest", mb)

	root := entry.Source.LocalDir
	if root == "" {
		// No on-disk content available (e.g. a synthetic catalog); the manifest
		// hash alone still anchors the entry.
		return digest(h), nil
	}

	// The entry body (e.g. SKILL.md) plus each declared files: dir, content-hashed
	// in deterministic order.
	var paths []string
	if entry.Manifest.Entry != "" {
		paths = append(paths, filepath.Join(root, entry.Manifest.Entry))
	}
	for _, f := range entry.Manifest.Files {
		dir := filepath.Join(root, f)
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			paths = append(paths, p)
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(paths)

	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		rel, _ := filepath.Rel(root, p)
		writeChunk(h, filepath.ToSlash(rel), b)
	}
	return digest(h), nil
}

// hashRecipe hashes the recipe manifest, canonically re-marshalled to YAML. The
// manifest already pins each delivery asset's own sha256, so hashing the manifest
// transitively pins the binaries it fetches.
func hashRecipe(entry registry.RecipeEntry) (string, error) {
	mb, err := yaml.Marshal(entry.Manifest)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	writeChunk(h, "manifest", mb)
	return digest(h), nil
}

// writeChunk feeds a length-prefixed, labelled chunk into h so concatenation is
// unambiguous (no two distinct (label, data) sequences produce the same stream).
func writeChunk(h hash.Hash, label string, data []byte) {
	fmt.Fprintf(h, "%s\x00%d\x00", label, len(data))
	_, _ = h.Write(data)
}

func digest(h hash.Hash) string {
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func findArtifact(cat *registry.Catalog, name string) *registry.ArtifactEntry {
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest.Name == name {
			return &cat.Artifacts[i]
		}
	}
	return nil
}

func findRecipe(cat *registry.Catalog, name string) *registry.RecipeEntry {
	for i := range cat.Recipes {
		if cat.Recipes[i].Manifest.Name == name {
			return &cat.Recipes[i]
		}
	}
	return nil
}

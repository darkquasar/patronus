package registry

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/darkquasar/patronus/internal/manifest"
)

// LocalRegistry reads the catalog directly from a Patronus repo's on-disk
// manifests (no published index.json required).
type LocalRegistry struct {
	root string
}

// NewLocalRegistry returns a registry rooted at a Patronus repo directory.
func NewLocalRegistry(root string) *LocalRegistry {
	return &LocalRegistry{root: root}
}

// Root reports the repo root this registry reads from.
func (r *LocalRegistry) Root() string { return r.root }

// DiscoverRoot walks up from start looking for a Patronus repo root — a
// directory containing both artifacts/ and adapters/ (most specific), falling
// back to a directory whose go.mod declares the patronus module.
func DiscoverRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if isDir(filepath.Join(dir, "artifacts")) && isDir(filepath.Join(dir, "adapters")) {
			return dir, nil
		}
		if hasPatronusModule(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a patronus repo (no artifacts/+adapters/ found above %s)", start)
		}
		dir = parent
	}
}

// Catalog walks the repo's artifacts/, recipes/, and profiles/ directories and
// returns the resolved catalog, sorted by name. It errors if a name is reused
// across artifacts and recipes (§5d relies on cross-type name uniqueness).
func (r *LocalRegistry) Catalog(ctx context.Context) (*Catalog, error) {
	cat := &Catalog{}

	if err := r.loadArtifacts(cat); err != nil {
		return nil, err
	}
	if err := r.loadRecipes(cat); err != nil {
		return nil, err
	}
	if err := r.loadProfiles(cat); err != nil {
		return nil, err
	}

	sort.Slice(cat.Artifacts, func(i, j int) bool {
		return cat.Artifacts[i].Manifest.Name < cat.Artifacts[j].Manifest.Name
	})
	sort.Slice(cat.Recipes, func(i, j int) bool {
		return cat.Recipes[i].Manifest.Name < cat.Recipes[j].Manifest.Name
	})
	sort.Slice(cat.Profiles, func(i, j int) bool {
		return cat.Profiles[i].Manifest.Name < cat.Profiles[j].Manifest.Name
	})

	if err := checkNameUniqueness(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (r *LocalRegistry) loadArtifacts(cat *Catalog) error {
	artifactsDir := filepath.Join(r.root, "artifacts")
	if !isDir(artifactsDir) {
		return nil
	}
	return filepath.WalkDir(artifactsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "patronus.yaml" {
			return nil
		}
		m, err := manifest.LoadArtifact(path)
		if err != nil {
			return err
		}
		cat.Artifacts = append(cat.Artifacts, ArtifactEntry{
			Manifest: m,
			Source:   Source{LocalDir: filepath.Dir(path)},
		})
		return nil
	})
}

func (r *LocalRegistry) loadRecipes(cat *Catalog) error {
	for _, path := range globYAML(filepath.Join(r.root, "recipes")) {
		m, err := manifest.LoadRecipe(path)
		if err != nil {
			return err
		}
		cat.Recipes = append(cat.Recipes, RecipeEntry{
			Manifest: m,
			Source:   Source{LocalDir: filepath.Dir(path)},
		})
	}
	return nil
}

func (r *LocalRegistry) loadProfiles(cat *Catalog) error {
	for _, path := range globYAML(filepath.Join(r.root, "profiles")) {
		m, err := manifest.LoadProfile(path)
		if err != nil {
			return err
		}
		cat.Profiles = append(cat.Profiles, ProfileEntry{
			Manifest: m,
			Source:   Source{LocalDir: filepath.Dir(path)},
		})
	}
	return nil
}

// checkNameUniqueness enforces that artifact and recipe names don't collide, so
// a bare name in a profile slot is unambiguous.
func checkNameUniqueness(cat *Catalog) error {
	seen := make(map[string]string)
	for _, a := range cat.Artifacts {
		seen[a.Manifest.Name] = "artifact"
	}
	for _, rc := range cat.Recipes {
		if kind, ok := seen[rc.Manifest.Name]; ok {
			return fmt.Errorf("name %q used by both %s and recipe", rc.Manifest.Name, kind)
		}
		seen[rc.Manifest.Name] = "recipe"
	}
	return nil
}

// globYAML returns sorted *.yaml and *.yml files directly under dir.
func globYAML(dir string) []string {
	var out []string
	for _, ext := range []string{"*.yaml", "*.yml"} {
		matches, _ := filepath.Glob(filepath.Join(dir, ext))
		out = append(out, matches...)
	}
	sort.Strings(out)
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasPatronusModule(goModPath string) bool {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}
	return filepath.Base(filepath.Dir(goModPath)) != "" &&
		containsLine(string(data), "module github.com/darkquasar/patronus")
}

func containsLine(content, prefix string) bool {
	for _, line := range splitLines(content) {
		if line == prefix || (len(line) > len(prefix) && line[:len(prefix)] == prefix) {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

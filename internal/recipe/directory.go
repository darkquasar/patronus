package recipe

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// directoryDiff constructs host intent; ownership inspection belongs to the CLI.
func directoryDiff(req Request, goos, goarch string) ([]diff.FileDiff, error) {
	rec := req.Recipe
	if err := manifest.ValidateRecipe(rec); err != nil {
		return nil, err
	}
	if req.Scope == "local" {
		return nil, fmt.Errorf("directory recipe %q requires global scope; --local is unsupported", rec.Name)
	}
	asset, err := rec.Delivery.ResolveAsset(goos, goarch)
	if err != nil {
		return nil, fmt.Errorf("directory recipe %q: %w", rec.Name, err)
	}
	home, err := filepath.Abs(req.Resolver.ExpandHome("~"))
	if err != nil {
		return nil, err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, ".patronus", "packages", rec.Name)
	spec := &diff.DirectorySpec{Recipe: rec.Name, RecipeVersion: rec.Version, URL: asset.URL, SHA256: "sha256:" + strings.ToLower(strings.TrimPrefix(asset.SHA256, "sha256:")), Root: root, PackageName: rec.Delivery.Package.Name, PackageVersion: rec.Delivery.Package.Version, OS: goos, Arch: goarch}
	return []diff.FileDiff{{Path: root, Action: diff.Fetch, Artifact: rec.Name, Version: rec.Version, Type: string(manifest.ShapeInstall), Role: string(rec.Role), Tool: TargetAgnostic, Scope: "global", Directory: spec}}, nil
}

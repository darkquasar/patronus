package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/registry"
)

// newBuildCmd is the CI-side `patronus build`: it reads the local checkout and
// emits the publishable registry — one portable-source tarball per artifact plus
// a metadata-only index.json (+ index.json.sha256). It ships SOURCE, not per-tool
// output: the installed binary transforms each artifact locally with the adapter
// engine, so the registry stays small and tool-agnostic.
func newBuildCmd() *cobra.Command {
	var (
		out             string
		registryVersion string
		baseURL         string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the publishable registry (index.json + portable-source tarballs) — CI use",
		Long: "Reads the local Patronus checkout and writes, into --out, one portable-source\n" +
			"tarball per artifact (patronus.yaml + entry body + files/) plus a metadata-only\n" +
			"index.json with every manifest inline and a tarball{url,sha256} pointer. This is\n" +
			"what build-registry.yml publishes as GitHub Release assets.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryVersion == "" {
				registryVersion = os.Getenv("GITHUB_REF_NAME")
			}
			if registryVersion == "" {
				registryVersion = "dev"
			}
			if baseURL == "" {
				baseURL = fmt.Sprintf("https://github.com/darkquasar/patronus/releases/download/%s", registryVersion)
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			cat, err := registry.NewLocalRegistry(root).Catalog(cmd.Context())
			if err != nil {
				return err
			}

			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}

			ix := &registry.Index{
				SchemaVersion:   registry.IndexSchemaVersion,
				RegistryVersion: registryVersion,
				Generated:       time.Now().UTC().Format(time.RFC3339),
			}

			for i := range cat.Artifacts {
				entry := &cat.Artifacts[i]
				files, err := collectArtifactFiles(entry)
				if err != nil {
					return fmt.Errorf("packaging %q: %w", entry.Manifest.Name, err)
				}
				tgz, err := archive.CreateTarGz(files)
				if err != nil {
					return err
				}
				name := fmt.Sprintf("%s-%s.tar.gz", entry.Manifest.Name, entry.Manifest.Version)
				if err := install.WriteFileAtomic(filepath.Join(out, name), tgz, 0o644); err != nil {
					return err
				}
				sum := sha256.Sum256(tgz)
				ix.Artifacts = append(ix.Artifacts, registry.IndexArtifact{
					Manifest: entry.Manifest,
					Tarball: registry.Tarball{
						URL:    baseURL + "/" + name,
						SHA256: "sha256:" + hex.EncodeToString(sum[:]),
					},
				})
				fmt.Fprintf(cmd.OutOrStdout(), "packaged %s\n", name)
			}
			for i := range cat.Recipes {
				ix.Recipes = append(ix.Recipes, registry.IndexRecipe{Manifest: cat.Recipes[i].Manifest})
			}
			for i := range cat.Profiles {
				ix.Profiles = append(ix.Profiles, registry.IndexProfile{Manifest: cat.Profiles[i].Manifest})
			}

			data, err := ix.Marshal()
			if err != nil {
				return err
			}
			if err := install.WriteFileAtomic(filepath.Join(out, "index.json"), data, 0o644); err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			shaLine := []byte("sha256:" + hex.EncodeToString(sum[:]) + "\n")
			if err := install.WriteFileAtomic(filepath.Join(out, "index.json.sha256"), shaLine, 0o644); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s/index.json (%d artifacts, %d recipes, %d profiles) @ %s\n",
				out, len(ix.Artifacts), len(ix.Recipes), len(ix.Profiles), registryVersion)
			return nil
		},
	}

	cmd.Flags().StringVar(&out, "out", "registry", "output directory for the built registry")
	cmd.Flags().StringVar(&registryVersion, "registry-version", "", "registry release tag (default: $GITHUB_REF_NAME or 'dev')")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "asset URL prefix for tarball links (default: GitHub Releases for this tag)")
	return cmd
}

// collectArtifactFiles builds the portable-source member set for one artifact:
// a canonical patronus.yaml (re-marshalled from the parsed manifest, matching the
// lock's hashing convention) plus the entry body and every file under files:,
// keyed by paths relative to the artifact dir.
func collectArtifactFiles(entry *registry.ArtifactEntry) (map[string][]byte, error) {
	root := entry.Source.LocalDir
	if root == "" {
		return nil, fmt.Errorf("artifact %q has no local source dir", entry.Manifest.Name)
	}
	out := map[string][]byte{}

	mb, err := yaml.Marshal(entry.Manifest)
	if err != nil {
		return nil, err
	}
	out["patronus.yaml"] = mb

	if entry.Manifest.Entry != "" {
		b, err := os.ReadFile(filepath.Join(root, entry.Manifest.Entry))
		if err != nil {
			return nil, err
		}
		out[path.Clean(entry.Manifest.Entry)] = b
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
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(rel)] = b
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

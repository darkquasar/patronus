package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
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
		out     string
		baseURL string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the publishable registry (catalog/ tree: index.json + per-item tarballs) — CI use",
		Long: "Reads the local Patronus checkout and writes, into --out, an R2-layout catalog/\n" +
			"tree: one portable-source tarball per artifact at catalog/<name>/<version>/ plus a\n" +
			"metadata-only catalog/index.json (every manifest inline + a tarball{url,sha256}\n" +
			"pointer) and its .sha256 sidecar. CI syncs this tree to the R2 bucket 1:1; the\n" +
			"per-item tarball keys are immutable (write-once), the index is discovery-only.\n\n" +
			"There is no registry-wide version — each item versions independently.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if baseURL == "" {
				baseURL = registry.DefaultRegistryURL
			}
			baseURL = strings.TrimRight(baseURL, "/")

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

			catalogDir := filepath.Join(out, "catalog")
			if err := os.MkdirAll(catalogDir, 0o755); err != nil {
				return err
			}

			ix := &registry.Index{
				SchemaVersion: registry.IndexSchemaVersion,
				Generated:     time.Now().UTC().Format(time.RFC3339),
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
				name, version := entry.Manifest.Name, entry.Manifest.Version
				// Immutable content-addressed key: catalog/<name>/<version>/<name>-<version>.tar.gz
				key := fmt.Sprintf("catalog/%s/%s/%s-%s.tar.gz", name, version, name, version)
				if err := install.WriteFileAtomic(filepath.Join(out, filepath.FromSlash(key)), tgz, 0o644); err != nil {
					return err
				}
				sum := sha256.Sum256(tgz)
				ix.Artifacts = append(ix.Artifacts, registry.IndexArtifact{
					Manifest: entry.Manifest,
					Tarball: registry.Tarball{
						URL:    baseURL + "/" + key,
						SHA256: "sha256:" + hex.EncodeToString(sum[:]),
					},
				})
				fmt.Fprintf(cmd.OutOrStdout(), "packaged %s\n", key)
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
			if err := install.WriteFileAtomic(filepath.Join(catalogDir, "index.json"), data, 0o644); err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			shaLine := []byte("sha256:" + hex.EncodeToString(sum[:]) + "\n")
			if err := install.WriteFileAtomic(filepath.Join(catalogDir, "index.json.sha256"), shaLine, 0o644); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s/catalog/index.json (%d artifacts, %d recipes, %d profiles)\n",
				out, len(ix.Artifacts), len(ix.Recipes), len(ix.Profiles))
			return nil
		},
	}

	cmd.Flags().StringVar(&out, "out", "registry", "output directory for the built registry tree")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "public base URL for tarball links (default: the official R2 registry)")
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

	// Vendored content (attribution set) must ship its NOTICE in the tarball so the
	// upstream license + copyright travels with the artifact (§3). Required, not
	// best-effort: a missing NOTICE on attributed content fails the build.
	if entry.Manifest.Attribution != nil {
		nb, err := os.ReadFile(filepath.Join(root, "NOTICE"))
		if err != nil {
			return nil, fmt.Errorf("artifact %q declares attribution but has no NOTICE file: %w", entry.Manifest.Name, err)
		}
		out["NOTICE"] = nb
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

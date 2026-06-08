package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
)

// newUpdateCmd is `patronus update` — in Phase 6 its job is to REFRESH THE
// REGISTRY CACHE: fetch the latest index.json and overwrite the local cache, the
// one explicit action that bypasses the otherwise apt-style "use the cache until
// told otherwise" policy. (DESIGN §8's other meaning of update — refreshing the
// installed items on disk — is added later as `update <name>` / `--installed`;
// the two are the same command growing a second job.)
func newUpdateCmd() *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh the registry cache (fetch the latest published index)",
		Long: "Fetches the latest (or a pinned --registry-version) index.json from the\n" +
			"GitHub Release and overwrites the local cache at ~/.patronus/cache. Day-to-day\n" +
			"commands read the cache offline; this is the explicit refresh that updates it.\n" +
			"If the network is unreachable but a cache already exists, the cache is kept.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Build a remote registry at the requested version and force a refresh,
			// regardless of any warm cache.
			cacheDir := filepath.Join(homeDir(), ".patronus", "cache")
			reg := registry.NewRemoteRegistry(fetcherForCommands, cacheDir, version)
			reg.Warnf = warnf

			cat, err := reg.Refresh(cmd.Context())
			if err != nil {
				return err
			}

			tag := version
			if tag == "" {
				tag = "latest"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated registry cache → %s (%d artifacts, %d recipes, %d profiles)\n",
				tag, len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
			return nil
		},
	}

	cmd.Flags().StringVar(&version, "registry-version", "", "refresh a specific registry release tag (default: latest)")
	return cmd
}

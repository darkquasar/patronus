package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
)

// newUpdateCmd is `patronus update` — its Phase-6/7 job is to REFRESH THE REGISTRY
// CACHE: fetch the latest discovery index.json from the R2 registry and overwrite
// the local cache, the one explicit action that bypasses the otherwise apt-style
// "use the cache until told otherwise" policy. (The other meaning of update —
// refreshing the installed items on disk — is a later phase as `update <name>` /
// `--installed`; the two are the same command growing a second job.)
func newUpdateCmd() *cobra.Command {
	var regSel registrySel

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh the registry cache (fetch the latest published discovery index)",
		Long: "Fetches the latest catalog/index.json from the registry and overwrites the\n" +
			"local cache at ~/.patronus/cache. Day-to-day commands read the cache offline;\n" +
			"this is the explicit refresh that updates it. If the network is unreachable but\n" +
			"a cache already exists, the cache is kept.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Force a refresh against the resolved registry base, regardless of any
			// warm cache. Uses the TLS-1.3 registry fetcher (the endpoint we control).
			cacheDir := filepath.Join(homeDir(), ".patronus", "cache")
			reg := registry.NewRemoteRegistry(registryFetcher, cacheDir, registryBaseURL(regSel.url))
			reg.Warnf = warnf

			cat, err := reg.Refresh(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated registry cache from %s (%d artifacts, %d recipes, %d profiles)\n",
				reg.Base(), len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
			return nil
		},
	}

	cmd.Flags().StringVar(&regSel.url, "registry-url", "", "registry base URL override (fork/mirror)")
	return cmd
}

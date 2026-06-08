package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func newLockCmd() *cobra.Command {
	var profileSel string

	cmd := &cobra.Command{
		Use:   "lock --profile <name>",
		Short: "Write/refresh patronus.lock from a profile's current catalog resolution",
		Long: "Re-resolves a profile against the catalog and writes patronus.lock — pinning\n" +
			"each resolved item's name, source provenance, version, and sha256 so a teammate\n" +
			"or fresh machine can reproduce the same environment (DESIGN §5d/§5e).\n\n" +
			"PROMOTE vs RESTORE: `patronus lock` is lock-follows-reality — it overwrites the\n" +
			"lock with whatever the profile resolves to NOW. The reverse direction\n" +
			"(reality-follows-lock: install reading the lock and refetching the pinned\n" +
			"source@resolvedRef) lands with the remote registry in Phase 6; today\n" +
			"`install --profile` resolves live from the catalog and does not read the lock.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileSel == "" {
				return fmt.Errorf("--profile is required (a lock pins what a profile resolved to)")
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			reg := registry.NewLocalRegistry(root)
			cat, err := reg.Catalog(cmd.Context())
			if err != nil {
				return err
			}

			res, err := profile.Resolve(cat, profileSel)
			if err != nil {
				return err
			}
			for _, w := range res.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}

			now := time.Now().UTC().Format(time.RFC3339)
			l, err := lock.FromResolved(cat, res, now)
			if err != nil {
				return err
			}

			// The lock is the shared, committed spec, so it lives at the project root
			// (cwd) — intentionally not the global/local split state.json uses.
			path := filepath.Join(wd, "patronus.lock")
			if err := lock.Save(path, l); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d entries)\n", path, len(l.Entries))
			return nil
		},
	}

	cmd.Flags().StringVar(&profileSel, "profile", "", "profile to lock (required)")
	return cmd
}

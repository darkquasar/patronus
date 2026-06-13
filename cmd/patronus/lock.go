package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/profile"
)

func newLockCmd() *cobra.Command {
	var (
		profileSel string
		regSel     registrySel
	)

	cmd := &cobra.Command{
		Use:   "lock --profile <name>",
		Short: "Write/refresh patronus.lock from a profile's current catalog resolution",
		Long: "Re-resolves a profile against the catalog and writes patronus.lock — pinning\n" +
			"each resolved item PER-ITEM (name, source provenance, version, content sha256,\n" +
			"and the published tarball sha) so a teammate or fresh machine reproduces the\n" +
			"exact same environment. There is no registry-wide version; reproducibility is\n" +
			"per item, independent of the tool version (the npm/pip model).\n\n" +
			"PROMOTE vs RESTORE: `patronus lock` is lock-follows-reality — it overwrites the\n" +
			"lock with whatever the profile resolves to NOW. The reverse (reality-follows-\n" +
			"lock) is `install --profile` against a committed lock: it fetches each item at\n" +
			"its locked version from the registry's immutable key and verifies the sha.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileSel == "" {
				return fmt.Errorf("--profile is required (a lock pins what a profile resolved to)")
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Resolve against the same registry install would use (the local checkout
			// in dev, or the remote R2 registry). The lock pins each item per-item
			// (version + content sha + tarball sha) — there is no registry-wide tag.
			reg, _, err := resolveRegistry(cmd.Context(), wd, regSel, homeDir(), warnf)
			if err != nil {
				return err
			}
			cat, err := reg.Catalog(cmd.Context())
			if err != nil {
				return err
			}

			res, err := profile.Resolve(cat, profileSel)
			if err != nil {
				return err
			}
			for _, w := range res.Warnings {
				warnf("%s", w)
			}

			// Hashing an artifact's content-fold reads its files from Source.LocalDir,
			// so a remote registry must materialize the selected items first (no-op for
			// a local checkout, where LocalDir is already set).
			if err := materializeSelected(cmd.Context(), reg, cat, res.Names()); err != nil {
				return err
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
	addRegistryFlags(cmd, &regSel)
	return cmd
}

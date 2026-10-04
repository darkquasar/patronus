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
		tool       string
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
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := dp06Target(tool); err != nil {
				return err
			}
			if profileSel == "" {
				return fmt.Errorf("--profile is required (a lock pins what a profile resolved to)")
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			m, err := acquireMutation(homeDir(), filepath.Join(wd, "patronus.lock"))
			if err != nil {
				return err
			}
			defer m.close(&err)
			prior, err := lock.Load(filepath.Join(wd, "patronus.lock"))
			if err != nil {
				return err
			}
			if prior.Target == "pi" {
				if !cmd.Flags().Changed("target") {
					tool = "pi"
				} else if tool != "pi" {
					return fmt.Errorf("existing Pi lock target conflicts with --target %s; explicit migration required", tool)
				}
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

			// tool pins per-tool flavours (§4); the default "all" pins the
			// tool-agnostic baseline (bare names only).
			res, err := profile.Resolve(cat, profileSel, tool)
			if err != nil {
				return err
			}
			if tool == "pi" {
				if err := dp06ProfileComplete(res); err != nil {
					return err
				}
				var old []string
				for _, e := range prior.Entries {
					old = append(old, e.Name)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Deliberate Pi lock regeneration; closure preview: pinned=%v selected=%v\n", old, res.Names())
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
			if err := m.saveLock(path, l); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d entries)\n", path, len(l.Entries))
			return nil
		},
	}

	cmd.Flags().StringVar(&profileSel, "profile", "", "profile to lock (required)")
	cmd.Flags().StringVar(&tool, "target", "all", "pin per-target flavours: claude|codex|opencode|pi|all")
	addRegistryFlags(cmd, &regSel)
	return cmd
}

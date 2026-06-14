package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
)

// newUpdateCmd is `patronus update`, a command with two jobs that share a cache
// refresh:
//
//	update            — REFRESH THE REGISTRY CACHE (Phase 6): fetch the latest
//	                    discovery index.json and overwrite the local cache, the one
//	                    explicit action that bypasses the apt-style cache policy.
//	update <name>...  — INSTALLED-ITEM REFRESH (Phase 8): after refreshing the
//	                    cache, compare each named installed item's recorded version
//	                    against the registry's latest and, if newer, re-drive its
//	                    install (re-fetch/rewire) at its recorded tool/scope. This is
//	                    a MANUAL, explicit action — Patronus never auto-updates.
//
// Like install, the installed-item refresh is a dry run unless --deploy.
func newUpdateCmd() *cobra.Command {
	var (
		regSel registrySel
		deploy bool
		dryRun bool
		all    bool
	)

	cmd := &cobra.Command{
		Use:   "update [name...]",
		Short: "Refresh the registry cache, or re-install named items at the latest version",
		Long: "With no arguments, fetches the latest catalog/index.json from the registry and\n" +
			"overwrites the local cache at ~/.patronus/cache (day-to-day commands read the\n" +
			"cache offline; this is the explicit refresh). If the network is unreachable but\n" +
			"a cache already exists, the cache is kept.\n\n" +
			"With one or more names (or --all), also compares each installed item's recorded\n" +
			"version against the registry's latest and, when newer, re-installs it at the\n" +
			"tool/scope it was originally installed to. Manual and explicit — nothing auto-\n" +
			"updates. Like install, this is a dry run unless --deploy.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deploy && dryRun {
				return fmt.Errorf("--deploy and --dry-run are mutually exclusive")
			}
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			home := homeDir()

			// Resolve the registry the same way install/list do (local checkout vs
			// remote R2), then refresh its catalog so the comparison sees the latest.
			reg, _, err := resolveRegistry(cmd.Context(), wd, regSel, home, warnf)
			if err != nil {
				return err
			}
			cat := refreshCatalog(cmd, reg, warnf)

			// No names → the classic cache-refresh job (already done above for remote;
			// report it).
			if len(args) == 0 && !all {
				if cat == nil {
					return fmt.Errorf("update: unable to refresh registry (offline and no cache)")
				}
				fmt.Fprintf(cmd.OutOrStdout(), "updated registry cache (%d artifacts, %d recipes, %d profiles)\n",
					len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
				return nil
			}
			if cat == nil {
				return fmt.Errorf("update: registry unavailable; cannot check for updates")
			}

			// Installed-item refresh. Gather candidate items from both scope state
			// files, filtered to the requested names (or all installed items).
			want := map[string]bool{}
			for _, n := range args {
				want[n] = true
			}
			type candidate struct {
				name, tool, scope, installed, latest string
			}
			var candidates []candidate
			anyInstalled := false
			for _, scope := range []string{"global", "local"} {
				sp := removeStatePath(scope, home, wd)
				s, err := state.Load(sp)
				if err != nil {
					return fmt.Errorf("load %s state: %w", scope, err)
				}
				for _, it := range s.Items {
					anyInstalled = true
					if !all && !want[it.Artifact] {
						continue
					}
					latest := latestVersion(cat, it.Artifact)
					candidates = append(candidates, candidate{
						name: it.Artifact, tool: it.Tool, scope: it.Scope,
						installed: it.ItemVersion, latest: latest,
					})
				}
			}

			if len(candidates) == 0 {
				if !anyInstalled {
					return fmt.Errorf("nothing is installed (no state recorded)")
				}
				return fmt.Errorf("not installed: %v", args)
			}

			out := cmd.OutOrStdout()
			updated := 0
			for _, c := range candidates {
				switch {
				case c.latest == "":
					fmt.Fprintf(out, "%s: not in registry (or unversioned) — leaving as-is\n", c.name)
				case c.installed == "":
					// We never recorded a version (pre-Phase-8 install, or a recipe).
					fmt.Fprintf(out, "%s: installed version unknown — refreshing to %s\n", c.name, c.latest)
					if err := reinstall(cmd, c.name, c.tool, c.scope, deploy); err != nil {
						return err
					}
					updated++
				case c.installed == c.latest:
					fmt.Fprintf(out, "%s: up to date (%s)\n", c.name, c.installed)
				default:
					fmt.Fprintf(out, "%s: %s -> %s\n", c.name, c.installed, c.latest)
					if err := reinstall(cmd, c.name, c.tool, c.scope, deploy); err != nil {
						return err
					}
					updated++
				}
			}
			if !deploy && updated > 0 {
				fmt.Fprintln(out, "\n(dry run — pass --deploy to apply updates)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&regSel.url, "registry-url", "", "registry base URL override (fork/mirror)")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually re-install updated items (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVar(&all, "all", false, "check every installed item for updates")
	return cmd
}

// refreshCatalog refreshes a remote registry's cache (the classic update job) and
// returns the resolved catalog. For a local registry there is no remote cache to
// refresh, so it just loads the catalog. Returns nil on failure (offline with no
// cache); the caller decides whether that is fatal for the requested job.
func refreshCatalog(cmd *cobra.Command, reg registry.Registry, warnf func(string, ...any)) *registry.Catalog {
	if rr, ok := reg.(*registry.RemoteRegistry); ok {
		rr.Warnf = warnf
		if cat, err := rr.Refresh(cmd.Context()); err == nil {
			return cat
		}
		// Refresh failed (offline); fall back to whatever the cache holds.
		warnf("registry refresh failed; using cached catalog if present")
	}
	cat, err := reg.Catalog(cmd.Context())
	if err != nil {
		return nil
	}
	return cat
}

// latestVersion returns the registry's advertised version for an artifact name, or
// "" if the name is not a versioned artifact in the catalog (e.g. a recipe, which
// carries no version field, or an unknown name).
func latestVersion(cat *registry.Catalog, name string) string {
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest != nil && cat.Artifacts[i].Manifest.Name == name {
			return cat.Artifacts[i].Manifest.Version
		}
	}
	return ""
}

// reinstall re-drives the install command for one item at its recorded tool/scope,
// reusing the entire materialize → plan → deploy → state-record pipeline (so the
// new version is recorded the same way a fresh install records it). Honors --deploy;
// without it, install renders a dry-run plan.
func reinstall(cmd *cobra.Command, name, tool, scope string, deploy bool) error {
	args := []string{name}
	if tool != "" {
		args = append(args, "--tool", tool)
	}
	switch scope {
	case "global":
		args = append(args, "--global")
	case "local":
		args = append(args, "--local")
	}
	if deploy {
		args = append(args, "--deploy", "--force") // an update intentionally overwrites the prior install
	}

	in := newInstallCmd()
	in.SetArgs(args)
	in.SetOut(cmd.OutOrStdout())
	in.SetErr(cmd.ErrOrStderr())
	in.SetIn(cmd.InOrStdin())
	in.SetContext(cmd.Context())
	return in.Execute()
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/requires"
	"github.com/darkquasar/patronus/internal/state"
)

// newUpdateCmd is `patronus update`, a command with two jobs that share a cache
// refresh:
//
//	update            — REFRESH THE REGISTRY CACHE (Phase 6): fetch the latest
//	                    discovery index.json and overwrite the local cache, the one
//	                    explicit action that bypasses the apt-style cache policy. A
//	                    local checkout has no cache, so there it just reads the
//	                    checkout and reports that.
//	update <name>...  — INSTALLED-ITEM REFRESH (Phase 8): after refreshing the
//	                    cache, compare each named installed item's recorded version
//	                    against the registry's latest and, if newer, re-drive its
//	                    install (re-fetch/rewire) at its recorded tool/scope. This is
//	                    a MANUAL, explicit action: Patronus never auto-updates.
//
// Like install, the installed-item refresh is a dry run unless --deploy.
func newUpdateCmd() *cobra.Command {
	var (
		regSel registrySel
		deploy bool
		dryRun bool
		all    bool
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "update [name...]",
		Short: "Refresh the registry cache, or re-install named items at the latest version",
		Long: "With no arguments and a remote registry, fetches the latest catalog/index.json and\n" +
			"overwrites the local cache at ~/.patronus/cache (day-to-day commands read the\n" +
			"cache offline; this is the explicit refresh). If the network is unreachable but\n" +
			"a cache already exists, the cache is kept. Against a local registry checkout\n" +
			"(--local-registry, or running inside the repo) there is no cache: the checkout\n" +
			"is read directly.\n\n" +
			"With one or more names (or --all), also compares each installed item's recorded\n" +
			"version against the registry's latest and, when newer, re-installs it at the\n" +
			"tool/scope it was originally installed to. Manual and explicit, nothing auto-\n" +
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
			reg, root, err := resolveRegistry(cmd.Context(), wd, regSel, home, warnf)
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
				// A local checkout has no cache to write; say what actually happened
				// rather than naming a file that was never touched.
				if root != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "read local registry checkout %s (%d artifacts, %d recipes, %d profiles)\n",
						root, len(cat.Artifacts), len(cat.Recipes), len(cat.Profiles))
					return nil
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

			// A profile leaves no state row: it installs its members. So an
			// update <profile> re-resolves the profile to its current member NAMES
			// (the "all" baseline — per-member targets come from state) and updates
			// each. The profile's own version is not compared (model A).
			for _, n := range args {
				if catalogHasProfile(cat, n) {
					res, err := profile.Resolve(cat, n, "all")
					if err != nil {
						return fmt.Errorf("resolve profile %q: %w", n, err)
					}
					delete(want, n)
					for _, m := range res.Names() {
						want[m] = true
					}
				}
			}
			type candidate struct {
				name, tool, scope, installed, latest string
			}
			var candidates []candidate
			anyInstalled := false

			// A recipe records several state rows (one MERGE per wired tool + a
			// tool-agnostic install row), so it needs collecting across rows before
			// we know which tools to refresh. recipeAgg accumulates the REAL tools a
			// recipe was installed on plus its recorded version, keyed by identity, so an
			// update refreshes exactly those tools — honoring "originally installed to",
			// the contract artifacts already get — and compares by the same version arm
			// as an artifact (ADR-0004: recipes are versioned, no special-case).
			type recipeKey struct{ name, scope string }
			type recipeAggEntry struct {
				tools     map[string]bool
				installed string // the recorded ItemVersion (same across a recipe's rows)
			}
			recipeAgg := map[recipeKey]*recipeAggEntry{}
			var recipeOrder []recipeKey // insertion order, for deterministic output

			receipts, err := packagestate.List(home)
			if err != nil {
				return fmt.Errorf("read package receipts: %w", err)
			}
			receiptByName := make(map[string]*packagestate.Receipt, len(receipts))
			for i := range receipts {
				receiptByName[receipts[i].Recipe] = &receipts[i]
			}
			for _, scope := range []string{"global", "local"} {
				sp := removeStatePath(scope, home, wd)
				s, err := state.Load(sp)
				if err != nil {
					return fmt.Errorf("load %s state: %w", scope, err)
				}
				if scope == "global" {
					for i := range receipts {
						state.Merge(s, []state.Item{directoryStateItem(&receipts[i])})
					}
				}
				for _, it := range s.Items {
					anyInstalled = true
					if !all && !want[it.Artifact] {
						continue
					}
					if it.PackageReceipt != "" && receiptByName[it.PackageReceipt] == nil {
						return fmt.Errorf("package %s: discovery reference has no receipt; restore the authoritative receipt before updating", it.Artifact)
					}
					if receipt := receiptByName[it.Artifact]; receipt != nil {
						if scope != "global" || it.Tool != recipe.TargetAgnostic {
							continue
						}
						rec := findRecipe(cat, it.Artifact)
						if rec != nil && (rec.Manifest.Delivery == nil || rec.Manifest.Delivery.Unpack != "directory") {
							return fmt.Errorf("package %s: selected catalog no longer provides directory delivery", it.Artifact)
						}
						it.ItemVersion = receipt.RecipeVersion
					}
					if catalogHasRecipe(cat, it.Artifact) {
						k := recipeKey{it.Artifact, it.Scope}
						e := recipeAgg[k]
						if e == nil {
							e = &recipeAggEntry{tools: map[string]bool{}}
							recipeAgg[k] = e
							recipeOrder = append(recipeOrder, k)
						}
						e.tools[it.Tool] = true // includes the agnostic install row
						if e.installed == "" {
							e.installed = it.ItemVersion
						}
						continue
					}
					candidates = append(candidates, candidate{
						name: it.Artifact, tool: it.Tool, scope: it.Scope,
						installed: it.ItemVersion, latest: latestVersion(cat, it.Artifact),
					})
				}
			}

			// Emit one recipe candidate per recorded REAL tool, dropping the agnostic
			// install row (its package install rides along with any real tool's
			// reinstall). A recipe wired on claude+codex refreshes claude and codex, not
			// opencode. When the agnostic row is the only one (an install-only recipe with no wiring),
			// keep a single no-tool refresh so the install still runs. Each carries the
			// recorded + latest version so the normal compare arm drives it.
			for _, k := range recipeOrder {
				e := recipeAgg[k]
				latest := latestVersion(cat, k.name)
				realTools := make([]string, 0, len(e.tools))
				for tl := range e.tools {
					if tl != recipe.TargetAgnostic {
						realTools = append(realTools, tl)
					}
				}
				sort.Strings(realTools) // deterministic order across runs (map iteration is not)
				if len(realTools) == 0 {
					candidates = append(candidates, candidate{name: k.name, tool: recipe.TargetAgnostic, scope: k.scope, installed: e.installed, latest: latest})
					continue
				}
				for _, tl := range realTools {
					candidates = append(candidates, candidate{name: k.name, tool: tl, scope: k.scope, installed: e.installed, latest: latest})
				}
			}

			if len(candidates) == 0 {
				if !anyInstalled {
					return fmt.Errorf("nothing is installed (no state recorded)")
				}
				return fmt.Errorf("not installed: %v", args)
			}

			out := cmd.OutOrStdout()
			var selected []candidate
			batch := &diff.ChangeSet{}
			for _, c := range candidates {
				rec := findRecipe(cat, c.name)
				directory := rec != nil && rec.Manifest.Delivery != nil && rec.Manifest.Delivery.Unpack == "directory"
				switch {
				case c.latest == "":
					fmt.Fprintf(out, "%s: not in registry — leaving as-is\n", c.name)
				case c.installed == c.latest && !(directory && force):
					if directory {
						tx, err := packagestate.ReadTransaction(home, c.name)
						if err != nil {
							return err
						}
						if tx != nil {
							fmt.Fprintf(out, "%s: pending recovery (%s); retry install %s --deploy\n", c.name, tx.Phase, c.name)
							continue
						}
					}
					fmt.Fprintf(out, "%s: up to date (%s)\n", c.name, c.installed)
				default:
					if c.installed == "" {
						fmt.Fprintf(out, "%s: installed version unknown — refreshing to %s\n", c.name, c.latest)
					} else {
						fmt.Fprintf(out, "%s: %s -> %s\n", c.name, c.installed, c.latest)
					}
					selected = append(selected, c)
				}
			}
			// A directory anywhere in the selected dependency closure adds a batch
			// barrier. Legacy-only updates retain planning immediately before apply.
			hasDirectory := false
			for _, c := range selected {
				for _, name := range requires.Expand([]string{c.name}, cat.Deps) {
					rec := findRecipe(cat, name)
					if rec != nil && rec.Manifest.Delivery != nil && rec.Manifest.Delivery.Unpack == "directory" {
						hasDirectory = true
					}
				}
			}
			planCandidate := func(c candidate) (plannedInstall, error) {
				tool := c.tool
				if tool == recipe.TargetAgnostic {
					tool = ""
				}
				return planInstall(cmd, installPlanRequest{Names: []string{c.name}, Tool: tool, Scope: c.scope, Home: home, ProjectDir: wd, Registry: regSel, Catalog: cat})
			}
			var planned []plannedInstall
			if hasDirectory {
				for _, c := range selected {
					p, err := planCandidate(c)
					if err != nil {
						return err
					}
					planned = append(planned, p)
					batch.Diffs = append(batch.Diffs, p.Changes.Diffs...)
				}
				if deploy && !jsonOutput {
					if err := preflightDirectories(home, batch, force); err != nil {
						return err
					}
				}
			}
			// Deploy the selected directory batch under one lock, before handing
			// any remaining file work to the legacy per-candidate applier.
			if hasDirectory && deploy && !jsonOutput {
				for _, p := range planned {
					p.Changes.DryRun = false
					render.PrintPlan(out, p.Changes, p.Resolver, false)
					printReadiness(out, readinessReport(p.Changes, exec.LookPath))
					printPathReadiness(out, pathReadiness(p.Changes, pathDirs(os.Getenv("PATH"))))
				}
				result, err := deployDirectories(cmd.Context(), home, batch, force)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Packages: %d committed, %d unchanged\n", result.Written, result.Skipped)
				printDirectoryReadiness(out, batch)
			}
			updated := 0
			for i, c := range selected {
				var p plannedInstall
				if hasDirectory && !deploy {
					p = planned[i]
				} else {
					p, err = planCandidate(c)
					if err != nil {
						return err
					}
				}
				p.Changes.DryRun = !deploy
				for _, w := range planWarnings(p.Changes) {
					warnf("%s", w)
				}
				if jsonOutput {
					if err := render.JSON(out, p.Changes); err != nil {
						return err
					}
					continue
				}
				if !hasDirectory || !deploy {
					render.PrintPlan(out, p.Changes, p.Resolver, false)
					printReadiness(out, readinessReport(p.Changes, exec.LookPath))
					printPathReadiness(out, pathReadiness(p.Changes, pathDirs(os.Getenv("PATH"))))
				}
				if deploy {
					if hasDirectory {
						legacy := &diff.ChangeSet{DryRun: false}
						for _, d := range p.Changes.Diffs {
							if d.Directory == nil {
								legacy.Diffs = append(legacy.Diffs, d)
							}
						}
						p.Changes = legacy
					}
					// Legacy overwrites retain existing update semantics; directory
					// force is exclusively the flag the user supplied.
					if err := runDeploy(cmd, p.Changes, p.Resolver, deployOptions{force: true, home: home, projectDir: wd}); err != nil {
						return err
					}
				}
				updated++
			}
			if !deploy && updated > 0 {
				fmt.Fprintln(out, "\n(dry run — pass --deploy to apply updates)")
			}
			return nil
		},
	}

	addRegistryFlags(cmd, &regSel) // --local-registry + --registry-url, same as list/install
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually re-install updated items (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVar(&force, "force", false, "replace edited owned directory package files; legacy updates already overwrite their files")
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

// latestVersion returns the registry's advertised version for an installable name,
// searching artifacts AND recipes (both carry a version since ADR-0004), or "" for
// an unknown name. Recipes compare by the same version arm as artifacts — there is
// no recipe special-case in update.
func latestVersion(cat *registry.Catalog, name string) string {
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest != nil && cat.Artifacts[i].Manifest.Name == name {
			return cat.Artifacts[i].Manifest.Version
		}
	}
	for i := range cat.Recipes {
		if cat.Recipes[i].Manifest != nil && cat.Recipes[i].Manifest.Name == name {
			return cat.Recipes[i].Manifest.Version
		}
	}
	return ""
}

// catalogHasRecipe reports whether name is a recipe in the catalog. update uses it
// to route an installed item's rows through recipe aggregation (one refresh per
// wired tool) — a wiring concern, distinct from versioning: a recipe compares by
// the same version arm as an artifact (ADR-0004).
func catalogHasRecipe(cat *registry.Catalog, name string) bool {
	for i := range cat.Recipes {
		if cat.Recipes[i].Manifest != nil && cat.Recipes[i].Manifest.Name == name {
			return true
		}
	}
	return false
}

// catalogHasProfile reports whether name is a profile in the catalog. update uses
// it to route a profile arg through member expansion (a profile leaves no state
// row of its own — it installs its members).
func catalogHasProfile(cat *registry.Catalog, name string) bool {
	for i := range cat.Profiles {
		if cat.Profiles[i].Manifest != nil && cat.Profiles[i].Manifest.Name == name {
			return true
		}
	}
	return false
}

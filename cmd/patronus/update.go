package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
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
		regSel                                          registrySel
		allowPkgInstalls, allowPiProject, trackExisting bool
		deploy                                          bool
		dryRun                                          bool
		all                                             bool
		force                                           bool
		target                                          string
		local, global                                   bool
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
			"updates. Like install, this is a dry run unless --deploy.\n\n" +
			"Pi updates require exactly one of --local or --global after target/provenance\n" +
			"selection. Pi dry previews use cached/local sources without network acquisition.\n" +
			"Pi --deploy acknowledges settled dependent work and the reload/restart obligation;\n" +
			"global updates require all-consumer review. Runtime activation remains unverified.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := dp06Target(target); err != nil {
				return err
			}
			for _, name := range args {
				if err := codexProfileTarget(name, target); err != nil {
					return err
				}
			}
			if target == "pi" && local == global {
				return fmt.Errorf("pi update scope requires exactly one of --local or --global")
			}
			if deploy && dryRun {
				return fmt.Errorf("--deploy and --dry-run are mutually exclusive")
			}
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			home := homeDir()
			if target != "" {
				if err := codexCheckLock(wd, target); err != nil {
					return err
				}
			}
			var mutation *mutation
			if deploy && !jsonOutput && (len(args) > 0 || all) {
				mutation, err = beginMutation(home, wd)
				if err != nil {
					return err
				}
				defer mutation.close(&err)
			}

			// Resolve the registry the same way install/list do (local checkout vs
			// remote R2), then refresh its catalog so the comparison sees the latest.
			reg, root, err := resolveRegistry(cmd.Context(), wd, regSel, home, warnf)
			if err != nil {
				return err
			}
			previewPi, err := dp06PotentialPi(args, all, target, home, wd)
			if err != nil {
				return err
			}
			var cat *registry.Catalog
			catalogRefreshed := false
			if previewPi && !(target == "pi" && deploy && !jsonOutput) {
				if rr, ok := reg.(*registry.RemoteRegistry); ok {
					rr.Fetcher = dp06OfflineFetcher{}
				}
				cat, err = reg.Catalog(cmd.Context())
				if err != nil {
					return err
				}
			} else {
				cat = refreshCatalog(cmd, reg, warnf)
				catalogRefreshed = true
			}

			// No names → the classic cache-refresh job (already done above for remote;
			// report it).
			if len(args) == 0 && !all {
				if target != "" || local || global {
					return fmt.Errorf("update target/scope requires names or --all")
				}
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

			piRoute, err := dp06UpdateTarget(cat, args, all, target, home, wd)
			if err != nil {
				return err
			}
			if piRoute {
				if local == global {
					return fmt.Errorf("pi update scope requires exactly one of --local or --global")
				}
				scope := "global"
				if local {
					scope = "local"
				}
				if deploy && !jsonOutput && !catalogRefreshed {
					if rr, ok := reg.(*registry.RemoteRegistry); ok {
						rr.Fetcher = registryFetcher
					}
					cat = refreshCatalog(cmd, reg, warnf)
					if cat == nil {
						return fmt.Errorf("update: registry unavailable")
					}
				}
				return dp06UpdatePi(cmd, cat, regSel, args, all, scope, home, wd, deploy, force, mutation, deployOptions{allowPkgInstalls: allowPkgInstalls, allowPiProject: allowPiProject, trackExisting: trackExisting})
			}
			if local || global {
				return fmt.Errorf("update scope flags require a proven Pi target")
			}
			if !catalogRefreshed {
				if rr, ok := reg.(*registry.RemoteRegistry); ok {
					rr.Fetcher = registryFetcher
				}
				cat = refreshCatalog(cmd, reg, warnf)
				if cat == nil {
					return fmt.Errorf("update: registry unavailable")
				}
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
					resolutionTarget := target
					if resolutionTarget == "" {
						resolutionTarget = "all"
					}
					res, err := profile.Resolve(cat, n, resolutionTarget)
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
					if err := mergeDirectoryDiscovery(home, s); err != nil {
						return err
					}
				}
				for _, it := range s.Items {
					anyInstalled = true
					if target != "" && target != "all" && it.Tool != target && it.Tool != recipe.TargetAgnostic {
						continue
					}
					if !all && !want[it.Artifact] {
						continue
					}
					if it.PackageReceipt != "" && (it.Artifact != it.PackageReceipt || scope != "global") {
						return fmt.Errorf("package %s: invalid discovery reference %q in %s scope", it.Artifact, it.PackageReceipt, scope)
					}
					if it.PackageReceipt != "" && receiptByName[it.PackageReceipt] == nil {
						tx, err := packagestate.ReadTransaction(home, it.PackageReceipt)
						if err != nil {
							return err
						}
						if tx == nil {
							return fmt.Errorf("package %s: discovery reference has no receipt; restore the authoritative receipt before updating", it.Artifact)
						}
						it.ItemVersion = ""
					}
					if it.PackageReceipt != "" {
						rec := findRecipe(cat, it.Artifact)
						if rec != nil && (rec.Manifest.Delivery == nil || rec.Manifest.Delivery.Unpack != "directory") {
							return fmt.Errorf("package %s: selected catalog no longer provides directory delivery", it.Artifact)
						}
					}
					if receipt := receiptByName[it.Artifact]; receipt != nil {
						if scope != "global" || it.Tool != recipe.TargetAgnostic {
							continue
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
				var pending *packagestate.Transaction
				if directory {
					var err error
					pending, err = packagestate.ReadTransaction(home, c.name)
					if err != nil {
						return err
					}
				}
				switch {
				case c.latest == "":
					fmt.Fprintf(out, "%s: not in registry — leaving as-is\n", c.name)
				case pending != nil:
					fmt.Fprintf(out, "%s: pending recovery (%s); deploy retries recovery under the package lock\n", c.name, pending.Phase)
					selected = append(selected, c)
				case c.installed == c.latest && !(directory && force):
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
			hasDirectory, piSelected := false, false
			for _, c := range selected {
				piSelected = piSelected || c.tool == "pi"
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
				if err := staticPiSelection(batch, piSelected); err != nil {
					return err
				}
				if err := mutation.checkPlan(batch, nil); err != nil {
					return err
				}
				result, err := deployDirectoriesLocked(cmd.Context(), home, batch, force, mutation, nil)
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
					// force is exclusively the flag the user supplied. runDeploy must
					// reconcile the complete owned set before this candidate counts as
					// updated; unresolved paths/state persistence stop the loop.
					if err := runDeployLocked(cmd, p.Changes, p.Resolver, deployOptions{mutation: mutation, target: c.tool, force: true, home: home, projectDir: wd}, runnerForCommands); err != nil {
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

	cmd.Flags().StringVar(&target, "target", "", "select target: claude|codex|opencode|pi|all (default: installed provenance)")
	cmd.Flags().BoolVar(&local, "local", false, "Pi update: select only project resources; verify global prerequisites")
	cmd.Flags().BoolVar(&global, "global", false, "Pi update: select only global resources, including agnostic dependencies")
	addRegistryFlags(cmd, &regSel) // --local-registry + --registry-url, same as list/install
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually re-install updated items (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVar(&allowPkgInstalls, "allow-package-installs", false, "authorize required native package installs/updates and dependency scripts")
	cmd.Flags().BoolVar(&allowPiProject, "allow-pi-project-config", false, "authorize native Pi project configuration/trust")
	cmd.Flags().BoolVar(&trackExisting, "track-existing", false, "enroll selected external native packages")
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

// dp06UpdateTarget resolves provenance before allowing any scope collection.
func dp06UpdateTarget(cat *registry.Catalog, args []string, all bool, target, home, wd string) (bool, error) {
	if err := dp06Target(target); err != nil {
		return false, err
	}
	wanted := map[string]bool{}
	for _, name := range args {
		wanted[name] = true
		if catalogHasProfile(cat, name) {
			l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
			if err != nil {
				return false, err
			}
			if l.Profile == name && l.Target != "" {
				if target != "" && target != l.Target {
					return false, fmt.Errorf("profile lock target %s differs from requested %s", l.Target, target)
				}
				target = l.Target
			}
			res, err := profile.Resolve(cat, name, "pi")
			if err != nil {
				return false, err
			}
			for _, n := range res.Names() {
				wanted[n] = true
			}
		}
	}
	if target != "" && target != "all" {
		return target == "pi", nil
	}
	for _, scope := range []string{"global", "local"} {
		s, err := state.Load(removeStatePath(scope, home, wd))
		if err != nil {
			return false, err
		}
		for _, it := range s.Items {
			if it.Tool == "pi" && (all || wanted[it.Artifact]) {
				if target == "all" {
					return false, fmt.Errorf("pi selection requires a separate --target pi update with explicit scope")
				}
				return true, nil
			}
		}
	}
	return false, nil
}

func dp06UpdatePi(cmd *cobra.Command, cat *registry.Catalog, reg registrySel, args []string, all bool, scope, home, wd string, deploy, force bool, mutation *mutation, nativeOpts deployOptions) error {
	out := cmd.OutOrStdout()
	installed, err := state.Load(removeStatePath(scope, home, wd))
	if err != nil {
		return err
	}
	if scope == "global" {
		if err := mergeDirectoryDiscovery(home, installed); err != nil {
			return err
		}
	}
	available := map[string]bool{}
	versions := map[string]string{}
	for _, it := range installed.Items {
		if _, ok := piProjectItem(it, piSelectedRoot(scope, home, wd)); !ok {
			continue
		}
		if it.Tool == "pi" || it.Tool == recipe.TargetAgnostic {
			available[it.Artifact] = true
			versions[it.Artifact] = it.ItemVersion
		}
	}
	wanted := map[string]bool{}
	explicit := map[string]bool{}
	for _, name := range args {
		if !catalogHasProfile(cat, name) {
			wanted[name] = true
			explicit[name] = true
			continue
		}
		res, err := profile.Resolve(cat, name, "pi")
		if err != nil {
			return err
		}
		if err := dp06ProfileComplete(res); err != nil {
			return err
		}
		if err := dp06CheckProfileLock(wd, name, "pi", cat, home, scope, out); err != nil {
			return err
		}
		for _, n := range res.Names() {
			wanted[n] = true
		}
		l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
		if err != nil {
			return err
		}
		if l.Profile == name {
			for _, e := range l.Entries {
				if !available[e.Name] {
					fmt.Fprintf(out, "%s: absent pinned member; not reinstalling\n", e.Name)
				}
			}
		}
	}
	var names []string
	for name := range wanted {
		if !available[name] {
			fmt.Fprintf(out, "%s: absent current member in %s scope; not reinstalling\n", name, scope)
			if explicit[name] {
				return fmt.Errorf("not installed in selected %s root: %s; update cannot relocate ownership", scope, name)
			}
		}
	}
	for name := range available {
		if all || wanted[name] {
			if latestVersion(cat, name) == "" {
				fmt.Fprintf(out, "%s: not in registry — leaving as-is\n", name)
				continue
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	// Validate the complete selected installed closure, even for up-to-date members.
	for _, name := range names {
		for _, dep := range requires.Expand([]string{name}, cat.Deps) {
			if available[dep] {
				continue
			}
			if scope == "local" {
				if rec := findRecipe(cat, dep); rec != nil && rec.Manifest.Delivery != nil && rec.Manifest.Wire.Method == "" {
					continue
				}
			}
			return fmt.Errorf("pi update dependency-incomplete: %s requires absent %s; separately preview and install the prerequisite", name, dep)
		}
	}
	var changed []string
	for _, name := range names {
		latest := latestVersion(cat, name)
		if latest == "" {
			fmt.Fprintf(out, "%s: not in registry — leaving as-is\n", name)
			continue
		}
		if latest == versions[name] && !force {
			fmt.Fprintf(out, "%s: up to date (%s)\n", name, latest)
		} else {
			fmt.Fprintf(out, "%s: %s -> %s\n", name, versions[name], latest)
			changed = append(changed, name)
		}
	}
	// Plan all selected members together: dependency validation and composition are
	// whole-selection barriers, not a per-item apply loop.
	if len(names) == 0 {
		return nil
	}
	p, err := planInstall(cmd, installPlanRequest{Names: names, Tool: "pi", Scope: scope, Home: home, ProjectDir: wd, Registry: reg, Catalog: cat, Force: force, Acquire: deploy && !jsonOutput})
	if err != nil {
		return err
	}
	if err := dp06UpdatePaths(p.Changes, installed.Items); err != nil {
		return err
	}
	needsFetch := false
	for _, d := range p.Changes.Diffs {
		needsFetch = needsFetch || d.Native != nil || (d.Fetch != nil && d.Action == diff.Fetch)
	}
	if len(changed) == 0 && !needsFetch {
		return nil
	}
	// Finalize has already combined shared file contributions. Retain verified
	// unchanged members in the batch so their ownership is never dropped.
	p.Changes.DryRun = !deploy
	if err := staticPiSelection(p.Changes, true); err != nil {
		return err
	}
	for _, w := range planWarnings(p.Changes) {
		fmt.Fprintln(cmd.ErrOrStderr(), w)
	}
	if jsonOutput {
		return render.JSON(out, p.Changes)
	}
	dp06PrintSelection(cmd, p.Changes, "pi", p.Resolver)
	render.PrintPlan(out, p.Changes, p.Resolver, true)
	if !deploy {
		return nil
	}
	dp06Acknowledge(cmd, "update")
	// Pi preflight has proved these ordinary replacements are unchanged owned
	// resources. They are updates, not discretionary overwrite prompts. Directory
	// force remains exclusively the operator flag.
	for i := range p.Changes.Diffs {
		if p.Changes.Diffs[i].Action == diff.Conflict {
			p.Changes.Diffs[i].Action = diff.Create
		}
	}
	return runDeployLocked(cmd, p.Changes, p.Resolver, deployOptions{allowPkgInstalls: nativeOpts.allowPkgInstalls, allowPiProject: nativeOpts.allowPiProject, trackExisting: nativeOpts.trackExisting, mutation: mutation, target: "pi", globalPrerequisites: p.GlobalPrerequisites, force: force, home: home, projectDir: wd}, runnerForCommands)
}

// Before catalog acquisition, conservatively recognize possible Pi provenance.
// Exact profile membership is resolved afterwards from available catalog bytes.
func dp06PotentialPi(args []string, all bool, target, home, wd string) (bool, error) {
	if target != "" && target != "all" {
		return target == "pi", nil
	}
	if len(args) == 0 && !all {
		return false, nil
	}
	l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
	if err != nil {
		return false, err
	}
	if l.Target == "pi" && (all || contains(args, l.Profile)) {
		return true, nil
	}
	for _, scope := range []string{"global", "local"} {
		s, err := state.Load(removeStatePath(scope, home, wd))
		if err != nil {
			return false, err
		}
		for _, it := range s.Items {
			if it.Tool == "pi" {
				return true, nil
			}
		}
	}
	return false, nil
}

// Changed environment roots cannot silently relocate an installed identity.
// Removal always uses the recorded paths; relocation needs a distinct migration.
func dp06UpdatePaths(cs *diff.ChangeSet, old []state.Item) error {
	desired := map[string]map[string]bool{}
	for _, d := range cs.Diffs {
		names := []string{d.Artifact}
		for _, c := range d.Contrib {
			names = append(names, c.Artifact)
		}
		for _, c := range d.SettingContrib {
			names = append(names, c.Artifact)
		}
		for _, name := range names {
			if desired[name] == nil {
				desired[name] = map[string]bool{}
			}
			desired[name][d.Path] = true
		}
	}
	for _, it := range old {
		paths, selected := desired[it.Artifact]
		if !selected || it.Tool != "pi" || len(it.Files) == 0 {
			continue
		}
		overlaps := false
		for _, f := range it.Files {
			overlaps = overlaps || paths[f.Path]
		}
		if !overlaps {
			return fmt.Errorf("pi update %s would relocate recorded root/path %s; restore the recorded root or authorize a separate migration", it.Artifact, it.Files[0].Path)
		}
	}
	return nil
}

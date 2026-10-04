package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/requires"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/source"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func newInstallCmd() *cobra.Command {
	var (
		tool                          string
		global                        bool
		local                         bool
		deploy                        bool
		dryRun                        bool
		verbose                       bool
		force                         bool
		yes                           bool
		recipeSel                     string
		profileSel                    string
		trackExisting, allowPiProject bool
		allowPkgInstalls              bool
		regSel                        registrySel
	)

	cmd := &cobra.Command{
		Use:   "install <name>...",
		Short: "Plan installation of artifact(s)/recipe(s) — dry-run by default; --deploy to write",
		Long: "Computes the exact set of changes installing one or more artifacts or recipes would\n" +
			"make, for each target tool and scope, and renders them as an artifact-centric summary\n" +
			"table, an ASCII tree, and (with --verbose) per-item unified diffs.\n\n" +
			"Artifacts translate+merge into each tool's on-disk layout (CREATE/APPEND/MERGE).\n" +
			"Recipes fetch+verify an external binary (FETCH), wire its MCP server into each tool\n" +
			"(MERGE), and/or run a self-wiring tool's post-install commands (EXEC).\n\n" +
			"SAFE BY DEFAULT: install is a dry run unless you pass --deploy. The absence of --deploy\n" +
			"(or an explicit --dry-run) means nothing is written, fetched, or executed.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if global && local {
				return fmt.Errorf("--global and --local are mutually exclusive")
			}
			if deploy && dryRun {
				return fmt.Errorf("--deploy and --dry-run are mutually exclusive")
			}
			// Exactly one of {positional names, --profile} selects what to install.
			if profileSel != "" {
				if len(args) > 0 {
					return fmt.Errorf("--profile and positional names are mutually exclusive")
				}
				if recipeSel != "" {
					return fmt.Errorf("--profile and --recipe are mutually exclusive")
				}
			} else if len(args) == 0 && recipeSel == "" {
				return fmt.Errorf("specify one or more artifact/recipe names, or --profile <name>")
			}
			scope := ""
			switch {
			case global:
				scope = "global"
			case local:
				scope = "local"
			}

			// --recipe explicitly selects a recipe by name. In Phase 4 the
			// positional name already is the selection; --recipe is accepted for
			// forward-compat and folded into the name list.
			names := args
			if recipeSel != "" && !contains(names, recipeSel) {
				names = append(names, recipeSel)
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			home := toolpath.HomeDir(os.LookupEnv)
			var mutation *mutation
			if deploy && !jsonOutput {
				mutation, err = beginMutation(home, wd)
				if err != nil {
					return err
				}
				defer mutation.close(&err)
			}
			planned, err := planInstall(cmd, installPlanRequest{Names: names, Profile: profileSel, Tool: tool, Scope: scope, Home: home, ProjectDir: wd, Registry: regSel, Force: force, Acquire: deploy && !jsonOutput})
			if err != nil {
				return err
			}
			cs, res := planned.Changes, planned.Resolver
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// DryRun drives the footer wording; only a real --deploy writes.
			cs.DryRun = !deploy

			// Surface any advisory a transform attached to a diff (e.g. an opencode
			// gate matcher token with no permission key was dropped). Dedupe so a
			// warning shared across a composed file's folded diffs prints once.
			for _, w := range planWarnings(cs) {
				warnf("%s", w)
			}

			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), cs)
			}
			dp06PrintSelection(cmd, cs, planned.Target, res)
			render.PrintPlan(cmd.OutOrStdout(), cs, res, verbose)

			// Always-on package-install readiness: for every package-install item,
			// report which candidate managers are present/missing on this host. Shown
			// on dry-run too, so the user sees what they'd need before deploying.
			printReadiness(cmd.OutOrStdout(), readinessReport(cs, exec.LookPath))

			// Always-on PATH readiness: warn when a FETCH-delivered binary
			// lands in a dir absent from the inherited $PATH, so a hook / MCP server /
			// shell that must execute it by bare name cannot resolve it. Shown on
			// dry-run too. A GUI-launched agent's $PATH is frozen at launch, so this is
			// exactly the gap that silently breaks a wired binary.
			printPathReadiness(cmd.OutOrStdout(), pathReadiness(cs, pathDirs(os.Getenv("PATH"))))

			// Without --deploy this is a safe dry run: plan shown, nothing written.
			if !deploy {
				return nil
			}
			return runDeployLocked(cmd, cs, res, deployOptions{trackExisting: trackExisting, allowPiProject: allowPiProject, profile: profileSel, mutation: mutation, target: planned.Target, globalPrerequisites: planned.GlobalPrerequisites, force: force, yes: yes, allowPkgInstalls: allowPkgInstalls, home: home, projectDir: wd}, runnerForCommands)
		},
	}

	cmd.Flags().StringVar(&tool, "target", "", "target runtime: claude|codex|opencode|pi|all (required for anything that wires to a runtime)")
	cmd.Flags().BoolVar(&global, "global", false, "install at global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "install at project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually write changes to disk (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "also show per-artifact unified diffs")
	cmd.Flags().BoolVar(&force, "force", false, "with --deploy: overwrite conflicting files without prompting")
	cmd.Flags().BoolVar(&yes, "yes", false, "with --deploy: non-interactive; legacy conflicts are skipped, directory conflicts remain errors")
	cmd.Flags().StringVar(&recipeSel, "recipe", "", "pick a specific recipe for a capability (e.g. memory-engram)")
	cmd.Flags().StringVar(&profileSel, "profile", "", "install a curated bundle across layers (§5d)")
	addRegistryFlags(cmd, &regSel)
	cmd.Flags().BoolVar(&trackExisting, "track-existing", false, "enroll compatible external native packages for future forced removal")
	cmd.Flags().BoolVar(&allowPiProject, "allow-pi-project-config", false, "authorize native Pi project configuration/trust at the selected cwd")
	cmd.Flags().BoolVar(&allowPkgInstalls, "allow-package-installs", false,
		"with --deploy: let Patronus run package-manager installs (npm/cargo/uv) non-interactively; all-or-nothing — errors if any required manager is absent")
	return cmd
}

// installPlanRequest separates planning from consent and deployment.
type installPlanRequest struct {
	Force                                  bool
	Names                                  []string
	Profile, Tool, Scope, Home, ProjectDir string
	Registry                               registrySel
	Catalog                                *registry.Catalog
	Acquire                                bool
}

type plannedInstall struct {
	Changes             *diff.ChangeSet
	Resolver            toolpath.Resolver
	Target              string
	GlobalPrerequisites []diff.FileDiff
}

func planInstall(cmd *cobra.Command, req installPlanRequest) (plannedInstall, error) {
	if err := dp06Target(req.Tool); err != nil {
		return plannedInstall{}, err
	}
	names := append([]string(nil), req.Names...)
	profileSel, tool, scope, home, wd, regSel := req.Profile, req.Tool, req.Scope, req.Home, req.ProjectDir, req.Registry
	warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

	reg, root, err := resolveRegistry(cmd.Context(), wd, regSel, home, warnf)
	if err != nil {
		return plannedInstall{}, err
	}

	if profileSel != "" && tool == "" {
		l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
		if err != nil {
			return plannedInstall{}, err
		}
		if l.Profile == profileSel && l.Target != "" {
			tool = l.Target
		}
	}
	if tool == "pi" && !req.Acquire {
		if rr, ok := reg.(*registry.RemoteRegistry); ok {
			rr.Fetcher = dp06OfflineFetcher{}
		}
	}
	// adapters/ comes from the checkout when local; loadAdapters falls back to
	// the embedded adapters when root is "" (installed-binary / remote case).
	adapters, err := loadAdapters(filepath.Join(root, "adapters"))
	if err != nil {
		return plannedInstall{}, err
	}

	// Load the catalog. When the user is installing ONLY out-of-tree sources
	// (git:/https:/file:) and no profile, the registry is not actually needed,
	// so a fetch failure (e.g. offline, no cache) degrades to an empty catalog
	// rather than blocking a self-contained sourced install.
	cat := req.Catalog
	if cat != nil {
		data, marshalErr := json.Marshal(cat)
		if marshalErr != nil {
			return plannedInstall{}, marshalErr
		}
		cat = &registry.Catalog{}
		if err := json.Unmarshal(data, cat); err != nil {
			return plannedInstall{}, err
		}
	} else {
		cat, err = reg.Catalog(cmd.Context())
	}
	if err != nil {
		// Only tolerate a registry failure when every name is a self-contained
		// sourced reference (git:/https:/file:) and there's no profile to resolve.
		if profileSel != "" || !allSourced(names) {
			return plannedInstall{}, err
		}
		warnf("registry unavailable (%v); proceeding with sourced references only", err)
		cat = &registry.Catalog{}
	}

	// --profile expands to the profile's resolved item names, which then flow
	// through the SAME artifact-vs-recipe dispatch a plain install uses.
	if profileSel != "" {
		if err := dp06CheckProfileLock(wd, profileSel, tool, cat, home, scope, cmd.OutOrStdout()); err != nil {
			return plannedInstall{}, err
		}
		// tool selects per-tool flavours (§4); "all" yields the tool-agnostic baseline.
		res, err := profile.Resolve(cat, profileSel, tool)
		if err != nil {
			return plannedInstall{}, err
		}
		if tool == "pi" {
			if err := dp06ProfileComplete(res); err != nil {
				return plannedInstall{}, err
			}
		}
		for _, w := range res.Warnings {
			warnf("%s", w)
		}
		names = res.Names()
		if len(names) == 0 {
			return plannedInstall{}, fmt.Errorf("profile %q resolved to no installable items", profileSel)
		}

		// Per-item reality-follows-lock: if a committed patronus.lock pins this
		// profile's items, rewrite the catalog so each is fetched at its LOCKED
		// version+sha from the registry's immutable key (not the index's latest).
		base := ""
		if rr, ok := reg.(*registry.RemoteRegistry); ok {
			base = rr.Base()
		}
		if err := applyLockPins(wd, profileSel, tool, base, cat, warnf); err != nil {
			return plannedInstall{}, err
		}
	}

	// Expand the `requires` closure: an item that needs another (a hook that
	// needs its binary recipe, an instruction that needs the binary it
	// documents) silently pulls that dependency in, dependency-before-dependent.
	// Pure over the catalog — sourced/unknown names contribute no edges and
	// pass through. Applies to BOTH the profile path (above) and a direct
	// `install <name>`, since both converge on `names` here. The profile's own
	// flavour/without selection has already run; requires works on base names.
	requested := append([]string(nil), names...)
	expanded := requires.Expand(names, cat.Deps)
	if pulled := requires.Pulled(names, expanded); len(pulled) > 0 {
		warnf("also installing required item(s): %s", strings.Join(pulled, ", "))
	}
	names = expanded

	// A positional name may be a sourced reference (file:, git:, https:, ...).
	// Resolve any sourced entries into the catalog so they dispatch like an
	// in-tree item; bare names are left untouched.
	names, err = mergeSourcedNames(cmd.Context(), cat, names, home, tool == "pi" && !req.Acquire)
	if err != nil {
		return plannedInstall{}, err
	}

	if tool == "pi" && scope == "" {
		scope, err = dp06DefaultPiScope(cat, names)
		if err != nil {
			return plannedInstall{}, err
		}
		out := cmd.OutOrStdout()
		if jsonOutput {
			out = cmd.ErrOrStderr()
		}
		fmt.Fprintf(out, "Pi resolved default scope: %s\n", scope)
	}
	// --target is required for anything that wires into a runtime. A
	// purely-agnostic item (binary/package-only recipe) may omit it.
	if tool == "" {
		var needing []string
		for _, n := range names {
			if itemNeedsTarget(cat, n) {
				needing = append(needing, n)
			}
		}
		if len(needing) > 0 {
			return plannedInstall{}, fmt.Errorf("--target is required (one of claude|codex|opencode|pi|all) for: %s", strings.Join(needing, ", "))
		}
	}

	// For a remote registry, fetch+unpack the selected artifacts' source so
	// the local adapter path can transform them (no-op for local/recipes).
	if err := materializeSelected(cmd.Context(), reg, cat, names); err != nil {
		return plannedInstall{}, err
	}

	inv, err := scan.Scan(scan.Options{ProjectDir: wd, Adapters: adapters})
	if err != nil {
		return plannedInstall{}, err
	}

	env := os.LookupEnv
	res := toolpath.New(env, home, wd)

	var globalPrerequisites []diff.FileDiff
	cs, err := computePlan(planInputs{
		cat:            cat,
		inv:            inv,
		adapters:       adapterMap(adapters),
		res:            res,
		names:          names,
		tool:           tool,
		scope:          scope,
		warnf:          warnf,
		verifiedGlobal: func(d diff.FileDiff) { globalPrerequisites = append(globalPrerequisites, d) },
	})
	if err != nil {
		return plannedInstall{}, err
	}

	if err := staticPiSelection(cs, tool == "pi"); err != nil {
		return plannedInstall{}, err
	}
	for i := range cs.Diffs {
		if cs.Diffs[i].Native != nil && !contains(requested, cs.Diffs[i].Artifact) {
			cs.Diffs[i].NativePrerequisite = true
		}
	}
	if err := inspectNative(cs, home, wd, req.Force, nil); err != nil {
		return plannedInstall{}, err
	}
	if err := inspectDirectoryPlan(home, cs); err != nil {
		return plannedInstall{}, err
	}
	if reviews, err := piPreflightPlan(cs, res, home, wd); err != nil {
		return plannedInstall{}, err
	} else if len(reviews) > 0 {
		warnf("Pi mixed-context conflict: operator-prepared combined file and separately scoped interactive consent required before any selected write")
	}
	return plannedInstall{Changes: cs, Resolver: res, Target: tool, GlobalPrerequisites: globalPrerequisites}, nil
}

// planInputs carries everything computePlan needs to build a change set across a
// mix of artifact and recipe names.
type planInputs struct {
	cat            *registry.Catalog
	inv            *scan.Inventory
	adapters       map[string]*manifest.Adapter
	res            toolpath.Resolver
	names          []string
	tool           string
	scope          string
	warnf          func(string, ...any)
	verifiedGlobal func(diff.FileDiff)

	// pluginProbe decides executed-vs-advised for plugin installs; a test seam.
	// Production leaves it nil → plugin.ExecProbe (real `<tool> plugin --help`).
	pluginProbe plugin.CLIProbe
}

// computePlan dispatches each requested name to the artifact path (adapter
// transform) or the recipe path (fetch + wire), concatenates the resulting
// diffs, and runs them through the one shared finalize tail (compose + classify +
// sort). This is the brief's "one spine": two producers, one ChangeSet, one
// applier — the dispatch by registry lookup also prefigures Phase-5 profile
// resolution. Names are unique across artifacts and recipes (enforced at catalog
// load), so a bare name is unambiguous.
func computePlan(in planInputs) (*diff.ChangeSet, error) {
	read := func(path string) ([]byte, bool, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return b, true, nil
	}

	var (
		artifactNames []string
		raw           []diff.FileDiff
	)
	seenDirectories := map[string]bool{}
	for _, name := range in.names {
		if pl := findPlugin(in.cat, name); pl != nil {
			if in.tool == "pi" {
				return nil, fmt.Errorf("pi static selection cannot provision plugin %s", name)
			}
			// Resolve scope and the target tool list the same way artifacts do:
			// an explicit flag wins, else the manifest's defaults; a bare --tool
			// "all"/"" fans out to the plugin's own Targets so a plain install
			// registers on every target (not zero, as a single "all" call would).
			scope := resolvePluginScope(in.scope, pl.Manifest)
			tools := resolvePluginTools(in.tool, pl.Manifest)
			probe := in.pluginProbe
			if probe == nil {
				probe = plugin.ExecProbe{}
			}
			raw = append(raw, pluginInstallDiffs(pl.Manifest, tools, scope, probe)...)
			continue
		}
		if rec := findRecipe(in.cat, name); rec != nil {
			if in.tool == "pi" && rec.Manifest.Wire.Method == manifest.WireMerge {
				ad := in.adapters["pi"]
				if ad == nil || ad.Layout.Mcp == nil {
					return nil, fmt.Errorf("pi MCP adapter layout absent")
				}
				scope := in.scope
				if scope == "" {
					scope = "global"
				}
				target, err := ad.Layout.Mcp.ResolveTarget(scope)
				if err != nil {
					return nil, err
				}
				if _, _, err := scan.ReadPiFile(in.res.ResolveMarker(target.File, "pi", scope)); err != nil {
					return nil, err
				}
			}
			if rec.Manifest.Delivery != nil && rec.Manifest.Delivery.Unpack == "directory" {
				if seenDirectories[name] {
					continue
				}
				seenDirectories[name] = true
			}
			recipeScope := in.scope
			if in.tool == "pi" && in.scope == "local" && rec.Manifest.Delivery != nil && rec.Manifest.Delivery.Unpack == "directory" {
				recipeScope = "global"
			}
			diffs, err := recipe.Compute(recipe.Request{
				Recipe:       rec.Manifest,
				Adapters:     in.adapters,
				Resolver:     in.res,
				Tool:         in.tool,
				Scope:        recipeScope,
				PlacedDigest: placedDigestFromState(in.inv),
				Warnf:        in.warnf,
			})
			if err != nil {
				return nil, err
			}
			if in.tool == "pi" && len(diffs) == 0 {
				return nil, fmt.Errorf("required recipe %s has no delivery outcome", name)
			}
			if in.tool == "pi" && in.scope == "local" {
				var localDiffs []diff.FileDiff
				for _, d := range diffs {
					if d.Scope == "global" && d.Tool == recipe.TargetAgnostic {
						if err := dp06VerifyGlobal(in.res.ExpandHome("~"), rec.Manifest.Version, d); err != nil {
							return nil, err
						}
						if in.verifiedGlobal != nil {
							d.Version = rec.Manifest.Version
							in.verifiedGlobal(d)
						}
						if in.warnf != nil {
							in.warnf("verified existing global prerequisite %s at %s; local selection makes no global writes", d.Artifact, d.Path)
						}
					} else {
						localDiffs = append(localDiffs, d)
					}
				}
				diffs = localDiffs
			}
			// Stamp each recipe diff with the recipe's own version so state records
			// its ItemVersion — the same thing the adapter engine does for artifacts
			// (internal/adapter/engine.go). Without this, an installed recipe has no
			// recorded version to compare on update (ADR-0004).
			for i := range diffs {
				diffs[i].Version = rec.Manifest.Version
			}
			raw = append(raw, diffs...)
			continue
		}
		artifactNames = append(artifactNames, name)
	}

	// Artifacts share the existing planner (it owns adapter transform + tool/scope
	// resolution); take its raw, un-finalized diffs and fold them in with recipes.
	if len(artifactNames) > 0 {
		acs, err := plan.Compute(plan.Request{
			Catalog:   in.cat,
			Inventory: in.inv,
			Adapters:  in.adapters,
			Resolver:  in.res,
			Names:     artifactNames,
			Tool:      in.tool,
			Scope:     in.scope,
		})
		if err != nil {
			return nil, err
		}
		raw = append(raw, acs.Diffs...)
	}

	cs, err := plan.Finalize(raw, read)
	if err != nil {
		return nil, err
	}
	// Structural ownership is independent of target selection: a row in the
	// other scope may still claim this same absolute config path.
	needsOwnership := in.tool == "pi"
	for _, d := range cs.Diffs {
		needsOwnership = needsOwnership || d.Setting != nil
	}
	if !needsOwnership {
		return cs, nil
	}
	var owners []state.Item
	roots := []string{in.res.ExpandHome("~"), in.res.ResolveMarker(".", "", "local")}
	seen := map[string]bool{}
	for _, root := range roots {
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		s, err := state.Load(filepath.Join(root, ".patronus", "state.json"))
		if err != nil {
			return nil, fmt.Errorf("read setting ownership in %s: %w", root, err)
		}
		owners = append(owners, s.Items...)
	}
	if in.tool == "pi" {
		if err := dp06AdmitFetches(cs, owners); err != nil {
			return nil, err
		}
	}
	stampPiRoots(cs.Diffs, in.res.ExpandHome("~"), in.res.ResolveMarker(".", "pi", "local"))
	return plan.AdmitSettings(cs, owners)
}

// planWarnings collects the distinct, non-empty advisories transforms attached to
// the change set's diffs, preserving first-seen order. A single warning is shared
// across a composed file's folded diffs, so dedup keeps it from printing twice.
func planWarnings(cs *diff.ChangeSet) []string {
	seen := map[string]bool{}
	var out []string
	for i := range cs.Diffs {
		w := cs.Diffs[i].Warning
		if w == "" || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

// mergeSourcedNames resolves each name as a sourced reference. Bare (registry)
// names pass through unchanged. A file:/git:/https: reference is fetched+loaded
// into the catalog so it dispatches exactly like an in-tree item, and its dispatch
// name becomes the resolved manifest's name. Names already present in the catalog
// (the common case) take the registry path with zero overhead.
func mergeSourcedNames(ctx context.Context, cat *registry.Catalog, names []string, home string, offline bool) ([]string, error) {
	rs := &source.Resolver{
		Fetcher:  fetcherForCommands,
		CacheDir: filepath.Join(home, ".patronus", "cache", "sources"),
	}
	if offline {
		rs.Fetcher = dp06OfflineFetcher{}
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		ref, err := source.Parse(n)
		if err != nil {
			return nil, err
		}
		if ref.Scheme == source.Registry {
			out = append(out, ref.Name)
			continue
		}
		resolved, err := rs.Resolve(ctx, ref)
		if err != nil {
			return nil, err
		}
		switch {
		case resolved.Artifact != nil:
			cat.Artifacts = append(cat.Artifacts, *resolved.Artifact)
			out = append(out, resolved.Artifact.Manifest.Name)
		case resolved.Recipe != nil:
			cat.Recipes = append(cat.Recipes, *resolved.Recipe)
			out = append(out, resolved.Recipe.Manifest.Name)
		case resolved.Plugin != nil:
			cat.Plugins = append(cat.Plugins, *resolved.Plugin)
			out = append(out, resolved.Plugin.Manifest.Name)
		default:
			return nil, fmt.Errorf("source %q resolved to nothing", n)
		}
	}
	return out, nil
}

// allSourced reports whether every name is an out-of-tree sourced reference (a
// scheme like git:/https:/file:), so a plain install of them needs no registry.
func allSourced(names []string) bool {
	if len(names) == 0 {
		return false
	}
	for _, n := range names {
		ref, err := source.Parse(n)
		if err != nil || ref.Scheme == source.Registry {
			return false
		}
	}
	return true
}

// applyLockPins implements PER-ITEM reality-follows-lock: when a committed
// patronus.lock pins this profile's items, it rewrites each matching catalog
// artifact or directory recipe entry so the install fetches the locked version+bytes from the
// registry's immutable name/version key, rather than whatever the (mutable)
// discovery index now advertises as latest. This is what makes a shared lock
// reproduce the exact environment even as the catalog moves on.
//
// It is a no-op when there's no lock, the lock is for a different profile, or an
// item isn't pinned — those follow the index latest, unchanged. base is the
// RemoteRegistry base URL (used to reconstruct the immutable item URL).
func applyLockPins(wd, profileName, target, base string, cat *registry.Catalog, warnf func(string, ...any)) error {
	l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
	if err != nil {
		return fmt.Errorf("read patronus.lock: %w", err)
	}
	if l.Profile != "" && l.Profile != profileName {
		return nil // an unrelated lock never silently pins this install
	}
	if l.Target != "" && l.Target != target {
		return fmt.Errorf("patronus.lock target %s differs from requested %s", l.Target, target)
	}
	pin := make(map[string]lock.Entry, len(l.Entries))
	for _, e := range l.Entries {
		pin[e.Name] = e
	}
	for _, e := range l.Entries {
		if e.NativeSource != "" {
			rec := findRecipe(cat, e.Name)
			if rec == nil || !manifest.IsPiDelivery(rec.Manifest.Delivery) {
				return fmt.Errorf("native lock pin %s is inapplicable", e.Name)
			}
			pinned := *rec.Manifest
			delivery := *pinned.Delivery
			delivery.Install = []manifest.InstallCandidate{{Manager: manifest.PMPi, Ref: e.NativeSource}}
			pinned.Version, pinned.Delivery = e.Version, &delivery
			if err := manifest.ValidateRecipe(&pinned); err != nil {
				return err
			}
			rec.Manifest = &pinned
		}
		if e.Delivery != nil {
			rec := findRecipe(cat, e.Name)
			if rec == nil || rec.Manifest.Delivery == nil || rec.Manifest.Delivery.Unpack != "directory" {
				return fmt.Errorf("patronus.lock: directory pin %q is inapplicable to the selected catalog", e.Name)
			}
		}
	}
	for i := range cat.Recipes {
		rec := &cat.Recipes[i]
		if rec.Manifest.Delivery == nil || rec.Manifest.Delivery.Unpack != "directory" {
			continue
		}
		e, ok := pin[rec.Manifest.Name]
		if !ok {
			continue
		}
		if e.Kind != "recipe" || e.Delivery == nil || e.Delivery.Unpack != "directory" {
			return fmt.Errorf("patronus.lock: directory recipe %q requires a complete directory delivery pin", e.Name)
		}
		pinned := *rec.Manifest
		pinned.Version, pinned.Delivery = e.Version, e.Delivery
		if err := manifest.ValidateRecipe(&pinned); err != nil {
			return fmt.Errorf("patronus.lock: %w", err)
		}
		rec.Manifest = &pinned
	}
	if base == "" {
		// Local artifact sources retain their current behavior.
		return nil
	}
	base = strings.TrimRight(base, "/")
	for i := range cat.Artifacts {
		a := &cat.Artifacts[i]
		e, ok := pin[a.Manifest.Name]
		if !ok || e.Version == "" || e.Kind != "artifact" {
			continue
		}
		if e.Version != a.Manifest.Version {
			warnf("pinning %s to %s from patronus.lock (index advertises %s)", e.Name, e.Version, a.Manifest.Version)
			a.Manifest.Version = e.Version // so Materialize's cache key is the locked version
		}
		// Reconstruct the immutable R2 key from name/version; pin the lock's tarball
		// sha so Materialize verifies the exact bytes. Clear LocalDir so a stale
		// latest materialization (if any) is not reused.
		a.Source.TarballURL = base + "/catalog/" + e.Name + "/" + e.Version + "/" + e.Name + "-" + e.Version + ".tar.gz"
		if e.TarballSha256 != "" {
			a.Source.SHA256 = e.TarballSha256
		}
		a.Source.LocalDir = ""
	}
	return nil
}

// contains reports whether ss includes s.
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// findPlugin returns the catalog plugin entry with the given name, or nil.
func findPlugin(cat *registry.Catalog, name string) *registry.PluginEntry {
	for i := range cat.Plugins {
		if cat.Plugins[i].Manifest.Name == name {
			return &cat.Plugins[i]
		}
	}
	return nil
}

// resolvePluginScope picks the install scope for a plugin: an explicit flag wins,
// else the manifest's defaults.scope, else local (project). Mirrors the artifact
// planner's resolveScope, normalizing "project" to "local". The only valid scopes
// are "global" and "local"; an unrecognized value falls through to local rather
// than registering a marketplace plugin in the wrong place.
func resolvePluginScope(flag string, p *manifest.Plugin) string {
	s := flag
	if s == "" {
		s = p.Defaults.Scope
	}
	if s == "project" {
		s = "local"
	}
	if s == "" {
		s = "local"
	}
	return s
}

// resolvePluginTools picks which tools to register a plugin on. A specific --target
// is used as-is (Compute resolves an unsupported target to an honest no-op). Only an
// explicit "all" fans out to the plugin's declared Targets. An empty flag can now
// reach here only on an agnostic path (the required-target gate errors first for a
// plugin), so it registers on nothing.
func resolvePluginTools(flag string, p *manifest.Plugin) []string {
	if flag != "" && flag != "all" {
		return []string{flag}
	}
	if flag == "all" {
		return p.Targets
	}
	return nil // no target: nothing to register (gate above already errored for a plugin)
}

// itemNeedsTarget reports whether installing name produces at least one targeted
// row — a recipe that wires an MCP/config entry into a runtime (Wire.Method ==
// merge), or a plugin, or an artifact. Such an item requires an explicit --target.
// A purely-agnostic recipe (WireExec/WireNone: a binary/package with no per-runtime
// wiring) does not. An unknown/sourced name is treated as needing a target (fail safe).
func itemNeedsTarget(cat *registry.Catalog, name string) bool {
	if rec := findRecipe(cat, name); rec != nil {
		return rec.Manifest.Wire.Method == manifest.WireMerge
	}
	if findPlugin(cat, name) != nil {
		return true
	}
	return true // artifacts + sourced/unknown: require a target
}

// findRecipe returns the catalog recipe entry with the given name, or nil.
func findRecipe(cat *registry.Catalog, name string) *registry.RecipeEntry {
	for i := range cat.Recipes {
		if cat.Recipes[i].Manifest.Name == name {
			return &cat.Recipes[i]
		}
	}
	return nil
}

// adapterMap keys the loaded adapters by tool for the planner.
func adapterMap(adapters []*manifest.Adapter) map[string]*manifest.Adapter {
	m := make(map[string]*manifest.Adapter, len(adapters))
	for _, ad := range adapters {
		m[ad.Tool] = ad
	}
	return m
}

// deployOptions carries the inputs runDeploy needs beyond the change set.
type deployOptions struct {
	trackExisting, allowPiProject bool
	profile                       string
	globalPrerequisites           []diff.FileDiff // verified read-only dependencies for a local operation
	target                        string          // requested runtime, even when all emitted rows are agnostic
	mutation                      *mutation
	saveState                     func(string, *state.State) error // nil uses atomic state.Save
	piConsents                    []piContextConsent               // operation-local prepared-byte consent; never persisted
	force                         bool
	yes                           bool
	allowPkgInstalls              bool   // --allow-package-installs: run package installs non-interactively, all-or-nothing
	home                          string // for ~/.patronus/state.json
	projectDir                    string // for <project>/.patronus/state.json
}

// commandRunner runs a self-wiring recipe's post-install command. The real impl
// shells out via os/exec; tests inject a fake so no test spawns a process.
type commandRunner interface {
	Run(argv []string) error
}

// runnerForCommands is the package-level seam for self-wiring post-install EXECs
// driven through the cobra commands (the runDeploy path takes no runner argument).
// Production leaves it nil → real os/exec. Integration tests that install a
// self-wiring recipe (e.g. memory-ai-memory) set it to a fake so the suite stays
// process-free — mirroring fetcherForCommands/registryFetcher. A test hook, not a
// user knob.
var runnerForCommands commandRunner

// fetcherForDeploy is the package-level seam for FETCH downloads on --deploy
// (binary recipes like gitleaks). Production uses the real HTTP fetcher;
// integration tests swap in the serving fetcher so a profile carrying a
// github-release binary installs fully offline — mirroring fetcherForCommands.
var fetcherForDeploy recipe.Fetcher = recipe.HTTPFetcher{}

// execRunner is the production commandRunner: it runs argv via os/exec, streaming
// output to the command's stdout/stderr.
type execRunner struct {
	cmd *cobra.Command
}

func (r execRunner) Run(argv []string) error {
	// Bind the command to the cobra context so a cancelled run (Ctrl-C, timeout)
	// also tears down the self-wiring post-install process.
	c := exec.CommandContext(r.cmd.Context(), argv[0], argv[1:]...)
	c.Stdout = r.cmd.OutOrStdout()
	c.Stderr = r.cmd.ErrOrStderr()
	return c.Run()
}

// runDeploy writes the change set to disk, performs FETCH downloads, runs
// self-wiring EXEC commands, and records what was installed. It is Terraform-style:
// a mid-apply failure stops, records what already succeeded in state, and returns
// the error. The runner is overridable for tests (nil => real os/exec).
func runDeploy(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions) error {
	return runDeployWith(cmd, cs, res, opts, runnerForCommands)
}

func runDeployWith(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions, runner commandRunner) (err error) {
	m, err := beginMutation(opts.home, opts.projectDir)
	if err != nil {
		return err
	}
	defer m.close(&err)
	opts.mutation = m
	return runDeployLocked(cmd, cs, res, opts, runner)
}

// runDeployLocked is called only by an owning command or runDeployWith.
func runDeployLocked(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions, runner commandRunner) error {
	if err := staticPiSelection(cs, opts.target == "pi"); err != nil {
		return err
	}
	// Read both ownership scopes before any mutation, including non-config Pi
	// installs. Unsupported/insufficient state cannot be repaired by equality.
	var owners []state.Item
	for _, scope := range []string{"global", "local"} {
		s, err := state.Load(statePath(scope, opts))
		if err != nil {
			return fmt.Errorf("load %s state: %w", scope, err)
		}
		owners = append(owners, s.Items...)
	}
	if err := preflightPiDeploy(cmd, cs, res, &opts); err != nil {
		return err
	}
	admitted, err := plan.AdmitSettings(cs, owners)
	if err != nil {
		return err
	}
	cs = admitted
	if err := opts.mutation.checkPlan(cs, opts.piConsents); err != nil {
		return err
	}
	out := cmd.OutOrStdout()

	consent := installConsent{
		allow: opts.allowPkgInstalls,
		yes:   opts.yes,
		look:  exec.LookPath,
		in:    bufio.NewReader(cmd.InOrStdin()),
		out:   out,
	}
	// Under --allow-package-installs we never deploy a subset: if any required item
	// has no manager on PATH, error before writing anything.
	if consent.allow {
		if err := preflightAllOrNothing(cs, consent.look); err != nil {
			return err
		}
	}

	var prepared *dp06Acquisition
	piSelected := opts.target == "pi"
	for _, d := range cs.Diffs {
		piSelected = piSelected || d.Tool == "pi"
	}
	if piSelected {
		if err := dp06AdmitFetches(cs, owners); err != nil {
			return err
		}
		if err := preflightDirectories(opts.home, cs, opts.force); err != nil {
			return err
		}
		prepared, err = dp06Acquire(cmd.Context(), cs)
		if err != nil {
			return err
		}
		defer prepared.close()
		// Acquisition can take time; recheck all snapshots before the first write.
		if err := opts.mutation.checkPlan(cs, opts.piConsents); err != nil {
			return err
		}
	}
	for _, d := range opts.globalPrerequisites {
		if err := dp06VerifyGlobal(opts.home, d.Version, d); err != nil {
			return err
		}
	}
	if err := applyNative(cmd, cs, opts, nil); err != nil {
		return err
	}
	if err := refreshNativeSettings(cs); err != nil {
		return err
	}
	admitted, err = plan.AdmitSettings(cs, owners)
	if err != nil {
		return err
	}
	cs = admitted
	var directoryResult directoryDeployResult
	if prepared != nil {
		directoryResult, err = deployDirectoriesLocked(cmd.Context(), opts.home, cs, opts.force, opts.mutation, prepared)
	} else {
		directoryResult, err = deployDirectoriesLocked(cmd.Context(), opts.home, cs, opts.force, opts.mutation, nil)
	}
	membershipErr := recordDirectoryProfile(cs, opts)
	if err != nil || membershipErr != nil {
		return errors.Join(err, membershipErr)
	}
	legacy := directoryResult.Legacy
	var authored []diff.FileDiff
	for _, d := range legacy.Diffs {
		if d.Native == nil {
			authored = append(authored, d)
		}
	}
	legacy = &diff.ChangeSet{Diffs: authored, DryRun: legacy.DryRun}
	app := &install.Applier{
		BeforeWrite: func(d diff.FileDiff) error { return opts.mutation.checkFile(d, opts.piConsents) },
		Force:       opts.force,
		Conflict:    conflictPrompt(cmd, res, opts.yes),
		Fetcher:     fetcherForDeploy,
		Ctx:         cmd.Context(),
	}
	if prepared != nil {
		app.Fetcher = prepared
	}
	result, applyErr := app.Apply(legacy)

	// Realize self-wiring post-install commands (EXEC diffs) only after the file
	// writes/fetches succeed, and only on --deploy (we are here). The applier
	// itself never spawns processes — it stays a pure file writer.
	realized := append([]diff.FileDiff(nil), result.Applied...)
	if applyErr == nil {
		if runner == nil {
			runner = execRunner{cmd: cmd}
		}
		ran, execErr := runExecs(cmd, legacy, runner, consent)
		realized = append(realized, ran...)
		if execErr != nil {
			applyErr = execErr
		}
	}

	// Record whatever succeeded BEFORE surfacing any error (state must reflect
	// reality even on partial failure).
	recordedResult := *result
	recordedResult.Applied = realized
	if applyErr == nil || len(realized) > 0 {
		if stateErr := recordStateLocked(legacy, &recordedResult, opts); stateErr != nil {
			applyErr = errors.Join(applyErr, stateErr)
		}
	}

	fmt.Fprintf(out, "\nApplied: %d written, %d skipped\n", len(result.Applied), len(result.Skipped))
	if directoryResult.Written+directoryResult.Skipped > 0 {
		fmt.Fprintf(out, "Packages: %d committed, %d unchanged\n", directoryResult.Written, directoryResult.Skipped)
		printDirectoryReadiness(out, cs)
	}
	if applyErr != nil {
		return fmt.Errorf("%w; %s; re-preview current bytes before repair (equality is not adoption authority)", applyErr, resultDiagnostics(result))
	}
	if piSelected {
		fmt.Fprintln(out, "Pi status: placed, runtime-unverified; eligible at next startup/reload (not proof of runtime registration).")
	}
	return nil
}

// runExecs runs each EXEC diff's command in order and returns the ones that ran
// (for state recording). The first failure stops, mirroring the applier's
// Terraform-style partial-on-failure.
//
// A package-install advisory (one carrying Candidates) consults consent: run it
// via the first present manager if consented, else surface it. Every other advisory
// (an external-actor self-wiring command) stays surface-only.
func runExecs(cmd *cobra.Command, cs *diff.ChangeSet, runner commandRunner, consent installConsent) ([]diff.FileDiff, error) {
	var ran []diff.FileDiff
	var skipped []string // package installs declined/unsatisfiable — summarised at the end
	for i := range cs.Diffs {
		d := cs.Diffs[i]
		if d.Directory != nil || d.Action != diff.Exec || d.Exec == nil {
			continue
		}

		// Package-install advisory: consult consent.
		if d.Exec.Advisory && len(d.Exec.Candidates) > 0 {
			chosen, preferred, ok := firstPresentCandidate(d.Exec.Candidates, consent.look)
			install := false
			switch {
			case !ok:
				// No manager on PATH — surface only (preflight already errored under --allow).
			case consent.allow:
				install = true
			case consent.yes:
				// Non-interactive without --allow: do not install, surface.
			default:
				install = promptInstall(consent, d.Artifact, chosen)
			}
			if install {
				if !preferred {
					fmt.Fprintf(consent.out, "note: %s not available; using %s instead\n",
						d.Exec.Candidates[0].Manager, chosen.Manager)
				}
				fmt.Fprintf(consent.out, "EXEC %s\n", chosen.Command)
				if err := runner.Run(strings.Fields(chosen.Command)); err != nil {
					return ran, fmt.Errorf("package install %q: %w", chosen.Command, err)
				}
				// Stamp the command Patronus actually ran onto the diff's Exec so
				// recordState persists it in Item.PostInstall (remove surfaces the
				// uninstall advisory from there). d.Exec is a pointer, so this reaches
				// the diff recordState later reads.
				d.Exec.Display = chosen.Command
				ran = append(ran, d)
				continue
			}
			skipped = append(skipped, d.Artifact)
			fmt.Fprintf(consent.out, "ADVISORY (run yourself): %s\n", d.Exec.Display)
			ran = append(ran, d)
			continue
		}

		// Non-package advisory (external-actor self-wiring) stays surface-only. It
		// is still recorded so state remembers the recipe and remove can report the
		// manual-cleanup.
		if d.Exec.Advisory {
			fmt.Fprintf(cmd.OutOrStdout(), "ADVISORY (run yourself): %s\n", d.Exec.Display)
			ran = append(ran, d)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "EXEC %s\n", d.Exec.Display)
		if err := runner.Run(d.Exec.Command); err != nil {
			return ran, fmt.Errorf("post-install %q: %w", d.Exec.Display, err)
		}
		ran = append(ran, d)
	}
	if len(skipped) > 0 {
		// One line naming what was not installed, so a per-item decline in
		// interactive mode is visible rather than lost in the scroll.
		fmt.Fprintf(consent.out, "\nSkipped package installs (run them yourself): %v\n", skipped)
	}
	return ran, nil
}

// recordState reconciles complete desired sets against actual outcomes. Directory
// receipt references are reloaded and preserved, never inferred from file diffs.
func recordState(desired *diff.ChangeSet, result *install.Result, opts deployOptions) (err error) {
	m, err := beginMutation(opts.home, opts.projectDir)
	if err != nil {
		return err
	}
	defer m.close(&err)
	opts.mutation = m
	return recordStateLocked(desired, result, opts)
}

func recordStateLocked(desired *diff.ChangeSet, result *install.Result, opts deployOptions) error {
	stampPiRoots(desired.Diffs, opts.home, opts.projectDir)
	stampPiRoots(result.Applied, opts.home, opts.projectDir)
	stampPiRoots(result.Skipped, opts.home, opts.projectDir)
	now := time.Now().UTC().Format(time.RFC3339)
	byScope := map[string][]diff.FileDiff{}
	var scopes []string
	for _, d := range desired.Diffs {
		if d.Directory != nil || d.Native != nil || d.IsDir {
			continue
		}
		if _, ok := byScope[d.Scope]; !ok {
			scopes = append(scopes, d.Scope)
		}
		byScope[d.Scope] = append(byScope[d.Scope], d)
	}
	save := opts.saveState
	if save == nil {
		save = state.Save
	}
	var outcomes []error
	for _, scope := range scopes {
		path := statePath(scope, opts)
		if err := opts.mutation.check(path); err != nil {
			return errors.Join(append(outcomes, fmt.Errorf("save state %s: %w; ownership uncertain; %s", path, err, resultDiagnostics(result)))...)
		}
		s, err := state.Load(path)
		if err != nil {
			return errors.Join(append(outcomes, err)...)
		}
		qualifyPiOwnership(s, byScope[scope])
		reconciled := state.Reconcile(state.ReconcileInput{Old: s.Items, Desired: byScope[scope], Result: *result, Now: now})
		s.Items = reconciled.Items
		if opts.target == "pi" {
			state.EnrollProfile(s, opts.profile, opts.target, scope, piSelectedRoot(scope, opts.home, opts.projectDir), byScope[scope])
		}
		outcomes = append(outcomes, reconciled.Unresolved...)
		if err := opts.mutation.saveState(path, s, save); err != nil {
			outcomes = append(outcomes, fmt.Errorf("save state %s: %w; ownership uncertain; %s", path, err, resultDiagnostics(result)))
			return errors.Join(outcomes...)
		}
	}
	return errors.Join(outcomes...)
}

func resultDiagnostics(result *install.Result) string {
	var applied, skipped []string
	for _, d := range result.Applied {
		applied = append(applied, string(d.Action)+" "+d.Path)
	}
	for _, d := range result.Skipped {
		skipped = append(skipped, string(d.Action)+" "+d.Path)
	}
	failed := "none"
	if result.Failed != nil {
		failed = string(result.Failed.Action) + " " + result.Failed.Path + " (write outcome uncertain)"
	}
	return fmt.Sprintf("verified committed=%q skipped=%q failed=%s", applied, skipped, failed)
}

// statePath returns the state file for a scope.
func statePath(scope string, opts deployOptions) string {
	if scope == "global" {
		return filepath.Join(opts.home, ".patronus", "state.json")
	}
	return filepath.Join(opts.projectDir, ".patronus", "state.json")
}

// placedDigestFromState builds the recipe.PlacedDigestFunc that classifyFetch uses to
// decide whether an already-present ARCHIVE-delivered binary is the one Patronus
// placed and verified — as opposed to some other file that merely happens to be at
// that path.
//
// The datum was already there. install/apply.go stamps FetchSpec.PlacedSHA256 with the
// digest of the binary it actually wrote (the extracted member, for an archive), and
// internal/state persists it as the FETCH row's Checksum ("sha256:<hex>") — with a
// comment naming this exact missing use: "so a later scan can tell 'unchanged' from
// 'user-replaced'." Nothing on the install path ever read it back, which is precisely
// why a poisoned binary at the dest was reported as "verified, up to date" forever.
//
// Both scopes are merged: ~/.patronus/bin/ is global by construction, but a recipe may
// set a project-scoped installTo, and a digest recorded in either state file is still a
// digest Patronus recorded. An unreadable or absent state yields no records at all, so
// classifyFetch FETCHes — fail closed, because "we have never verified this" is not the
// same as "this is fine".
func placedDigestFromState(inv *scan.Inventory) recipe.PlacedDigestFunc {
	byDest := map[string]string{}
	if inv == nil {
		return func(string) (string, bool) { return "", false }
	}
	for _, dir := range []string{inv.Home, inv.ProjectDir} {
		if dir == "" {
			continue
		}
		s, err := state.Load(filepath.Join(dir, ".patronus", "state.json"))
		if err != nil {
			continue // unreadable state -> we know nothing about it -> verify everything
		}
		for _, it := range s.Items {
			for _, f := range it.Files {
				if f.Action == string(diff.Fetch) && f.Checksum != "" {
					byDest[f.Path] = strings.TrimPrefix(f.Checksum, "sha256:")
				}
			}
		}
	}
	return func(dest string) (string, bool) {
		sum, ok := byDest[dest]
		return sum, ok
	}
}

// conflictPrompt builds the interactive resolver for CONFLICT files. In
// non-interactive mode (--yes) it returns nil, which the Applier treats as
// "skip every conflict" — never a silent overwrite.
func conflictPrompt(cmd *cobra.Command, res toolpath.Resolver, yes bool) install.ConflictFunc {
	if yes {
		return nil
	}
	in := bufio.NewReader(cmd.InOrStdin())
	out := cmd.OutOrStdout()
	return func(d diff.FileDiff) (install.Resolution, error) {
		fmt.Fprintf(out, "\nCONFLICT: %s already exists and differs.\n", res.CollapseHome(d.Path))
		fmt.Fprint(out, "  [s]kip (default) / [o]verwrite / [d]iff: ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			return install.Skip, nil
		}
		switch strings.TrimSpace(strings.ToLower(line)) {
		case "o", "overwrite":
			return install.Overwrite, nil
		case "d", "diff":
			fmt.Fprintln(out, d.Unified())
			return conflictPrompt(cmd, res, false)(d) // re-prompt after showing the diff
		default:
			return install.Skip, nil
		}
	}
}

func dp06Target(target string) error {
	switch target {
	case "", "all", "claude", "codex", "opencode", "pi":
		return nil
	}
	return fmt.Errorf("unknown target %q; use claude|codex|opencode|pi|all", target)
}

func dp06ProfileComplete(res *profile.Resolved) error {
	for _, w := range res.Warnings {
		if strings.Contains(w, "not resolvable") {
			return fmt.Errorf("required profile member unavailable: %s", w)
		}
	}
	return nil
}

// Legacy desired pins are not target provenance. Consistent installed rows can
// establish Pi; otherwise regeneration is a separate deliberate lock command.
func dp06CheckProfileLock(wd, profileName, target string, cat *registry.Catalog, home, scope string, out io.Writer) error {
	l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
	if err != nil {
		return err
	}
	if l.Profile != "" && l.Profile != profileName {
		return nil
	}
	if l.Target != "" {
		if l.Target != target {
			return fmt.Errorf("patronus.lock target %s differs from requested %s", l.Target, target)
		}
		return nil
	}
	if target != "pi" || len(l.Entries) == 0 {
		return nil
	}
	res, err := profile.Resolve(cat, profileName, "pi")
	if err != nil {
		return err
	}
	if scope == "" {
		scope, err = dp06DefaultPiScope(cat, res.Names())
		if err != nil {
			return err
		}
	}
	s, err := state.Load(removeStatePath(scope, home, wd))
	if err != nil {
		return err
	}
	proven := false
	for _, it := range s.Items {
		if !contains(res.Names(), it.Artifact) {
			continue
		}
		if it.Tool != "pi" && it.Tool != recipe.TargetAgnostic {
			proven = false
			break
		}
		proven = proven || it.Tool == "pi"
	}
	if proven {
		return nil
	}
	var old []string
	for _, e := range l.Entries {
		old = append(old, e.Name)
	}
	fmt.Fprintf(out, "Pi lock closure preview: pinned=%v selected=%v\n", old, res.Names())
	return fmt.Errorf("target-less lock lacks consistent installed Pi provenance; review closure difference, then deliberately regenerate with lock --profile %s --target pi", profileName)
}

// FETCH planning equality is not replacement authority. Check the original
// owned bytes independently of the selected payload, including agnostic rows.
func dp06AdmitFetches(cs *diff.ChangeSet, owners []state.Item) error {
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Fetch == nil || d.Directory != nil {
			continue
		}
		owned := false
		for _, it := range owners {
			for _, f := range it.Files {
				if f.Path != d.Path {
					continue
				}
				if owned || it.Artifact != d.Artifact || it.Tool != d.Tool || it.Scope != d.Scope || f.Action != string(diff.Fetch) {
					return fmt.Errorf("pi FETCH ownership conflict at %s; explicit migration required", d.Path)
				}
				if d.Before == nil || f.Checksum != fmt.Sprintf("sha256:%x", sha256.Sum256(d.Before)) {
					return fmt.Errorf("pi owned FETCH drift at %s; preserve edits and resolve ownership before replacement", d.Path)
				}
				owned = true
			}
		}
		if !owned && d.Before != nil && (d.Action != diff.Skip || d.Fetch.Archive != "") {
			return fmt.Errorf("pi FETCH destination %s is unmanaged; explicit migration required", d.Path)
		}
		if d.Fetch.Archive != "" && d.Action == diff.Skip {
			// An installed member digest proves neither the selected archive pin
			// nor its member. Authorized apply must acquire and decode that pin.
			d.Action = diff.Fetch
			d.Note = "selected archive pin requires acquisition and member verification"
		}
	}
	return nil
}

func dp06VerifyGlobal(home, version string, d diff.FileDiff) error {
	refuse := func(reason string) error {
		return fmt.Errorf("global prerequisite %s: %s; run a separate global install/update before local selection", d.Artifact, reason)
	}
	s, err := state.Load(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		return err
	}
	items := s.Find(d.Artifact, recipe.TargetAgnostic, "global")
	if len(items) != 1 || items[0].ItemVersion != version {
		return refuse("missing or stale owned state")
	}
	if d.Directory != nil {
		req := directoryRequest(d.Directory)
		in, err := (&packagedelivery.Service{Home: home}).Inspect(req)
		if err != nil {
			return refuse(err.Error())
		}
		r := in.Receipt
		if items[0].PackageReceipt != d.Artifact || r == nil || in.Pending != nil || len(in.Changed)+len(in.Missing)+len(in.Unknown) > 0 {
			return refuse("missing, drifted or unowned receipt")
		}
		if r.RecipeVersion != req.RecipeVersion || r.Identity != req.Identity || r.Root != req.Root || r.URL != req.URL || r.ArchiveSHA256 != req.SHA256 {
			return refuse("incompatible receipt pin")
		}
		return nil
	}
	if d.Fetch != nil && d.Fetch.Archive != "" {
		return fmt.Errorf("global prerequisite %s: legacy archive ownership records only member digest/version, not the selected archive pin; Pi-local reliance requires raw/directory delivery with pin evidence or a separately qualified migration (ordinary global reinstall cannot add this evidence)", d.Artifact)
	}
	if d.Fetch == nil || d.Action != diff.Skip {
		return refuse("missing or incompatible pinned delivery")
	}
	for _, f := range items[0].Files {
		if f.Path == d.Path && f.Action == string(diff.Fetch) && f.Checksum == fmt.Sprintf("sha256:%x", sha256.Sum256(d.Before)) {
			if err := install.CheckUnchanged(d.Path, d.Before); err != nil {
				return refuse(err.Error())
			}
			return nil
		}
	}
	return refuse("missing ownership or changed installed digest")
}

func dp06Acknowledge(cmd *cobra.Command, operation string) {
	fmt.Fprintf(cmd.OutOrStdout(), "Pi %s --deploy acknowledges: settle dependent work (all consumers for global changes); reload/restart is required. Loaded sessions may retain old bytes. Pi/npm exclusively manage native package declarations and files.\n", operation)
}

// No fallback: Pi dry previews can consume only already available source bytes.
type dp06OfflineFetcher struct{}

func (dp06OfflineFetcher) Fetch(context.Context, string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("pi dry preview is network-free; required source is not cached; acquire it in a separately authorized operation")
}

func dp06PrintSelection(cmd *cobra.Command, cs *diff.ChangeSet, target string, res toolpath.Resolver) {
	pi := target == "pi"
	for _, d := range cs.Diffs {
		pi = pi || d.Tool == "pi"
	}
	if !pi {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Pi selection: global root=%s; project root=%s; static resources become eligible at startup/reload, runtime-unverified.\n", res.ResolveMarker("~/.pi/agent", "pi", "global"), res.ResolveMarker(".pi", "pi", "local"))
	for _, d := range cs.Diffs {
		if d.Tool == recipe.TargetAgnostic {
			fmt.Fprintf(cmd.OutOrStdout(), "Global shared dependency effect: %s %s at %s; no independent profile lifetime.\n", d.Action, d.Artifact, d.Path)
		}
	}
}

// Scope inference applies only to Pi with no explicit scope. Agnostic delivery
// rows are prerequisites, not evidence that a local resource selection is global.
func dp06DefaultPiScope(cat *registry.Catalog, names []string) (string, error) {
	selected := ""
	for _, name := range names {
		scope := ""
		for _, a := range cat.Artifacts {
			if a.Manifest.Name == name {
				scope = a.Manifest.Defaults.Scope
				if scope == "" || scope == "project" {
					scope = "local"
				}
			}
		}
		if rec := findRecipe(cat, name); rec != nil && rec.Manifest.Wire.Method == manifest.WireMerge {
			scope = "global"
		}
		if scope == "" {
			continue
		}
		if scope != "global" && scope != "local" {
			return "", fmt.Errorf("invalid Pi default scope %q for %s", scope, name)
		}
		if selected != "" && selected != scope {
			return "", fmt.Errorf("pi selection has mixed local/global resource defaults; select explicit --local or --global, or preview separate operations")
		}
		selected = scope
	}
	if selected == "" {
		selected = "global"
	}
	return selected, nil
}

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// newRemoveCmd is `patronus remove` (alias `revert`): the inverse of install. It
// reads what Patronus recorded in state.json and undoes it on the shared change-set
// spine — delete CREATEs, un-APPEND sections by marker, restore MERGEs to their
// pre-install bytes. Safe by default: a dry run unless --deploy. User edits since
// install are detected by the recorded checksum and skipped unless --force.
func newRemoveCmd(aliases []string) *cobra.Command {
	var (
		tool           string
		global         bool
		local          bool
		deploy         bool
		dryRun         bool
		verbose        bool
		force          bool
		allowPiProject bool
	)

	cmd := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: aliases,
		Short:   "Uninstall tracked item(s) — dry-run by default; --deploy to apply",
		Long: "Undoes a previous install by reading ~/.patronus/state.json (global) and\n" +
			"<project>/.patronus/state.json (local): CREATEd files are deleted, APPENDed\n" +
			"sections are removed by their patronus markers (surrounding prose untouched),\n" +
			"and MERGEd configs are restored to their pre-install bytes.\n\n" +
			"SAFE BY DEFAULT: remove is a dry run unless you pass --deploy. Files edited\n" +
			"since install are detected (via the recorded checksum) and skipped — pass\n" +
			"--force to remove them anyway. Self-wired recipes cannot be auto-reverted and\n" +
			"are reported for manual cleanup.\n\n" +
			"Pi --deploy acknowledges settled dependent work and reload/restart obligations.\n" +
			"Native packages are removed by Pi/npm; internal manual edits are not checked.\n" +
			"Stored profiles select historical concrete effects. Desired lock pins and\n" +
			"unrelated roots/configuration remain unchanged.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := dp06Target(tool); err != nil {
				return err
			}
			if global && local {
				return fmt.Errorf("--global and --local are mutually exclusive")
			}
			if deploy && dryRun {
				return fmt.Errorf("--deploy and --dry-run are mutually exclusive")
			}
			scopeFilter := ""
			switch {
			case global:
				scopeFilter = "global"
			case local:
				scopeFilter = "local"
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			home := homeDir()
			var mutation *mutation
			if deploy && !jsonOutput {
				mutation, err = beginMutation(home, wd)
				if err != nil {
					return err
				}
				defer mutation.close(&err)
			}
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Which scopes' state files to consult. Default = both.
			scopes := []string{"global", "local"}
			if scopeFilter != "" {
				scopes = []string{scopeFilter}
			}

			// Collect the matching state items across the selected scopes, tracking
			// which scope's file each came from so we can rewrite it after a deploy.
			selectedProfiles := map[string]bool{}
			var selected []state.Item
			loaded := map[string]*state.State{}
			anyKnown := map[string]bool{} // name -> seen anywhere
			for _, scope := range scopes {
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
				loaded[scope] = s
				for _, name := range args {
					items := s.Find(name, "", "")
					root := piSelectedRoot(scope, home, wd)
					for _, p := range s.Profiles {
						if p.Name == name && p.Scope == scope && p.Root == root && (tool == "" || tool == "all" || tool == p.Tool) {
							anyKnown[name] = true
							selectedProfiles[p.Name+"\x00"+p.Root] = true
							items = append(items, currentProfileMembers(s, p)...)
						}
					}
					for _, it := range items {
						var ok bool
						it, ok = piProjectItem(it, root)
						if !ok {
							continue
						}
						if tool != "" && tool != "all" && it.Tool != tool && it.PackageReceipt == "" && !(tool == "pi" && it.Tool == recipe.TargetAgnostic && scope == "global") {
							continue
						}
						anyKnown[name] = true
						selected = append(selected, it)
					}
				}
			}

			// Report any requested name that is not installed in the selected scope(s),
			// listing what IS installed so the user can correct the name.
			var unknown []string
			for _, name := range args {
				if !anyKnown[name] {
					unknown = append(unknown, name)
				}
			}
			if len(unknown) > 0 {
				return fmt.Errorf("not installed: %v\n%s", unknown, installedSummary(loaded))
			}

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

			// The sole-contributor gate on pre-compose MERGE rows must see EVERY
			// recorded contributor, not just the selected ones and not just the
			// scopes this command touches: an artifact still wired into a shared
			// config is exactly the sibling a wholesale restore would destroy, and
			// a --global remove would otherwise be blind to a local record naming
			// the same absolute path.
			occupancy, err := fullOccupancy(home, wd, loaded)
			if err != nil {
				return err
			}
			// Multiple names/profiles may select the same concrete effect.
			selected = deduplicateEffects(selected)
			var retainedShared error
			var nativeItems []state.Item
			var packageItems []state.Item
			var legacyItems []state.Item
			for _, item := range selected {
				if item.Native != nil {
					nativeItems = append(nativeItems, item)
				} else if item.PackageReceipt != "" {
					if !force && sharedDirectoryProfile(loaded[item.Scope], item.PackageReceipt, selectedProfiles) {
						retainedShared = errors.Join(retainedShared, fmt.Errorf("package %s retained: shared profile effect (use --force)", item.Artifact))
					} else {
						packageItems = append(packageItems, item)
					}
				} else {
					legacyItems = append(legacyItems, item)
				}
			}
			piSelected := tool == "pi"
			for _, it := range selected {
				piSelected = piSelected || it.Tool == "pi"
			}
			selected = legacyItems
			packagePlans, err := planDirectoryRemovals(home, packageItems, force)
			if err != nil {
				return err
			}
			var removable []state.Item
			var shared []state.Item
			for _, it := range selected {
				ready, held := it, it
				ready.Files, held.Files = nil, nil
				for _, f := range it.Files {
					if !force && sharedProfileEffect(loaded[it.Scope], it, f, selectedProfiles) {
						held.Files = append(held.Files, f)
					} else {
						ready.Files = append(ready.Files, f)
					}
				}
				if len(held.Files) > 0 {
					shared = append(shared, held)
				}
				if len(ready.Files) > 0 || len(it.Files) == 0 {
					removable = append(removable, ready)
				}
			}
			computed, err := remove.ComputeWithForce(removable, read, occupancy, force)
			if err != nil {
				return err
			}
			for _, it := range shared {
				for _, f := range it.Files {
					current, _, err := read(f.Path)
					if err != nil {
						return err
					}
					computed.ChangeSet.Diffs = append(computed.ChangeSet.Diffs, diff.FileDiff{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope, Path: f.Path, Action: diff.Skip, Before: current, Note: "shared profile effect retained (use --force)"})
					computed.Ledger = append(computed.Ledger, remove.LedgerEntry{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope, Path: f.Path, Effect: f, Outcome: remove.DriftSkipped})
				}
			}
			cs, warnings, ledger := computed.ChangeSet, computed.Warnings, computed.Ledger

			// Symmetric plugin teardown: for any selected item that is a tracked
			// plugin, append the tool's uninstall EXEC(s) (advisory when its CLI is
			// absent). The v1 orphan `plugins.<name>` MERGE, if any, is already
			// reverted by remove.Compute's Prior-restore path — no extra code.
			if pluginDiffs := pluginRemoveDiffs(cmd, wd, selected, warnf); len(pluginDiffs) > 0 {
				cs.Diffs = append(cs.Diffs, pluginDiffs...)
			}

			if force {
				computed = remove.Promote(computed)
				cs, warnings, ledger = computed.ChangeSet, computed.Warnings, computed.Ledger
			}
			if retainedShared != nil {
				warnf("%v", retainedShared)
			}
			for _, w := range warnings {
				if w.Path != "" {
					warnf("%s (%s): %s", w.Item, w.Path, w.Message)
				} else {
					warnf("%s: %s", w.Item, w.Message)
				}
			}

			for _, it := range nativeItems {
				op := it.Native.Operation
				op.Kind = "remove"
				// A profile reference is identity, not a stale authority snapshot.
				if index, err := nativeItem(loaded[it.Scope], op.Identity); err != nil {
					return err
				} else if index >= 0 {
					op.Source = loaded[it.Scope].Items[index].Native.Operation.Source
				}
				// Ownership remains root-qualified, while invocation context is always
				// refreshed from this removal rather than stale install-time values.
				op.Identity.Project = wd
				if op.Identity.Scope == "local" {
					op.Identity.AgentRoot = piSelectedRoot("global", home, wd)
				}
				cs.Diffs = append(cs.Diffs, diff.FileDiff{Action: diff.Native, Artifact: it.Artifact, Tool: "pi", Scope: it.Scope, Root: it.Root, Path: op.Identity.MetadataPath(), Native: &op})
			}
			if err := inspectNative(cs, home, wd, force, selectedProfiles); err != nil {
				return err
			}
			cs.DryRun = !deploy

			env := os.LookupEnv
			res := toolpath.New(env, home, wd)
			if err := codexPreflightPlan(cs, res, home, wd); err != nil {
				return err
			}
			if jsonOutput {
				if len(packagePlans) == 0 {
					return render.JSON(cmd.OutOrStdout(), cs)
				}
				return render.JSON(cmd.OutOrStdout(), struct {
					*diff.ChangeSet
					Packages []directoryRemovalPlan `json:"packages"`
				}{cs, packagePlans})
			}
			dp06PrintSelection(cmd, cs, tool, res)
			render.PrintPlan(cmd.OutOrStdout(), cs, res, verbose)
			for _, plan := range packagePlans {
				fmt.Fprintf(cmd.OutOrStdout(), "Package %s: delete %v; retain %v; unknown leftovers %v; pending recovery %t\n", plan.Recipe, plan.Delete, plan.Retain, plan.Leftovers, plan.Pending)
			}

			if !deploy {
				return nil
			}
			if piSelected {
				dp06Acknowledge(cmd, "remove")
				for _, p := range packagePlans {
					if !force && (len(p.Retain) > 0 || len(p.Leftovers) > 0) {
						return fmt.Errorf("pi whole-selection removal conflict: package %s has retained or unknown files; resolve before removal", p.Recipe)
					}
				}
			}
			if err := staticPiSelection(cs, piSelected); err != nil {
				return err
			}
			if err := mutation.checkPlan(cs, nil); err != nil {
				return err
			}
			if err := deployDirectoryRemovalsLocked(cmd, home, packagePlans, force, mutation); err != nil {
				return err
			}
			if len(packagePlans) > 0 {
				refreshed, err := state.Load(removeStatePath("global", home, wd))
				if err != nil {
					return err
				}
				loaded["global"] = refreshed
			}
			return runRemoveLocked(cmd, cs, ledger, selected, loaded, removeStateOpts{retained: retainedShared, allowPiProject: allowPiProject, profiles: selectedProfiles, mutation: mutation, home: home, projectDir: wd, force: force})
		},
	}

	cmd.Flags().StringVar(&tool, "target", "", "limit to a target runtime: claude|codex|opencode|pi|all (default: installed provenance)")
	cmd.Flags().BoolVar(&global, "global", false, "limit to global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "limit to project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually undo the changes on disk (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "also show per-item unified diffs")
	cmd.Flags().BoolVar(&allowPiProject, "allow-pi-project-config", false, "authorize native Pi project configuration/trust")
	cmd.Flags().BoolVar(&force, "force", false, "with --deploy: undo files edited since install (overrides drift skips)")
	return cmd
}

// pluginRemoveDiffs builds the uninstall EXEC diffs for any selected item that is
// a tracked plugin, grouping the recorded (tool,scope) items under each plugin's
// manifest. It loads the catalog to resolve each plugin's source/ecosystem; if the
// catalog is unavailable, no plugin is a known plugin here and it returns nil
// (the file-revert path still runs). It never fails the remove.
func pluginRemoveDiffs(cmd *cobra.Command, wd string, selected []state.Item, warnf func(string, ...any)) []diff.FileDiff {
	cat := scanCatalogFn(cmd.Context(), wd, warnf)
	if cat == nil {
		return nil
	}
	// Group recorded items by plugin name so one plugin's per-tool items build one
	// uninstall pass. Non-plugin items (findPlugin==nil) are left to remove.Compute.
	byPlugin := map[string][]state.Item{}
	for _, it := range selected {
		if findPlugin(cat, it.Artifact) != nil {
			byPlugin[it.Artifact] = append(byPlugin[it.Artifact], it)
		}
	}
	if len(byPlugin) == 0 {
		return nil
	}
	probe := plugin.ExecProbe{}
	var out []diff.FileDiff
	for name, items := range byPlugin {
		pl := findPlugin(cat, name)
		out = append(out, pluginUninstallDiffs(pl.Manifest, items, probe)...)
	}
	return out
}

// fullOccupancy builds the contributor index from BOTH scopes' state files,
// reusing whatever this command already loaded and reading the rest. A scope
// filter narrows what gets REMOVED; it must not narrow what the safety gate can
// see, or a --global remove would restore a snapshot over a local record's
// wiring simply because it never looked.
func fullOccupancy(home, projectDir string, loaded map[string]*state.State) (remove.Occupancy, error) {
	states := make([]*state.State, 0, 2)
	for _, scope := range []string{"global", "local"} {
		if s, ok := loaded[scope]; ok {
			states = append(states, s)
			continue
		}
		s, err := state.Load(removeStatePath(scope, home, projectDir))
		if err != nil {
			return nil, fmt.Errorf("load %s state: %w", scope, err)
		}
		states = append(states, s)
	}
	return occupancyOf(states), nil
}

// occupancyOf indexes recorded MERGE and APPEND rows across the states by path,
// listing the contributors wired into each. remove.Compute uses it to tell a
// config file this record owns alone from one it shares, which is what makes a
// pre-compose whole-file restore safe or unsafe.
//
// Recorded paths are absolute, so a row from any scope's state file counts:
// what the gate protects is the FILE, and a contributor recorded in the scope
// this command did not load is exactly the one the user cannot see coming.
func occupancyOf(states []*state.State) remove.Occupancy {
	occ := remove.Occupancy{}
	for _, s := range states {
		if s == nil {
			continue
		}
		for _, it := range s.Items {
			// State identity is (artifact, tool, scope): the same artifact installed
			// for two tools is two independent records, and one must not vouch for
			// the other's contribution to a shared file.
			c := remove.Contributor{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope}
			for _, f := range it.Files {
				if f.Action != string(diff.Merge) && f.Action != string(diff.Append) {
					continue
				}
				c.Setting = f.Setting
				c.Section = f.Section
				occ[f.Path] = append(occ[f.Path], c)
			}
		}
	}
	return occ
}

// removeStateOpts carries what runRemove needs to rewrite state after an undo.
type removeStateOpts struct {
	retained       error
	allowPiProject bool
	profiles       map[string]bool
	mutation       *mutation
	saveState      func(string, *state.State) error // nil uses atomic state.Save
	home           string
	projectDir     string
	force          bool
}

// runRemove applies the inverse change set and, on success, drops the fully-undone
// items from their scope's state file. It mirrors runDeploy's structure: apply via
// the shared install.Applier (no EXEC — undo has none), then persist state to match
// the new reality. An item is dropped from state only when every one of its files
// was actually undone (not skipped as drift); a partially-skipped item stays so a
// later --force can finish it.
func runRemove(cmd *cobra.Command, cs *diff.ChangeSet, ledger remove.Ledger, selected []state.Item, loaded map[string]*state.State, opts removeStateOpts) (err error) {
	m, err := beginMutation(opts.home, opts.projectDir)
	if err != nil {
		return err
	}
	defer m.close(&err)
	opts.mutation = m
	return runRemoveLocked(cmd, cs, ledger, selected, loaded, opts)
}

func runRemoveLocked(cmd *cobra.Command, cs *diff.ChangeSet, ledger remove.Ledger, selected []state.Item, loaded map[string]*state.State, opts removeStateOpts) error {
	if err := codexPreflightPlan(cs, toolpath.New(os.LookupEnv, opts.home, opts.projectDir), opts.home, opts.projectDir); err != nil {
		return err
	}
	piSelected := false
	for _, item := range selected {
		piSelected = piSelected || item.Tool == "pi"
	}
	if err := staticPiSelection(cs, piSelected); err != nil {
		return err
	}
	if err := opts.mutation.checkPlan(cs, nil); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	nativeErr := opts.retained
	var authored, natives []diff.FileDiff
	changedSettings := map[string]bool{}
	for _, d := range cs.Diffs {
		if d.Native == nil {
			authored = append(authored, d)
			continue
		}
		natives = append(natives, d)
		changedSettings[d.Native.Identity.SettingsPath()] = true
	}
	// Native policy and runtime probes are a whole-selection admission barrier.
	// In particular, do not let an early singleton removal or authored revert land
	// before a later selected native member fails admission.
	if len(natives) > 0 {
		if err := applyNative(cmd, &diff.ChangeSet{Diffs: natives}, deployOptions{mutation: opts.mutation, saveState: opts.saveState, home: opts.home, projectDir: opts.projectDir, force: opts.force, allowPiProject: opts.allowPiProject}, opts.profiles); err != nil {
			return errors.Join(nativeErr, err)
		}
	}
	cs = &diff.ChangeSet{Diffs: authored}
	if len(changedSettings) > 0 {
		for scope := range loaded {
			s, err := state.Load(removeStatePath(scope, opts.home, opts.projectDir))
			if err != nil {
				return errors.Join(nativeErr, err)
			}
			loaded[scope] = s
		}
		// Recompute only selected authored settings after the manager changed its
		// packages array. Sharing gates still apply to the concrete effects.
		var settings []state.Item
		for _, it := range selected {
			row := it
			row.Files = nil
			for _, f := range it.Files {
				if changedSettings[f.Path] && (opts.force || !sharedProfileEffect(loaded[it.Scope], it, f, opts.profiles)) {
					row.Files = append(row.Files, f)
				}
			}
			if len(row.Files) > 0 {
				settings = append(settings, row)
			}
		}
		if len(settings) > 0 {
			fresh, err := remove.ComputeWithForce(settings, func(path string) ([]byte, bool, error) {
				b, e := os.ReadFile(path)
				if os.IsNotExist(e) {
					return nil, false, nil
				}
				return b, e == nil, e
			}, occupancyOf([]*state.State{loaded["global"], loaded["local"]}), opts.force)
			if err != nil {
				return errors.Join(nativeErr, err)
			}
			if opts.force {
				fresh = remove.Promote(fresh)
			}
			var keep []diff.FileDiff
			for _, d := range cs.Diffs {
				if !changedSettings[d.Path] {
					keep = append(keep, d)
				}
			}
			cs.Diffs = append(keep, fresh.ChangeSet.Diffs...)
			var keptLedger remove.Ledger
			for _, e := range ledger {
				if !changedSettings[e.Path] {
					keptLedger = append(keptLedger, e)
				}
			}
			ledger = append(keptLedger, fresh.Ledger...)
		}
	}

	app := &install.Applier{BeforeWrite: func(d diff.FileDiff) error { return opts.mutation.checkFile(d, nil) }}
	result, applyErr := app.Apply(cs)
	applyErr = errors.Join(nativeErr, applyErr)
	var ranExecs []diff.FileDiff

	// Run plugin uninstall EXECs (the applier skips EXEC diffs — it stays a pure
	// file writer). Only after the file reverts succeed, mirroring runDeploy. An
	// advisory exec (CLI absent) is shown, not run. A failure is surfaced but does
	// not block dropping the file-reverted state below.
	if applyErr == nil {
		runner := runnerForCommands
		if runner == nil {
			runner = execRunner{cmd: cmd}
		}
		// remove never installs packages: a no-install consent (yes, not allow) keeps
		// every package-install advisory surface-only.
		consent := installConsent{yes: true, look: exec.LookPath, out: cmd.OutOrStdout()}
		var execErr error
		ranExecs, execErr = runExecs(cmd, cs, runner, consent)
		if execErr != nil {
			applyErr = execErr
		}
	}

	// Determine which (artifact,tool,scope) items were fully undone. Completion is
	// keyed by ARTIFACT IDENTITY, not by path: composition folds several artifacts'
	// contributions into one physical write, so a landed path no longer identifies
	// who was undone. The ledger answers that per contributor; the applier's
	// Applied set confirms the write it predicted actually happened.
	verifiedSkippedPaths := map[string]bool{}
	for _, d := range result.Skipped {
		verifiedSkippedPaths[d.Tool+"\x00"+d.Scope+"\x00"+d.Path] = true
	}
	writtenPaths := map[string]bool{}
	for _, d := range result.Applied {
		writtenPaths[d.Tool+"\x00"+d.Scope+"\x00"+d.Path] = true
	}
	// One artifact can record SEVERAL edits on one path — an OpenCode gate whose
	// matcher maps to more than one permission key is the standing case — so the
	// identity tuple is not unique per ledger row. Accumulate with AND: every
	// outcome on the tuple must be settled, or the row stays open. Overwriting
	// instead would let one landed edit vouch for a refused sibling, retiring an
	// artifact whose wiring is still on disk.
	undone := map[string]bool{} // artifact+tool+scope+path -> EVERY contribution there is settled
	for _, e := range ledger {
		k := removalEffectKey(e.Artifact, e.Tool, e.Scope, e.Effect)
		if e.Effect.Path == "" {
			k = removalEffectKey(e.Artifact, e.Tool, e.Scope, state.FileState{Path: e.Path})
		}
		settled := e.Outcome.Complete() && verifiedSkippedPaths[e.Tool+"\x00"+e.Scope+"\x00"+e.Path]
		if e.Outcome == remove.Applied {
			// Predicted to be written; credit it only if the write actually landed.
			settled = writtenPaths[e.Tool+"\x00"+e.Scope+"\x00"+e.Path]
		}
		if prev, seen := undone[k]; seen {
			settled = settled && prev
		}
		undone[k] = settled
	}

	dirty := map[string]bool{} // scopes whose state file changed
	for _, it := range selected {
		fullyUndone := true
		for _, f := range it.Files {
			if !undone[removalEffectKey(it.Artifact, it.Tool, it.Scope, f)] {
				fullyUndone = false
				break
			}
		}
		// A self-wired recipe with no files is never "removed" — its wiring can't be
		// auto-reverted, so we leave its record for manual cleanup. When Patronus
		// installed a package for it (with consent), surface the manual-uninstall
		// reminder: global-ish package state may be shared, so we never auto-uninstall.
		if it.SelfWired && len(it.Files) == 0 {
			fullyUndone = false
			surfaceUninstallAdvisory(out, it)
		}
		var settledFiles []state.FileState
		for _, f := range it.Files {
			if undone[removalEffectKey(it.Artifact, it.Tool, it.Scope, f)] {
				settledFiles = append(settledFiles, f)
			}
		}
		if !fullyUndone && len(settledFiles) > 0 && (it.Tool == "pi" || it.Tool == "codex") {
			state.ForgetEffects(loaded[it.Scope], it, settledFiles)
			dirty[it.Scope] = true
		}
		if !fullyUndone {
			shared := false
			for _, f := range it.Files {
				shared = shared || sharedProfileEffect(loaded[it.Scope], it, f, opts.profiles)
			}
			if it.Tool == "pi" || it.Tool == "codex" || shared {
				applyErr = errors.Join(applyErr, fmt.Errorf("removal conflict unresolved: %s (%s/%s); ownership retained", it.Artifact, it.Tool, it.Scope))
			}
			continue
		}
		// Every tracked deletion for this item is settled, so a directory-shaped
		// artifact's now-empty tree can be pruned. This is the only point that sees
		// at once the full state.Item, which deletes actually landed, and whether
		// the removal was complete — Compute runs before Apply and cannot know, and
		// the applier is deliberately a per-diff writer with no artifact grouping.
		//
		// A prune failure that is NOT "directory not empty" keeps the state row:
		// retiring it while an owned directory survives would make Patronus forget
		// a directory it owns and failed to clean.
		pruneWarnings, err := remove.Prune(it)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			if applyErr == nil {
				applyErr = err
			}
			continue
		}
		for _, w := range pruneWarnings {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s (%s): %s\n", w.Item, w.Path, w.Message)
		}
		if s := loaded[it.Scope]; s != nil {
			if it.Tool == "pi" || it.Tool == "codex" {
				state.ForgetEffects(s, it, it.Files)
			} else {
				s.Remove(it.Artifact, it.Tool, it.Scope)
			}
			dirty[it.Scope] = true
		}
	}

	for scope, s := range loaded {
		if state.RefreshSectionChecksums(s.Items, result.Applied) {
			dirty[scope] = true
		}
		if len(opts.profiles) > 0 {
			settleProfiles(s, opts.profiles)
			dirty[scope] = true
		}
	}
	// Persist the trimmed state files (only those that changed).
	for _, scope := range []string{"global", "local"} {
		if !dirty[scope] {
			continue
		}
		sp := removeStatePath(scope, opts.home, opts.projectDir)
		save := opts.saveState
		if save == nil {
			save = state.Save
		}
		if err := opts.mutation.saveState(sp, loaded[scope], save); err != nil {
			applyErr = errors.Join(applyErr, fmt.Errorf("save %s state: %w; ownership uncertain; %s; re-preview before repair", sp, err, resultDiagnostics(result)))
			break
		}
	}

	// Report LOGICAL contributions, not physical writes. Several artifacts can be
	// reversed by one composed write, and counting the writes would say "1 undone"
	// after removing three — under-reporting the change beneath a table that
	// already shows a row per contributor. Install hit this same trap on its
	// composed MERGE footer and resolved it the same way.
	//
	// Count from the settled-contribution view rather than the applier's own
	// tally, which answers a different question: a file that was already gone
	// needs no write and lands in Skipped, yet its removal is done, and reporting
	// it as skipped would tell the user work remains when none does.
	undoneCount, skippedCount := 0, 0
	for _, settled := range undone {
		if settled {
			undoneCount++
			continue
		}
		skippedCount++
	}
	// A plugin's uninstall command is a real contribution with no recorded file
	// behind it, so it has no ledger entry. Count what actually ran, or removing a
	// file-less plugin would report "0 undone" after doing the work.
	undoneCount += len(ranExecs)
	fmt.Fprintf(out, "\nRemoved: %d undone, %d skipped\n", undoneCount, skippedCount)
	if applyErr != nil {
		return fmt.Errorf("%w; %s; re-preview current bytes before repair", applyErr, resultDiagnostics(result))
	}
	return nil
}

// surfaceUninstallAdvisory prints a manual-uninstall reminder for a package-install
// item. Patronus installed the package (with consent) but does NOT auto-uninstall —
// global-ish package state may be shared. It surfaces each recorded install command
// so the user can reverse it deliberately.
func surfaceUninstallAdvisory(out io.Writer, it state.Item) {
	for _, cmd := range it.PostInstall {
		fmt.Fprintf(out, "ADVISORY (uninstall yourself): package for %q was installed via `%s` — remove it manually if unused\n", it.Artifact, cmd)
	}
}

// removeStatePath returns the state file for a scope (mirrors install's statePath
// but takes home/projectDir directly so remove has no dependency on deployOptions).
func removeStatePath(scope, home, projectDir string) string {
	if scope == "global" {
		return filepath.Join(home, ".patronus", "state.json")
	}
	return filepath.Join(projectDir, ".patronus", "state.json")
}

// installedSummary lists what is currently recorded across the loaded scopes, for
// a helpful "not installed" error.
func installedSummary(loaded map[string]*state.State) string {
	names := map[string]bool{}
	for _, s := range loaded {
		for _, it := range s.Items {
			names[it.Artifact] = true
		}
	}
	if len(names) == 0 {
		return "nothing is currently installed (no state recorded)"
	}
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	sort.Strings(list)
	return "installed: " + strings.Join(list, ", ")
}

// directoryRemovalPlan never passes package identities to legacy fileUndo.
type directoryRemovalPlan struct {
	Recipe    string   `json:"recipe"`
	Delete    []string `json:"delete"`
	Retain    []string `json:"retain"`
	Leftovers []string `json:"leftovers"`
	Pending   bool     `json:"pending"`
}

func planDirectoryRemovals(home string, items []state.Item, force bool) ([]directoryRemovalPlan, error) {
	var plans []directoryRemovalPlan
	seen := map[string]bool{}
	service := directoryServiceForDeploy(home)
	for _, item := range items {
		name := item.PackageReceipt
		if item.Artifact != name || item.Scope != "global" {
			return nil, fmt.Errorf("package %s: invalid discovery reference %q in %s scope", item.Artifact, name, item.Scope)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		receipt, err := packagestate.Load(home, name)
		if err != nil {
			return nil, err
		}
		tx, err := packagestate.ReadTransaction(home, name)
		if err != nil {
			return nil, err
		}
		p := directoryRemovalPlan{Recipe: name, Pending: tx != nil}
		if tx != nil {
			plans = append(plans, p)
			continue
		}
		if receipt == nil {
			return nil, fmt.Errorf("package %s: discovery reference has no receipt; no deletion authorized", name)
		}
		in, err := service.Inspect(packagedelivery.Request{Recipe: name, RecipeVersion: receipt.RecipeVersion, Root: receipt.Root, URL: receipt.URL, SHA256: receipt.ArchiveSHA256, Identity: receipt.Identity})
		if err != nil {
			return nil, err
		}
		changed := map[string]bool{}
		for _, path := range in.Changed {
			changed[path] = true
		}
		for _, entry := range receipt.Files {
			if changed[entry.Path] && !force {
				p.Retain = append(p.Retain, entry.Path)
			} else {
				p.Delete = append(p.Delete, entry.Path)
			}
		}
		p.Leftovers = in.Unknown
		plans = append(plans, p)
	}
	return plans, nil
}

func deployDirectoryRemovals(cmd *cobra.Command, home string, plans []directoryRemovalPlan, force bool) (err error) {
	m, err := acquireMutation(home, filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		return err
	}
	defer m.close(&err)
	return deployDirectoryRemovalsLocked(cmd, home, plans, force, m)
}

func deployDirectoryRemovalsLocked(cmd *cobra.Command, home string, plans []directoryRemovalPlan, force bool, m *mutation) (err error) {
	if len(plans) == 0 {
		return nil
	}
	service := directoryServiceForDeploy(home)
	for _, plan := range plans {
		// A prior removal may already have deleted its receipt. Reconcile it first,
		// then only start a fresh removal if authoritative ownership still exists.
		if err := recoverDirectory(cmd.Context(), service, plan.Recipe, m); err != nil {
			return err
		}
		receipt, err := packagestate.Load(home, plan.Recipe)
		if err != nil {
			return err
		}
		if receipt == nil {
			continue
		}
		result, err := service.Remove(cmd.Context(), plan.Recipe, force)
		if err != nil {
			return directoryDiagnostic(err)
		}
		if err := recoverDirectory(cmd.Context(), service, plan.Recipe, m); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Package %s removed; retained owned paths: %v; unowned leftovers: %v\n", plan.Recipe, result.Retained, result.Leftovers)
	}
	return nil
}

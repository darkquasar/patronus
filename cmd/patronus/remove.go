package main

import (
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
	"github.com/darkquasar/patronus/internal/plugin"
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
func newRemoveCmd(use string, aliases []string) *cobra.Command {
	var (
		tool    string
		global  bool
		local   bool
		deploy  bool
		dryRun  bool
		verbose bool
		force   bool
	)

	cmd := &cobra.Command{
		Use:     use + " <name>...",
		Aliases: aliases,
		Short:   "Uninstall tracked item(s) — dry-run by default; --deploy to apply",
		Long: "Undoes a previous install by reading ~/.patronus/state.json (global) and\n" +
			"<project>/.patronus/state.json (local): CREATEd files are deleted, APPENDed\n" +
			"sections are removed by their patronus markers (surrounding prose untouched),\n" +
			"and MERGEd configs are restored to their pre-install bytes.\n\n" +
			"SAFE BY DEFAULT: remove is a dry run unless you pass --deploy. Files edited\n" +
			"since install are detected (via the recorded checksum) and skipped — pass\n" +
			"--force to remove them anyway. Self-wired recipes cannot be auto-reverted and\n" +
			"are reported for manual cleanup.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Which scopes' state files to consult. Default = both.
			scopes := []string{"global", "local"}
			if scopeFilter != "" {
				scopes = []string{scopeFilter}
			}

			// Collect the matching state items across the selected scopes, tracking
			// which scope's file each came from so we can rewrite it after a deploy.
			var selected []state.Item
			loaded := map[string]*state.State{}
			anyKnown := map[string]bool{} // name -> seen anywhere
			for _, scope := range scopes {
				sp := removeStatePath(scope, home, wd)
				s, err := state.Load(sp)
				if err != nil {
					return fmt.Errorf("load %s state: %w", scope, err)
				}
				loaded[scope] = s
				for _, name := range args {
					items := s.Find(name, tool, "")
					for _, it := range items {
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
			computed, err := remove.Compute(selected, read, occupancy)
			if err != nil {
				return err
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
				cs, ledger = computed.ChangeSet, computed.Ledger
			}
			for _, w := range warnings {
				if w.Path != "" {
					warnf("%s (%s): %s", w.Item, w.Path, w.Message)
				} else {
					warnf("%s: %s", w.Item, w.Message)
				}
			}

			cs.DryRun = !deploy

			env := os.LookupEnv
			res := toolpath.New(env, home, wd)
			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), cs)
			}
			render.PrintPlan(cmd.OutOrStdout(), cs, res, verbose)

			if !deploy {
				return nil
			}
			return runRemove(cmd, cs, ledger, selected, loaded, removeStateOpts{home: home, projectDir: wd, force: force})
		},
	}

	cmd.Flags().StringVar(&tool, "target", "", "limit to a target runtime: claude|codex|opencode (default: all)")
	cmd.Flags().BoolVar(&global, "global", false, "limit to global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "limit to project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually undo the changes on disk (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "also show per-item unified diffs")
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

// occupancyOf indexes every recorded MERGE row across the given states by path,
// listing the contributors wired into each. remove.Compute uses it to tell a
// config file this record owns alone from one it shares, which is what makes a
// pre-compose whole-file restore safe or unsafe.
//
// Recorded paths are absolute, so a row from any scope's state file counts:
// what the gate protects is the FILE, and a contributor recorded in the scope
// this command did not load is exactly the one the user cannot see coming.
func occupancyOf(states []*state.State) remove.Occupancy {
	occ := remove.Occupancy{}
	seen := map[remove.Contributor]map[string]bool{}
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
				if f.Action != string(diff.Merge) {
					continue
				}
				if seen[c] == nil {
					seen[c] = map[string]bool{}
				}
				if seen[c][f.Path] {
					continue // one record may hold several edits to one file
				}
				seen[c][f.Path] = true
				occ[f.Path] = append(occ[f.Path], c)
			}
		}
	}
	return occ
}

// removeStateOpts carries what runRemove needs to rewrite state after an undo.
type removeStateOpts struct {
	home       string
	projectDir string
	force      bool
}

// runRemove applies the inverse change set and, on success, drops the fully-undone
// items from their scope's state file. It mirrors runDeploy's structure: apply via
// the shared install.Applier (no EXEC — undo has none), then persist state to match
// the new reality. An item is dropped from state only when every one of its files
// was actually undone (not skipped as drift); a partially-skipped item stays so a
// later --force can finish it.
func runRemove(cmd *cobra.Command, cs *diff.ChangeSet, ledger remove.Ledger, selected []state.Item, loaded map[string]*state.State, opts removeStateOpts) error {
	out := cmd.OutOrStdout()

	app := &install.Applier{}
	result, applyErr := app.Apply(cs)

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
		if _, execErr := runExecs(cmd, cs, runner, consent); execErr != nil {
			applyErr = execErr
		}
	}

	// Determine which (artifact,tool,scope) items were fully undone. Completion is
	// keyed by ARTIFACT IDENTITY, not by path: composition folds several artifacts'
	// contributions into one physical write, so a landed path no longer identifies
	// who was undone. The ledger answers that per contributor; the applier's
	// Applied set confirms the write it predicted actually happened.
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
		k := e.Artifact + "\x00" + e.Tool + "\x00" + e.Scope + "\x00" + e.Path
		settled := e.Outcome.Complete()
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
			if !undone[it.Artifact+"\x00"+it.Tool+"\x00"+it.Scope+"\x00"+f.Path] {
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
		if fullyUndone {
			if s := loaded[it.Scope]; s != nil {
				s.Remove(it.Artifact, it.Tool, it.Scope)
				dirty[it.Scope] = true
			}
		}
	}

	// Persist the trimmed state files (only those that changed).
	for scope := range dirty {
		sp := removeStatePath(scope, opts.home, opts.projectDir)
		if err := state.Save(sp, loaded[scope]); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to update %s state: %v\n", scope, err)
		}
	}

	// Report LOGICAL contributions, not physical writes. Several artifacts can be
	// reversed by one composed write, and counting the writes would say "1 undone"
	// after removing three — under-reporting the change beneath a table that
	// already shows a row per contributor. Install hit this same trap on its
	// composed MERGE footer and resolved it the same way.
	undoneCount, skippedCount := 0, 0
	for _, d := range result.Applied {
		undoneCount += 1 + len(d.RestoreContrib)
	}
	for _, d := range result.Skipped {
		skippedCount += 1 + len(d.RestoreContrib)
	}
	fmt.Fprintf(out, "\nRemoved: %d undone, %d skipped\n", undoneCount, skippedCount)
	return applyErr
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

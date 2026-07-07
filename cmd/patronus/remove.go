package main

import (
	"fmt"
	"os"
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

			cs, warnings, err := remove.Compute(selected, read)
			if err != nil {
				return err
			}

			// Symmetric plugin teardown: for any selected item that is a tracked
			// plugin, append the tool's uninstall EXEC(s) (advisory when its CLI is
			// absent). The v1 orphan `plugins.<name>` MERGE, if any, is already
			// reverted by remove.Compute's Prior-restore path — no extra code.
			if pluginDiffs := pluginRemoveDiffs(cmd, wd, selected, warnf); len(pluginDiffs) > 0 {
				cs.Diffs = append(cs.Diffs, pluginDiffs...)
			}

			if force {
				remove.Promote(cs)
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
			return runRemove(cmd, cs, selected, loaded, removeStateOpts{home: home, projectDir: wd, force: force})
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "", "limit to a target tool: claude|codex|opencode (default: all)")
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
func runRemove(cmd *cobra.Command, cs *diff.ChangeSet, selected []state.Item, loaded map[string]*state.State, opts removeStateOpts) error {
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
		if _, execErr := runExecs(cmd, cs, runner); execErr != nil {
			applyErr = execErr
		}
	}

	// Determine which (artifact,tool,scope) items were fully undone: every recorded
	// file for that item must appear in result.Applied (not skipped). We key applied
	// paths by the same identity the diffs carry.
	appliedPaths := map[string]bool{}
	for _, d := range result.Applied {
		appliedPaths[d.Tool+"\x00"+d.Scope+"\x00"+d.Path] = true
	}

	dirty := map[string]bool{} // scopes whose state file changed
	for _, it := range selected {
		fullyUndone := true
		for _, f := range it.Files {
			if !appliedPaths[it.Tool+"\x00"+it.Scope+"\x00"+f.Path] {
				fullyUndone = false
				break
			}
		}
		// A self-wired recipe with no files is never "removed" — its wiring can't be
		// auto-reverted, so we leave its record for manual cleanup.
		if it.SelfWired && len(it.Files) == 0 {
			fullyUndone = false
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

	fmt.Fprintf(out, "\nRemoved: %d undone, %d skipped\n", len(result.Applied), len(result.Skipped))
	return applyErr
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

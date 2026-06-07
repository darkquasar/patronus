package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func newInstallCmd() *cobra.Command {
	var (
		tool    string
		global  bool
		local   bool
		deploy  bool
		dryRun  bool
		verbose bool
		force   bool
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "install <name>...",
		Short: "Plan installation of artifact(s) — dry-run by default; --deploy to write",
		Long: "Computes the exact set of file changes installing one or more artifacts would make,\n" +
			"for each target tool and scope, and renders them as an ASCII tree, an artifact-centric\n" +
			"summary table, and (with --verbose) per-artifact unified diffs.\n\n" +
			"SAFE BY DEFAULT: install is a dry run unless you pass --deploy. The absence of --deploy\n" +
			"(or an explicit --dry-run) means nothing is written to disk.\n\n" +
			"NOTE: the apply path is not yet implemented (Phase 3), so --deploy currently refuses to\n" +
			"run rather than writing a partial result.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if global && local {
				return fmt.Errorf("--global and --local are mutually exclusive")
			}
			if deploy && dryRun {
				return fmt.Errorf("--deploy and --dry-run are mutually exclusive")
			}
			scope := ""
			switch {
			case global:
				scope = "global"
			case local:
				scope = "local"
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}

			adapters, err := loadAdapters(filepath.Join(root, "adapters"))
			if err != nil {
				return err
			}

			reg := registry.NewLocalRegistry(root)
			cat, err := reg.Catalog(cmd.Context())
			if err != nil {
				return err
			}

			inv, err := scan.Scan(scan.Options{ProjectDir: wd, Adapters: adapters})
			if err != nil {
				return err
			}

			env := os.LookupEnv
			res := toolpath.New(env, toolpath.HomeDir(env), wd)

			cs, err := plan.Compute(plan.Request{
				Catalog:   cat,
				Inventory: inv,
				Adapters:  adapterMap(adapters),
				Resolver:  res,
				Names:     args,
				Tool:      tool,
				Scope:     scope,
			})
			if err != nil {
				return err
			}

			// DryRun drives the footer wording; only a real --deploy writes.
			cs.DryRun = !deploy

			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), cs)
			}
			render.PrintPlan(cmd.OutOrStdout(), cs, res, verbose)

			// Without --deploy this is a safe dry run: plan shown, nothing written.
			if !deploy {
				return nil
			}
			return runDeploy(cmd, cs, res, deployOptions{force: force, yes: yes, home: toolpath.HomeDir(env), projectDir: wd})
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "all", "target tool: claude|codex|opencode|all")
	cmd.Flags().BoolVar(&global, "global", false, "install at global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "install at project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually write changes to disk (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "also show per-artifact unified diffs")
	cmd.Flags().BoolVar(&force, "force", false, "with --deploy: overwrite conflicting files without prompting")
	cmd.Flags().BoolVar(&yes, "yes", false, "with --deploy: assume non-interactive (conflicts are skipped, never overwritten)")
	return cmd
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
	force      bool
	yes        bool
	home       string // for ~/.patronus/state.json
	projectDir string // for <project>/.patronus/state.json
}

// runDeploy writes the change set to disk and records what was installed. It is
// Terraform-style: a mid-apply failure stops, records the successful writes in
// state, and returns the error.
func runDeploy(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions) error {
	out := cmd.OutOrStdout()

	app := &install.Applier{
		Force:    opts.force,
		Conflict: conflictPrompt(cmd, res, opts.yes),
	}
	result, applyErr := app.Apply(cs)

	// Record whatever succeeded BEFORE surfacing any error (state must reflect
	// reality even on partial failure).
	if stateErr := recordState(result.Applied, opts); stateErr != nil {
		// A state-write failure shouldn't mask an apply failure, but report it.
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to write state: %v\n", stateErr)
	}

	fmt.Fprintf(out, "\nApplied: %d written, %d skipped\n", len(result.Applied), len(result.Skipped))
	if applyErr != nil {
		return applyErr
	}
	return nil
}

// recordState groups applied diffs by scope and upserts them into the matching
// scope's state file (~/.patronus for global, <project>/.patronus for local).
func recordState(applied []diff.FileDiff, opts deployOptions) error {
	if len(applied) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)

	byScope := map[string][]diff.FileDiff{}
	for _, d := range applied {
		byScope[d.Scope] = append(byScope[d.Scope], d)
	}
	for scope, diffs := range byScope {
		path := statePath(scope, opts)
		s, err := state.Load(path)
		if err != nil {
			return err
		}
		state.Merge(s, state.FromChangeSet(diffs, now))
		if err := state.Save(path, s); err != nil {
			return err
		}
	}
	return nil
}

// statePath returns the state file for a scope.
func statePath(scope string, opts deployOptions) string {
	if scope == "global" {
		return filepath.Join(opts.home, ".patronus", "state.json")
	}
	return filepath.Join(opts.projectDir, ".patronus", "state.json")
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

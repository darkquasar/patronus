package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
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

			// The footer's "dry run" note reflects intent: only a successful
			// --deploy (Phase 3) would write. Today --deploy still can't write,
			// so this stays a dry run in practice — see the refusal below.
			cs.DryRun = !deploy

			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), cs)
			}
			render.PrintPlan(cmd.OutOrStdout(), cs, res, verbose)

			// --deploy is the explicit opt-in to write. The applier is not built
			// yet (Phase 3), so refuse rather than risk a partial write. Without
			// --deploy this is always a safe dry run.
			if deploy {
				return fmt.Errorf("--deploy: apply is not yet implemented (Phase 3); plan shown above, nothing was written")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "all", "target tool: claude|codex|opencode|all")
	cmd.Flags().BoolVar(&global, "global", false, "install at global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "install at project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "actually write changes to disk (default: dry run only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "explicitly plan only (the default; no-op without --deploy)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "also show per-artifact unified diffs")
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

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func newInstallCmd() *cobra.Command {
	var (
		tool            string
		global          bool
		local           bool
		deploy          bool
		dryRun          bool
		verbose         bool
		force           bool
		yes             bool
		recipeSel       string
		preferSystemPkg bool
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

			// --recipe explicitly selects a recipe by name. In Phase 4 the
			// positional name already is the selection; --recipe is accepted for
			// forward-compat (profiles, Phase 5) and folded into the name list.
			names := args
			if recipeSel != "" && !contains(names, recipeSel) {
				names = append(names, recipeSel)
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

			cs, err := computePlan(planInputs{
				cat:             cat,
				inv:             inv,
				adapters:        adapterMap(adapters),
				res:             res,
				names:           names,
				tool:            tool,
				scope:           scope,
				preferSystemPkg: preferSystemPkg,
				warnf:           func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) },
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
	cmd.Flags().StringVar(&recipeSel, "recipe", "", "pick a specific recipe for a capability (e.g. memory-engram)")
	cmd.Flags().BoolVar(&preferSystemPkg, "prefer-system-pkg", false, "use brew/scoop/winget if present (Phase 8; currently falls through to github-release)")
	return cmd
}

// planInputs carries everything computePlan needs to build a change set across a
// mix of artifact and recipe names.
type planInputs struct {
	cat             *registry.Catalog
	inv             *scan.Inventory
	adapters        map[string]*manifest.Adapter
	res             toolpath.Resolver
	names           []string
	tool            string
	scope           string
	preferSystemPkg bool
	warnf           func(string, ...any)
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
	for _, name := range in.names {
		if rec := findRecipe(in.cat, name); rec != nil {
			diffs, err := recipe.Compute(recipe.Request{
				Recipe:          rec.Manifest,
				Adapters:        in.adapters,
				Resolver:        in.res,
				Tool:            in.tool,
				Scope:           in.scope,
				PreferSystemPkg: in.preferSystemPkg,
				Warnf:           in.warnf,
			})
			if err != nil {
				return nil, err
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

	return plan.Finalize(raw, read)
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
	force      bool
	yes        bool
	home       string // for ~/.patronus/state.json
	projectDir string // for <project>/.patronus/state.json
}

// commandRunner runs a self-wiring recipe's post-install command. The real impl
// shells out via os/exec; tests inject a fake so no test spawns a process.
type commandRunner interface {
	Run(argv []string) error
}

// execRunner is the production commandRunner: it runs argv via os/exec, streaming
// output to the command's stdout/stderr.
type execRunner struct {
	cmd *cobra.Command
}

func (r execRunner) Run(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.Stdout = r.cmd.OutOrStdout()
	c.Stderr = r.cmd.ErrOrStderr()
	return c.Run()
}

// runDeploy writes the change set to disk, performs FETCH downloads, runs
// self-wiring EXEC commands, and records what was installed. It is Terraform-style:
// a mid-apply failure stops, records what already succeeded in state, and returns
// the error. The runner is overridable for tests (nil => real os/exec).
func runDeploy(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions) error {
	return runDeployWith(cmd, cs, res, opts, nil)
}

func runDeployWith(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts deployOptions, runner commandRunner) error {
	out := cmd.OutOrStdout()

	app := &install.Applier{
		Force:    opts.force,
		Conflict: conflictPrompt(cmd, res, opts.yes),
		Fetcher:  recipe.HTTPFetcher{},
		Ctx:      cmd.Context(),
	}
	result, applyErr := app.Apply(cs)

	// Realize self-wiring post-install commands (EXEC diffs) only after the file
	// writes/fetches succeed, and only on --deploy (we are here). The applier
	// itself never spawns processes — it stays a pure file writer.
	realized := append([]diff.FileDiff(nil), result.Applied...)
	if applyErr == nil {
		if runner == nil {
			runner = execRunner{cmd: cmd}
		}
		ran, execErr := runExecs(cmd, cs, runner)
		realized = append(realized, ran...)
		if execErr != nil {
			applyErr = execErr
		}
	}

	// Record whatever succeeded BEFORE surfacing any error (state must reflect
	// reality even on partial failure).
	if stateErr := recordState(realized, opts); stateErr != nil {
		// A state-write failure shouldn't mask an apply failure, but report it.
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to write state: %v\n", stateErr)
	}

	fmt.Fprintf(out, "\nApplied: %d written, %d skipped\n", len(result.Applied), len(result.Skipped))
	if applyErr != nil {
		return applyErr
	}
	return nil
}

// runExecs runs each EXEC diff's command in order and returns the ones that ran
// (for state recording). The first failure stops, mirroring the applier's
// Terraform-style partial-on-failure.
func runExecs(cmd *cobra.Command, cs *diff.ChangeSet, runner commandRunner) ([]diff.FileDiff, error) {
	var ran []diff.FileDiff
	for _, d := range cs.Diffs {
		if d.Action != diff.Exec || d.Exec == nil {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "EXEC %s\n", d.Exec.Display)
		if err := runner.Run(d.Exec.Command); err != nil {
			return ran, fmt.Errorf("post-install %q: %w", d.Exec.Display, err)
		}
		ran = append(ran, d)
	}
	return ran, nil
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

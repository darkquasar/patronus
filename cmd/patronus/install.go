package main

import (
	"bufio"
	"context"
	"fmt"
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
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/source"
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
		profileSel      string
		preferSystemPkg bool
		regSel          registrySel
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			home := toolpath.HomeDir(os.LookupEnv)
			reg, root, err := resolveRegistry(cmd.Context(), wd, regSel, home, warnf)
			if err != nil {
				return err
			}

			// adapters/ comes from the checkout when local; loadAdapters falls back to
			// the embedded adapters when root is "" (installed-binary / remote case).
			adapters, err := loadAdapters(filepath.Join(root, "adapters"))
			if err != nil {
				return err
			}

			// Load the catalog. When the user is installing ONLY out-of-tree sources
			// (git:/https:/file:) and no profile, the registry is not actually needed,
			// so a fetch failure (e.g. offline, no cache) degrades to an empty catalog
			// rather than blocking a self-contained sourced install.
			cat, err := reg.Catalog(cmd.Context())
			if err != nil {
				// Only tolerate a registry failure when every name is a self-contained
				// sourced reference (git:/https:/file:) and there's no profile to resolve.
				if profileSel != "" || !allSourced(names) {
					return err
				}
				warnf("registry unavailable (%v); proceeding with sourced references only", err)
				cat = &registry.Catalog{}
			}

			// --profile expands to the profile's resolved item names, which then flow
			// through the SAME artifact-vs-recipe dispatch a plain install uses.
			if profileSel != "" {
				// tool selects per-tool flavours (§4); "all" yields the tool-agnostic baseline.
				res, err := profile.Resolve(cat, profileSel, tool)
				if err != nil {
					return err
				}
				for _, w := range res.Warnings {
					warnf("%s", w)
				}
				names = res.Names()
				if len(names) == 0 {
					return fmt.Errorf("profile %q resolved to no installable items", profileSel)
				}

				// Per-item reality-follows-lock: if a committed patronus.lock pins this
				// profile's items, rewrite the catalog so each is fetched at its LOCKED
				// version+sha from the registry's immutable key (not the index's latest).
				if rr, ok := reg.(*registry.RemoteRegistry); ok {
					applyLockPins(wd, profileSel, rr.Base(), cat, warnf)
				}
			}

			// A positional name may be a sourced reference (file:, git:, https:, ...).
			// Resolve any sourced entries into the catalog so they dispatch like an
			// in-tree item; bare names are left untouched.
			names, err = mergeSourcedNames(cmd.Context(), cat, names, home)
			if err != nil {
				return err
			}

			// For a remote registry, fetch+unpack the selected artifacts' source so
			// the local adapter path can transform them (no-op for local/recipes).
			if err := materializeSelected(cmd.Context(), reg, cat, names); err != nil {
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
				warnf:           warnf,
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
	cmd.Flags().StringVar(&profileSel, "profile", "", "install a curated bundle across layers (§5d)")
	addRegistryFlags(cmd, &regSel)
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

// mergeSourcedNames resolves each name as a sourced reference. Bare (registry)
// names pass through unchanged. A file:/git:/https: reference is fetched+loaded
// into the catalog so it dispatches exactly like an in-tree item, and its dispatch
// name becomes the resolved manifest's name. Names already present in the catalog
// (the common case) take the registry path with zero overhead.
func mergeSourcedNames(ctx context.Context, cat *registry.Catalog, names []string, home string) ([]string, error) {
	rs := &source.Resolver{
		Fetcher:  fetcherForCommands,
		CacheDir: filepath.Join(home, ".patronus", "cache", "sources"),
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
// artifact entry so the install fetches the LOCKED version+bytes from the
// registry's immutable name/version key, rather than whatever the (mutable)
// discovery index now advertises as latest. This is what makes a shared lock
// reproduce the exact environment even as the catalog moves on.
//
// It is a no-op when there's no lock, the lock is for a different profile, or an
// item isn't pinned — those follow the index latest, unchanged. base is the
// RemoteRegistry base URL (used to reconstruct the immutable item URL).
func applyLockPins(wd, profileName, base string, cat *registry.Catalog, warnf func(string, ...any)) {
	l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
	if err != nil || len(l.Entries) == 0 {
		return // no lock → follow index latest
	}
	if l.Profile != "" && l.Profile != profileName {
		return // an unrelated lock never silently pins this install
	}
	pin := make(map[string]lock.Entry, len(l.Entries))
	for _, e := range l.Entries {
		pin[e.Name] = e
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

// runnerForCommands is the package-level seam for self-wiring post-install EXECs
// driven through the cobra commands (the runDeploy path takes no runner argument).
// Production leaves it nil → real os/exec. Integration tests that install a
// self-wiring recipe (e.g. memory-ai-memory) set it to a fake so the suite stays
// process-free — mirroring fetcherForCommands/registryFetcher. A test hook, not a
// user knob.
var runnerForCommands commandRunner

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
		if d.Exec.Advisory {
			// Display-only: an install-only recipe's package-install line. Patronus
			// never runs a global package install on the user's behalf — it surfaces
			// the command so the user (or a future --prefer-system-pkg path) runs it.
			// It is still recorded (in `ran`) so state remembers the recipe was
			// installed and remove can report the manual-cleanup command.
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

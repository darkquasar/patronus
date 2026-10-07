package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Relocate recorded Patronus payloads between native roots",
	}
	cmd.AddCommand(newMigrateCodexSkillsCmd())
	return cmd
}

// codexMigrateOpts exposes write/persist seams for failure-path tests.
type codexMigrateOpts struct {
	home, project string
	scopes        []string
	deploy        bool
	beforeWrite   func(diff.FileDiff) error
	saveState     func(string, *state.State) error
}

func newMigrateCodexSkillsCmd() *cobra.Command {
	var global, local, deploy bool
	cmd := &cobra.Command{
		Use:   "codex-skills",
		Short: "Move recorded Codex skills from .codex skill roots to .agents/skills — preview by default",
		Long: "Moves Patronus-recorded Codex skill payloads (SKILL.md and owned sidecars) from\n" +
			"legacy ~/.codex/skills, $CODEX_HOME/skills or <project>/.codex/skills into the\n" +
			"selected .agents/skills root. Preview is the default and inspects both scopes.\n" +
			"--deploy requires exactly one of --global or --local as explicit scope consent.\n" +
			"Unowned, edited, colliding, linked or ambiguous payloads block the whole scope;\n" +
			"there is no --force bypass. Destination writes are verified and ownership is\n" +
			"persisted before any legacy source is retired. Failures retain old files and\n" +
			"ownership. Static placement is runtime-unverified.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if global && local {
				return fmt.Errorf("--global and --local are mutually exclusive")
			}
			scopes := []string{"global", "local"}
			switch {
			case global:
				scopes = []string{"global"}
			case local:
				scopes = []string{"local"}
			case deploy:
				return fmt.Errorf("--deploy requires explicit scope consent: pass --global or --local")
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCodexMigrate(cmd.OutOrStdout(), codexMigrateOpts{home: homeDir(), project: wd, scopes: scopes, deploy: deploy})
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "select global (user) scope")
	cmd.Flags().BoolVar(&local, "local", false, "select project (local) scope")
	cmd.Flags().BoolVar(&deploy, "deploy", false, "apply the selected scope's migration (default: preview only)")
	return cmd
}

func codexMigrationScope(scope, home, project string) plan.CodexMigrationScope {
	if scope == "global" {
		config := toolpath.New(os.LookupEnv, home, project).ResolveMarker("~/.codex", "codex", "global")
		legacy := []string{filepath.Join(home, ".codex/skills")}
		if c := filepath.Join(config, "skills"); c != legacy[0] {
			legacy = append(legacy, c)
		}
		return plan.CodexMigrationScope{Scope: scope, Legacy: legacy, Dest: filepath.Join(home, ".agents/skills")}
	}
	return plan.CodexMigrationScope{Scope: scope, Legacy: []string{filepath.Join(project, ".codex/skills")}, Dest: filepath.Join(project, ".agents/skills")}
}

func runCodexMigrate(out io.Writer, opts codexMigrateOpts) (err error) {
	var m *mutation
	if opts.deploy {
		if len(opts.scopes) != 1 {
			return fmt.Errorf("--deploy requires exactly one selected scope")
		}
		if m, err = beginMutation(opts.home, opts.project); err != nil {
			return err
		}
		defer m.close(&err)
	}
	config := toolpath.New(os.LookupEnv, opts.home, opts.project).ResolveMarker("~/.codex", "codex", "global")
	for _, p := range []string{opts.home, opts.project, config} {
		if err := scan.CodexSafePath(p); err != nil {
			return err
		}
	}
	states := map[string]*state.State{}
	for _, scope := range []string{"global", "local"} {
		sp := removeStatePath(scope, opts.home, opts.project)
		if err := scan.CodexSafePath(sp); err != nil {
			return err
		}
		if states[scope], err = state.Load(sp); err != nil {
			return fmt.Errorf("load %s state: %w", scope, err)
		}
	}
	roots := scan.CodexSkillRoots(opts.home, opts.project, config)
	for _, scope := range opts.scopes {
		in := plan.CodexMigrationInput{Scope: codexMigrationScope(scope, opts.home, opts.project), States: states, Roots: roots}
		moves, err := plan.CodexSkillMigration(in)
		if err != nil {
			return err
		}
		if len(moves) == 0 {
			fmt.Fprintf(out, "%s: no recorded legacy Codex skills\n", scope)
			continue
		}
		for _, mv := range moves {
			fmt.Fprintf(out, "MIGRATE %s (%s)\n", mv.Source.Artifact, scope)
			for _, d := range mv.Writes {
				fmt.Fprintf(out, "  CREATE %s\n", d.Path)
			}
			for _, d := range mv.Retire {
				fmt.Fprintf(out, "  DELETE %s (after verified write and ownership persistence)\n", d.Path)
			}
		}
		if !opts.deploy {
			fmt.Fprintf(out, "Preview only; rerun with --%s --deploy to apply.\n", scope)
			continue
		}
		if err := deployCodexMigration(out, m, opts, scope, states[scope], moves); err != nil {
			return err
		}
	}
	return nil
}

// deployCodexMigration writes and verifies every destination, persists old and
// new ownership, retires verified sources, then persists destination-only
// ownership. Each failure persists only verified facts and retains old data.
func deployCodexMigration(out io.Writer, m *mutation, opts codexMigrateOpts, scope string, s *state.State, moves []plan.CodexSkillMove) error {
	sp := removeStatePath(scope, opts.home, opts.project)
	save := func() error { return m.saveState(sp, s, opts.saveState) }
	var writes, retire []diff.FileDiff
	for _, mv := range moves {
		writes = append(writes, mv.Writes...)
		retire = append(retire, mv.Retire...)
	}
	app := &install.Applier{BeforeWrite: func(d diff.FileDiff) error {
		if err := m.checkFile(d, nil); err != nil {
			return err
		}
		if opts.beforeWrite != nil {
			return opts.beforeWrite(d)
		}
		return nil
	}}
	cs := &diff.ChangeSet{Diffs: writes}
	if err := m.checkPlan(&diff.ChangeSet{Diffs: append(append([]diff.FileDiff(nil), writes...), retire...)}, nil); err != nil {
		return err
	}
	res, applyErr := app.Apply(cs)
	if applyErr != nil {
		// Record only verified destination writes, beside the unchanged old rows.
		written := map[string]bool{}
		for _, d := range res.Applied {
			written[d.Path] = true
		}
		for _, mv := range moves {
			it := mv.Source
			for _, f := range mv.Staged.Files {
				if written[f.Path] {
					it.Files = append(it.Files, f)
				}
			}
			s.Items[mv.Index] = it
		}
		if len(written) > 0 {
			if err := save(); err != nil {
				return fmt.Errorf("codex migration write: %w; persist %s: %w; written destinations %s have unresolved ownership; old sources retained", applyErr, sp, err, codexPaths(res.Applied))
			}
		}
		return fmt.Errorf("codex migration write: %w; old sources and ownership retained; verified partial destinations recorded beside old rows", applyErr)
	}
	for _, mv := range moves {
		s.Items[mv.Index] = mv.Staged
	}
	if err := save(); err != nil {
		return fmt.Errorf("codex migration ownership: %w; destinations %s have unresolved ownership; old sources and ownership retained", err, codexPaths(writes))
	}
	res, retireErr := app.Apply(&diff.ChangeSet{Diffs: retire})
	if retireErr != nil {
		retired := map[string]bool{}
		for _, d := range res.Applied {
			retired[d.Path] = true
		}
		for _, mv := range moves {
			it := mv.Staged
			it.Files = nil
			for _, f := range mv.Staged.Files {
				if !retired[f.Path] {
					it.Files = append(it.Files, f)
				}
			}
			s.Items[mv.Index] = it
		}
		if err := save(); err != nil {
			return errors.Join(fmt.Errorf("codex migration retirement: %w", retireErr), fmt.Errorf("persist %s: %w; ownership uncertain; fresh preview required", sp, err))
		}
		return fmt.Errorf("codex migration retirement: %w; unretired sources and their ownership retained", retireErr)
	}
	for _, mv := range moves {
		for _, dir := range mv.Dirs {
			_ = os.Remove(dir) // prune only when empty; a non-empty dir stays
		}
		s.Items[mv.Index] = mv.Final
	}
	if err := save(); err != nil {
		return fmt.Errorf("codex migration final ownership: %w; retired source rows remain recorded; fresh preview required", err)
	}
	fmt.Fprintf(out, "Migrated: %d skill(s) in %s scope\n", len(moves), scope)
	return nil
}

func codexPaths(ds []diff.FileDiff) string {
	var paths []string
	for _, d := range ds {
		paths = append(paths, d.Path)
	}
	return strings.Join(paths, ", ")
}

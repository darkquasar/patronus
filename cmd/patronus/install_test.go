package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// runInstall executes the ordinary install command against the caller's
// fixture environment and returns stdout, stderr and the command error.
func runInstall(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newInstallCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func TestInstallSkillDryRun(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	out, _, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	for _, want := range []string{"fix-skill", "SKILL.md", "CREATE", "skill", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallVerboseShowsDiff(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	out, _, err := runInstall(t, "fix-instruction-global", "--target", "claude", "--local", "--verbose", "--dry-run")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	// The invented instruction emits APPEND with a unified diff body.
	if !strings.Contains(out, "APPEND") {
		t.Errorf("expected APPEND:\n%s", out)
	}
	if !strings.Contains(out, "@@") {
		t.Errorf("verbose mode should show unified diff hunks:\n%s", out)
	}
}

func TestInstallMutuallyExclusiveScope(t *testing.T) {
	_, _, err := runInstall(t, "fix-skill", "--global", "--local")
	if err == nil {
		t.Error("expected error for --global and --local together")
	}
}

func TestInstallProfileCloudflareDryRun(t *testing.T) {
	root := fixtureCatalog(t)
	mp := filepath.Join(root, "profiles/fix-all.yaml")
	if err := os.WriteFile(mp, append(mustRead(t, mp), []byte("status: stub\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("HOME", t.TempDir())
	out, errOut, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("profile install failed: %v\n%s", err, errOut)
	}
	// The invented profile populates several delivery layers;
	// every populated slot's item should appear in the combined plan.
	for _, want := range []string{"fix-instruction", "fix-instruction-2", "fix-skill", "fix-style", "fix-hook"} {
		if !strings.Contains(out, want) {
			t.Errorf("profile plan missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "dry run") {
		t.Errorf("expected dry-run footer:\n%s", out)
	}
	// status: stub profile must warn (on stderr).
	if !strings.Contains(errOut, "stub") {
		t.Errorf("expected stub warning on stderr:\n%s", errOut)
	}
}

func TestInstallProfileAndPositionalMutuallyExclusive(t *testing.T) {
	_, _, err := runInstall(t, "fix-skill", "--profile", "fix-all")
	if err == nil {
		t.Error("expected error for --profile with positional names")
	}
}

func TestInstallNoTargetSpecified(t *testing.T) {
	_, _, err := runInstall(t)
	if err == nil {
		t.Error("expected error when neither names nor --profile given")
	}
}

func TestInstallUnknownArtifact(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	_, _, err := runInstall(t, "does-not-exist")
	if err == nil {
		t.Error("expected error for unknown artifact")
	}
}

func TestInstallDefaultIsDryRun(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	// No --deploy, no --dry-run: must be a safe dry run, no error, plan shown.
	out, _, err := runInstall(t, "fix-skill", "--target", "claude", "--global")
	if err != nil {
		t.Fatalf("default install should succeed as dry run: %v", err)
	}
	if !strings.Contains(out, "dry run") {
		t.Errorf("default run should be a dry run:\n%s", out)
	}
}

func TestInstallDeployWritesFilesAndState(t *testing.T) {
	// Drive the deploy machinery directly with a constructed change set into
	// isolated temp dirs (the full cobra path needs the repo registry; the write
	// + state behavior is what matters here).
	home := t.TempDir()
	proj := t.TempDir()
	res := toolpath.New(func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}, home, proj)

	skillPath := filepath.Join(home, ".claude", "skills", "s", "SKILL.md")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: skillPath, Action: diff.Create, After: []byte("BODY"),
			Artifact: "s", Type: "skill", Tool: "claude", Scope: "global"},
	}}

	cmd := newInstallCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runDeploy(cmd, cs, res, deployOptions{home: home, projectDir: proj}); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// File written.
	if b, err := os.ReadFile(skillPath); err != nil || string(b) != "BODY" {
		t.Fatalf("skill not written: %v %q", err, b)
	}
	// State recorded with a checksum.
	statePath := filepath.Join(home, ".patronus", "state.json")
	sb, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("state not written: %v", err)
	}
	for _, want := range []string{`"artifact": "s"`, `"action": "CREATE"`, "sha256:"} {
		if !strings.Contains(string(sb), want) {
			t.Errorf("state missing %q:\n%s", want, sb)
		}
	}
}

func TestRecordStateSplitsByScope(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	opts := deployOptions{home: home, projectDir: proj}
	applied := []diff.FileDiff{
		{Path: filepath.Join(home, ".claude/skills/g/SKILL.md"), Action: diff.Create, After: []byte("g"), Artifact: "g", Tool: "claude", Scope: "global"},
		{Path: filepath.Join(proj, ".claude/skills/l/SKILL.md"), Action: diff.Create, After: []byte("l"), Artifact: "l", Tool: "claude", Scope: "local"},
	}
	if err := recordState(&diff.ChangeSet{Diffs: applied}, &install.Result{Applied: applied}, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus", "state.json")); err != nil {
		t.Errorf("global state missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".patronus", "state.json")); err != nil {
		t.Errorf("local state missing: %v", err)
	}
}

func TestInstallDeployAndDryRunMutuallyExclusive(t *testing.T) {
	_, _, err := runInstall(t, "fix-skill", "--deploy", "--dry-run")
	if err == nil {
		t.Error("expected error for --deploy and --dry-run together")
	}
}

func TestInstallJSON(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	// --json is a persistent root flag; set it on the package global directly
	// since we run the subcommand in isolation here.
	jsonOutput = true
	defer func() { jsonOutput = false }()
	out, _, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !strings.Contains(out, `"action": "CREATE"`) || !strings.Contains(out, `"dryRun": true`) {
		t.Errorf("unexpected json:\n%s", out)
	}
	// Before/After bytes must not leak into JSON.
	if strings.Contains(out, `"before"`) || strings.Contains(out, `"after"`) {
		t.Errorf("raw content leaked into json:\n%s", out)
	}
}

// --- Phase 4: recipe dispatch + self-wiring EXEC -----------------------------

func TestInstallRecipeRemoteMcpDryRun(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	// The invented HTTP MCP recipe is pure MERGE, with no fetch.
	out, _, err := runInstall(t, "fix-mcp-two", "--target", "claude", "--local", "--dry-run")
	if err != nil {
		t.Fatalf("install github failed: %v", err)
	}
	for _, want := range []string{"fix-mcp-two", ".mcp.json", "MERGE", "mcp"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallRecipeFetchDryRun(t *testing.T) {
	t.Chdir(fixtureCatalog(t))
	t.Setenv("HOME", t.TempDir())
	// The invented archive recipe emits FETCH and a per-tool MERGE.
	out, _, err := runInstall(t, "fix-mcp-bin", "--target", "all", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install memory-engram failed: %v", err)
	}
	for _, want := range []string{"fix-mcp-bin", "FETCH", "MERGE"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// TestInstallPathReadinessWarningFires is the PATH-readiness acceptance check: installing a
// recipe that FETCHes a binary into a dir absent from the inherited $PATH prints the
// PATH-readiness warning, and installing with that dir ON $PATH does not. Uses the
// fixture raw-delivery recipe (fix-bin → ~/.patronus/bin) with $PATH controlled per
// run. Dry-run suffices — the warning is always-on.
func TestInstallPathReadinessWarningFires(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixRawURL] = fixRawBinary
	home := withRemoteEnv(t, f)
	binDir := filepath.Join(home, ".patronus", "bin")

	// (1) ~/.patronus/bin OFF PATH → the warning fires.
	t.Setenv("PATH", "/usr/bin:/bin")
	out, _, err := runInstall(t, "fix-bin", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install (off PATH): %v", err)
	}
	if !strings.Contains(out, "PATH readiness:") || !strings.Contains(out, binDir) {
		t.Errorf("expected a PATH-readiness warning naming %q:\n%s", binDir, out)
	}

	// (2) the SAME dir ON PATH → no warning.
	t.Setenv("PATH", binDir+":/usr/bin")
	out2, _, err := runInstall(t, "fix-bin", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install (on PATH): %v", err)
	}
	if strings.Contains(out2, "PATH readiness:") {
		t.Errorf("did not expect a PATH-readiness warning when the dir is on PATH:\n%s", out2)
	}
}

// fakeRunner records the argvs it was asked to run and never spawns a process.
type fakeRunner struct{ ran [][]string }

func (f *fakeRunner) Run(argv []string) error {
	f.ran = append(f.ran, argv)
	return nil
}

func TestRunDeployRunsExecAndRecordsSelfWired(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	res := toolpath.New(func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}, home, proj)

	// A NON-advisory exec (an actor:patronus recipe whose commands Patronus does run)
	// — proves runDeployWith executes it and records the provenance. (An actor:external
	// recipe's exec is advisory and would be surfaced, not run; that path is covered
	// by the recipe-layer test + the advisory branch in runExecs.)
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{
		Path: "demo-tool install-mcp --client claude --apply", Action: diff.Exec,
		Artifact: "demo-run-recipe", Type: "fetch+run", Tool: "claude", Scope: "global",
		Exec: &diff.ExecSpec{
			Command:     []string{"demo-tool", "install-mcp", "--client", "claude", "--apply"},
			Display:     "demo-tool install-mcp --client claude --apply",
			SelfManaged: true,
		},
	}}}

	cmd := newInstallCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	runner := &fakeRunner{}
	if err := runDeployWith(cmd, cs, res, deployOptions{home: home, projectDir: proj}, runner); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// The post-install command ran exactly once with the right argv.
	if len(runner.ran) != 1 || runner.ran[0][1] != "install-mcp" {
		t.Fatalf("runner.ran = %v", runner.ran)
	}
	// State records the recipe as self-wired with the command.
	sb, err := os.ReadFile(filepath.Join(home, ".patronus", "state.json"))
	if err != nil {
		t.Fatalf("state not written: %v", err)
	}
	for _, want := range []string{`"selfWired": true`, "install-mcp --client claude --apply"} {
		if !strings.Contains(string(sb), want) {
			t.Errorf("state missing %q:\n%s", want, sb)
		}
	}
}

func TestRunExecsStopsOnFailure(t *testing.T) {
	cmd := newInstallCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Action: diff.Exec, Exec: &diff.ExecSpec{Command: []string{"a"}, Display: "a"}},
		{Action: diff.Exec, Exec: &diff.ExecSpec{Command: []string{"b"}, Display: "b"}},
	}}
	ran, err := runExecs(cmd, cs, failRunner{}, installConsent{look: exec.LookPath, out: &out})
	if err == nil {
		t.Fatal("expected failure")
	}
	if len(ran) != 0 {
		t.Errorf("no command should be recorded as run, got %v", ran)
	}
}

type failRunner struct{}

func (failRunner) Run([]string) error { return os.ErrPermission }

// pkgInstallCS builds a change set with one package-install advisory carrying the
// given ordered candidates.
func pkgInstallCS(artifact string, cands ...diff.InstallCandidateSpec) *diff.ChangeSet {
	return &diff.ChangeSet{Diffs: []diff.FileDiff{{
		Action: diff.Exec, Artifact: artifact,
		Exec: &diff.ExecSpec{Advisory: true, Display: cands[0].Command, Candidates: cands},
	}}}
}

// fetchCS builds a change set with one FETCH diff placing a binary at dest.
func fetchCS(artifact, dest string) *diff.ChangeSet {
	return &diff.ChangeSet{Diffs: []diff.FileDiff{{
		Action: diff.Fetch, Artifact: artifact, Path: dest,
		Fetch: &diff.FetchSpec{Dest: dest},
	}}}
}

func TestPathReadinessFlagsOffPathDest(t *testing.T) {
	tests := []struct {
		name     string
		dest     string
		pathDirs []string
		wantRow  bool
	}{
		{
			name:     "dest dir off PATH warns",
			dest:     "/home/u/.patronus/bin/tk",
			pathDirs: []string{"/usr/bin", "/home/u/.local/bin"},
			wantRow:  true,
		},
		{
			name:     "dest dir on PATH is clean",
			dest:     "/home/u/.local/bin/tk",
			pathDirs: []string{"/usr/bin", "/home/u/.local/bin"},
			wantRow:  false,
		},
		{
			name:     "empty PATH warns (nothing is reachable)",
			dest:     "/home/u/.patronus/bin/tk",
			pathDirs: nil,
			wantRow:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := pathReadiness(fetchCS(tt.name, tt.dest), tt.pathDirs)
			if tt.wantRow && (len(rows) != 1 || rows[0].Artifact != tt.name) {
				t.Fatalf("want 1 row for %q, got %+v", tt.name, rows)
			}
			if !tt.wantRow && len(rows) != 0 {
				t.Fatalf("want no rows, got %+v", rows)
			}
			if tt.wantRow && rows[0].Dir != filepath.Dir(tt.dest) {
				t.Errorf("row Dir = %q, want %q", rows[0].Dir, filepath.Dir(tt.dest))
			}
		})
	}
}

func TestReadinessReportFlagsUnsatisfiable(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	rows := readinessReport(cs, func(string) (string, error) { return "", errors.New("nope") })
	if len(rows) != 1 || rows[0].Satisfiable {
		t.Fatalf("want 1 unsatisfiable row, got %+v", rows)
	}
	if len(rows[0].Missing) != 1 || rows[0].Missing[0] != "uv" {
		t.Errorf("want uv missing, got %+v", rows[0])
	}
}

func TestPreflightAllOrNothingErrorsWhenMissing(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	if err := preflightAllOrNothing(cs, func(string) (string, error) { return "", errors.New("no") }); err == nil {
		t.Fatal("want preflight error when no manager present")
	}
	if err := preflightAllOrNothing(cs, func(string) (string, error) { return "/x", nil }); err != nil {
		t.Fatalf("want no error when a manager is present, got %v", err)
	}
}

func TestConsentAllowFlagRunsWhenManagerPresent(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	runner := &fakeRunner{}
	consent := installConsent{allow: true, look: func(string) (string, error) { return "/x", nil }, out: io.Discard}
	if _, err := runExecs(newInstallCmd(), cs, runner, consent); err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 1 || runner.ran[0][0] != "uv" {
		t.Fatalf("want the uv install to run, ran=%v", runner.ran)
	}
}

func TestConsentAllowFlagFallsOffPreferredManager(t *testing.T) {
	// uv (preferred) is absent; brew (fallback) is present → install via brew.
	cs := pkgInstallCS("demo-recipe",
		diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"},
		diff.InstallCandidateSpec{Manager: "brew", Command: "brew install mypkg"},
	)
	runner := &fakeRunner{}
	var out bytes.Buffer
	look := func(bin string) (string, error) {
		if bin == "brew" {
			return "/opt/brew", nil
		}
		return "", errors.New("absent")
	}
	if _, err := runExecs(newInstallCmd(), cs, runner, installConsent{allow: true, look: look, out: &out}); err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 1 || runner.ran[0][0] != "brew" {
		t.Fatalf("want the brew fallback to run, ran=%v", runner.ran)
	}
	if !strings.Contains(out.String(), "uv not available; using brew") {
		t.Errorf("want a fell-off-preferred note, got:\n%s", out.String())
	}
}

func TestConsentSurfacesWhenNoManagerPresent(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	runner := &fakeRunner{}
	var out bytes.Buffer
	consent := installConsent{allow: true, look: func(string) (string, error) { return "", errors.New("no") }, out: &out}
	ran, err := runExecs(newInstallCmd(), cs, runner, consent)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 0 {
		t.Fatalf("nothing should run when no manager present, ran=%v", runner.ran)
	}
	// Still recorded (state remembers the recipe) and surfaced + summarised.
	if len(ran) != 1 {
		t.Fatalf("advisory should still be recorded, got %d", len(ran))
	}
	if !strings.Contains(out.String(), "Skipped package installs") {
		t.Errorf("want a skip summary, got:\n%s", out.String())
	}
}

func TestConsentInteractiveDeclineSurfaces(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	runner := &fakeRunner{}
	var out bytes.Buffer
	consent := installConsent{
		look: func(string) (string, error) { return "/x", nil },
		in:   bufio.NewReader(strings.NewReader("n\n")),
		out:  &out,
	}
	if _, err := runExecs(newInstallCmd(), cs, runner, consent); err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 0 {
		t.Fatalf("a declined install must not run, ran=%v", runner.ran)
	}
}

func TestConsentInteractiveAcceptRuns(t *testing.T) {
	cs := pkgInstallCS("demo-recipe", diff.InstallCandidateSpec{Manager: "uv", Command: "uv tool install mypkg"})
	runner := &fakeRunner{}
	var out bytes.Buffer
	consent := installConsent{
		look: func(string) (string, error) { return "/x", nil },
		in:   bufio.NewReader(strings.NewReader("y\n")),
		out:  &out,
	}
	if _, err := runExecs(newInstallCmd(), cs, runner, consent); err != nil {
		t.Fatal(err)
	}
	if len(runner.ran) != 1 {
		t.Fatalf("an accepted install must run, ran=%v", runner.ran)
	}
}

func TestAllowPackageInstallsFlagRegistered(t *testing.T) {
	cmd := newInstallCmd()
	if cmd.Flags().Lookup("allow-package-installs") == nil {
		t.Fatal("--allow-package-installs not registered")
	}
	if cmd.Flags().Lookup("prefer-system-pkg") != nil {
		t.Fatal("--prefer-system-pkg should be removed")
	}
}

// TestComputePlanDispatchesPlugin proves a plugin name routes to the plugin
// install path (not the artifact fall-through). The plugin has a "claude-code"
// source and the injected probe reports the claude CLI present, so it dispatches
// as a non-advisory EXEC diff that rides the shared apply spine.
func TestComputePlanDispatchesPlugin(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	res := toolpath.New(func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}, home, proj)

	adapters, err := loadAdapters(filepath.Join(t.TempDir(), "no-adapters-dir"))
	if err != nil {
		t.Fatalf("loadAdapters: %v", err)
	}

	cat := &registry.Catalog{
		Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
			Meta:    manifest.Meta{APIVersion: manifest.APIVersion, Family: manifest.FamilyPlugin, Name: "demo-plugin"},
			Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Ref: "v2.1.0"}},
		}}},
	}

	cs, err := computePlan(planInputs{
		cat:         cat,
		adapters:    adapterMap(adapters),
		res:         res,
		names:       []string{"demo-plugin"},
		tool:        "claude",
		scope:       "global",
		pluginProbe: fakeProbe{present: map[string]bool{"claude": true}},
	})
	if err != nil {
		t.Fatalf("computePlan: %v", err)
	}
	if cs == nil || len(cs.Diffs) == 0 {
		t.Fatal("expected an install EXEC diff for the plugin, got none")
	}
	// A claude-code source with the claude CLI present dispatches as a non-advisory
	// EXEC diff (Patronus runs it on --deploy), typed "plugin" and keyed on claude.
	var found bool
	for _, d := range cs.Diffs {
		if d.Action == diff.Exec && d.Type == "plugin" && d.Tool == "claude" &&
			d.Exec != nil && !d.Exec.Advisory {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a non-advisory plugin EXEC diff for claude, got %+v", cs.Diffs)
	}
}

// TestComputePlanPluginAllExpandsTargets covers FIX C1+I1: a bare install (tool
// "all", no scope flag) must fan out to the plugin's Targets and honor the
// manifest's defaults.scope, emitting EXEC diffs for EVERY target instead of the
// zero diffs a single "all" call would yield.
func TestComputePlanPluginAllExpandsTargets(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	res := toolpath.New(func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}, home, proj)

	adapters, err := loadAdapters(filepath.Join(t.TempDir(), "no-adapters-dir"))
	if err != nil {
		t.Fatalf("loadAdapters: %v", err)
	}

	cat := &registry.Catalog{
		Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
			Meta:     manifest.Meta{APIVersion: manifest.APIVersion, Family: manifest.FamilyPlugin, Name: "demo-plugin"},
			Sources:  map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Ref: "v2.1.0"}},
			Targets:  []string{"claude", "codex"},
			Defaults: manifest.PluginDefaults{Scope: "global"},
		}}},
	}

	cs, err := computePlan(planInputs{
		cat:         cat,
		adapters:    adapterMap(adapters),
		res:         res,
		names:       []string{"demo-plugin"},
		tool:        "all", // the default; must expand to Targets
		scope:       "",    // no flag; must fall back to defaults.scope=global
		pluginProbe: fakeProbe{present: map[string]bool{}},
	})
	if err != nil {
		t.Fatalf("computePlan: %v", err)
	}
	if cs == nil || len(cs.Diffs) == 0 {
		t.Fatal("expected install EXEC diffs for the plugin, got none")
	}
	for _, d := range cs.Diffs {
		if d.Scope != "global" {
			t.Errorf("diff %s scope = %q, want global (from defaults.scope)", d.Path, d.Scope)
		}
	}
	// The bare "all" install fans out to both declared targets: claude (a
	// claude-code source => install commands) and codex (no codex source => an
	// honest skip line). Both surface as plugin EXEC diffs.
	tools := map[string]bool{}
	for _, d := range cs.Diffs {
		if d.Type == "plugin" {
			tools[d.Tool] = true
		}
	}
	if !tools["claude"] || !tools["codex"] {
		t.Errorf("expected plugin EXEC diffs for both claude and codex, got tools %v", tools)
	}
}

// TestMergeSourcedNamesPlugin covers FIX I2: a file: reference to a plugin
// manifest must resolve into cat.Plugins and return the plugin's name, rather than
// failing with "resolved to nothing".
func TestMergeSourcedNamesPlugin(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "demo-plugin.yaml")
	body := "apiVersion: patronus/v2\n" +
		"family: plugin\n" +
		"role: lifecycle\n" +
		"name: demo-plugin\n" +
		"version: 2.1.0\n" +
		"sources:\n" +
		"  claude-code:\n" +
		"    kind: marketplace\n" +
		"    ref: v2.1.0\n" +
		"targets: [claude, codex]\n"
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cat := &registry.Catalog{}
	names, err := mergeSourcedNames(context.Background(), cat, []string{"file:" + manifestPath}, t.TempDir(), false)
	if err != nil {
		t.Fatalf("mergeSourcedNames: %v", err)
	}
	if len(names) != 1 || names[0] != "demo-plugin" {
		t.Fatalf("names = %v, want [demo-plugin]", names)
	}
	if len(cat.Plugins) != 1 || cat.Plugins[0].Manifest.Name != "demo-plugin" {
		t.Fatalf("cat.Plugins = %+v, want one demo-plugin plugin", cat.Plugins)
	}
}

// TestInstallWiredRecipeRequiresTarget: a recipe that wires an MCP entry into a
// runtime (fix-mcp-bin: fetch + MERGE) needs an explicit --target; installing it
// bare must error rather than silently fanning out to every backend.
func TestInstallWiredRecipeRequiresTarget(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	withRemoteEnv(t, f)

	_, _, err := runInstall(t, "fix-mcp-bin", "--global", "--dry-run") // wires an MCP entry
	if err == nil {
		t.Fatal("expected error: a wired recipe needs an explicit --target")
	}
	if !strings.Contains(err.Error(), "--target is required") {
		t.Errorf("error should come from the required-target gate, got: %v", err)
	}
}

// TestInstallAgnosticRecipeNeedsNoTarget: a binary-only recipe (fix-bin: raw fetch,
// no wiring row) belongs to no runtime by nature, so it installs with no --target.
func TestInstallAgnosticRecipeNeedsNoTarget(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixRawURL] = fixRawBinary
	withRemoteEnv(t, f)

	if _, _, err := runInstall(t, "fix-bin", "--global", "--dry-run"); err != nil {
		t.Errorf("agnostic binary recipe should install with no --target: %v", err)
	}
}

// TestInstallProfileWiresBothMcpServers is the end-to-end gate for composed MCP
// wiring: a profile naming TWO MCP recipes that target ONE config file must wire
// BOTH. Before the compose fix exactly one survived planning.
//
// It binds to the FIXTURE catalog deliberately. Asserting that the real code-intel
// profile contains particular servers would test the catalog, not the mechanism;
// the real profile is confirmed by a manual check instead.
func TestInstallProfileWiresBothMcpServers(t *testing.T) {
	root := fixtureCatalog(t)
	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "--profile", "fix-two-mcp", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	raw := mustRead(t, filepath.Join(home, ".claude.json"))
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse config: %v (%s)", err, raw)
	}
	servers, _ := m["mcpServers"].(map[string]any)
	for _, want := range []string{"fix-mcp-bin", "fix-mcp-two"} {
		if _, ok := servers[want]; !ok {
			t.Errorf("mcpServers.%s missing; a same-path sibling overwrote it:\n%s", want, raw)
		}
	}
}

func TestApplyLockPinsDirectoryRoles(t *testing.T) {
	for _, role := range []manifest.Role{manifest.RoleSandbox, manifest.RoleOrchestration, manifest.RoleTools} {
		t.Run(string(role), func(t *testing.T) {
			f := newDirectoryFixture(t)
			rec := f.recipe(t, "sample-replay", "1.0.0", "invented payload")
			rec.Role = role
			f.saveRecipe(t, rec)
			pin := &lock.Lock{Version: 2, Profile: "sample-profile", Entries: []lock.Entry{{Name: rec.Name, Version: rec.Version, Kind: "recipe", Source: "registry", SHA256: "sha256:" + strings.Repeat("a", 64), Delivery: rec.Delivery}}}
			path := filepath.Join(f.root, "patronus.lock")
			if err := lock.Save(path, pin); err != nil {
				t.Fatal(err)
			}
			before := mustRead(t, path)
			newer := *rec
			newer.Version = "2.0.0"
			cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: &newer}}}
			if err := applyLockPins(f.root, "sample-profile", "all", "", cat, nil); err != nil {
				t.Fatal(err)
			}
			if got := cat.Recipes[0].Manifest; got.Version != "1.0.0" || got.Role != role || got.Delivery.Package.Name != rec.Delivery.Package.Name {
				t.Fatalf("replayed recipe = %+v", got)
			}
			if !bytes.Equal(before, mustRead(t, path)) {
				t.Fatal("pin replay rewrote legacy lock")
			}
			requireNoPackageWrites(t, f.home)
			for _, tc := range []struct {
				name   string
				mutate func(*manifest.Recipe)
			}{
				{"role", func(r *manifest.Recipe) { r.Role = manifest.RoleMemory }},
				{"local", func(r *manifest.Recipe) { r.Scope = &manifest.RecipeScope{Marker: ".sample"} }},
				{"wire", func(r *manifest.Recipe) { r.Wire.Tools = []string{"pi"} }},
				{"version", func(r *manifest.Recipe) { r.APIVersion = "patronus/v2" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					invalid := *rec
					tc.mutate(&invalid)
					cat.Recipes[0].Manifest = &invalid
					if err := applyLockPins(f.root, "sample-profile", "all", "", cat, nil); err == nil {
						t.Fatal("invalid catalog recipe accepted at pin replay")
					}
					if !bytes.Equal(before, mustRead(t, path)) {
						t.Fatal("refused replay rewrote lock")
					}
					requireNoPackageWrites(t, f.home)
				})
			}
			rec.Delivery.Assets[0].SHA256 = "corrupt"
			if err := lock.Save(path, pin); err != nil {
				t.Fatal(err)
			}
			cat.Recipes[0].Manifest = rec
			if err := applyLockPins(f.root, "sample-profile", "all", "", cat, nil); err == nil {
				t.Fatal("corrupt restored pin accepted")
			}
			requireNoPackageWrites(t, f.home)
		})
	}
}

func TestRecordStateSaveFailureIsOperationFailureForEveryTarget(t *testing.T) {
	for _, tool := range []string{"pi", "claude", "codex"} {
		t.Run(tool, func(t *testing.T) {
			f := dp02Setup(t)
			path := filepath.Join(f.home, "."+tool, "prompts", "fixture.md")
			if tool == "pi" {
				path = filepath.Join(f.home, ".pi/agent/prompts/fixture.md")
			}
			d := diff.FileDiff{Artifact: "fixture", Type: "command", Tool: tool, Scope: "global", Version: "1", Path: path, Action: diff.Create, After: []byte("initial fixture")}
			opts := deployOptions{home: f.home, projectDir: f.root, force: true, yes: true}
			res := toolpath.New(os.LookupEnv, f.home, f.root)
			cmd := newInstallCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := runDeploy(cmd, &diff.ChangeSet{Diffs: []diff.FileDiff{d}}, res, opts); err != nil {
				t.Fatal(err)
			}
			durablePath := statePath("global", opts)
			before := mustRead(t, durablePath)
			d.Before = d.After
			d.After = []byte("updated fixture")
			d.Action = diff.Conflict
			d.Version = "2"
			opts.saveState = func(string, *state.State) error { return errors.New("injected state persistence failure") }
			err := runDeploy(cmd, &diff.ChangeSet{Diffs: []diff.FileDiff{d}}, res, opts)
			if err == nil || !strings.Contains(err.Error(), "ownership uncertain") || !strings.Contains(err.Error(), path) {
				t.Fatalf("missing failure/result diagnostic: %v", err)
			}
			if !bytes.Equal(before, mustRead(t, durablePath)) || !bytes.Equal(d.After, mustRead(t, path)) {
				t.Fatal("failure lost durable state or committed bytes")
			}
			// A retry sees equality but cannot repair the stale ownership by adoption.
			opts.saveState = nil
			d.Before = d.After
			d.Action = diff.Skip
			if err := runDeploy(cmd, &diff.ChangeSet{Diffs: []diff.FileDiff{d}}, res, opts); err == nil {
				t.Fatal("equal retry adopted uncertain bytes")
			}
			loaded, err := state.Load(durablePath)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Items[0].ItemVersion != "1" {
				t.Fatal("uncertain install advanced version")
			}
		})
	}
}

func TestPiOwnershipUnsupportedStateRefusesBeforeAnyWrite(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-prompt", "command", "Fixture prompt\n")
	p := filepath.Join(f.home, ".patronus/state.json")
	dp02File(t, p, `{"version":99,"items":[]}`)
	before := dp02Snapshot(t, f.home, f.root)
	if _, _, err := runInstall(t, "fixture-prompt", "--target", "pi", "--global", "--deploy", "--yes"); err == nil || !strings.Contains(err.Error(), "unsupported state version") {
		t.Fatalf("unsupported state admitted: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("refusal mutated files")
	}
}

func TestPiOwnershipReplacementRefusesInvalidContinuity(t *testing.T) {
	for _, fault := range []string{"changed absolute path", "changed structural path", "missing prior", "invalid prior", "overlap", "drift"} {
		t.Run(fault, func(t *testing.T) {
			f := dp02Setup(t)
			path := filepath.Join(f.home, ".pi/agent/settings.json")
			edit := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "fixture", ScalarValue: "old"}
			before := []byte(`{"fixture":"old","user":"keep"}`)
			d := diff.FileDiff{Artifact: "fixture-setting", Type: "setting", Tool: "pi", Scope: "global", Version: "1", Path: path, Action: diff.Merge, After: before, Setting: edit}
			opts := deployOptions{home: f.home, projectDir: f.root, force: true, yes: true}
			old := state.FromChangeSet([]diff.FileDiff{d}, "first")
			switch fault {
			case "changed absolute path":
				d.Path = filepath.Join(f.home, ".pi/agent/other.json")
			case "changed structural path":
				copyEdit := *edit
				copyEdit.Dotted = "other"
				d.Setting = &copyEdit
			case "invalid prior":
				old[0].Files[0].Setting.PriorValue = "unproven"
			case "overlap":
				other := old[0]
				other.Artifact = "other-owner"
				old = append(old, other)
			case "drift":
				before = []byte(`{"fixture":"user edit","user":"keep"}`)
			}
			dp02File(t, path, string(before))
			if err := state.Save(statePath("global", opts), &state.State{Version: state.Version, Items: old}); err != nil {
				t.Fatal(err)
			}
			if fault == "missing prior" {
				sp := statePath("global", opts)
				raw := mustRead(t, sp)
				raw = bytes.Replace(raw, []byte(`"PriorPresent": false`), []byte(`"MissingPrior": false`), 1)
				dp02File(t, sp, string(raw))
			}
			d.Before = before
			d.After = []byte(`{"fixture":"new","user":"keep"}`)
			d.Version = "2"
			copyEdit := *d.Setting
			copyEdit.ScalarValue = "new"
			d.Setting = &copyEdit
			snapshot := dp02Snapshot(t, f.home, f.root)
			cmd := newInstallCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := runDeploy(cmd, &diff.ChangeSet{Diffs: []diff.FileDiff{d}}, toolpath.New(os.LookupEnv, f.home, f.root), opts); err == nil {
				t.Fatal("unsafe replacement admitted")
			}
			if !reflect.DeepEqual(snapshot, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("unsafe replacement mutated files")
			}
		})
	}
}

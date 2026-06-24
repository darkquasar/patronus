package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// runInstall executes the install command with args against the real repo
// (DiscoverRoot walks up from the cwd, which is this package's dir inside the
// repo). It returns combined stdout and the error.
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
	// Isolate HOME so --global resolves to an empty sandbox (not the developer's
	// real ~/.claude, where team-research may already be installed → SKIP not CREATE).
	t.Setenv("HOME", t.TempDir())
	out, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	for _, want := range []string{"team-research", "SKILL.md", "CREATE", "skill", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallVerboseShowsDiff(t *testing.T) {
	out, _, err := runInstall(t, "agent-principles", "--tool", "claude", "--local", "--verbose", "--dry-run")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	// agent-principles is an Instruction -> APPEND with a unified diff body.
	if !strings.Contains(out, "APPEND") {
		t.Errorf("expected APPEND:\n%s", out)
	}
	if !strings.Contains(out, "@@") {
		t.Errorf("verbose mode should show unified diff hunks:\n%s", out)
	}
}

func TestInstallMutuallyExclusiveScope(t *testing.T) {
	_, _, err := runInstall(t, "team-research", "--global", "--local")
	if err == nil {
		t.Error("expected error for --global and --local together")
	}
}

func TestInstallProfileCloudflareDryRun(t *testing.T) {
	out, errOut, err := runInstall(t, "--profile", "cloudflare", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("profile install failed: %v\n%s", err, errOut)
	}
	// The cloudflare profile spans instructions + capabilities + context + memory;
	// every populated slot's item should appear in the combined plan.
	for _, want := range []string{"agent-principles", "team-research", "team-implement", "pattern-cloudflare", "memory-ai-memory"} {
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
	_, _, err := runInstall(t, "team-research", "--profile", "golang")
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
	_, _, err := runInstall(t, "does-not-exist")
	if err == nil {
		t.Error("expected error for unknown artifact")
	}
}

func TestInstallDefaultIsDryRun(t *testing.T) {
	// No --deploy, no --dry-run: must be a safe dry run, no error, plan shown.
	out, _, err := runInstall(t, "team-research", "--tool", "claude", "--global")
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
	if err := recordState(applied, opts); err != nil {
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
	_, _, err := runInstall(t, "team-research", "--deploy", "--dry-run")
	if err == nil {
		t.Error("expected error for --deploy and --dry-run together")
	}
}

func TestInstallJSON(t *testing.T) {
	// Isolate HOME so --global is a clean sandbox (see TestInstallSkillDryRun).
	t.Setenv("HOME", t.TempDir())
	// --json is a persistent root flag; set it on the package global directly
	// since we run the subcommand in isolation here.
	jsonOutput = true
	defer func() { jsonOutput = false }()
	out, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--dry-run")
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
	// github is a remote http MCP recipe: pure MERGE, no fetch.
	out, _, err := runInstall(t, "github", "--tool", "claude", "--local", "--dry-run")
	if err != nil {
		t.Fatalf("install github failed: %v", err)
	}
	for _, want := range []string{"github", ".mcp.json", "MERGE", "mcp"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallRecipeFetchDryRun(t *testing.T) {
	// engram is a github-release recipe: FETCH the binary + MERGE per tool.
	out, _, err := runInstall(t, "memory-engram", "--tool", "all", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("install memory-engram failed: %v", err)
	}
	for _, want := range []string{"memory-engram", "FETCH", "engram", "MERGE"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
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

	// A NON-advisory exec (e.g. a mode: run recipe whose commands Patronus does run)
	// — proves runDeployWith executes it and records the provenance. (A mode: self
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
	ran, err := runExecs(cmd, cs, failRunner{})
	if err == nil {
		t.Fatal("expected failure")
	}
	if len(ran) != 0 {
		t.Errorf("no command should be recorded as run, got %v", ran)
	}
}

type failRunner struct{}

func (failRunner) Run([]string) error { return os.ErrPermission }

// TestComputePlanDispatchesPlugin proves a plugin name routes to plugin.Compute
// (not the artifact fall-through). The plugin is claude-native (a "claude-code"
// source), so the registration rides the claude adapter's setting MERGE and
// yields an applicable diff. The adapters + resolver are real (loadAdapters'
// builtin claude + a HOME-backed toolpath resolver) because post-fix
// plugin.Compute routes the registration through adapter.Engine.Transform and
// would hit a nil adapter without them.
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
			Meta:    manifest.Meta{APIVersion: manifest.APIVersion, Family: manifest.FamilyPlugin, Name: "superpowers"},
			Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Ref: "v2.1.0"}},
		}}},
	}

	cs, contribs, err := computePlan(planInputs{
		cat:      cat,
		adapters: adapterMap(adapters),
		res:      res,
		names:    []string{"superpowers"},
		tool:     "claude",
		scope:    "global",
	})
	if err != nil {
		t.Fatalf("computePlan: %v", err)
	}
	if cs == nil || len(cs.Diffs) == 0 {
		t.Fatal("expected a registration diff for the plugin, got none")
	}
	// The Contribution is the dry-run trust surface: it must come back so the plan
	// can state the per-target disposition (a claude-code source => native).
	if len(contribs) != 1 || len(contribs[0].contribs) != 1 {
		t.Fatalf("expected one plugin group with one contribution, got %+v", contribs)
	}
	if got := contribs[0].contribs[0]; got.Tool != "claude" || got.Mode != plugin.ModeNative {
		t.Errorf("contribution = %+v, want Tool=claude Mode=native", got)
	}
}

// TestComputePlanPluginAllExpandsTargets covers FIX C1+I1: a bare install (tool
// "all", no scope flag) must fan out to the plugin's Targets and honor the
// manifest's defaults.scope, registering on EVERY target instead of yielding zero
// diffs the way a single ResolveMode("all") call would.
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
			Meta:     manifest.Meta{APIVersion: manifest.APIVersion, Family: manifest.FamilyPlugin, Name: "superpowers"},
			Sources:  map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Ref: "v2.1.0"}},
			Targets:  []string{"claude", "codex"},
			Defaults: manifest.PluginDefaults{Scope: "global"},
		}}},
	}

	cs, contribs, err := computePlan(planInputs{
		cat:      cat,
		adapters: adapterMap(adapters),
		res:      res,
		names:    []string{"superpowers"},
		tool:     "all", // the default; must expand to Targets
		scope:    "",    // no flag; must fall back to defaults.scope=global
	})
	if err != nil {
		t.Fatalf("computePlan: %v", err)
	}
	if cs == nil || len(cs.Diffs) == 0 {
		t.Fatal("expected registration diffs for the plugin, got none")
	}
	for _, d := range cs.Diffs {
		if d.Scope != "global" {
			t.Errorf("diff %s scope = %q, want global (from defaults.scope)", d.Path, d.Scope)
		}
	}
	if len(contribs) != 1 || len(contribs[0].contribs) != 2 {
		t.Fatalf("expected one plugin group with two contributions (claude+codex), got %+v", contribs)
	}
	byTool := map[string]plugin.Mode{}
	for _, c := range contribs[0].contribs {
		byTool[c.Tool] = c.Mode
	}
	if byTool["claude"] != plugin.ModeNative {
		t.Errorf("claude mode = %q, want native", byTool["claude"])
	}
	if byTool["codex"] != plugin.ModeTranslate {
		t.Errorf("codex mode = %q, want translate (claude-code-only source)", byTool["codex"])
	}
}

// TestMergeSourcedNamesPlugin covers FIX I2: a file: reference to a plugin
// manifest must resolve into cat.Plugins and return the plugin's name, rather than
// failing with "resolved to nothing".
func TestMergeSourcedNamesPlugin(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "superpowers.yaml")
	body := "apiVersion: patronus/v2\n" +
		"family: plugin\n" +
		"role: lifecycle\n" +
		"name: superpowers\n" +
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
	names, err := mergeSourcedNames(context.Background(), cat, []string{"file:" + manifestPath}, t.TempDir())
	if err != nil {
		t.Fatalf("mergeSourcedNames: %v", err)
	}
	if len(names) != 1 || names[0] != "superpowers" {
		t.Fatalf("names = %v, want [superpowers]", names)
	}
	if len(cat.Plugins) != 1 || cat.Plugins[0].Manifest.Name != "superpowers" {
		t.Fatalf("cat.Plugins = %+v, want one superpowers plugin", cat.Plugins)
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
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
			Artifact: "s", Capability: "skill", Tool: "claude", Scope: "global"},
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

	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{
		Path: "ai-memory install-mcp --client claude --apply", Action: diff.Exec,
		Artifact: "memory-ai-memory", Capability: "self-wire", Tool: "claude", Scope: "global",
		Exec: &diff.ExecSpec{
			Command: []string{"ai-memory", "install-mcp", "--client", "claude", "--apply"},
			Display: "ai-memory install-mcp --client claude --apply",
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

package main

import (
	"bytes"
	"strings"
	"testing"
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

func TestInstallDeployRefusesUntilPhase3(t *testing.T) {
	// --deploy is the explicit write opt-in; until the applier exists it must
	// refuse (fail-safe), AND it must still print the plan first.
	out, _, err := runInstall(t, "team-research", "--tool", "claude", "--global", "--deploy")
	if err == nil {
		t.Fatal("--deploy must refuse until apply is implemented")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "CREATE") {
		t.Errorf("plan should be shown even when --deploy refuses:\n%s", out)
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

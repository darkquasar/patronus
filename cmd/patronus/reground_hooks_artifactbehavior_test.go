//go:build artifactbehavior

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runPlacedHook(t *testing.T, home, script string) (string, error) {
	t.Helper()
	p := filepath.Join(home, ".claude", "hooks", script)
	cmd := exec.CommandContext(context.Background(), "bash", p)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	return string(out), err
}

func TestPlacedHookScriptRunsAndListsSkills(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "--profile", "fix-all", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}

	out, err := runPlacedHook(t, home, "fix-hook-claude.sh")
	if err != nil {
		t.Fatalf("running the placed hook script: %v", err)
	}
	var emitted struct {
		InstalledSkills string `json:"installedSkills"`
	}
	if err := json.Unmarshal([]byte(out), &emitted); err != nil {
		t.Fatalf("hook output is not valid JSON: %v\n%s", err, out)
	}
	if !strings.Contains(emitted.InstalledSkills, "fix-skill") {
		t.Errorf("the placed hook should enumerate the skill installed beside it, got %q", emitted.InstalledSkills)
	}
}

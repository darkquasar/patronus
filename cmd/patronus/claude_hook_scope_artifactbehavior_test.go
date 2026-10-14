//go:build artifactbehavior

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Run the distributed payload at its installed hook path, not in its source
// directory. HOME, cwd and the selected scope deliberately have different bytes.
func claudeScopeHook(t *testing.T, home, cwd, scope, name, script string) string {
	t.Helper()
	raw := mustRead(t, filepath.Join(applicationSourceRoot(t), "artifacts", "hooks", name, script))
	placed := filepath.Join(scope, ".claude", "hooks", name+".sh")
	codexWrite(t, placed, raw)
	return claudeScopePlacedOutput(t, home, cwd, placed)
}

func claudeScopePlacedOutput(t *testing.T, home, cwd, placed string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "bash", placed)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("placed %s: %v\n%s", placed, err, out)
	}
	var emitted struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &emitted); err != nil {
		t.Fatalf("invalid hook JSON: %v\n%s", err, out)
	}
	if emitted.HookSpecificOutput.HookEventName == "" {
		t.Fatal("hook emitted no event")
	}
	return emitted.HookSpecificOutput.AdditionalContext
}

func TestClaudeHookScopeHeartbeatLocal(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	codexWrite(t, filepath.Join(home, ".claude", "skills", "global-only-cl", "SKILL.md"), []byte("global"))
	codexWrite(t, filepath.Join(project, ".claude", "skills", "local-only-cl", "SKILL.md"), []byte("local"))
	out := claudeScopeHook(t, home, home, project, "skills-heartbeat", "skills-heartbeat.sh")
	if !strings.Contains(out, "local-only-cl") || strings.Contains(out, "global-only-cl") {
		t.Fatalf("local heartbeat must list only its own scope, got %q", out)
	}
}

func TestClaudeHookScopeHeartbeatMissing(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	codexWrite(t, filepath.Join(home, ".claude", "skills", "global-only-cl", "SKILL.md"), []byte("global"))
	out := claudeScopeHook(t, home, home, project, "skills-heartbeat", "skills-heartbeat.sh")
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, filepath.Join(project, ".claude")) || strings.Contains(out, "global-only-cl") {
		t.Fatalf("missing selected skills directory must be visible without fallback, got %q", out)
	}
}

func TestClaudeHookScopeHeartbeatGlobalIgnoresProject(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	codexWrite(t, filepath.Join(home, ".claude", "skills", "global-only-cl", "SKILL.md"), []byte("global"))
	codexWrite(t, filepath.Join(project, ".claude", "skills", "project-shadow-cl", "SKILL.md"), []byte("project"))
	out := claudeScopeHook(t, home, project, home, "skills-heartbeat", "skills-heartbeat.sh")
	if !strings.Contains(out, "global-only-cl") || strings.Contains(out, "project-shadow-cl") {
		t.Fatalf("global heartbeat must ignore project input/cwd, got %q", out)
	}
}

func TestClaudeHookScopeDispatchLocal(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	// The old unsuffixed name is a rejected foreign/retired candidate.
	codexWrite(t, filepath.Join(home, ".claude", "skills", "skills-dispatch", "SKILL.md"), []byte("global legacy shadow"))
	codexWrite(t, filepath.Join(home, ".claude", "skills", "skills-dispatch-cl", "SKILL.md"), []byte("global current shadow"))
	local := "local dispatch: \"quoted\"\\path\nUnicode λ\tend"
	codexWrite(t, filepath.Join(project, ".claude", "skills", "skills-dispatch-cl", "SKILL.md"), []byte(local))
	out := claudeScopeHook(t, home, home, project, "skills-dispatch-activate-cl", "inject-skills-dispatch.sh")
	if !strings.Contains(out, local) || strings.Contains(out, "global") {
		t.Fatalf("local activation must inject its exact suffixed skill bytes, got %q", out)
	}
}

func TestClaudeHookScopeDispatchGlobalIgnoresProject(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	codexWrite(t, filepath.Join(home, ".claude", "skills", "skills-dispatch-cl", "SKILL.md"), []byte("global selected dispatch"))
	codexWrite(t, filepath.Join(project, ".claude", "skills", "skills-dispatch-cl", "SKILL.md"), []byte("malicious project shadow"))
	out := claudeScopeHook(t, home, project, home, "skills-dispatch-activate-cl", "inject-skills-dispatch.sh")
	if !strings.Contains(out, "global selected dispatch") || strings.Contains(out, "malicious project shadow") {
		t.Fatalf("global activation must ignore project shadow/cwd, got %q", out)
	}
}

func TestClaudeHookScopeDispatchMissing(t *testing.T) {
	for _, scope := range []string{"local", "global"} {
		t.Run(scope, func(t *testing.T) {
			home, project := t.TempDir(), t.TempDir()
			selected, other := project, home
			if scope == "global" {
				selected, other = home, project
			}
			codexWrite(t, filepath.Join(other, ".claude", "skills", "skills-dispatch-cl", "SKILL.md"), []byte("alternate scope must not leak"))
			codexWrite(t, filepath.Join(selected, ".claude", "skills", "skills-dispatch", "SKILL.md"), []byte("legacy alias must not leak"))
			out := claudeScopeHook(t, home, other, selected, "skills-dispatch-activate-cl", "inject-skills-dispatch.sh")
			if !strings.Contains(out, "unavailable") || !strings.Contains(out, filepath.Join(selected, ".claude")) || strings.Contains(out, "must not leak") {
				t.Fatalf("missing exact dispatch must be visible with no fallback/alias, got %q", out)
			}
		})
	}
}

func TestClaudeHookScopeNoFlagInstall(t *testing.T) {
	// Keep the real artifact payloads opt-in. The ordinary planner and real
	// adapter resolve no-flag defaults into invented HOME/project paths.
	root, home := fixtureCatalog(t), t.TempDir()
	project := root
	for _, rel := range []string{"artifacts/skills/skills-dispatch-cl", "artifacts/hooks/skills-dispatch-activate-cl", "artifacts/hooks/skills-heartbeat"} {
		copyDir(t, filepath.Join(applicationSourceRoot(t), rel), filepath.Join(root, rel))
	}
	t.Setenv("HOME", home)
	t.Chdir(root)
	_, stderr, err := runInstall(t, "skills-dispatch-activate-cl", "skills-heartbeat", "--target", "claude", "--local-registry", "--deploy", "--yes")
	if err != nil {
		t.Fatalf("no-flag install: %v\n%s", err, stderr)
	}
	for _, rel := range []string{".claude/skills/skills-dispatch-cl/SKILL.md", ".claude/hooks/skills-dispatch-activate-cl.sh", ".claude/hooks/skills-heartbeat.sh"} {
		if _, err := os.Stat(filepath.Join(project, rel)); err != nil {
			t.Fatalf("no-flag dependency/placement absent: %s", rel)
		}
		if _, err := os.Stat(filepath.Join(home, rel)); !os.IsNotExist(err) {
			t.Fatalf("no-flag install unexpectedly wrote global path %s: %v", rel, err)
		}
	}
	out := claudeScopePlacedOutput(t, home, home, filepath.Join(project, ".claude/hooks/skills-dispatch-activate-cl.sh"))
	body := string(mustRead(t, filepath.Join(project, ".claude/skills/skills-dispatch-cl/SKILL.md")))
	if !strings.Contains(out, strings.TrimSuffix(body, "\n")) || strings.Contains(out, "unavailable") {
		t.Fatalf("ordinary no-flag placement did not inject its matched dependency: %q", out)
	}
}

func TestClaudeHookScopeExplicitInstall(t *testing.T) {
	for _, scope := range []string{"local", "global"} {
		t.Run(scope, func(t *testing.T) {
			root, home := fixtureCatalog(t), t.TempDir()
			for _, rel := range []string{"artifacts/skills/skills-dispatch-cl", "artifacts/hooks/skills-dispatch-activate-cl", "artifacts/hooks/skills-heartbeat"} {
				copyDir(t, filepath.Join(applicationSourceRoot(t), rel), filepath.Join(root, rel))
			}
			t.Setenv("HOME", home)
			t.Chdir(root)
			_, stderr, err := runInstall(t, "skills-dispatch-activate-cl", "skills-heartbeat", "--target", "claude", "--local-registry", "--"+scope, "--deploy", "--yes")
			if err != nil {
				t.Fatalf("%s install: %v\n%s", scope, err, stderr)
			}
			selected, other := root, home
			if scope == "global" {
				selected, other = home, root
			}
			codexWrite(t, filepath.Join(other, ".claude/skills/skills-dispatch-cl/SKILL.md"), []byte("MALICIOUS_OTHER_SCOPE_BODY"))
			codexWrite(t, filepath.Join(other, ".claude/skills/other-scope-only-cl/SKILL.md"), []byte("other scope"))
			out := claudeScopePlacedOutput(t, home, other, filepath.Join(selected, ".claude/hooks/skills-dispatch-activate-cl.sh"))
			body := string(mustRead(t, filepath.Join(selected, ".claude/skills/skills-dispatch-cl/SKILL.md")))
			if !strings.Contains(out, strings.TrimSuffix(body, "\n")) || strings.Contains(out, "MALICIOUS_OTHER_SCOPE_BODY") {
				t.Fatalf("%s installed activation did not respect selected scope: %q", scope, out)
			}
			out = claudeScopePlacedOutput(t, home, other, filepath.Join(selected, ".claude/hooks/skills-heartbeat.sh"))
			if !strings.Contains(out, "skills-dispatch-cl") || strings.Contains(out, "other-scope-only-cl") {
				t.Fatalf("%s installed heartbeat did not respect selected scope: %q", scope, out)
			}
		})
	}
}

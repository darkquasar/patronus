//go:build artifactbehavior

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// These are invented payloads, not observed native Codex events. Execute only
// the self-contained script the ordinary adapter places, with a scratch HOME.
func placedCodexGuard(t *testing.T, name, scope string) (string, string) {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("..", "..", "artifacts", "hooks", name))
	if err != nil {
		t.Fatal(err)
	}
	art, err := manifest.LoadArtifact(filepath.Join(src, "patronus.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", "codex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	home, project, config := t.TempDir(), t.TempDir(), t.TempDir()
	env := func(k string) (string, bool) {
		if k == "CODEX_HOME" {
			return config, true
		}
		return "", false
	}
	eng := adapter.New(toolpath.New(env, home, project))
	ds, err := eng.Transform(art, ad, scope, src, func(string) ([]byte, bool, error) { return []byte("model = 'fixture-model'\n"), true, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 {
		t.Fatalf("selected hook emitted %d diffs", len(ds))
	}
	base := config
	if scope == "local" {
		base = filepath.Join(project, ".codex")
	}
	want := filepath.Join(base, "hooks", name+filepath.Ext(art.Hook.Script))
	if ds[0].Path != want || ds[0].Mode != 0755 {
		t.Fatalf("placement: %+v", ds[0])
	}
	if ds[1].Path != filepath.Join(base, "config.toml") || ds[1].Setting == nil || !strings.Contains(ds[1].Warning, "runtime-unverified") {
		t.Fatalf("registration: %+v", ds[1])
	}
	value, present, err := adapter.ReadDotted(ds[1].After, ad.Layout.Hook.ForScope(scope), "hooks.PreToolUse")
	if err != nil || !present {
		t.Fatalf("TOML hooks: %v %v", value, err)
	}
	groups, ok := value.([]any)
	if !ok || len(groups) != 1 {
		t.Fatalf("hook groups: %#v", value)
	}
	group := groups[0].(map[string]any)
	if group["matcher"] != art.Hook.Matcher {
		t.Fatalf("matcher: %#v", group)
	}
	handlers := group["hooks"].([]any)
	if handlers[0].(map[string]any)["command"] != strings.ReplaceAll(art.Hook.Command, "{script}", want) {
		t.Fatalf("registered wrong script: %#v", handlers)
	}
	if err := os.MkdirAll(filepath.Dir(want), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, ds[0].After, ds[0].Mode); err != nil {
		t.Fatal(err)
	}
	return want, home
}

func runCodexGuard(t *testing.T, script, home, cwd, payload string, env ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, script)
	cmd.Dir = cwd
	// Do not inherit real HOME or scanner configuration. PATH contains only test
	// prerequisites and (when provided) the invented scanner.
	prerequisites := t.TempDir()
	for _, name := range []string{"python3", "git"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("test prerequisite %s: %v", name, err)
		}
		if err := os.Symlink(path, filepath.Join(prerequisites, name)); err != nil {
			t.Fatal(err)
		}
	}
	cmd.Env = append([]string{"HOME=" + home, "PATH=" + prerequisites, "PYTHONDONTWRITEBYTECODE=1"}, env...)
	cmd.Stdin = strings.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("guard timeout: %v", ctx.Err())
	}
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return code, string(output)
}

func guardPayload(t *testing.T, tool string, input any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"tool_name": tool, "tool_input": input})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCodexBlockSecretsApplyPatchPayload(t *testing.T) {
	script, home := placedCodexGuard(t, "block-secrets-cx", "local")
	token := "ghp_" + strings.Repeat("A", 36) // invented, never an auth value
	patch := "*** Begin Patch\n*** Update File: fixture.txt\n@@\n+" + token + "\n*** End Patch\n"
	cases := []struct {
		name, payload string
		code          int
		diagnostic    string
	}{
		{"apply_patch", guardPayload(t, "apply_patch", map[string]any{"patch": patch}), 2, "BLOCKED"},
		{"Write_matcher_alias", `{"matcher":"Write","tool_name":"apply_patch","tool_input":{"input":` + string(mustJSON(t, patch)) + `}}`, 2, "BLOCKED"},
		{"freeform_patch", guardPayload(t, "apply_patch", patch), 2, "BLOCKED"},
		{"add_file_secret", guardPayload(t, "apply_patch", strings.ReplaceAll(patch, "Update File", "Add File")), 2, "patch"},
		{"valid_add_file_secret", guardPayload(t, "apply_patch", strings.ReplaceAll(strings.ReplaceAll(patch, "Update File", "Add File"), "@@\n", "")), 2, "BLOCKED"},
		{"rename_context_only", guardPayload(t, "apply_patch", "*** Begin Patch\n*** Update File: fixture.txt\n*** Move to: new.txt\n@@\n "+token+"\n+benign\n*** End of File\n*** End Patch\n"), 0, ""},
		{"benign_patch", guardPayload(t, "apply_patch", map[string]any{"patch": strings.ReplaceAll(patch, token, "fixture-nonce")}), 0, ""},
		{"removed_secret", guardPayload(t, "apply_patch", map[string]any{"patch": strings.ReplaceAll(patch, "+"+token, "-"+token+"\n+removed")}), 0, ""},
		{"context_secret", guardPayload(t, "apply_patch", map[string]any{"patch": strings.ReplaceAll(patch, "+"+token, " "+token+"\n+benign")}), 0, ""},
		{"delete_file", guardPayload(t, "apply_patch", map[string]any{"patch": "*** Begin Patch\n*** Delete File: fixture.txt\n*** End Patch\n"}), 0, ""},
		{"Write", guardPayload(t, "Write", map[string]any{"content": token}), 2, "BLOCKED"},
		{"Edit", guardPayload(t, "Edit", map[string]any{"old_string": "fixture", "new_string": token}), 2, "BLOCKED"},
		{"MultiEdit", guardPayload(t, "MultiEdit", map[string]any{"edits": []any{map[string]any{"new_string": token}}}), 2, "BLOCKED"},
		{"legacy_benign", guardPayload(t, "Edit", map[string]any{"old_string": token, "new_string": "removed"}), 0, ""},
		{"empty_write", guardPayload(t, "Write", map[string]any{"content": ""}), 0, ""},
		{"malformed_json", "{", 2, "payload"},
		{"missing_patch", guardPayload(t, "apply_patch", map[string]any{}), 2, "payload"},
		{"malformed_patch", guardPayload(t, "apply_patch", "not a patch"), 2, "patch"},
		{"invalid_MultiEdit", guardPayload(t, "MultiEdit", map[string]any{"edits": []any{map[string]any{"old_string": "x"}}}), 2, "payload"},
		{"unsupported_matched_tool", guardPayload(t, "unknown_writer", map[string]any{"content": "benign"}), 2, "unsupported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, output := runCodexGuard(t, script, home, home, tc.payload)
			if code != tc.code || !strings.Contains(output, tc.diagnostic) {
				t.Fatalf("exit=%d want=%d output=%q diagnostic=%q", code, tc.code, output, tc.diagnostic)
			}
			if strings.Contains(output, token) {
				t.Fatal("diagnostic leaked token content")
			}
		})
	}
}

func TestCodexBlockSecretsPatterns(t *testing.T) {
	script, home := placedCodexGuard(t, "block-secrets-cx", "global")
	tokens := []struct{ name, token string }{
		{"PEM", "-----BEGIN PRIVATE KEY-----"},
		{"AWS", "AKIA" + strings.Repeat("A", 16)},
		{"AWS_temporary", "ASIA" + strings.Repeat("A", 16)},
		{"GitLab", "glpat-" + strings.Repeat("A", 20)},
		{"Slack", "xoxb-" + strings.Repeat("A", 10)},
		{"OpenAI_style", "sk-" + strings.Repeat("A", 32)},
		{"Google", "AIza" + strings.Repeat("A", 35)},
	}
	for _, tc := range tokens {
		t.Run(tc.name, func(t *testing.T) {
			code, output := runCodexGuard(t, script, home, home, guardPayload(t, "Write", map[string]any{"content": tc.token}))
			if code != 2 || !strings.Contains(output, "BLOCKED") || strings.Contains(output, tc.token) {
				t.Fatalf("exit=%d output=%q", code, output)
			}
		})
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func scratchGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("scratch git %v: %v\n%s", args, err, output)
	}
}

func TestCodexGitleaksGuardBashEvent(t *testing.T) {
	script, home := placedCodexGuard(t, "gitleaks-guard-cx", "global")
	repo := t.TempDir()
	scratchGit(t, repo, "init", "-q")
	staged := "invented staged fixture nonce\n"
	if err := os.WriteFile(filepath.Join(repo, "fixture.txt"), []byte(staged), 0644); err != nil {
		t.Fatal(err)
	}
	scratchGit(t, repo, "add", "fixture.txt")
	if err := os.WriteFile(filepath.Join(repo, "fixture.txt"), []byte("unstaged nonce\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin, capture := t.TempDir(), filepath.Join(t.TempDir(), "scanner-input")
	fake := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$CAPTURE.args\"\ncat > \"$CAPTURE\"\nexit \"${FAKE_EXIT:-0}\"\n"
	if err := os.WriteFile(filepath.Join(bin, "gitleaks"), []byte(fake), 0755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, command     string
		scannerExit, code int
		diagnostic        string
		scan              bool
	}{
		{"benign_commit", "git commit -m fixture", 0, 0, "", true},
		{"found_secret", "git commit -m fixture", 1, 2, "BLOCKED", true},
		{"scanner_failure", "git commit -m fixture", 3, 2, "scanner", true},
		{"noncommit", "git status", 1, 0, "", false},
		{"quoted_noncommit", "echo 'git commit'", 1, 0, "", false},
		{"compound_commit", "git status && git commit -m fixture", 0, 0, "", true},
		{"global_option", "git -c user.name=Fixture commit -m fixture", 0, 0, "", true},
		{"git_C", "git -C '" + repo + "' commit -m fixture", 0, 0, "", true},
		{"cd_commit", "cd '" + repo + "' && git commit -m fixture", 0, 0, "", true},
		{"newline_commit", "git status\ngit commit -m fixture", 0, 0, "", true},
		{"unsupported_override", "git --work-tree=other commit -m fixture", 0, 2, "unsupported", false},
		{"unsupported_override_value", "git --git-dir other commit -m fixture", 0, 2, "unsupported", false},
		{"unsupported_config", "git -c core.worktree=other commit -m fixture", 0, 2, "unsupported", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(capture)
			code, output := runCodexGuard(t, script, home, repo, guardPayload(t, "Bash", map[string]any{"command": tc.command}), "PATH="+bin+":/usr/bin:/bin", "CAPTURE="+capture, "FAKE_EXIT="+string(mustJSON(t, tc.scannerExit)), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
			if code != tc.code || !strings.Contains(output, tc.diagnostic) {
				t.Fatalf("exit=%d want=%d output=%q", code, tc.code, output)
			}
			b, err := os.ReadFile(capture)
			if tc.scan {
				if err != nil || !bytes.Contains(b, []byte("+invented staged fixture nonce")) || bytes.Contains(b, []byte("unstaged nonce")) {
					t.Fatalf("not staged diff: %q %v", b, err)
				}
				args := string(mustRead(t, capture+".args"))
				if !strings.Contains(args, "stdin") || !strings.Contains(args, "--exit-code 1") {
					t.Fatal(args)
				}
			} else if err == nil {
				t.Fatal("noncommit invoked scanner")
			}
		})
	}
	t.Run("git_C_from_other_cwd", func(t *testing.T) {
		code, output := runCodexGuard(t, script, home, home, guardPayload(t, "Bash", map[string]any{"command": "git -C '" + repo + "' commit -m fixture"}), "PATH="+bin+":/usr/bin:/bin", "CAPTURE="+capture)
		if code != 0 {
			t.Fatalf("exit=%d output=%q", code, output)
		}
		if !bytes.Contains(mustRead(t, capture), []byte("+invented staged fixture nonce")) {
			t.Fatal("wrong repository scanned")
		}
	})
	t.Run("placed_scanner_preferred", func(t *testing.T) {
		dir := filepath.Join(home, ".patronus", "bin")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "gitleaks"), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dir)
		code, output := runCodexGuard(t, script, home, repo, guardPayload(t, "Bash", map[string]any{"command": "git commit -m fixture"}), "PATH="+bin+":/usr/bin:/bin", "CAPTURE="+capture)
		if code != 2 || !strings.Contains(output, "BLOCKED") {
			t.Fatalf("exit=%d output=%q", code, output)
		}
	})
	t.Run("missing_scanner", func(t *testing.T) {
		code, output := runCodexGuard(t, script, home, repo, guardPayload(t, "Bash", map[string]any{"command": "git commit -m fixture"}))
		if code != 2 || !strings.Contains(output, "install") {
			t.Fatalf("exit=%d output=%q", code, output)
		}
	})
	t.Run("malformed_payload", func(t *testing.T) {
		code, output := runCodexGuard(t, script, home, repo, `{"tool_name":"Bash","tool_input":{}}`)
		if code != 2 || !strings.Contains(output, "payload") {
			t.Fatalf("exit=%d output=%q", code, output)
		}
	})
	t.Run("git_failure", func(t *testing.T) {
		code, output := runCodexGuard(t, script, home, home, guardPayload(t, "Bash", map[string]any{"command": "git commit -m fixture"}), "PATH="+bin+":/usr/bin:/bin", "CAPTURE="+capture)
		if code != 2 || !strings.Contains(output, "staged diff") {
			t.Fatalf("exit=%d output=%q", code, output)
		}
	})
}

func TestCodexGuardArtifactsPlacement(t *testing.T) {
	for _, name := range []string{"block-secrets-cx", "gitleaks-guard-cx"} {
		for _, scope := range []string{"global", "local"} {
			t.Run(name+"/"+scope, func(t *testing.T) {
				script, home := placedCodexGuard(t, name, scope)
				input := guardPayload(t, "Write", map[string]any{"content": "fixture-nonce"})
				if name == "gitleaks-guard-cx" {
					input = guardPayload(t, "Bash", map[string]any{"command": "echo fixture-nonce"})
				}
				code, output := runCodexGuard(t, script, home, home, input)
				if code != 0 {
					t.Fatalf("placed script exit=%d output=%q", code, output)
				}
			})
		}
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHardenedProfileSandboxFlavourDiverges is the §6b acceptance for P7.5.5: the
// hardened profile (extends core) wires the L6 sandbox layer with a per-tool
// flavour, and the resolver picks exactly one per --tool. THIS is the proof that
// the P7.4 sandbox list-ify + the P7.5 settings-merge compose:
//
//	claude   -> native settings `sandbox` object in settings.json
//	codex    -> native `sandbox_mode = workspace-write` in config.toml
//	opencode -> the sandbox-runtime (srt) install advisory (no native switch)
func TestHardenedProfileSandboxFlavourDiverges(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		f := builtRegistry(t)
		home := withRemoteEnv(t, f)
		withFakeRunner(t)
		stubBinary(t, home, "gitleaks")
		stubBinary(t, home, "bd") // core wires beads -> requires bd (github-release FETCH SKIPs offline)

		if _, e, err := runInstall(t, "--profile", "hardened", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
			t.Fatalf("install: %v\n%s", err, e)
		}
		root := map[string]any{}
		if err := json.Unmarshal(mustRead(t, filepath.Join(home, ".claude", "settings.json")), &root); err != nil {
			t.Fatalf("settings.json: %v", err)
		}
		sb, ok := root["sandbox"].(map[string]any)
		if !ok || sb["enabled"] != true {
			t.Errorf("claude should get the native sandbox settings object: %v", root["sandbox"])
		}
		// Codex/opencode flavours must NOT have leaked.
		if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); err == nil {
			t.Error("claude install should not write codex config")
		}
	})

	t.Run("codex", func(t *testing.T) {
		f := builtRegistry(t)
		home := withRemoteEnv(t, f)
		withFakeRunner(t)
		stubBinary(t, home, "gitleaks")
		stubBinary(t, home, "bd") // core wires beads -> requires bd (github-release FETCH SKIPs offline)

		if _, e, err := runInstall(t, "--profile", "hardened", "--tool", "codex", "--global", "--deploy", "--yes"); err != nil {
			t.Fatalf("install: %v\n%s", err, e)
		}
		toml := string(mustRead(t, filepath.Join(home, ".codex", "config.toml")))
		if !strings.Contains(toml, "sandbox_mode") || !strings.Contains(toml, "workspace-write") {
			t.Errorf("codex should get sandbox_mode = workspace-write:\n%s", toml)
		}
	})

	t.Run("opencode", func(t *testing.T) {
		f := builtRegistry(t)
		home := withRemoteEnv(t, f)
		withFakeRunner(t)
		stubBinary(t, home, "gitleaks")
		stubBinary(t, home, "bd") // core wires beads -> requires bd (github-release FETCH SKIPs offline)

		out, e, err := runInstall(t, "--profile", "hardened", "--tool", "opencode", "--global", "--deploy", "--yes")
		if err != nil {
			t.Fatalf("install: %v\n%s", err, e)
		}
		// opencode's flavour is the srt install advisory (no native settings switch).
		if !strings.Contains(out, "npm install -g @anthropic-ai/sandbox-runtime") {
			t.Errorf("opencode should get the sandbox-runtime install advisory:\n%s", out)
		}
	})
}

// TestHardIsolationAddsMicrosandbox proves the hard-isolation overlay (extends
// hardened) adds the microsandbox MCP for every tool uniformly — the microVM tier
// on top of the per-tool OS sandboxes.
func TestHardIsolationAddsMicrosandbox(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	withFakeRunner(t)
	stubBinary(t, home, "gitleaks")
	stubBinary(t, home, "bd") // core wires beads -> requires bd (github-release FETCH SKIPs offline)

	out, e, err := runInstall(t, "--profile", "hard-isolation", "--tool", "claude", "--global", "--dry-run", "--verbose")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, e)
	}
	// microsandbox MCP wires into the Claude MCP config; it also keeps the native
	// sandbox (inherited from hardened).
	if !strings.Contains(out, "microsandbox") {
		t.Errorf("hard-isolation should wire microsandbox:\n%s", out)
	}
	if !strings.Contains(out, "sandbox") {
		t.Errorf("hard-isolation should keep the native sandbox from hardened:\n%s", out)
	}
}

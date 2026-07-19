package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// This is the §6b acceptance gate for P7.5.1 (the settings-merge primitive): it
// drives the REAL CLI (build → serve → install → idempotent re-install → remove)
// in a temp dir against a served catalog, proving a type:hook artifact MERGEs a
// matcher-group into the agent's settings.json and reverts by stripping exactly
// that element. The fixture is throwaway (injected into the served index + a
// hand-built tarball) — no repo artifact is authored, per Build Constraint §0.
// The real first-class hooks (tdd-guard, the guardrails) land in P7.5.2+.

// hookManifestYAML is the portable source of a trivial hook artifact: a
// PreToolUse command hook with no body file (a hook is pure settings data).
const hookManifestYAML = `apiVersion: patronus/v2
family: artifact
type: hook
role: eval
name: smoke-hook
description: smoke pre-tool hook
version: 1.0.0
targets: [claude, codex, opencode]
defaults:
  scope: global
hook:
  event: PreToolUse
  matcher: Edit|Write
  command: smoke-guard --check
`

// serveHookFixture builds the FIXTURE baseline registry, injects the throwaway
// smoke-hook artifact (+ its tarball) into the served index, and refreshes the
// client cache. Returns the temp HOME.
//
// The baseline is the fixture catalog, not the real one. This file already INVENTS
// its item (smoke-hook) — it only ever needed a valid catalog to inject into, and
// building the real one dragged every real recipe, and every real upstream pin, in
// for no reason.
func serveHookFixture(t *testing.T) string {
	t.Helper()
	outDir := t.TempDir()
	t.Chdir(fixtureCatalog(t)) // baseline = the fixture; no real pins inherited
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	home := withRemoteEnv(t, f)

	tgz := mustTarGz(t, map[string][]byte{"patronus.yaml": []byte(hookManifestYAML)})
	url := testRegistryBase + "/catalog/smoke-hook/1.0.0/smoke-hook-1.0.0.tar.gz"

	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	ix.Artifacts = append(ix.Artifacts, registry.IndexArtifact{
		Manifest: &manifest.Artifact{
			Meta:     manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyArtifact, Name: "smoke-hook", Description: "smoke pre-tool hook", Version: "1.0.0", Role: manifest.RoleEval},
			Type:     manifest.TypeHook,
			Targets:  []string{"claude", "codex", "opencode"},
			Defaults: manifest.ArtifactDefaults{Scope: "global"},
			Hook:     &manifest.HookSpec{Event: "PreToolUse", Matcher: "Edit|Write", Command: "smoke-guard --check"},
		},
		Tarball: registry.Tarball{URL: url, SHA256: shaOf(tgz)},
	})
	mutated, err := ix.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")
	f.bodies[url] = tgz

	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}
	return home
}

// preToolUseHooks decodes the hooks.PreToolUse matcher-group array from a
// settings.json file.
func preToolUseHooks(t *testing.T, path string) []any {
	t.Helper()
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, path), &root); err != nil {
		t.Fatalf("decode settings.json: %v", err)
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		return nil
	}
	list, _ := hooks["PreToolUse"].([]any)
	return list
}

// TestHookArtifactMergesIntoClaudeSettings proves the primitive end-to-end on
// Claude: install MERGEs the hook into ~/.claude/settings.json, it is idempotent,
// state records the MERGE, and remove strips exactly the element (the file
// survives, the user's other settings untouched).
func TestHookArtifactMergesIntoClaudeSettings(t *testing.T) {
	home := serveHookFixture(t)

	// Seed a pre-existing user setting the install must preserve.
	settings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte("{\n  \"model\": \"opus\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, errOut, err := runInstall(t, "smoke-hook", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	list := preToolUseHooks(t, settings)
	if len(list) != 1 {
		t.Fatalf("want 1 PreToolUse matcher-group, got %d", len(list))
	}
	grp := list[0].(map[string]any)
	if grp["matcher"] != "Edit|Write" {
		t.Errorf("matcher = %v, want Edit|Write", grp["matcher"])
	}
	inner := grp["hooks"].([]any)[0].(map[string]any)
	if inner["command"] != "smoke-guard --check" {
		t.Errorf("command = %v, want smoke-guard --check", inner["command"])
	}
	// The user's pre-existing setting is preserved.
	root := map[string]any{}
	_ = json.Unmarshal(mustRead(t, settings), &root)
	if root["model"] != "opus" {
		t.Errorf("user setting clobbered: %v", root)
	}

	// state.json recorded the hook MERGE.
	st := string(mustRead(t, filepath.Join(home, ".patronus", "state.json")))
	if !strings.Contains(st, "smoke-hook") {
		t.Errorf("state missing smoke-hook:\n%s", st)
	}

	// Idempotent re-run → SKIP.
	out, _, err := runInstall(t, "smoke-hook", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// remove strips exactly our element; settings.json survives with the user key.
	if _, errOut, err := execRemove(t, "smoke-hook", "--global", "--deploy"); err != nil {
		t.Fatalf("remove: %v\n%s", err, errOut)
	}
	if got := preToolUseHooks(t, settings); len(got) != 0 {
		t.Errorf("hook element not stripped on remove: %v", got)
	}
	root = map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings.json gone or corrupt after remove: %v", err)
	}
	if root["model"] != "opus" {
		t.Errorf("user setting lost after remove: %v", root)
	}
}

// TestHookArtifactSkipsToolsWithoutHookSurface proves the honest per-tool
// divergence: Codex/OpenCode model no hook surface, so installing the same hook
// there writes no settings file and is a clean no-op (not an error).
func TestHookArtifactSkipsToolsWithoutHookSurface(t *testing.T) {
	for _, tool := range []string{"codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			home := serveHookFixture(t)
			out, errOut, err := runInstall(t, "smoke-hook", "--tool", tool, "--global", "--deploy", "--yes")
			if err != nil {
				t.Fatalf("install: %v\n%s", err, errOut)
			}
			// Nothing to write — the plan is empty of file changes for this artifact.
			if strings.Contains(out, "MERGE") {
				t.Errorf("%s: hook should produce no MERGE (no hook surface):\n%s", tool, out)
			}
			// No Claude settings leaked into this tool's tree either.
			if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err == nil {
				t.Errorf("%s: unexpected claude settings.json written", tool)
			}
		})
	}
}

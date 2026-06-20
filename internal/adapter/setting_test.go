package adapter

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func settingArtifact(name, path string, value any) *manifest.Artifact {
	return &manifest.Artifact{
		Meta:    manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleObservability},
		Type:    manifest.TypeSetting,
		Setting: &manifest.SettingSpec{Path: path, Value: value},
	}
}

// A setting artifact MERGEs its value at the dotted path in settings.json,
// preserving surrounding keys, and is idempotent.
func TestTransformSettingClaudeMerges(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))

	art := settingArtifact("ccusage-statusline", "statusLine",
		map[string]any{"type": "command", "command": "ccusage statusline"})

	// Seed an existing user setting the merge must preserve.
	existing := []byte("{\n  \"model\": \"opus\"\n}\n")
	diffs, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", existingBytes(existing))
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Action != diff.Merge {
		t.Fatalf("want 1 MERGE, got %+v", diffs)
	}
	want := filepath.Join(home, ".claude", "settings.json")
	if diffs[0].Path != want {
		t.Errorf("path = %q, want %q", diffs[0].Path, want)
	}
	// A scalar setting carries a SCALAR SettingEdit (IdentityKey "") so the planner
	// re-folds it onto accumulated config and remove deletes exactly its key.
	if diffs[0].Setting == nil || diffs[0].Setting.IdentityKey != "" {
		t.Errorf("scalar setting should carry a scalar SettingEdit (IdentityKey \"\"), got %+v", diffs[0].Setting)
	}
	if diffs[0].Setting != nil && diffs[0].Setting.Dotted != "statusLine" {
		t.Errorf("setting edit dotted = %q, want statusLine", diffs[0].Setting.Dotted)
	}

	root := map[string]any{}
	if err := json.Unmarshal(diffs[0].After, &root); err != nil {
		t.Fatal(err)
	}
	if root["model"] != "opus" {
		t.Errorf("user setting clobbered: %v", root)
	}
	sl, ok := root["statusLine"].(map[string]any)
	if !ok || sl["command"] != "ccusage statusline" {
		t.Errorf("statusLine not set correctly: %v", root["statusLine"])
	}
}

// settingFor applies per-tool overrides (path + value) over the base setting, so
// ONE artifact carries divergent native switches (native-sandbox: Claude's object
// vs Codex's string).
func TestSettingForPerToolOverrides(t *testing.T) {
	art := settingArtifact("native-sandbox", "sandbox", map[string]any{"enabled": true})
	art.Overrides = map[string]map[string]interface{}{
		"codex": {"path": "sandbox_mode", "value": "workspace-write"},
	}

	// Base (claude, no override): the sandbox object at `sandbox`.
	p, v := settingFor(art, "claude")
	if p != "sandbox" {
		t.Errorf("claude path = %q, want sandbox", p)
	}
	if m, ok := v.(map[string]any); !ok || m["enabled"] != true {
		t.Errorf("claude value = %v, want the sandbox object", v)
	}

	// Codex override: a different path AND value.
	p, v = settingFor(art, "codex")
	if p != "sandbox_mode" || v != "workspace-write" {
		t.Errorf("codex override = (%q, %v), want (sandbox_mode, workspace-write)", p, v)
	}
}

// On a tool with no setting layout (OpenCode — Codex models config.toml now), a
// setting artifact is an honest no-op, so a flavoured setting diverges cleanly.
func TestTransformSettingNoSurfaceSkips(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := settingArtifact("ccusage-statusline", "statusLine", "x")

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Errorf("opencode: unexpected error %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("opencode: want 0 diffs (no setting surface), got %d", len(diffs))
	}
}

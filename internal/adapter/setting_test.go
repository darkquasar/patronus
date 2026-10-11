package adapter

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestComputedSettingEmitsNoAdapterDiff(t *testing.T) {
	eng := New(toolpath.New(testEnv(t.TempDir()), t.TempDir(), t.TempDir()))
	art := settingArtifact("invented-mask", "permission.skill", nil)
	art.Setting.Computed = "skill-mask"
	diffs, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 0 {
		t.Fatalf("computed setting emitted placeholder: %+v", diffs)
	}
}

func TestSplitLeavesEmitQuotedKeys(t *testing.T) {
	ad, err := manifest.DecodeAdapter([]byte("apiVersion: patronus/v2\nfamily: adapter\ntool: toy\nlayout:\n  setting:\n    project: {file: toy.json, format: json}\n    splitLeaves: [permission.edit]\n"))
	if err != nil {
		t.Fatal(err)
	}
	eng := New(toolpath.New(testEnv(t.TempDir()), t.TempDir(), t.TempDir()))
	art := settingArtifact("invented-defaults", "permission.edit", map[string]any{"*": "allow", "*.env": "deny", ".env.*": "deny"})
	before := []byte(`{"permission":{"edit":{"*.secret":"ask"},"bash":"ask"}}`)
	diffs, err := eng.Transform(art, ad, "local", "", existingBytes(before))
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 3 {
		t.Fatalf("want three leaves, got %d", len(diffs))
	}
	merged := before
	for _, d := range diffs {
		if d.Setting == nil || d.Setting.PriorPresent {
			t.Fatalf("not independent new leaf: %+v", d.Setting)
		}
		merged, err = ApplySettingEdit(merged, d.Setting)
		if err != nil {
			t.Fatal(err)
		}
	}
	var root map[string]any
	if err := json.Unmarshal(merged, &root); err != nil {
		t.Fatal(err)
	}
	permission := root["permission"].(map[string]any)
	edit := permission["edit"].(map[string]any)
	if len(edit) != 4 || edit["*"] != "allow" || edit["*.env"] != "deny" || edit[".env.*"] != "deny" || edit["*.secret"] != "ask" || permission["bash"] != "ask" {
		t.Fatalf("wrong written keys/siblings: %s", merged)
	}
	ad.Layout.Setting.SplitLeaves = nil
	legacy, err := eng.Transform(art, ad, "local", "", noExisting)
	if err != nil || len(legacy) != 1 || legacy[0].Setting.Dotted != "permission.edit" {
		t.Fatalf("legacy object changed: %+v, %v", legacy, err)
	}
}

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

// A scalar setting captures whatever the user already had at that key, so remove
// restores their value instead of deleting the key outright.
func TestSettingDiffCapturesPrior(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := settingArtifact("ccusage-statusline", "statusLine",
		map[string]any{"type": "command", "command": "ccusage statusline"})

	// The user already configured a statusLine of their own.
	existing := []byte(`{"statusLine":{"type":"command","command":"my-own"}}`)
	diffs, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", existingBytes(existing))
	if err != nil {
		t.Fatal(err)
	}
	d := diffs[0]

	if d.Setting == nil {
		t.Fatal("Setting is nil")
	}
	if !d.Setting.PriorPresent {
		t.Fatal("PriorPresent = false; the user's existing statusLine was not captured")
	}
	prior, ok := d.Setting.PriorValue.(map[string]any)
	if !ok || prior["command"] != "my-own" {
		t.Fatalf("PriorValue = %#v, want the user's statusLine; remove would delete it", d.Setting.PriorValue)
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

// On a tool with no setting layout, a setting artifact is an honest no-op, so a
// flavoured setting diverges cleanly. Every shipped adapter now models a
// setting surface, so the case is proven over an invented adapter.
func TestTransformSettingNoSurfaceSkips(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := settingArtifact("ccusage-statusline", "statusLine", "x")
	noSurface, err := manifest.DecodeAdapter([]byte(`apiVersion: patronus/v2
family: adapter
tool: toy
layout:
  skill:
    global: "~/.toy/skills/{name}/SKILL.md"
    project: ".toy/skills/{name}/SKILL.md"
`))
	if err != nil {
		t.Fatal(err)
	}

	diffs, err := eng.Transform(art, noSurface, "global", "", noExisting)
	if err != nil {
		t.Errorf("no setting surface: unexpected error %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("no setting surface: want 0 diffs, got %d", len(diffs))
	}
}

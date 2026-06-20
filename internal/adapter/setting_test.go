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

// On a tool with no setting layout (codex/opencode), a setting artifact is an
// honest no-op — so a @claude-flavoured statusline diverges cleanly.
func TestTransformSettingNoSurfaceSkips(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := settingArtifact("ccusage-statusline", "statusLine", "x")

	for _, tool := range []string{"codex", "opencode"} {
		diffs, err := eng.Transform(art, loadAdapter(t, tool), "global", "", noExisting)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tool, err)
		}
		if len(diffs) != 0 {
			t.Errorf("%s: want 0 diffs (no setting surface), got %d", tool, len(diffs))
		}
	}
}

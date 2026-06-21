package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func hookArtifact(name, event, matcher, command string) *manifest.Artifact {
	return &manifest.Artifact{
		Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleEval},
		Type: manifest.TypeHook,
		Hook: &manifest.HookSpec{Event: event, Matcher: matcher, Command: command},
	}
}

// hooksAt decodes the matcher-group array at hooks.{event} from settings bytes.
func hooksAt(t *testing.T, b []byte, event string) []any {
	t.Helper()
	root := map[string]any{}
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("decode settings: %v\n%s", err, b)
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		return nil
	}
	list, _ := hooks[event].([]any)
	return list
}

// On Claude a hook artifact MERGEs one matcher-group into settings.json at
// hooks.{event}, stamped with a patronus identity, with the command nested in the
// inner hooks array.
func TestTransformHookClaudeMerges(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))

	art := hookArtifact("tdd-guard", "PreToolUse", "Edit|Write", "tdd-guard")
	diffs, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff, got %d", len(diffs))
	}
	d := diffs[0]
	if d.Action != diff.Merge {
		t.Errorf("action = %s, want MERGE", d.Action)
	}
	want := filepath.Join(home, ".claude", "settings.json")
	if d.Path != want {
		t.Errorf("path = %q, want %q", d.Path, want)
	}
	if d.Setting == nil {
		t.Fatal("hook diff carries no SettingEdit")
	}
	if d.Setting.Dotted != "hooks.PreToolUse" {
		t.Errorf("dotted = %q, want hooks.PreToolUse", d.Setting.Dotted)
	}
	if d.Setting.IdentityKey != patronusHookID || d.Setting.Identity == "" {
		t.Errorf("identity not stamped: %+v", d.Setting)
	}

	list := hooksAt(t, d.After, "PreToolUse")
	if len(list) != 1 {
		t.Fatalf("want 1 matcher-group, got %d", len(list))
	}
	grp := list[0].(map[string]any)
	if grp["matcher"] != "Edit|Write" {
		t.Errorf("matcher = %v, want Edit|Write", grp["matcher"])
	}
	if grp[patronusHookID] != d.Setting.Identity {
		t.Errorf("element id %v != edit identity %v", grp[patronusHookID], d.Setting.Identity)
	}
	inner := grp["hooks"].([]any)[0].(map[string]any)
	if inner["type"] != "command" || inner["command"] != "tdd-guard" {
		t.Errorf("inner handler wrong: %v", inner)
	}
}

// A hook is idempotent: transforming against settings that already contain its
// element produces identical bytes (SKIP-worthy).
func TestTransformHookIdempotent(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("tdd-guard", "PreToolUse", "Edit", "tdd-guard")

	first, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", noExisting)
	if err != nil {
		t.Fatal(err)
	}
	prior := first[0].After
	second, err := eng.Transform(art, loadAdapter(t, "claude"), "global", "", existingBytes(prior))
	if err != nil {
		t.Fatal(err)
	}
	if string(second[0].After) != string(prior) {
		t.Errorf("re-merge not idempotent:\n%s\nvs\n%s", second[0].After, prior)
	}
}

// Codex/OpenCode model no hook surface (null Hook target), so a hook artifact is
// an honest no-op there rather than an error — a cross-tool profile installs
// cleanly and only hook-capable tools get the hook.
func TestTransformHookNoSurfaceSkips(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("tdd-guard", "PreToolUse", "Edit", "tdd-guard")

	for _, tool := range []string{"codex", "opencode"} {
		diffs, err := eng.Transform(art, loadAdapter(t, tool), "global", "", noExisting)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tool, err)
		}
		if len(diffs) != 0 {
			t.Errorf("%s: want 0 diffs (no hook surface), got %d", tool, len(diffs))
		}
	}
}

// A script-bearing hook places its helper script (CREATE, executable) into the
// tool's hook-script dir AND registers a hook whose command invokes the placed
// path (the {script} token resolves to it).
func TestTransformHookPlacesScript(t *testing.T) {
	home := t.TempDir()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "guard.sh"), []byte("#!/bin/bash\nexit 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))

	art := &manifest.Artifact{
		Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "git-guardrails", Role: manifest.RoleGuardrail},
		Type: manifest.TypeHook,
		Hook: &manifest.HookSpec{Event: "PreToolUse", Matcher: "Bash", Command: "{script}", Script: "guard.sh"},
	}
	diffs, err := eng.Transform(art, loadAdapter(t, "claude"), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 2 {
		t.Fatalf("want 2 diffs (CREATE script + MERGE settings), got %d", len(diffs))
	}

	// First diff: the placed, executable script.
	script := diffs[0]
	wantPath := filepath.Join(home, ".claude", "hooks", "git-guardrails.sh")
	if script.Action != diff.Create || script.Path != wantPath {
		t.Errorf("script diff = %s %q, want CREATE %q", script.Action, script.Path, wantPath)
	}
	if script.Mode != 0o755 {
		t.Errorf("hook script mode = %o, want 0755 (executable)", script.Mode)
	}

	// Second diff: the settings hook, command resolved to the placed script path.
	cmd := hooksAt(t, diffs[1].After, "PreToolUse")[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"]
	if cmd != wantPath {
		t.Errorf("hook command = %v, want the placed script path %q", cmd, wantPath)
	}
}

// existingBytes is a ReadExisting that returns fixed bytes for any path.
func existingBytes(b []byte) ReadExisting {
	return func(string) ([]byte, bool, error) { return b, true, nil }
}

package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// A nudge hook wires natively on Codex (Claude-style hooks in config.toml) but is
// a no-op on OpenCode (which has no nudge mechanism — the paired instruction
// carries it). So the same nudge yields a Codex diff and no OpenCode diff.
func TestTransformNudgeHookPerTool(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("tdd-guard", "PreToolUse", "Edit", "tdd-guard") // default intent: nudge

	codex, err := eng.Transform(art, loadAdapter(t, "codex"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("codex: %v", err)
	}
	if len(codex) != 1 || codex[0].Action != diff.Merge {
		t.Errorf("codex nudge: want 1 MERGE diff, got %+v", codex)
	}
	if !strings.Contains(codex[0].Path, "config.toml") {
		t.Errorf("codex hook should target config.toml, got %q", codex[0].Path)
	}

	oc, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("opencode: %v", err)
	}
	if len(oc) != 0 {
		t.Errorf("opencode nudge: want 0 diffs (instruction carries it), got %d", len(oc))
	}
}

// A gate hook on OpenCode maps to a deny rule in the declarative permission config
// (permission.<matcher> = "deny"), OpenCode having no hooks block.
func TestTransformGateHookOpenCode(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("no-bash", "PreToolUse", "bash", "block")
	art.Hook.Intent = manifest.HookGate

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("opencode gate: %v", err)
	}
	if len(diffs) != 1 || diffs[0].Action != diff.Merge {
		t.Fatalf("want 1 MERGE diff, got %+v", diffs)
	}
	if diffs[0].Setting == nil || diffs[0].Setting.Dotted != "permission.bash" || diffs[0].Setting.ScalarValue != "deny" {
		t.Errorf("gate should deny permission.bash, got %+v", diffs[0].Setting)
	}
	if !strings.Contains(string(diffs[0].After), "deny") {
		t.Errorf("merged config should contain a deny:\n%s", diffs[0].After)
	}
}

// A gate whose matcher is a pipe-alternation of Claude tool names maps to ONE
// OpenCode permission key per distinct tool: OpenCode keys are single lowercase
// tool names, never alternations, and its `edit` permission covers write/edit/
// patch — so Write|Edit|MultiEdit collapses to a single permission.edit = "deny".
func TestTransformGateOpenCodeMapsAndCollapsesMatcher(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("block-secrets", "PreToolUse", "Write|Edit|MultiEdit", "block")
	art.Hook.Intent = manifest.HookGate

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("opencode gate: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("Write|Edit|MultiEdit should collapse to one permission.edit diff, got %d: %+v", len(diffs), diffs)
	}
	if got := diffs[0].Setting.Dotted; got != "permission.edit" {
		t.Errorf("dotted = %q, want permission.edit", got)
	}
	if diffs[0].Setting.ScalarValue != "deny" {
		t.Errorf("value = %v, want deny", diffs[0].Setting.ScalarValue)
	}
}

// Distinct tools in the matcher yield one deny diff each (Bash → permission.bash
// is separate from an edit gate).
func TestTransformGateOpenCodeMultipleKeys(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("wide-gate", "PreToolUse", "Edit|Bash", "block")
	art.Hook.Intent = manifest.HookGate

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("opencode gate: %v", err)
	}
	if len(diffs) != 2 {
		t.Fatalf("Edit|Bash should yield 2 permission diffs, got %d", len(diffs))
	}
	got := map[string]bool{diffs[0].Setting.Dotted: true, diffs[1].Setting.Dotted: true}
	if !got["permission.edit"] || !got["permission.bash"] {
		t.Errorf("want permission.edit and permission.bash, got %v", got)
	}
}

// A matcher with a mappable AND an unmappable token (Claude-only TodoWrite) still
// wires the mappable part and carries a warning naming the dropped token — never a
// silent skip.
func TestTransformGateOpenCodePartialWireWarns(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("tdd-guard-hook", "PreToolUse", "Write|Edit|MultiEdit|TodoWrite", "tdd-guard")
	art.Hook.Intent = manifest.HookGate

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err != nil {
		t.Fatalf("opencode gate: %v", err)
	}
	if len(diffs) != 1 || diffs[0].Setting.Dotted != "permission.edit" {
		t.Fatalf("want one permission.edit diff, got %+v", diffs)
	}
	if !strings.Contains(diffs[0].Warning, "TodoWrite") {
		t.Errorf("partial wire should warn about the dropped TodoWrite token, got %q", diffs[0].Warning)
	}
}

// A gate whose EVERY token is unmappable to OpenCode is an error — there is no
// honest permission key to deny, so silently wiring nothing would recreate the
// unmappable-token no-op.
func TestTransformGateOpenCodeAllUnmappableErrors(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("todo-only", "PreToolUse", "TodoWrite", "block")
	art.Hook.Intent = manifest.HookGate

	_, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", noExisting)
	if err == nil {
		t.Fatal("a fully-unmappable gate matcher should error, not silently wire nothing")
	}
}

// An OpenCode permission gate captures whatever the user already had at that
// permission key, so remove restores their value instead of deleting the key.
func TestGateDiffCapturesPrior(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := hookArtifact("no-edit", "PreToolUse", "Edit", "block")
	art.Hook.Intent = manifest.HookGate

	// The user already set permission.edit themselves.
	existing := []byte(`{"permission":{"edit":"allow"}}`)
	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", "", existingBytes(existing))
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) == 0 {
		t.Fatal("no gate diffs produced")
	}
	d := diffs[0]
	if d.Setting == nil {
		t.Fatal("Setting is nil")
	}
	if !d.Setting.PriorPresent {
		t.Fatal("PriorPresent = false; the user's existing permission.edit was not captured")
	}
	if d.Setting.PriorValue != "allow" {
		t.Fatalf("PriorValue = %#v, want \"allow\"", d.Setting.PriorValue)
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

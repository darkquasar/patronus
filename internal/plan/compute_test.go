package plan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// --- test fixtures -----------------------------------------------------------

func loadAdapters(t *testing.T) map[string]*manifest.Adapter {
	t.Helper()
	out := map[string]*manifest.Adapter{}
	for _, tool := range []string{"claude", "codex", "opencode"} {
		ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", tool+".yaml"))
		if err != nil {
			t.Fatalf("load %s adapter: %v", tool, err)
		}
		out[tool] = ad
	}
	return out
}

func env(home string) toolpath.EnvLookup {
	return func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}
}

// skillArtifact writes a SKILL.md into a fresh source dir and returns a catalog
// entry pointing at it.
func skillArtifact(t *testing.T, name string, targets []string, scope string) registry.ArtifactEntry {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("BODY:"+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return registry.ArtifactEntry{
		Manifest: &manifest.Artifact{
			Meta:  manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleCapability},
			Type:  manifest.TypeSkill,
			Entry: "SKILL.md", Targets: targets, Defaults: manifest.ArtifactDefaults{Scope: scope},
		},
		Source: registry.Source{LocalDir: src},
	}
}

func instructionArtifact(t *testing.T, name string, targets []string) registry.ArtifactEntry {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "INSTRUCTIONS.md"), []byte("rules:"+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return registry.ArtifactEntry{
		Manifest: &manifest.Artifact{
			Meta:  manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleInstruction},
			Type:  manifest.TypeInstruction,
			Entry: "INSTRUCTIONS.md", Targets: targets, Defaults: manifest.ArtifactDefaults{Scope: "local"},
		},
		Source: registry.Source{LocalDir: src},
	}
}

func baseReq(t *testing.T, home, proj string, entries ...registry.ArtifactEntry) Request {
	t.Helper()
	return Request{
		Catalog:   &registry.Catalog{Artifacts: entries},
		Inventory: &scan.Inventory{},
		Adapters:  loadAdapters(t),
		Resolver:  toolpath.New(env(home), home, proj),
	}
}

// --- classification ----------------------------------------------------------

func TestComputeCreateWhenAbsent(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "claude"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 || cs.Diffs[0].Action != diff.Create {
		t.Fatalf("want 1 CREATE, got %+v", cs.Diffs)
	}
}

func TestComputeSkipWhenIdentical(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	entry := skillArtifact(t, "s", []string{"claude"}, "global")
	req := baseReq(t, home, proj, entry)
	req.Names = []string{"s"}
	req.Tool = "claude"

	// Pre-create the target with identical bytes.
	target := filepath.Join(home, ".claude", "skills", "s", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("BODY:s"), 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Skip {
		t.Errorf("want SKIP, got %s", cs.Diffs[0].Action)
	}
}

func TestComputeConflictWhenDiffers(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "claude"

	target := filepath.Join(home, ".claude", "skills", "s", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("DIFFERENT"), 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Conflict {
		t.Errorf("want CONFLICT, got %s", cs.Diffs[0].Action)
	}
}

func TestComputeInstructionAppendThenSkip(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	entry := instructionArtifact(t, "ap", []string{"claude"})
	req := baseReq(t, home, proj, entry)
	req.Names = []string{"ap"}
	req.Tool = "claude"
	req.Scope = "local"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Append {
		t.Fatalf("want APPEND, got %s", cs.Diffs[0].Action)
	}

	// Write the computed result to disk, re-plan -> SKIP (idempotent).
	target := cs.Diffs[0].Path
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, cs.Diffs[0].After, 0o644); err != nil {
		t.Fatal(err)
	}
	cs2, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if cs2.Diffs[0].Action != diff.Skip {
		t.Errorf("re-plan want SKIP, got %s", cs2.Diffs[0].Action)
	}
}

// --- tool resolution ---------------------------------------------------------

func TestResolveToolsSpecificNotTargeted(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "codex" // not in targets
	if _, err := Compute(req); err == nil {
		t.Error("expected error: artifact does not target codex")
	}
}

func TestResolveToolsAllFallsBackWhenNoneDetected(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude", "opencode", "codex"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "all" // no detection -> fall back to all targets

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	// One SKILL.md per tool, distinct paths, ordered claude, opencode, codex.
	wantTools := []string{"claude", "opencode", "codex"}
	if len(cs.Diffs) != 3 {
		t.Fatalf("want 3 diffs, got %d", len(cs.Diffs))
	}
	for i, d := range cs.Diffs {
		if d.Tool != wantTools[i] {
			t.Errorf("diff[%d].Tool = %s, want %s (ordering)", i, d.Tool, wantTools[i])
		}
	}
}

func TestResolveToolsAllPrefersDetected(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude", "codex"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "all"
	// Only codex detected at global.
	req.Inventory = &scan.Inventory{Tools: []scan.ToolStatus{
		{Tool: "codex", Global: &scan.Detection{Detected: true}},
		{Tool: "claude", Global: &scan.Detection{Detected: false}},
	}}

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 || cs.Diffs[0].Tool != "codex" {
		t.Fatalf("want only detected codex, got %+v", toolList(cs))
	}
}

// --- scope resolution --------------------------------------------------------

func TestScopeDefaultsToArtifact(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "claude" // no scope flag -> artifact default "global"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if got := cs.Diffs[0].Scope; got != "global" {
		t.Errorf("scope = %s, want global (artifact default)", got)
	}
	if !strings.HasPrefix(cs.Diffs[0].Path, home) {
		t.Errorf("global path should be under home: %s", cs.Diffs[0].Path)
	}
}

func TestScopeFlagOverrides(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj, skillArtifact(t, "s", []string{"claude"}, "global"))
	req.Names = []string{"s"}
	req.Tool = "claude"
	req.Scope = "local"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cs.Diffs[0].Path, proj) {
		t.Errorf("local path should be under project: %s", cs.Diffs[0].Path)
	}
}

// --- cross-tool compose ------------------------------------------------------

func TestComposeSharedAgentsFile(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	// codex + opencode both append to the project AGENTS.md.
	entry := instructionArtifact(t, "ap", []string{"codex", "opencode"})
	req := baseReq(t, home, proj, entry)
	req.Names = []string{"ap"}
	req.Tool = "all"
	req.Scope = "local"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	// Both resolve to <proj>/AGENTS.md -> a single composed diff.
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed diff, got %d: %v", len(cs.Diffs), paths(cs.Diffs))
	}
	d := cs.Diffs[0]
	if d.Path != filepath.Join(proj, "AGENTS.md") {
		t.Errorf("path = %s", d.Path)
	}
	if d.Tool != "opencode+codex" && d.Tool != "codex+opencode" {
		t.Errorf("tool label should show both: %s", d.Tool)
	}
}

// TestComposeTwoArtifactsOneFile covers the multi-instruction case the visual/core
// profiles introduce: two DISTINCT artifacts append to the same CLAUDE.md. They
// compose into one physical diff (one write), but the second contributor is
// recorded in Contrib so state/remove can track each section independently.
func TestComposeTwoArtifactsOneFile(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	a := instructionArtifact(t, "demo-instr", []string{"claude"})
	b := instructionArtifact(t, "demo-rules", []string{"claude"})
	req := baseReq(t, home, proj, a, b)
	req.Names = []string{"demo-instr", "demo-rules"}
	req.Tool = "claude"
	req.Scope = "local"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed diff, got %d: %v", len(cs.Diffs), paths(cs.Diffs))
	}
	d := cs.Diffs[0]
	// First contributor stays on the diff; the second lives in Contrib.
	if d.Artifact != "demo-instr" {
		t.Errorf("primary artifact = %q, want demo-instr", d.Artifact)
	}
	if len(d.Contrib) != 1 || d.Contrib[0].Artifact != "demo-rules" || d.Contrib[0].Section != "demo-rules" {
		t.Fatalf("want one contrib for demo-rules, got %+v", d.Contrib)
	}
	// Both fenced sections are present in the single composed file.
	for _, want := range []string{"patronus:start demo-instr", "patronus:start demo-rules"} {
		if !bytes.Contains(d.After, []byte(want)) {
			t.Errorf("composed file missing %q:\n%s", want, d.After)
		}
	}
	// Contrib.Prior is the file BEFORE demo-rules folded in — it has demo-instr but not demo-rules,
	// so remove can reverse exactly the demo-rules section.
	if !bytes.Contains(d.Contrib[0].Prior, []byte("patronus:start demo-instr")) ||
		bytes.Contains(d.Contrib[0].Prior, []byte("patronus:start demo-rules")) {
		t.Errorf("contrib prior should hold demo-instr-only:\n%s", d.Contrib[0].Prior)
	}
}

// hookEntry builds a hook artifact registry entry for the plan tests. A hook
// reads no body file, so the source dir is irrelevant.
func hookEntry(t *testing.T, name, event, matcher, command string, targets []string) registry.ArtifactEntry {
	t.Helper()
	return registry.ArtifactEntry{
		Manifest: &manifest.Artifact{
			Meta:     manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleEval},
			Type:     manifest.TypeHook,
			Targets:  targets,
			Defaults: manifest.ArtifactDefaults{Scope: "global"},
			Hook:     &manifest.HookSpec{Event: event, Matcher: matcher, Command: command},
		},
		Source: registry.Source{LocalDir: t.TempDir()},
	}
}

// Two hooks on the same event land in ONE settings.json with BOTH array elements
// present (the merge-fold fix): without re-folding onto accumulated bytes, the
// second hook would clobber the first. The second is recorded as a SettingContrib
// so remove can strip exactly it.
func TestComposeTwoHooksOneSettingsFile(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	a := hookEntry(t, "demo-hook", "PreToolUse", "Edit", "demo-hook", []string{"claude"})
	b := hookEntry(t, "demo-hook2", "PreToolUse", "Bash", "demo-cmd2", []string{"claude"})
	req := baseReq(t, home, proj, a, b)
	req.Names = []string{"demo-hook", "demo-hook2"}
	req.Tool = "claude"
	req.Scope = "global"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed settings diff, got %d: %v", len(cs.Diffs), paths(cs.Diffs))
	}
	d := cs.Diffs[0]
	if d.Artifact != "demo-hook" {
		t.Errorf("primary = %q, want demo-hook", d.Artifact)
	}
	if len(d.SettingContrib) != 1 || d.SettingContrib[0].Artifact != "demo-hook2" {
		t.Fatalf("want one setting-contrib for demo-hook2, got %+v", d.SettingContrib)
	}
	// Both commands survive in the single composed settings file.
	for _, want := range []string{"demo-hook", "demo-cmd2"} {
		if !bytes.Contains(d.After, []byte(want)) {
			t.Errorf("composed settings missing %q:\n%s", want, d.After)
		}
	}
}

// An opencode gate whose matcher maps to two permission keys (Edit|Bash →
// permission.edit + permission.bash) composes into ONE opencode.json with BOTH
// denies, and the second key is recorded as a SettingContrib under the same
// artifact so remove strips both (not just the first). Without the contrib the
// permission.bash deny would leak on remove.
func TestComposeMultiKeyOpenCodeGate(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	g := hookEntry(t, "wide-gate", "PreToolUse", "Edit|Bash", "block", []string{"opencode"})
	g.Manifest.Hook.Intent = manifest.HookGate
	req := baseReq(t, home, proj, g)
	req.Names = []string{"wide-gate"}
	req.Tool = "opencode"
	req.Scope = "global"

	cs, err := Compute(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed opencode.json diff, got %d: %v", len(cs.Diffs), paths(cs.Diffs))
	}
	d := cs.Diffs[0]
	// The owning diff carries the first key; the second rides a SettingContrib.
	if d.Setting == nil || d.Setting.Dotted != "permission.edit" {
		t.Errorf("owning edit = %+v, want permission.edit", d.Setting)
	}
	if len(d.SettingContrib) != 1 || d.SettingContrib[0].Edit.Dotted != "permission.bash" {
		t.Fatalf("want a permission.bash setting-contrib, got %+v", d.SettingContrib)
	}
	for _, want := range []string{"\"edit\"", "\"bash\"", "deny"} {
		if !bytes.Contains(d.After, []byte(want)) {
			t.Errorf("composed opencode.json missing %q:\n%s", want, d.After)
		}
	}
}

func TestUnknownArtifactErrors(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	req := baseReq(t, home, proj)
	req.Names = []string{"nope"}
	if _, err := Compute(req); err == nil {
		t.Error("expected error for unknown artifact")
	}
}

// mcpMergeDiff builds the diff an MCP recipe produces: a MERGE on one shared
// config path carrying a scalar SettingEdit. Every one is computed from the SAME
// base bytes, which is exactly the condition that made last-wins lose merges.
func mcpMergeDiff(t *testing.T, path string, base []byte, name, cmd string) diff.FileDiff {
	t.Helper()
	ft := manifest.FileTarget{File: ".claude.json", Format: "json"}
	obj := map[string]any{"type": "stdio", "command": cmd}
	after, err := adapter.MergeSettings(base, ft, "mcpServers."+name, obj)
	if err != nil {
		t.Fatal(err)
	}
	return diff.FileDiff{
		Path: path, Action: diff.Merge, Before: base, After: after,
		Artifact: name, Type: "recipe", Role: "tools", Tool: "claude",
		Setting: &diff.SettingEdit{
			Target: diff.FileTargetRef{File: ft.File, Format: ft.Format},
			Dotted: "mcpServers." + name, ScalarValue: obj,
		},
	}
}

// servers unmarshals a composed config and returns its mcpServers map.
func servers(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse composed config: %v (%s)", err, b)
	}
	s, _ := m["mcpServers"].(map[string]any)
	return s
}

// existingBytes serves fixed bytes as the current content of every path, which is
// what Finalize needs to classify a diff without touching the real filesystem.
func existingBytes(b []byte) adapter.ReadExisting {
	return func(string) ([]byte, bool, error) { return b, true, nil }
}

func TestComposeAccumulatesSamePathMcpMerges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	base := []byte(`{"mcpServers":{"context7":{"command":"c7"}}}`)
	if err := os.WriteFile(path, base, 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := Finalize([]diff.FileDiff{
		mcpMergeDiff(t, path, base, "graphify", "gq"),
		mcpMergeDiff(t, path, base, "serena", "uvx"),
	}, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed diff, got %d", len(cs.Diffs))
	}
	d := cs.Diffs[0]

	got := servers(t, d.After)
	for _, want := range []string{"context7", "graphify", "serena"} {
		if _, ok := got[want]; !ok {
			t.Errorf("mcpServers.%s missing from the composed result:\n%s", want, d.After)
		}
	}

	// graphify enters byPath first, so it owns the row; serena folds in as a
	// contributor under its OWN identity. Without the contrib, remove strips the
	// wrong block and the plan renders one row for two artifacts.
	if d.Artifact != "graphify" {
		t.Errorf("owner = %q, want graphify (first in)", d.Artifact)
	}
	if len(d.SettingContrib) != 1 || d.SettingContrib[0].Artifact != "serena" {
		t.Fatalf("want one setting-contrib for serena, got %+v", d.SettingContrib)
	}
}

// TestComposeThreeWayMcpMerges guards a fold that re-applies against the wrong
// base. A fold that re-derived from `Before` instead of the accumulated `After`
// would keep only the last two servers, which two merges cannot distinguish.
func TestComposeThreeWayMcpMerges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	base := []byte(`{"mcpServers":{"context7":{"command":"c7"}}}`)
	if err := os.WriteFile(path, base, 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := Finalize([]diff.FileDiff{
		mcpMergeDiff(t, path, base, "alpha", "a"),
		mcpMergeDiff(t, path, base, "beta", "b"),
		mcpMergeDiff(t, path, base, "gamma", "g"),
	}, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	got := servers(t, cs.Diffs[0].After)
	if len(got) != 4 {
		t.Fatalf("want 4 servers (context7 + 3 folded), got %d:\n%s", len(got), cs.Diffs[0].After)
	}
	if n := len(cs.Diffs[0].SettingContrib); n != 2 {
		t.Errorf("want 2 contributors (alpha owns the row), got %d", n)
	}
}

// TestComposeFoldIsIdempotent pins the invariant behind the double-Finalize path.
// install.go finalizes the combined artifact+recipe set after plan.Compute already
// finalized the artifact half, so an artifact and a recipe touching one path fold
// twice. ApplySettingEdit is idempotent for a scalar set, and this keeps it so.
func TestComposeFoldIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	base := []byte(`{"mcpServers":{"context7":{"command":"c7"}}}`)
	if err := os.WriteFile(path, base, 0o644); err != nil {
		t.Fatal(err)
	}
	raw := []diff.FileDiff{
		mcpMergeDiff(t, path, base, "graphify", "gq"),
		mcpMergeDiff(t, path, base, "serena", "uvx"),
	}

	once, err := Finalize(raw, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Finalize(once.Diffs, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(once.Diffs[0].After, twice.Diffs[0].After) {
		t.Errorf("re-folding changed the bytes:\n once  %s\n twice %s", once.Diffs[0].After, twice.Diffs[0].After)
	}
	if a, b := len(once.Diffs[0].SettingContrib), len(twice.Diffs[0].SettingContrib); a != b {
		t.Errorf("re-folding duplicated contributors: %d -> %d", a, b)
	}
}

// TestComposedMcpPlanRowsShowBothArtifacts is the acceptance gate for the
// row-versus-bytes split: before the fix, one row was attributed to graphify
// while the bytes written were serena's, so neither artifact was honestly
// represented. Each contributor must carry its OWN Type and Role rather than
// inheriting the owning diff's.
func TestComposedMcpPlanRowsShowBothArtifacts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	base := []byte(`{}`)
	if err := os.WriteFile(path, base, 0o644); err != nil {
		t.Fatal(err)
	}

	graphify := mcpMergeDiff(t, path, base, "graphify", "gq")
	serena := mcpMergeDiff(t, path, base, "serena", "uvx")
	serena.Role = "memory" // a DIFFERENT role, so inheritance is detectable

	cs, err := Finalize([]diff.FileDiff{graphify, serena}, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	d := cs.Diffs[0]

	if d.Artifact != "graphify" || d.Role != "tools" {
		t.Errorf("owner row = %s/%s, want graphify/tools", d.Artifact, d.Role)
	}
	c := d.SettingContrib[0]
	if c.Artifact != "serena" {
		t.Fatalf("contributor = %q, want serena", c.Artifact)
	}
	if c.Role != "memory" {
		t.Errorf("contributor Role = %q, want memory: the row inherited the owner's "+
			"instead of carrying its own", c.Role)
	}
	if c.Type != "recipe" {
		t.Errorf("contributor Type = %q, want recipe", c.Type)
	}
}

// --- helpers -----------------------------------------------------------------

func toolList(cs *diff.ChangeSet) []string {
	out := make([]string, len(cs.Diffs))
	for i, d := range cs.Diffs {
		out[i] = d.Tool
	}
	return out
}

func paths(diffs []diff.FileDiff) []string {
	out := make([]string, len(diffs))
	for i, d := range diffs {
		out[i] = d.Path
	}
	return out
}

func TestPiCompositionRefoldFailureDiscardsPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	first := mcpMergeDiff(t, path, nil, "fixture-first", "one")
	second := mcpMergeDiff(t, path, nil, "fixture-second", "two")
	// Inject a broken intermediate result, not a broken standalone transform.
	first.After = []byte(`{"broken":`)
	cs, err := Finalize([]diff.FileDiff{first, second}, existingBytes(nil))
	if err == nil || cs != nil {
		t.Fatalf("refold failure returned success/partial plan: cs=%+v err=%v", cs, err)
	}
}

func TestPiCompositionStructuralConflicts(t *testing.T) {
	for _, tt := range []struct {
		name, owner, dotted string
		value               any
	}{
		{"different owner equal value", "other", "mcpServers.fixture", map[string]any{"command": "one", "type": "stdio"}},
		{"same owner different value", "fixture", "mcpServers.fixture", false},
		{"ancestor", "other", "mcpServers", false},
		{"descendant", "other", "mcpServers.fixture.command", "one"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			first := mcpMergeDiff(t, path, nil, "fixture", "one")
			second := first
			edit := *first.Setting
			edit.Dotted, edit.ScalarValue = tt.dotted, tt.value
			second.Artifact, second.Setting = tt.owner, &edit
			cs, err := Finalize([]diff.FileDiff{first, second}, existingBytes(nil))
			if err == nil || cs != nil {
				t.Fatalf("overlap accepted: %+v, %v", cs, err)
			}
		})
	}
}

func TestPiCompositionRejectsHeterogeneousIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, firstTool, secondTool, firstScope, secondScope string
	}{
		{"scope", "pi", "pi", "global", "local"},
		{"tool", "claude", "codex", "global", "global"},
	} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", tc.name, reverse), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "settings.json")
				installed := mcpMergeDiff(t, path, nil, "fixture-b", "one")
				owner := state.Item{Artifact: "fixture-b", Tool: tc.firstTool, Scope: tc.firstScope, Files: []state.FileState{{Path: path, Action: string(diff.Merge), Setting: installed.Setting}}}
				first := mcpMergeDiff(t, path, installed.After, "fixture-a", "fresh")
				second := mcpMergeDiff(t, path, installed.After, "fixture-b", "two")
				first.Tool, first.Scope = tc.firstTool, tc.firstScope
				second.Tool, second.Scope = tc.secondTool, tc.secondScope
				rows := []diff.FileDiff{first, second}
				if reverse {
					rows[0], rows[1] = rows[1], rows[0]
				}
				cs, err := Finalize(rows, existingBytes(installed.After))
				if err == nil {
					admitted, admissionErr := AdmitSettings(cs, []state.Item{owner})
					t.Fatalf("identity lost before admission: admitted=%t admissionErr=%v", admitted != nil, admissionErr)
				}
				if cs != nil {
					t.Fatal("conflict returned a partial plan")
				}
			})
		}
	}
}

func TestPiCompositionDuplicateOwnerIsIdempotent(t *testing.T) {
	d := mcpMergeDiff(t, filepath.Join(t.TempDir(), "settings.json"), nil, "fixture", "one")
	cs, err := Finalize([]diff.FileDiff{d, d}, existingBytes(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 || len(cs.Diffs[0].SettingContrib) != 0 {
		t.Fatalf("duplicate ownership: %+v", cs)
	}
}

func TestPiCompositionScalarMCPOrderIndependent(t *testing.T) {
	base := []byte(`{"unrelated":{"legacy":true},"mcpServers":{"external":{"url":"https://fixture.invalid"}}}`)
	path := filepath.Join(t.TempDir(), "settings.json")
	mcp := mcpMergeDiff(t, path, base, "fixture", "one")
	scalar := mcp
	scalar.Artifact = "switch"
	scalar.Setting = &diff.SettingEdit{Target: mcp.Setting.Target, Dotted: "ui.color", ScalarValue: false}
	var err error
	scalar.After, err = adapter.ApplySettingEdit(base, scalar.Setting)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Finalize([]diff.FileDiff{mcp, scalar}, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Finalize([]diff.FileDiff{scalar, mcp}, existingBytes(base))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Diffs[0].After, second.Diffs[0].After) {
		t.Fatal("order changed result")
	}
	for _, dotted := range []string{"unrelated.legacy", "mcpServers.external.url", "mcpServers.fixture.command", "ui.color"} {
		_, present, err := adapter.ReadDotted(first.Diffs[0].After, manifest.FileTarget{File: "settings.json", Format: "json"}, dotted)
		if err != nil || !present {
			t.Fatalf("missing sibling %s: %v", dotted, err)
		}
	}
}

func TestPiCompositionOwnershipAdmission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := mcpMergeDiff(t, path, nil, "fixture", "one")
	owner := state.Item{Artifact: original.Artifact, Tool: original.Tool, Scope: original.Scope, Files: []state.FileState{{Path: path, Action: string(diff.Merge), Setting: original.Setting}}}
	current := original.After
	desired := mcpMergeDiff(t, path, current, "fixture", "two")
	cs, err := Finalize([]diff.FileDiff{desired}, existingBytes(current))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		owners []state.Item
	}{
		{"unmanaged differing", nil},
		{"other owner", []state.Item{{Artifact: "other", Tool: owner.Tool, Scope: owner.Scope, Files: owner.Files}}},
		{"other scope", []state.Item{{Artifact: owner.Artifact, Tool: owner.Tool, Scope: "local", Files: owner.Files}}},
		{"duplicate evidence", []state.Item{owner, owner}},
		{"legacy ownership", []state.Item{{Artifact: "legacy", Files: []state.FileState{{Path: path, Action: string(diff.Merge)}}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AdmitSettings(cs, tt.owners)
			if err == nil || got != nil {
				t.Fatalf("unsafe admission: %+v %v", got, err)
			}
		})
	}
	admitted, err := AdmitSettings(cs, []state.Item{owner})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(admitted.Diffs[0].After, desired.After) {
		t.Fatal("same owner update changed result")
	}
	edited := mcpMergeDiff(t, path, []byte(`{"mcpServers":{"fixture":{"command":"edited"}}}`), "fixture", "two")
	if _, err := AdmitSettings(&diff.ChangeSet{Diffs: []diff.FileDiff{edited}}, []state.Item{owner}); err == nil {
		t.Fatal("drift accepted")
	}
}

func TestPiCompositionExternalEqualNeverBecomesContributor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	external := mcpMergeDiff(t, path, nil, "external-fixture", "one")
	base := external.After
	external = mcpMergeDiff(t, path, base, "external-fixture", "one")
	fresh := mcpMergeDiff(t, path, base, "managed-fixture", "two")
	for _, raw := range [][]diff.FileDiff{{external, fresh}, {fresh, external}} {
		cs, err := Finalize(raw, existingBytes(base))
		if err != nil {
			t.Fatal(err)
		}
		admitted, err := AdmitSettings(cs, nil)
		if err != nil {
			t.Fatal(err)
		}
		admitted, err = AdmitSettings(admitted, nil)
		if err != nil {
			t.Fatalf("repeat admission rejected equal unmanaged sibling: %v", err)
		}
		var writes []diff.FileDiff
		for _, d := range admitted.Diffs {
			if d.Action != diff.Skip {
				writes = append(writes, d)
			}
		}
		if len(writes) != 1 || writes[0].Artifact != "managed-fixture" || len(writes[0].SettingContrib) != 0 {
			t.Fatalf("external gained authority: %+v", writes)
		}
		recorded := state.FromChangeSet(writes, "")
		if len(recorded) != 1 || recorded[0].Artifact != "managed-fixture" {
			t.Fatalf("external recorded: %+v", recorded)
		}
		if _, ok := servers(t, writes[0].After)["external-fixture"]; !ok {
			t.Fatal("external lost")
		}
	}
}

func TestPiCompositionAdmissionStillRejectsNonstructuralWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	structural := mcpMergeDiff(t, path, nil, "fixture-setting", "one")
	file := structural
	file.Artifact, file.Action, file.Setting = "fixture-file", diff.Create, nil
	file.After = []byte(`{"replacement":true}`)
	for _, rows := range [][]diff.FileDiff{{structural, file}, {file, structural}} {
		if got, err := AdmitSettings(&diff.ChangeSet{Diffs: rows}, nil); err == nil || got != nil {
			t.Fatalf("real structural/nonstructural conflict admitted: %+v %v", got, err)
		}
	}
}

func TestPiCompositionInvalidPriorAndChangedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	d := mcpMergeDiff(t, path, nil, "fixture", "one")
	d.Before = []byte(`null`)
	if got, err := Finalize([]diff.FileDiff{d}, existingBytes(d.Before)); err == nil || got != nil {
		t.Fatal("invalid prior admitted")
	}
	d.Before = nil
	owner := state.Item{Artifact: d.Artifact, Tool: d.Tool, Files: []state.FileState{{Path: path + ".old", Action: string(diff.Merge), Setting: d.Setting}}}
	if got, err := AdmitSettings(&diff.ChangeSet{Diffs: []diff.FileDiff{d}}, []state.Item{owner}); err == nil || got != nil {
		t.Fatal("relocated path admitted")
	}
}

func TestPiCompositionDuplicateComposedOwnerIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	cs, err := Finalize([]diff.FileDiff{mcpMergeDiff(t, path, nil, "first", "one"), mcpMergeDiff(t, path, nil, "second", "two")}, existingBytes(nil))
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Finalize([]diff.FileDiff{cs.Diffs[0], cs.Diffs[0]}, existingBytes(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(repeated.Diffs) != 1 || len(repeated.Diffs[0].SettingContrib) != 1 {
		t.Fatalf("duplicated composite ownership: %+v", repeated)
	}
}

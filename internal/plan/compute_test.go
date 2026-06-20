package plan

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
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
	a := instructionArtifact(t, "spine", []string{"claude"})
	b := instructionArtifact(t, "rules", []string{"claude"})
	req := baseReq(t, home, proj, a, b)
	req.Names = []string{"spine", "rules"}
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
	if d.Artifact != "spine" {
		t.Errorf("primary artifact = %q, want spine", d.Artifact)
	}
	if len(d.Contrib) != 1 || d.Contrib[0].Artifact != "rules" || d.Contrib[0].Section != "rules" {
		t.Fatalf("want one contrib for rules, got %+v", d.Contrib)
	}
	// Both fenced sections are present in the single composed file.
	for _, want := range []string{"patronus:start spine", "patronus:start rules"} {
		if !bytes.Contains(d.After, []byte(want)) {
			t.Errorf("composed file missing %q:\n%s", want, d.After)
		}
	}
	// Contrib.Prior is the file BEFORE rules folded in — it has spine but not rules,
	// so remove can reverse exactly the rules section.
	if !bytes.Contains(d.Contrib[0].Prior, []byte("patronus:start spine")) ||
		bytes.Contains(d.Contrib[0].Prior, []byte("patronus:start rules")) {
		t.Errorf("contrib prior should hold spine-only:\n%s", d.Contrib[0].Prior)
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

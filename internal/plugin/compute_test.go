package plugin

import (
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func testEnv(home string) toolpath.EnvLookup {
	return func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}
}

func claudeAdapter(t *testing.T) *manifest.Adapter {
	t.Helper()
	ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", "claude.yaml"))
	if err != nil {
		t.Fatalf("load claude adapter: %v", err)
	}
	return ad
}

func noExisting(string) ([]byte, bool, error) { return nil, false, nil }

// TestComputeUnsupportedIsNoOp: an unsupported target (a tool with no plugin
// construct) is an honest no-op resolved BEFORE the engine is touched, so the
// engine/adapter can be nil here.
func TestComputeUnsupportedIsNoOp(t *testing.T) {
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers", Family: manifest.FamilyPlugin},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Ref: "v1"}},
	}
	diffs, contrib, err := Compute(Request{Plugin: p, Tool: "opencode", Scope: "user"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("unsupported produced %d diffs, want 0", len(diffs))
	}
	if contrib.Mode != ModeUnsupported {
		t.Errorf("mode = %q, want unsupported", contrib.Mode)
	}
	if contrib.Tool != "opencode" {
		t.Errorf("tool = %q, want opencode", contrib.Tool)
	}
	if contrib.Ecosystem != "" {
		t.Errorf("ecosystem = %q, want empty", contrib.Ecosystem)
	}
}

// TestComputeNativeProducesApplicableDiff is the regression test for the broken
// hand-built diff (empty Path → diff.Classify SKIP, a silent no-op). Routing the
// registration through the adapter setting transform yields an APPLICABLE diff:
// a real Path, distinct Before/After, and a Setting edit at plugins.<name>.
func TestComputeNativeProducesApplicableDiff(t *testing.T) {
	home := t.TempDir()
	eng := adapter.New(toolpath.New(testEnv(home), home, t.TempDir()))
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers", Family: manifest.FamilyPlugin},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Ref: "v2.1.0"}},
	}
	req := Request{
		Plugin:       p,
		Tool:         "claude",
		Scope:        "global",
		Engine:       eng,
		Adapter:      claudeAdapter(t),
		ReadExisting: noExisting,
	}
	diffs, contrib, err := Compute(req)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if contrib.Mode != ModeNative {
		t.Errorf("mode = %q, want native", contrib.Mode)
	}
	if contrib.Ecosystem != "claude-code" {
		t.Errorf("ecosystem = %q, want claude-code", contrib.Ecosystem)
	}
	if contrib.Tool != "claude" {
		t.Errorf("tool = %q, want claude", contrib.Tool)
	}
	if len(diffs) != 1 {
		t.Fatalf("native produced %d diffs, want 1 (registration)", len(diffs))
	}

	d := diffs[0]
	if d.Action != diff.Merge {
		t.Errorf("action = %q, want %q", d.Action, diff.Merge)
	}
	// THE KEY ASSERTION: an applicable diff has a real target path. The broken
	// hand-built diff left this empty, which made diff.Classify return SKIP.
	if d.Path == "" {
		t.Fatalf("Path is empty — inapplicable diff (the bug)")
	}
	if d.After == nil {
		t.Fatalf("After is nil — nothing to write")
	}
	if string(d.After) == string(d.Before) {
		t.Fatalf("After == Before — diff.Classify would SKIP this (the bug)")
	}
	// Prove non-SKIP through the real classifier (fresh file, does not exist).
	if got := diff.Classify(d.Action, d.Before, d.After, false); got == diff.Skip {
		t.Fatalf("Classify = SKIP, want an applicable action")
	}
	if d.Artifact != "superpowers" {
		t.Errorf("artifact = %q, want superpowers", d.Artifact)
	}
	if d.Tool != "claude" {
		t.Errorf("tool = %q, want claude", d.Tool)
	}
	if d.Scope != "global" {
		t.Errorf("scope = %q, want global", d.Scope)
	}
	if d.Setting == nil {
		t.Fatalf("Setting edit is nil, want scalar MERGE")
	}
	if d.Setting.Dotted != "plugins.superpowers" {
		t.Errorf("dotted = %q, want plugins.superpowers", d.Setting.Dotted)
	}
	val, ok := d.Setting.ScalarValue.(map[string]any)
	if !ok {
		t.Fatalf("ScalarValue type = %T, want map[string]any", d.Setting.ScalarValue)
	}
	if val["enabled"] != true {
		t.Errorf("enabled = %v, want true", val["enabled"])
	}
	if val["ecosystem"] != "claude-code" {
		t.Errorf("ecosystem = %v, want claude-code", val["ecosystem"])
	}
}

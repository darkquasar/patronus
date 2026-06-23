package plugin

import (
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

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

func TestComputeNativeProducesOneDiff(t *testing.T) {
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers", Family: manifest.FamilyPlugin},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Ref: "v2.1.0"}},
	}
	diffs, contrib, err := Compute(Request{Plugin: p, Tool: "claude", Scope: "user"})
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
	if d.Artifact != "superpowers" {
		t.Errorf("artifact = %q, want superpowers", d.Artifact)
	}
	if d.Type != "plugin" {
		t.Errorf("type = %q, want plugin", d.Type)
	}
	if d.Tool != "claude" {
		t.Errorf("tool = %q, want claude", d.Tool)
	}
	if d.Scope != "user" {
		t.Errorf("scope = %q, want user", d.Scope)
	}
	if d.Setting == nil {
		t.Fatalf("Setting edit is nil, want scalar MERGE")
	}
	if d.Setting.Dotted != "plugins.superpowers" {
		t.Errorf("dotted = %q, want plugins.superpowers", d.Setting.Dotted)
	}
	if d.Setting.IdentityKey != "" {
		t.Errorf("identityKey = %q, want empty (scalar set)", d.Setting.IdentityKey)
	}
	val, ok := d.Setting.ScalarValue.(map[string]any)
	if !ok {
		t.Fatalf("ScalarValue type = %T, want map[string]any", d.Setting.ScalarValue)
	}
	if val["enabled"] != true {
		t.Errorf("enabled = %v, want true", val["enabled"])
	}
}

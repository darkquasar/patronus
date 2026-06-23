package plugin

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func mkPlugin(ecosystems ...string) *manifest.Plugin {
	p := &manifest.Plugin{Sources: map[string]manifest.PluginSource{}}
	for _, e := range ecosystems {
		p.Sources[e] = manifest.PluginSource{Kind: "marketplace", Ref: "v1"}
	}
	return p
}

func TestResolveMode(t *testing.T) {
	cases := []struct {
		name     string
		plugin   *manifest.Plugin
		tool     string
		wantMode Mode
		wantEco  string
	}{
		{"claude native", mkPlugin("claude-code"), "claude", ModeNative, "claude-code"},
		{"codex native", mkPlugin("codex"), "codex", ModeNative, "codex"},
		{"both native on claude", mkPlugin("claude-code", "codex"), "claude", ModeNative, "claude-code"},
		{"both native on codex", mkPlugin("claude-code", "codex"), "codex", ModeNative, "codex"},
		{"claude-only translated to codex", mkPlugin("claude-code"), "codex", ModeTranslate, "claude-code"},
		{"codex-only translated to claude", mkPlugin("codex"), "claude", ModeTranslate, "codex"},
		{"opencode unsupported", mkPlugin("claude-code"), "opencode", ModeUnsupported, ""},
		{"no sources unsupported", mkPlugin(), "claude", ModeUnsupported, ""},
	}
	for _, tc := range cases {
		gotMode, gotEco := ResolveMode(tc.plugin, tc.tool)
		if gotMode != tc.wantMode {
			t.Errorf("%s: mode = %q, want %q", tc.name, gotMode, tc.wantMode)
		}
		if gotEco != tc.wantEco {
			t.Errorf("%s: eco = %q, want %q", tc.name, gotEco, tc.wantEco)
		}
	}
}

func TestEcosystemFor(t *testing.T) {
	if eco, ok := EcosystemFor("claude"); !ok || eco != "claude-code" {
		t.Errorf("claude -> %q,%v; want claude-code,true", eco, ok)
	}
	if eco, ok := EcosystemFor("codex"); !ok || eco != "codex" {
		t.Errorf("codex -> %q,%v; want codex,true", eco, ok)
	}
	if _, ok := EcosystemFor("opencode"); ok {
		t.Error("opencode -> ok=true; want false (no plugin construct)")
	}
}

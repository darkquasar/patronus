package toolpath

import (
	"path/filepath"
	"testing"
)

func envFrom(m map[string]string) EnvLookup {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestResolveMarkerHomeExpansion(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "/proj")
	got := r.ResolveMarker("~/.claude/", "claude", ScopeGlobal)
	want := filepath.Join("/home/u", ".claude")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerProjectScope(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "/proj")
	got := r.ResolveMarker(".mcp.json", "claude", ScopeLocal)
	want := filepath.Join("/proj", ".mcp.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerCodexHomeOverride(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u", "CODEX_HOME": "/custom/codex"}), "/home/u", "/proj")
	got := r.ResolveMarker("~/.codex/config.toml", "codex", ScopeGlobal)
	want := filepath.Join("/custom/codex", "config.toml")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerOpencodeXDGOverride(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u", "XDG_CONFIG_HOME": "/xdg"}), "/home/u", "/proj")
	got := r.ResolveMarker("~/.config/opencode/", "opencode", ScopeGlobal)
	want := filepath.Join("/xdg", "opencode")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerOpencodeConfigDirWins(t *testing.T) {
	r := New(envFrom(map[string]string{
		"HOME":                "/home/u",
		"XDG_CONFIG_HOME":     "/xdg",
		"OPENCODE_CONFIG_DIR": "/oc",
	}), "/home/u", "/proj")
	got := r.ResolveMarker("~/.config/opencode/", "opencode", ScopeGlobal)
	if got != "/oc" {
		t.Errorf("got %q, want /oc", got)
	}
}

func TestCollapseHome(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "/proj")
	cases := map[string]string{
		"/home/u/.claude/skills/x/SKILL.md": "~/.claude/skills/x/SKILL.md",
		"/home/u":                           "~",
		"/etc/passwd":                       "/etc/passwd",
	}
	for in, want := range cases {
		if got := r.CollapseHome(filepath.FromSlash(in)); got != want {
			t.Errorf("CollapseHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRelativeTo(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "/proj")
	cases := map[string]string{
		// Inside the project: relative.
		"/proj/.claude/skills/x": ".claude/skills/x",
		"/proj":                  ".",
		// Escapes the project: degrades to the absolute input.
		"/home/u/.claude/skills/x": "/home/u/.claude/skills/x",
		"/proj-other/skills/x":     "/proj-other/skills/x",
	}
	for in, want := range cases {
		if got := r.RelativeTo(filepath.FromSlash(in)); got != filepath.FromSlash(want) {
			t.Errorf("RelativeTo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRelativeToNoProjectDir(t *testing.T) {
	r := New(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "")
	in := filepath.FromSlash("/anywhere/x")
	if got := r.RelativeTo(in); got != in {
		t.Errorf("RelativeTo(%q) = %q, want unchanged", in, got)
	}
}

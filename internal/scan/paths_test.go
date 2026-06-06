package scan

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
	env := envFrom(map[string]string{"HOME": "/home/u"})
	r := newResolver(env, "/home/u", "/proj")
	got := r.resolveMarker("~/.claude/", "claude", ScopeGlobal)
	want := filepath.Join("/home/u", ".claude")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerProjectScope(t *testing.T) {
	env := envFrom(map[string]string{"HOME": "/home/u"})
	r := newResolver(env, "/home/u", "/proj")
	got := r.resolveMarker(".mcp.json", "claude", ScopeLocal)
	want := filepath.Join("/proj", ".mcp.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerCodexHomeOverride(t *testing.T) {
	env := envFrom(map[string]string{"HOME": "/home/u", "CODEX_HOME": "/custom/codex"})
	r := newResolver(env, "/home/u", "/proj")
	got := r.resolveMarker("~/.codex/config.toml", "codex", ScopeGlobal)
	want := filepath.Join("/custom/codex", "config.toml")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerOpencodeXDGOverride(t *testing.T) {
	env := envFrom(map[string]string{"HOME": "/home/u", "XDG_CONFIG_HOME": "/xdg"})
	r := newResolver(env, "/home/u", "/proj")
	got := r.resolveMarker("~/.config/opencode/", "opencode", ScopeGlobal)
	want := filepath.Join("/xdg", "opencode")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkerOpencodeConfigDirWins(t *testing.T) {
	env := envFrom(map[string]string{
		"HOME":                "/home/u",
		"XDG_CONFIG_HOME":     "/xdg",
		"OPENCODE_CONFIG_DIR": "/oc",
	})
	r := newResolver(env, "/home/u", "/proj")
	got := r.resolveMarker("~/.config/opencode/", "opencode", ScopeGlobal)
	if got != "/oc" {
		t.Errorf("got %q, want /oc", got)
	}
}

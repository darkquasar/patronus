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

func TestPiPathsOverrideNormalization(t *testing.T) {
	for _, tc := range []struct{ value, want string }{{"", "/home/fixture/.pi/agent/skills/item/SKILL.md"}, {"~", "/home/fixture/skills/item/SKILL.md"}, {"~/kit", "/home/fixture/kit/skills/item/SKILL.md"}, {"relative-kit", "/project/relative-kit/skills/item/SKILL.md"}, {"../kit", "/kit/skills/item/SKILL.md"}, {"/relocated", "/relocated/skills/item/SKILL.md"}} {
		t.Run(tc.value, func(t *testing.T) {
			r := New(envFrom(map[string]string{"PI_CODING_AGENT_DIR": tc.value}), "/home/fixture", "/project")
			if got := r.ResolveMarker("~/.pi/agent/skills/item/SKILL.md", "pi", ScopeGlobal); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
			if got := r.ResolveMarker(".pi/skills/item/SKILL.md", "pi", ScopeLocal); got != "/project/.pi/skills/item/SKILL.md" {
				t.Fatalf("local path redirected: %s", got)
			}
			if got := r.ResolveMarker("~/.claude/skills/item/SKILL.md", "claude", ScopeGlobal); got != "/home/fixture/.claude/skills/item/SKILL.md" {
				t.Fatalf("legacy path redirected: %s", got)
			}
		})
	}
	r := New(envFrom(map[string]string{"PI_CODING_AGENT_DIR": "relative-kit"}), "/home/fixture", "")
	if got := r.ResolveMarker("~/.pi/agent/AGENTS.md", "pi", ScopeGlobal); filepath.IsAbs(got) {
		t.Fatalf("invented working base: %s", got)
	}
	// Do not silently trim host/MCP-disagreeing spelling; scan preflight rejects it.
	r = New(envFrom(map[string]string{"PI_CODING_AGENT_DIR": "/relocated "}), "/home/fixture", "/project")
	if got := r.ResolveMarker("~/.pi/agent/AGENTS.md", "pi", ScopeGlobal); got != "/relocated /AGENTS.md" {
		t.Fatalf("silently trimmed override: %s", got)
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

func TestResolveMarkerPi(t *testing.T) {
	home, project, override := t.TempDir(), t.TempDir(), t.TempDir()
	for _, tc := range []struct{ name, override, marker, scope, want string }{
		{"default", "", "~/.pi/agent/skills/demo/SKILL.md", ScopeGlobal, filepath.Join(home, ".pi/agent/skills/demo/SKILL.md")},
		{"override", override, "~/.pi/agent/skills/demo/SKILL.md", ScopeGlobal, filepath.Join(override, "skills/demo/SKILL.md")},
		{"root", override, "~/.pi/agent/", ScopeGlobal, override},
		{"project", override, ".pi/prompts/demo.md", ScopeLocal, filepath.Join(project, ".pi/prompts/demo.md")},
		{"context", override, "AGENTS.md", ScopeLocal, filepath.Join(project, "AGENTS.md")},
		{"prefix boundary", override, "~/.pi/agent-other/file", ScopeGlobal, filepath.Join(home, ".pi/agent-other/file")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := New(envFrom(map[string]string{"PI_CODING_AGENT_DIR": tc.override}), home, project)
			if got := r.ResolveMarker(tc.marker, "pi", tc.scope); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

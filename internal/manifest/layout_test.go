package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// adaptersDir locates the repo's adapters/ directory relative to this package.
func adaptersDir(t *testing.T) string {
	t.Helper()
	// internal/manifest -> repo root is two levels up.
	return filepath.Join("..", "..", "adapters")
}

func loadAdapterForTest(t *testing.T, tool string) *Adapter {
	t.Helper()
	ad, err := LoadAdapter(filepath.Join(adaptersDir(t), tool+".yaml"))
	if err != nil {
		t.Fatalf("LoadAdapter(%s): %v", tool, err)
	}
	return ad
}

func TestLayoutDecodeClaude(t *testing.T) {
	ad := loadAdapterForTest(t, "claude")

	if ad.Layout.Skill == nil {
		t.Fatal("Skill layout missing")
	}
	if got := ad.Layout.Skill.Global; !got.Set || got.Path != "~/.claude/skills/{name}/SKILL.md" {
		t.Errorf("Skill.global = %+v", got)
	}
	if !ad.Layout.Skill.Frontmatter.Passthrough {
		t.Error("Skill.frontmatter should be passthrough")
	}

	// Instruction is an object with action: appendSection.
	if ad.Layout.Instruction == nil {
		t.Fatal("Instruction layout missing")
	}
	if got := ad.Layout.Instruction.Global; got.File != "~/.claude/CLAUDE.md" || got.Action != "appendSection" {
		t.Errorf("Instruction.global = %+v", got)
	}

	// MCP: Claude stdio transport carries a literal type:"stdio".
	if ad.Layout.Mcp == nil || ad.Layout.Mcp.Transports == nil {
		t.Fatal("Mcp transports missing")
	}
	stdio := ad.Layout.Mcp.Transports["stdio"]
	if stdio.Keys == nil {
		t.Fatal("stdio keys missing")
	}
	if v := stdio.Keys.Vals["type"]; v != "stdio" {
		t.Errorf("claude stdio type = %q, want stdio", v)
	}
	if got := ad.Layout.Mcp.User; got.File != "~/.claude.json" {
		t.Errorf("Mcp.user = %+v", got)
	}
}

func TestLayoutDecodeCodexShapeByKey(t *testing.T) {
	ad := loadAdapterForTest(t, "codex")

	// Codex Agent is TOML.
	if ad.Layout.Agent == nil || ad.Layout.Agent.Format != "toml" {
		t.Errorf("Agent.format = %+v, want toml", ad.Layout.Agent)
	}
	if ad.Layout.Agent.BodyIs != "developer_instructions" {
		t.Errorf("Agent.bodyIs = %q", ad.Layout.Agent.BodyIs)
	}

	// Command.project is null -> no usable project path (yaml.v3 leaves the
	// field zero for an explicit null, which is functionally "no target").
	if ad.Layout.Command == nil {
		t.Fatal("Command layout missing")
	}
	if got := ad.Layout.Command.Project; got.Path != "" {
		t.Errorf("Command.project path = %q, want empty (null)", got.Path)
	}
	if got := ad.Layout.Command.Global; got.Path == "" {
		t.Error("Command.global should be set")
	}

	// §9.9: Codex stdio transport has NO type key.
	stdio := ad.Layout.Mcp.Transports["stdio"]
	if stdio.Keys == nil {
		t.Fatal("codex stdio keys missing")
	}
	if _, ok := stdio.Keys.Vals["type"]; ok {
		t.Error("codex stdio must NOT carry a type key")
	}
	// http transport keyed by url/bearer_token_env_var/http_headers.
	http := ad.Layout.Mcp.Transports["http"]
	for _, k := range []string{"url", "bearer_token_env_var", "http_headers"} {
		if _, ok := http.Keys.Vals[k]; !ok {
			t.Errorf("codex http missing key %q", k)
		}
	}

	// Hook is null for Codex -> no usable file target.
	if ad.Layout.Hook == nil || ad.Layout.Hook.Global.File != "" {
		t.Errorf("Codex Hook.global should be null: %+v", ad.Layout.Hook)
	}
}

func TestLayoutAccessors(t *testing.T) {
	ad := loadAdapterForTest(t, "claude")

	// ForScope returns global vs project per scope.
	if g := ad.Layout.Skill.ForScope("global"); !g.OK() || g.Path != "~/.claude/skills/{name}/SKILL.md" {
		t.Errorf("Skill.ForScope(global) = %+v", g)
	}
	if l := ad.Layout.Skill.ForScope("local"); !l.OK() {
		t.Errorf("Skill.ForScope(local) not OK: %+v", l)
	}
	if ad.Layout.Agent.ForScope("global").Path == "" {
		t.Error("Agent.ForScope(global) empty")
	}
	if ad.Layout.Command.ForScope("local").Path == "" {
		t.Error("Command.ForScope(local) empty")
	}
	// Claude's Mcp uses project + user (no global); project must be OK.
	if !ad.Layout.Mcp.ForScope("local").OK() {
		t.Error("Mcp.ForScope(local) not OK")
	}
	if !ad.Layout.Instruction.ForScope("local").OK() {
		t.Error("Instruction.ForScope(local) not OK")
	}

	// OK on an empty target is false.
	if (PathTarget{}).OK() {
		t.Error("empty PathTarget should not be OK")
	}
	if (FileTarget{}).OK() {
		t.Error("empty FileTarget should not be OK")
	}
}

func TestLoadAdapterErrors(t *testing.T) {
	if _, err := LoadAdapter(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing adapter file")
	}
	// Present file without a tool field.
	p := filepath.Join(t.TempDir(), "notool.yaml")
	if err := os.WriteFile(p, []byte("apiVersion: patronus/v1\nkind: Adapter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAdapter(p); err == nil {
		t.Error("expected error for adapter missing tool")
	}
}

func TestLayoutDecodeOpencodeArrayAndKeyOrder(t *testing.T) {
	ad := loadAdapterForTest(t, "opencode")

	// OpenCode Agent frontmatter is an allow-list, not passthrough.
	if ad.Layout.Agent == nil {
		t.Fatal("Agent layout missing")
	}
	if ad.Layout.Agent.Frontmatter.Passthrough {
		t.Error("OpenCode Agent frontmatter should be an allow-list")
	}
	want := []string{"mode", "model", "prompt", "permission"}
	if got := ad.Layout.Agent.Frontmatter.Allow; len(got) != len(want) {
		t.Errorf("Agent frontmatter allow = %v, want %v", got, want)
	}

	// §9.9: OpenCode stdio uses type:"local" + command:"{commandArray}".
	stdio := ad.Layout.Mcp.Transports["stdio"]
	if stdio.Keys == nil {
		t.Fatal("opencode stdio keys missing")
	}
	if v := stdio.Keys.Vals["type"]; v != "local" {
		t.Errorf("opencode stdio type = %q, want local", v)
	}
	if v := stdio.Keys.Vals["command"]; v != "{commandArray}" {
		t.Errorf("opencode stdio command = %q, want {commandArray}", v)
	}

	// Key order is preserved as authored: type, command, environment.
	wantOrder := []string{"type", "command", "environment"}
	for i, k := range wantOrder {
		if i >= len(stdio.Keys.Keys) || stdio.Keys.Keys[i] != k {
			t.Errorf("opencode stdio key order = %v, want %v", stdio.Keys.Keys, wantOrder)
			break
		}
	}
}

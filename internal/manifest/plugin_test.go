package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlugin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "superpowers.yaml")
	content := `apiVersion: patronus/v2
family: plugin
role: lifecycle
name: superpowers
description: Brainstorming and planning plugin.
version: 2.1.0
sources:
  claude-code:
    kind: marketplace
    marketplace: claude-plugins-official
    plugin: superpowers
    ref: v2.1.0
    sha: "abc123"
  codex:
    kind: marketplace
    marketplace: codex-curated
    plugin: superpowers
    ref: v2.0.4
    sha: "def456"
targets: [claude, codex]
defaults:
  scope: user
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPlugin(path)
	if err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	if p.Family != FamilyPlugin {
		t.Errorf("family = %s, want plugin", p.Family)
	}
	if p.Name != "superpowers" {
		t.Errorf("name = %s, want superpowers", p.Name)
	}
	if len(p.Sources) != 2 {
		t.Fatalf("sources = %d, want 2", len(p.Sources))
	}
	cc, ok := p.Sources["claude-code"]
	if !ok {
		t.Fatal("missing claude-code source")
	}
	if cc.Marketplace != "claude-plugins-official" || cc.Ref != "v2.1.0" {
		t.Errorf("claude-code source wrong: %+v", cc)
	}
	if len(p.Targets) != 2 || p.Targets[0] != "claude" {
		t.Errorf("targets = %v, want [claude codex]", p.Targets)
	}
	if p.Defaults.Scope != "user" {
		t.Errorf("defaults.scope = %s, want user", p.Defaults.Scope)
	}
}

func TestLoadPluginWrongFamily(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	content := `apiVersion: patronus/v2
family: recipe
name: bad
description: d
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPlugin(path); err == nil {
		t.Fatal("LoadPlugin accepted family: recipe, want error")
	}
}

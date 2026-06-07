package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func agentEngine(t *testing.T, home, proj string) *Engine {
	t.Helper()
	env := func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}
	return New(toolpath.New(env, home, proj))
}

const sampleAgentMD = `---
name: reviewer
description: Reviews code
mode: subagent
model: opus
permission: read
extra: dropme
---
You are a careful reviewer.
`

func writeAgent(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte(sampleAgentMD), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAgentClaudePassthrough(t *testing.T) {
	src := t.TempDir()
	writeAgent(t, src)
	home := t.TempDir()
	eng := agentEngine(t, home, t.TempDir())
	art := &manifest.Artifact{Kind: manifest.KindAgent, Name: "reviewer", Entry: "agent.md", Role: manifest.RoleCapability}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude", "agents", "reviewer.md")
	if diffs[0].Path != want {
		t.Errorf("path = %q, want %q", diffs[0].Path, want)
	}
	s := string(diffs[0].After)
	// Passthrough keeps the body and frontmatter (incl. extra key).
	if !strings.Contains(s, "You are a careful reviewer.") {
		t.Errorf("body missing:\n%s", s)
	}
}

func TestAgentOpencodeFrontmatterAllowList(t *testing.T) {
	src := t.TempDir()
	writeAgent(t, src)
	home := t.TempDir()
	eng := agentEngine(t, home, t.TempDir())
	art := &manifest.Artifact{Kind: manifest.KindAgent, Name: "reviewer", Entry: "agent.md"}

	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	s := string(diffs[0].After)
	// Allowed keys kept; disallowed "extra" and "description"/"name" dropped from
	// frontmatter (allow-list is [mode, model, prompt, permission]).
	if !strings.Contains(s, "mode: subagent") || !strings.Contains(s, "model: opus") || !strings.Contains(s, "permission: read") {
		t.Errorf("allowed frontmatter keys missing:\n%s", s)
	}
	if strings.Contains(s, "extra:") || strings.Contains(s, "dropme") {
		t.Errorf("disallowed key leaked:\n%s", s)
	}
	// bodyIs: prompt -> body lands in the prompt frontmatter key.
	if !strings.Contains(s, "prompt:") || !strings.Contains(s, "careful reviewer") {
		t.Errorf("body not mapped to prompt:\n%s", s)
	}
	want := filepath.Join(home, ".config", "opencode", "agent", "reviewer.md")
	if diffs[0].Path != want {
		t.Errorf("path = %q, want %q", diffs[0].Path, want)
	}
}

func TestAgentCodexTOML(t *testing.T) {
	src := t.TempDir()
	writeAgent(t, src)
	home := t.TempDir()
	eng := agentEngine(t, home, t.TempDir())
	art := &manifest.Artifact{
		Kind: manifest.KindAgent, Name: "reviewer", Description: "Reviews code", Entry: "agent.md",
		Overrides: map[string]map[string]interface{}{"codex": {"model": "o1"}},
	}

	diffs, err := eng.Transform(art, loadAdapter(t, "codex"), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".codex", "agents", "reviewer.toml")
	if diffs[0].Path != want {
		t.Errorf("path = %q, want %q", diffs[0].Path, want)
	}
	var doc map[string]any
	if err := toml.Unmarshal(diffs[0].After, &doc); err != nil {
		t.Fatalf("invalid toml:\n%s\n%v", diffs[0].After, err)
	}
	if doc["name"] != "reviewer" {
		t.Errorf("name = %v", doc["name"])
	}
	if di, _ := doc["developer_instructions"].(string); !strings.Contains(di, "careful reviewer") {
		t.Errorf("developer_instructions = %v", doc["developer_instructions"])
	}
	// Override applied.
	if doc["model"] != "o1" {
		t.Errorf("override model = %v, want o1", doc["model"])
	}
}

func TestSplitFrontmatterNone(t *testing.T) {
	fm, body := splitFrontmatter([]byte("no frontmatter here"))
	if len(fm) != 0 || string(body) != "no frontmatter here" {
		t.Errorf("expected no frontmatter; fm=%v body=%q", fm, body)
	}
}

func TestSplitFrontmatterMalformed(t *testing.T) {
	// Opening --- but no close: treat whole input as body.
	raw := []byte("---\nname: x\nstill going")
	fm, body := splitFrontmatter(raw)
	if len(fm) != 0 || string(body) != string(raw) {
		t.Errorf("malformed frontmatter mishandled; fm=%v body=%q", fm, body)
	}
}

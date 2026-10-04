package adapter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func TestPiAgentNativeGrammar(t *testing.T) {
	valid := []string{
		"", "# before\n\n", "model: provider/model\noutput: result.md\n", "model: 'small'\n",
		"tools:\nexcludeTools:\nskills:\nextensions:\nsubagentOnlyExtensions:\nskillPath:\ndefaultReads:\n",
		"tools: read,bash,write\nexcludeTools: edit\n", "tools:\n  - read\n  - bash\n# next entry\nasync: true\n",
		"extensions: /srv/extension.js\nsubagentOnlyExtensions:\n  - /srv/second.ts\nskillPath: /srv/skills\ndefaultReads: /srv/input.md\n",
		"inheritProjectContext: true\ninheritGlobalContext: true\ninheritSkills: false\nallowNestedSubagents: false\nasync: false\ndefaultProgress: true\n",
		"systemPromptMode: append\ndefaultContext: fork\noutputMode: file-only\nacceptanceRole: writer\ntimeoutMs: 2147483647\ntoolTimeoutMs: 1\nmaxSubagentDepth: 0\n",
		"systemPromptMode: replace\ndefaultContext: fresh\noutputMode: inline\nacceptanceRole: read-only\n",
		"outputSchema: {\"type\":\"object\",\"properties\":{\"x\":{\"type\":\"string\"}}}\ntoolBudget: {}\n",
		"tools:\n# no continuation\nmodel: 'provider/model'\noutput: \"result.md\"\n",
	}
	for _, fields := range valid {
		t.Run("admit/"+fields, func(t *testing.T) {
			raw := []byte("---\nname: sample\ndescription: \"Local fixture\"\n" + fields + "---\nPrompt.\n")
			if _, err := ValidatePiAgent("sample", raw); err != nil {
				t.Fatal(err)
			}
		})
	}
	invalid := []string{
		"unknown: true", "name: sample", "runner: native", "permission: read", "acceptance:\n  role: writer", "memory: {}", "fallbackModels: small", "defaultProvider: p",
		"tools: []", "tools: [read]", "tools: null", "tools: ~", "tools: 'read'", "tools: \"read\"", "tools: read,,bash", "tools: read,./helper.js", "tools: helper.ts", "tools: helper.js", "tools: /helper", "tools: C:\\helper", "tools: mcp:query", "excludeTools: mcp:query", "excludeTools: ./helper", "tools: read # comment",
		"tools:\n  - read\n  # comment\n  - bash", "tools:\n  - read\n  \n  - bash", "tools:\n  - read\n\n  - bash", "tools:\n  - read\n# end\n  - bash", "tools:\n# empty\n  - read", "tools:\n  - read\n   - bash", "tools:\n\t- read", "tools:\n  - read,bash", "tools:\n  - ''",
		"extensions: ./helper.js", "extensions: ../helper.js", "skillPath: ~/skills", "defaultReads: /srv/../input", "subagentOnlyExtensions: ${ROOT}/x", "extensions: /srv/{x}",
		"async: yes", "inheritSkills: null", "inheritGlobalContext: true", "timeoutMs: 0", "timeoutMs: -1", "timeoutMs: +1", "toolTimeoutMs: 2147483648", "timeoutMs: '1'", "maxSubagentDepth: -1", "maxSubagentDepth:",
		"systemPromptMode: inherit", "defaultContext: unknown", "outputMode: file", "acceptanceRole: admin", "outputSchema: []", "outputSchema: null", "outputSchema: '{\"x\":1}'", "outputSchema: {bad}", "outputSchema: {\"x\":1,\"x\":2}", "toolBudget: {\"nested\":{\"x\":1,\"x\":2}}", "toolBudget:\n  x: 1",
		"model:", "model: null", "model: |", "model: >", "model: &anchor", "model: *anchor", "model: !tag x", "model: \"escape\\n\"", "model: 'a'b'", "model: 'unterminated", "model: abc # comment", "model: x\n  continued", "----", "---not-delimiter",
	}
	for _, fields := range invalid {
		t.Run("refuse/"+fields, func(t *testing.T) {
			_, err := ValidatePiAgent("sample", []byte("---\nname: sample\ndescription: Local\n"+fields+"\n---\nPrompt.\n"))
			if err == nil {
				t.Fatal("unsupported native shape admitted")
			}
			for _, part := range []string{"sample", "target pi", "field", "line"} {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("missing diagnostic %q: %v", part, err)
				}
			}
		})
	}
	for _, raw := range []string{"body", "---\nname: sample\ndescription: x", "---\nname: sample\ndescription: x\n---\n  ", "---\nname: other\ndescription: x\n---\nBody", "---\nname: sample\n---\nBody", "---\ndescription: x\n---\nBody", "---\nname: sample\ndescription: x\n---\n\xff"} {
		t.Run("document/"+raw, func(t *testing.T) {
			if _, err := ValidatePiAgent("sample", []byte(raw)); err == nil {
				t.Fatal("malformed document admitted")
			}
		})
	}
}

func TestPiAgentNativeLayout(t *testing.T) {
	cases := []struct {
		name   string
		change func(*manifest.Artifact, *manifest.Adapter)
	}{
		{"missing format", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.Format = "" }},
		{"legacy format", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.Format = "markdown" }},
		{"unknown format", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.Format = "other" }},
		{"non pi", func(a *manifest.Artifact, d *manifest.Adapter) { d.Tool = "claude" }},
		{"multiple targets", func(a *manifest.Artifact, d *manifest.Adapter) { a.Targets = append(a.Targets, "claude") }},
		{"no targets", func(a *manifest.Artifact, d *manifest.Adapter) { a.Targets = nil }},
		{"bodyIs", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.BodyIs = "prompt" }},
		{"frontmatter", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.Frontmatter.Passthrough = true }},
		{"empty allow policy", func(a *manifest.Artifact, d *manifest.Adapter) { d.Layout.Agent.Frontmatter.Allow = []string{} }},
		{"missing entry", func(a *manifest.Artifact, d *manifest.Adapter) { a.Entry = "" }},
		{"escaping entry", func(a *manifest.Artifact, d *manifest.Adapter) { a.Entry = "../agent.md" }},
		{"non markdown", func(a *manifest.Artifact, d *manifest.Adapter) { a.Entry = "agent.txt" }},
		{"files", func(a *manifest.Artifact, d *manifest.Adapter) { a.Files = []string{"data"} }},
		{"pi overrides", func(a *manifest.Artifact, d *manifest.Adapter) { a.Overrides = map[string]map[string]any{"pi": {}} }},
		{"other overrides", func(a *manifest.Artifact, d *manifest.Adapter) {
			a.Overrides = map[string]map[string]any{"claude": {"model": "x"}}
		}},
		{"wrong basename", func(a *manifest.Artifact, d *manifest.Adapter) {
			d.Layout.Agent.Global.Path = "~/.pi/agent/agents/alias.md"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := t.TempDir()
			if err := os.WriteFile(filepath.Join(src, "agent.md"), []byte("---\nname: sample\ndescription: Local\n---\nPrompt.\n"), 0600); err != nil {
				t.Fatal(err)
			}
			art := &manifest.Artifact{Meta: manifest.Meta{Name: "sample"}, Type: manifest.TypeAgent, Targets: []string{"pi"}, Entry: "agent.md"}
			ad := loadAdapter(t, "pi")
			tc.change(art, ad)
			if _, err := agentEngine(t, t.TempDir(), t.TempDir()).Transform(art, ad, "global", src, noExisting); err == nil {
				t.Fatal("incompatible layout admitted")
			}
		})
	}
}

func TestPiAgentNativeBytes(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(newline, func(t *testing.T) {
			src := t.TempDir()
			raw := bytes.ReplaceAll([]byte("---\nname: sample\ndescription: A local fixture\ntools:\noutputSchema: {\"type\":\"object\"}\n---\n\nKeep {name} verbatim.\n"), []byte("\n"), []byte(newline))
			if err := os.WriteFile(filepath.Join(src, "agent.md"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			art := &manifest.Artifact{Meta: manifest.Meta{Name: "sample"}, Type: manifest.TypeAgent, Targets: []string{"pi"}, Entry: "agent.md"}
			ds, err := agentEngine(t, t.TempDir(), t.TempDir()).Transform(art, loadAdapter(t, "pi"), "global", src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			if len(ds) != 1 || !bytes.Equal(ds[0].After, raw) {
				t.Fatalf("not raw-byte copy: %+v", ds)
			}
		})
	}
}

package adapter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestPiPathsIdentityAndSidecars(t *testing.T) {
	for _, kind := range []manifest.ArtifactType{manifest.TypeSkill, manifest.TypeCommand} {
		t.Run(string(kind), func(t *testing.T) {
			src := t.TempDir()
			mustWrite(t, filepath.Join(src, "entry.md"), "---\nname: wrong-name\ndescription: Fixture\n---\nBody\n")
			art := &manifest.Artifact{Meta: manifest.Meta{Name: "fixture-name"}, Type: kind, Entry: "entry.md"}
			if _, err := agentEngine(t, t.TempDir(), t.TempDir()).Transform(art, loadAdapter(t, "pi"), "global", src, noExisting); err == nil {
				t.Fatal("identity mismatch admitted")
			}
		})
	}
	for _, sidecar := range []string{"../outside", "alias", "support"} {
		t.Run(sidecar, func(t *testing.T) {
			src, outside := t.TempDir(), t.TempDir()
			mustWrite(t, filepath.Join(src, "SKILL.md"), "---\nname: fixture-skill\ndescription: Fixture\n---\nBody\n")
			mustWrite(t, filepath.Join(outside, "secret.md"), "must not copy")
			if err := os.Symlink(outside, filepath.Join(src, "alias")); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(src, "support"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(src, "support/leak.md")); err != nil {
				t.Fatal(err)
			}
			art := &manifest.Artifact{Meta: manifest.Meta{Name: "fixture-skill"}, Type: manifest.TypeSkill, Entry: "SKILL.md", Files: []string{sidecar}}
			if _, err := agentEngine(t, t.TempDir(), t.TempDir()).Transform(art, loadAdapter(t, "pi"), "global", src, noExisting); err == nil {
				t.Fatal("escaping or symlinked sidecar admitted")
			}
		})
	}
}

func TestPiLayoutNativeDeclaration(t *testing.T) {
	ad := loadAdapter(t, "pi")
	a := ad.Layout.Agent
	if a == nil || a.Format != "pi-subagents-markdown" || a.BodyIs != "" || a.Frontmatter.Passthrough || len(a.Frontmatter.Allow) != 0 || !reflect.DeepEqual(a.Required, []string{"name", "description"}) {
		t.Fatalf("native agent declaration = %+v", a)
	}
	if a.Global.Path != "~/.pi/agent/agents/{name}.md" || a.Project.Path != ".pi/agents/{name}.md" {
		t.Fatalf("agent destinations = %+v", a)
	}
	i := ad.Layout.Instruction
	if i.Global.File != "~/.pi/agent/AGENTS.md" || i.Project.File != "AGENTS.md" || i.Global.Action != "appendSection" || i.Project.Action != "appendSection" || i.PointerDirGlobal.OK() || i.PointerDirProject.OK() {
		t.Fatalf("inline context declaration = %+v", i)
	}
}

func TestPiResourcePathsAndSkillBytes(t *testing.T) {
	home, project, override, src := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	body := "---\nname: sample-resource\ndescription: Example resource\nlicense: MIT\n---\nInvented body.\n"
	mustWrite(t, filepath.Join(src, "SKILL.md"), body)
	mustWrite(t, filepath.Join(src, "prompt.md"), body)
	mustWrite(t, filepath.Join(src, "helpers", "sample.sh"), "#!/bin/sh\nprintf fixture\n")
	if err := os.Chmod(filepath.Join(src, "helpers", "sample.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	ad := loadAdapter(t, "pi")
	for _, tc := range []struct{ name, scope, override, root string }{
		{"default", "global", "", filepath.Join(home, ".pi", "agent")},
		{"override", "global", override, override},
		{"local", "local", override, filepath.Join(project, ".pi")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := func(k string) (string, bool) { return tc.override, k == "PI_CODING_AGENT_DIR" }
			eng := New(toolpath.New(env, home, project))
			art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "sample-resource", Version: "1.0.0", Role: manifest.RoleCapability}, Type: manifest.TypeSkill, Entry: "SKILL.md", Files: []string{"helpers"}}
			rows, err := eng.Transform(art, ad, tc.scope, src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 || rows[0].Path != filepath.Join(tc.root, "skills/sample-resource/SKILL.md") || string(rows[0].After) != body || rows[1].Mode != 0o755 || string(rows[1].After) != "#!/bin/sh\nprintf fixture\n" || rows[1].Path != filepath.Join(tc.root, "skills/sample-resource/helpers/sample.sh") {
				t.Fatalf("skill copy = %+v", rows)
			}
			for _, row := range rows {
				if row.Action != diff.Create || row.Tool != "pi" || row.Scope != tc.scope || row.Artifact != art.Name || row.Version != art.Version || row.Role != string(art.Role) {
					t.Fatalf("metadata lost: %+v", row)
				}
			}
			art.Type, art.Entry, art.Files = manifest.TypeCommand, "prompt.md", nil
			rows, err = eng.Transform(art, ad, tc.scope, src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0].Path != filepath.Join(tc.root, "prompts/sample-resource.md") || string(rows[0].After) != body {
				t.Fatalf("prompt copy = %+v", rows)
			}
			art.Type, art.Setting = manifest.TypeSetting, &manifest.SettingSpec{Path: "sampleFlag", Value: true}
			rows, err = eng.Transform(art, ad, tc.scope, src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || rows[0].Path != filepath.Join(tc.root, "settings.json") || !bytes.Contains(rows[0].After, []byte(`"sampleFlag": true`)) || rows[0].Setting == nil || rows[0].Setting.Target.Format != "json" {
				t.Fatalf("settings merge = %+v", rows)
			}
		})
	}
}

func TestPiUnsupportedSurfaces(t *testing.T) {
	eng := agentEngine(t, t.TempDir(), t.TempDir())
	for _, tc := range []struct {
		kind    manifest.ArtifactType
		message string
	}{
		{manifest.TypeHook, `adapter "pi": no Hook layout`},
		{manifest.TypeOutputStyle, `adapter "pi": no output-style layout`},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			_, err := eng.Transform(&manifest.Artifact{Meta: manifest.Meta{Name: "sample-unsupported"}, Type: tc.kind}, loadAdapter(t, "pi"), "global", t.TempDir(), noExisting)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error = %v, want %q", err, tc.message)
			}
		})
	}
}

func TestPiMCPTransportTemplates(t *testing.T) {
	ad := loadAdapter(t, "pi")
	for _, scope := range []string{"global", "local"} {
		t.Run(scope, func(t *testing.T) {
			ft, err := ad.Layout.Mcp.ResolveTarget(scope)
			if err != nil {
				t.Fatal(err)
			}
			wantFile := "~/.pi/agent/mcp-adapter.json"
			if scope == "local" {
				wantFile = ".pi/mcp-adapter.json"
			}
			if ft.File != wantFile || ft.Format != "json" {
				t.Fatalf("MCP target = %+v", ft)
			}
			for _, tc := range []struct {
				transport string
				values    map[string]any
			}{
				{"http", map[string]any{"url": "https://fixture.invalid/mcp"}},
				{"stdio", map[string]any{"command": "fixture-server", "args": []any{"--read-only"}}},
			} {
				t.Run(tc.transport, func(t *testing.T) {
					data, err := MergeConfig([]byte(`{"keep":true}`), ft, ad.Layout.Mcp.Transports[tc.transport], ServerSpec{Name: "sample-server", Transport: tc.transport, Values: tc.values})
					if err != nil {
						t.Fatal(err)
					}
					var got map[string]any
					if err := json.Unmarshal(data, &got); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{"keep": true, "mcpServers": map[string]any{"sample-server": tc.values}}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("merge = %s, want %+v", data, want)
					}
				})
			}
		})
	}
}

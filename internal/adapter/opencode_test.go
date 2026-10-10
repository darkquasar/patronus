package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// decodeAdapter builds an invented adapter from inline YAML.
func decodeAdapter(t *testing.T, body string) *manifest.Adapter {
	t.Helper()
	ad, err := manifest.DecodeAdapter([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return ad
}

func inlineInstructionArtifact(name string) *manifest.Artifact {
	return &manifest.Artifact{
		Meta:  manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Role: manifest.RoleInstruction},
		Type:  manifest.TypeInstruction,
		Entry: "INSTRUCTIONS.md",
	}
}

// TestInstructionSectionPrefixFromLayout: the write-side section name comes
// from the adapter's InstructionLayout.SectionPrefix for every tool. An
// invented prefix yields prefix+name, the Pi adapter still yields pi:<name>
// with today's bytes, and an adapter without the field yields the bare name.
func TestInstructionSectionPrefixFromLayout(t *testing.T) {
	src := writeInstruction(t, "body\n")
	home, proj := t.TempDir(), t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, proj))
	toy := decodeAdapter(t, `apiVersion: patronus/v2
family: adapter
tool: toy
layout:
  instruction:
    global: { file: "~/.toy/TOY.md", action: appendSection }
    project: { file: "TOY.md", action: appendSection }
    sectionPrefix: "toy:"
`)
	bare := decodeAdapter(t, `apiVersion: patronus/v2
family: adapter
tool: bare
layout:
  instruction:
    global: { file: "~/.bare/BARE.md", action: appendSection }
    project: { file: "BARE.md", action: appendSection }
`)
	for _, tc := range []struct {
		name    string
		ad      *manifest.Adapter
		section string
	}{
		{"invented prefix", toy, "toy:demo"},
		{"pi keeps pi: prefix", loadAdapter(t, "pi"), "pi:demo"},
		{"no prefix yields the bare name", bare, "demo"},
		{"claude has no prefix", loadAdapter(t, "claude"), "demo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diffs, err := eng.Transform(inlineInstructionArtifact("demo"), tc.ad, "local", src, noExisting)
			if err != nil {
				t.Fatal(err)
			}
			if len(diffs) != 1 || diffs[0].Section == nil || diffs[0].Section.Name != tc.section {
				t.Fatalf("diffs = %+v", diffs)
			}
			if want := AppendSection(nil, tc.section, []byte("body\n")); string(diffs[0].After) != string(want) {
				t.Fatalf("section bytes\n got: %q\nwant: %q", diffs[0].After, want)
			}
		})
	}
}

// TestDeclarativeGateWritesSettingFile: a script-less gate on an adapter whose
// hook registration file and setting file differ writes permission.<key> =
// "deny" into the SETTING file (declarative permission lives in the agent's
// settings, not in a hooks registry) and nothing into the hook file; an adapter
// with no setting layout is refused rather than written into the hook file.
func TestDeclarativeGateWritesSettingFile(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, proj))
	split := decodeAdapter(t, `apiVersion: patronus/v2
family: adapter
tool: opencode
layout:
  hook:
    global: { file: "~/.toy/hooks.json", format: json, path: "hooks.{event}", action: merge }
    project: { file: ".toy/hooks.json", format: json, path: "hooks.{event}", action: merge }
  setting:
    global: { file: "~/.toy/settings.json", format: jsonc }
    project: { file: "toy.json", format: jsonc }
`)
	art := hookArtifact("no-bash", "PreToolUse", "bash", "block")
	art.Hook.Intent = manifest.HookGate
	diffs, err := eng.Transform(art, split, "local", "", noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Path != filepath.Join(proj, "toy.json") {
		t.Fatalf("declarative gate landed at %+v", diffs)
	}
	if diffs[0].Setting == nil || diffs[0].Setting.Dotted != "permission.bash" || diffs[0].Setting.ScalarValue != "deny" || diffs[0].Setting.Target.File != "toy.json" {
		t.Fatalf("setting edit = %+v", diffs[0].Setting)
	}
	if !strings.Contains(string(diffs[0].After), `"deny"`) {
		t.Fatalf("after = %s", diffs[0].After)
	}

	noSetting := decodeAdapter(t, `apiVersion: patronus/v2
family: adapter
tool: opencode
layout:
  hook:
    global: { file: "~/.toy/hooks.json", format: json, path: "hooks.{event}", action: merge }
    project: { file: ".toy/hooks.json", format: json, path: "hooks.{event}", action: merge }
`)
	_, err = eng.Transform(art, noSetting, "local", "", noExisting)
	if err == nil || err.Error() != `adapter "opencode": declarative gate "no-bash" needs a setting layout` {
		t.Fatalf("no setting layout: %v", err)
	}
}

// TestAgentAllowListDropsUnknownKeysWithWarning: every allow-listed key
// survives (a nested permission map intact), an unknown key is dropped and the
// diff carries a Warning naming it; `name` is the file identity and is dropped
// silently.
func TestAgentAllowListDropsUnknownKeysWithWarning(t *testing.T) {
	src := t.TempDir()
	agent := "---\nname: demo\ndescription: OpenCode only. Demo role\nmode: subagent\nmodel: toy/model\ninvented_key: 1\npermission:\n  edit: deny\n  bash:\n    \"*\": ask\n    \"git status*\": allow\n---\nYou are the demo role.\n"
	if err := os.WriteFile(filepath.Join(src, "agent.md"), []byte(agent), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "demo"}, Type: manifest.TypeAgent, Entry: "agent.md"}
	diffs, err := eng.Transform(art, loadAdapter(t, "opencode"), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Action != diff.Create {
		t.Fatalf("diffs = %+v", diffs)
	}
	s := string(diffs[0].After)
	for _, want := range []string{"description: OpenCode only. Demo role", "mode: subagent", "model: toy/model", "permission:", "edit: deny", "git status*", "prompt: You are the demo role."} {
		if !strings.Contains(s, want) {
			t.Errorf("emitted frontmatter missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "invented_key") || strings.Contains(s, "name: demo") {
		t.Errorf("dropped key leaked:\n%s", s)
	}
	if want := `agent "demo": frontmatter key(s) invented_key are not in the opencode allow-list and were dropped`; diffs[0].Warning != want {
		t.Fatalf("warning = %q, want %q", diffs[0].Warning, want)
	}
	// A clean frontmatter carries no warning.
	clean := strings.Replace(strings.Replace(agent, "invented_key: 1\n", "", 1), "name: demo\n", "", 1)
	if err := os.WriteFile(filepath.Join(src, "agent.md"), []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}
	diffs, err = eng.Transform(art, loadAdapter(t, "opencode"), "local", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if diffs[0].Warning != "" {
		t.Fatalf("clean frontmatter warned: %q", diffs[0].Warning)
	}
}

// TestOpenCodeRootContract pins where every OpenCode surface lands at both
// scopes under the three global-base states (OPENCODE_CONFIG_DIR, XDG_CONFIG_HOME,
// neither), and that both adapter copies declare the runner dirs, the setting
// layout with its redact and split-leaf data, leaf MCP ownership and the oc:
// section prefix.
func TestOpenCodeRootContract(t *testing.T) {
	ad := loadAdapter(t, "opencode")
	home, project, cfg, xdg := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	for _, env := range []struct {
		name string
		env  toolpath.EnvLookup
		base string
	}{
		{"OPENCODE_CONFIG_DIR", func(k string) (string, bool) {
			switch k {
			case "OPENCODE_CONFIG_DIR":
				return cfg, true
			case "XDG_CONFIG_HOME":
				return xdg, true
			}
			return "", false
		}, cfg},
		{"XDG_CONFIG_HOME", func(k string) (string, bool) {
			if k == "XDG_CONFIG_HOME" {
				return xdg, true
			}
			return "", false
		}, filepath.Join(xdg, "opencode")},
		{"neither", testEnv(home), filepath.Join(home, ".config/opencode")},
	} {
		t.Run(env.name, func(t *testing.T) {
			res := toolpath.New(env.env, home, project)
			resolve := func(marker, scope string) string {
				return res.ResolveMarker(strings.ReplaceAll(marker, "{name}", "demo-oc"), "opencode", scope)
			}
			for _, tc := range []struct{ surface, got, want string }{
				{"skill global", resolve(ad.Layout.Skill.Global.Path, "global"), filepath.Join(env.base, "skills/demo-oc/SKILL.md")},
				{"skill project", resolve(ad.Layout.Skill.Project.Path, "local"), filepath.Join(project, ".opencode/skills/demo-oc/SKILL.md")},
				{"agent global", resolve(ad.Layout.Agent.Global.Path, "global"), filepath.Join(env.base, "agents/demo-oc.md")},
				{"agent project", resolve(ad.Layout.Agent.Project.Path, "local"), filepath.Join(project, ".opencode/agents/demo-oc.md")},
				{"command global", resolve(ad.Layout.Command.Global.Path, "global"), filepath.Join(env.base, "commands/demo-oc.md")},
				{"command project", resolve(ad.Layout.Command.Project.Path, "local"), filepath.Join(project, ".opencode/commands/demo-oc.md")},
				{"hook script dir global", resolve(ad.Layout.Hook.ScriptDirFor("global").Path, "global"), filepath.Join(env.base, "patronus/hooks")},
				{"hook script dir project", resolve(ad.Layout.Hook.ScriptDirFor("local").Path, "local"), filepath.Join(project, ".opencode/patronus/hooks")},
				{"runner dir global", resolve(ad.Layout.Hook.RunnerDirFor("global").Path, "global"), filepath.Join(env.base, "plugins")},
				{"runner dir project", resolve(ad.Layout.Hook.RunnerDirFor("local").Path, "local"), filepath.Join(project, ".opencode/plugins")},
				{"hooks.json global", resolve(ad.Layout.Hook.ForScope("global").File, "global"), filepath.Join(env.base, "patronus/hooks.json")},
				{"hooks.json project", resolve(ad.Layout.Hook.ForScope("local").File, "local"), filepath.Join(project, ".opencode/patronus/hooks.json")},
				{"setting global", resolve(ad.Layout.Setting.ForScope("global").File, "global"), filepath.Join(env.base, "opencode.json")},
				{"setting project", resolve(ad.Layout.Setting.ForScope("local").File, "local"), filepath.Join(project, "opencode.json")},
				{"mcp global", resolve(ad.Layout.Mcp.Global.File, "global"), filepath.Join(env.base, "opencode.json")},
				{"mcp project", resolve(ad.Layout.Mcp.Project.File, "local"), filepath.Join(project, "opencode.json")},
				{"pointer dir global", resolve(ad.Layout.Instruction.PointerDirFor("global").Path, "global"), filepath.Join(env.base, "patronus/instructions/demo-oc.md")},
				{"pointer dir project", resolve(ad.Layout.Instruction.PointerDirFor("local").Path, "local"), filepath.Join(project, ".opencode/patronus/instructions/demo-oc.md")},
				{"context global", resolve(ad.Layout.Instruction.Global.File, "global"), filepath.Join(env.base, "AGENTS.md")},
				{"context project", resolve(ad.Layout.Instruction.Project.File, "local"), filepath.Join(project, "AGENTS.md")},
			} {
				if tc.got != tc.want {
					t.Errorf("%s = %s, want %s", tc.surface, tc.got, tc.want)
				}
			}
		})
	}
	if ad.Layout.Mcp.Ownership != manifest.OwnershipLeaf || ad.Layout.Instruction.SectionPrefix != "oc:" {
		t.Fatalf("ownership = %q, sectionPrefix = %q", ad.Layout.Mcp.Ownership, ad.Layout.Instruction.SectionPrefix)
	}
	if got := strings.Join(ad.Layout.Setting.Redact, ","); got != "provider.*.options" {
		t.Fatalf("redact = %q", got)
	}
	if got := strings.Join(ad.Layout.Setting.SplitLeaves, ","); got != "permission.edit" {
		t.Fatalf("splitLeaves = %q", got)
	}
	if ad.Layout.Setting.Global.Format != "jsonc" || ad.Layout.Hook.Global.Format != "json" || ad.Layout.Hook.Global.Path != "hooks.{event}" {
		t.Fatalf("formats: setting %q hook %q path %q", ad.Layout.Setting.Global.Format, ad.Layout.Hook.Global.Format, ad.Layout.Hook.Global.Path)
	}
}

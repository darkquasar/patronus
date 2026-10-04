package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dp02Write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func dp02Env(values map[string]string) EnvLookup {
	return func(key string) (string, bool) { v, ok := values[key]; return v, ok }
}

func TestPiDiscoveryContext(t *testing.T) {
	root := t.TempDir()
	dp02Write(t, filepath.Join(root, "CLAUDE.md"), "Shared prior\n")
	context, err := DiscoverPiContext(root, ReadPiFile)
	if err != nil {
		t.Fatal(err)
	}
	if context.Path != filepath.Join(root, "CLAUDE.md") || !context.Mixed {
		t.Fatalf("wrong effective context: %+v", context)
	}
	dp02Write(t, filepath.Join(root, "AGENTS.override.md"), "Prepared shared content\n")
	context, err = DiscoverPiContext(root, ReadPiFile)
	if err != nil {
		t.Fatal(err)
	}
	// Case-insensitive filesystems expose CLAUDE.md under both discovery names.
	wantCandidates := 2
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.MD")); err == nil {
		wantCandidates++
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if !strings.HasSuffix(context.Path, "AGENTS.override.md") || context.Warning == "" || len(context.Candidates) != wantCandidates || !context.Mixed {
		t.Fatalf("wrong precedence: %+v", context)
	}
	injected := func(path string) ([]byte, bool, error) {
		if filepath.Base(path) == "AGENTS.override.md" {
			return nil, true, fmt.Errorf("unreadable earlier candidate")
		}
		return ReadPiFile(path)
	}
	if _, err := DiscoverPiContext(root, injected); err == nil || !strings.Contains(err.Error(), "AGENTS.override.md") {
		t.Fatalf("earlier unreadable candidate admitted: %v", err)
	}
}

func TestPiDiscoverySources(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	agent := filepath.Join(home, "relocated")
	if err := os.MkdirAll(filepath.Join(project, ".pi"), 0700); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(root, "package")
	dp02Write(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-kit","pi":{"skills":["skills"],"prompts":["prompts"],"subagents":{"agents":["roles"]}}}`)
	dp02Write(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"packages":[%q]}`, pkg))
	dp02Write(t, filepath.Join(pkg, "skills/package-skill/SKILL.md"), "---\nname: package-skill\ndescription: Fixture\n---\nBody\n")
	dp02Write(t, filepath.Join(pkg, "prompts/package-prompt.md"), "Prompt bytes\n")
	dp02Write(t, filepath.Join(pkg, "roles/package-agent.md"), "---\nname: package-agent\ndescription: Fixture\n---\nBody\n")
	dp02Write(t, filepath.Join(agent, "skills/native-skill/SKILL.md"), "---\nname: native-skill\ndescription: Fixture\n---\nBody\n")
	dp02Write(t, filepath.Join(project, ".pi/agents/project-agent.md"), "---\nname: project-agent\ndescription: Fixture\n---\nBody\n")
	builtin := filepath.Join(root, "builtin")
	dp02Write(t, filepath.Join(builtin, "builtin-agent.md"), "---\nname: builtin-agent\ndescription: Fixture\n---\nBody\n")
	roots, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": "TrUe"}))
	if err != nil {
		t.Fatal(err)
	}
	roots = append([]PiResourceRoot{{Kind: "agent", Path: builtin, Origin: "qualified builtin"}}, roots...)
	inventory, err := DiscoverPiResources(roots)
	if err != nil {
		t.Fatal(err)
	}
	positions := map[string]int{}
	for i, r := range inventory {
		positions[r.Name] = i
	}
	for _, name := range []string{"package-skill", "native-skill", "package-agent", "project-agent", "builtin-agent"} {
		if _, ok := positions[name]; !ok {
			t.Errorf("missing source %s: %+v", name, inventory)
		}
	}
	if positions["package-skill"] >= positions["native-skill"] || positions["builtin-agent"] >= positions["package-agent"] || positions["package-agent"] >= positions["project-agent"] {
		t.Fatalf("Pi/subagents tier order: %+v", inventory)
	}
	for _, v := range []string{"", "false", "0"} {
		var warnings []string
		if _, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": v}), func(message string) { warnings = append(warnings, message) }); err != nil {
			t.Errorf("runtime-only global npm discovery blocked static admission for %q: %v", v, err)
		}
		if !strings.Contains(strings.Join(warnings, "\n"), "global npm discovery is runtime-unverified") {
			t.Errorf("missing global npm boundary for %q: %v", v, warnings)
		}
	}
	for _, v := range []string{"1", "YES", "true"} {
		if _, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": v})); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := PiResourceRoots(home, project, agent, false, dp02Env(nil)); err != nil {
		t.Fatalf("unrelated resource route requires offline: %v", err)
	}
	selected := PiResource{Kind: "skill", Name: "package-skill", Path: filepath.Join(project, ".pi/skills/package-skill/SKILL.md"), Origin: "selected local"}
	err = PiResourceConflict(inventory, selected)
	if err == nil || !strings.Contains(err.Error(), pkg) || !strings.Contains(err.Error(), selected.Path) {
		t.Fatalf("collision missing both origins: %v", err)
	}
	// Identical name in user/project origins is never adopted or silently shadowed.
	err = PiResourceConflict([]PiResource{{Kind: "agent", Name: "same", Path: "/global/same.md", Origin: "user"}}, PiResource{Kind: "agent", Name: "same", Path: "/project/same.md", Origin: "project"})
	if err == nil {
		t.Fatal("project/global collision accepted")
	}
}

func TestPiDiscoveryNativeAgentAliases(t *testing.T) {
	t.Run("installed oracle metadata shape", func(t *testing.T) {
		root := t.TempDir()
		dp02Write(t, filepath.Join(root, "oracle.md"), `---
name: oracle
aliases: advisor
description: High-context decision-consistency oracle that protects inherited state and prevents drift
tools: read, grep, find, ls, bash
thinking: high
systemPromptMode: replace
inheritProjectContext: true
inheritSkills: false
defaultContext: fork
---

Invented compatibility body.
`)
		inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "package fixture"}})
		if err != nil {
			t.Fatalf("scalar alias compatibility: %v", err)
		}
		if len(inventory) != 1 || inventory[0].Name != "oracle" || len(inventory[0].Aliases) != 1 || inventory[0].Aliases[0] != "advisor" {
			t.Fatalf("unexpected inventory: %+v", inventory)
		}
	})

	t.Run("supported scalar fields filter empty values", func(t *testing.T) {
		root := t.TempDir()
		dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\nalias: first\ndescription: Fixture\n---\nBody\n")
		dp02Write(t, filepath.Join(root, "two.md"), "---\nname: two\naliases: second, , third,\ndescription: Fixture\n---\nBody\n")
		inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
		if err != nil || len(inventory) != 2 {
			t.Fatalf("supported scalar aliases: %+v, %v", inventory, err)
		}
		if got := inventory[1].Aliases; len(got) != 2 || got[0] != "second" || got[1] != "third" {
			t.Fatalf("scalar alias empty filtering mismatch: %v", got)
		}
	})

	for _, tc := range []struct {
		name, firstName, firstAliases, secondName, secondAliases string
	}{
		{name: "primary-primary", firstName: "shared", secondName: "shared"},
		{name: "primary-alias", firstName: "one", firstAliases: "shared", secondName: "shared"},
		{name: "alias-primary", firstName: "shared", secondName: "two", secondAliases: "shared"},
		{name: "alias-alias", firstName: "one", firstAliases: "shared", secondName: "two", secondAliases: "shared"},
	} {
		t.Run(tc.name+" collision", func(t *testing.T) {
			root := t.TempDir()
			first := fmt.Sprintf("---\nname: %s\ndescription: Fixture\n", tc.firstName)
			if tc.firstAliases != "" {
				first += "aliases: " + tc.firstAliases + "\n"
			}
			second := fmt.Sprintf("---\nname: %s\ndescription: Fixture\n", tc.secondName)
			if tc.secondAliases != "" {
				second += "aliases: " + tc.secondAliases + "\n"
			}
			dp02Write(t, filepath.Join(root, "a.md"), first+"---\nBody\n")
			dp02Write(t, filepath.Join(root, "b.md"), second+"---\nBody\n")
			_, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
			if err == nil || !strings.Contains(err.Error(), "shared") || !strings.Contains(err.Error(), "a.md") || !strings.Contains(err.Error(), "b.md") {
				t.Fatalf("cross-source collision admitted or poorly diagnosed: %v", err)
			}
		})
	}

	t.Run("same source aliases", func(t *testing.T) {
		root := t.TempDir()
		dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\naliases: one, helper, helper\ndescription: Fixture\n---\nBody\n")
		inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
		if err != nil {
			t.Fatalf("same-source aliases self-conflicted: %v", err)
		}
		if len(inventory) != 1 || len(inventory[0].Aliases) != 1 || inventory[0].Aliases[0] != "helper" {
			t.Fatalf("same-source aliases not deduplicated: %+v", inventory)
		}
	})

	for _, aliases := range []string{"../outside", "/absolute", `back\\slash`, "package:agent", ".", "..", "[first, second]", "{first: second}"} {
		t.Run("unsafe "+aliases, func(t *testing.T) {
			root := t.TempDir()
			dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\naliases: "+aliases+"\ndescription: Fixture\n---\nBody\n")
			if _, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}}); err == nil {
				t.Fatalf("unsafe or malformed aliases %q admitted", aliases)
			}
		})
	}

	t.Run("supported block list", func(t *testing.T) {
		root := t.TempDir()
		dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\naliases:\n  - advisor\n  - second, , third\ndescription: Fixture\n---\nBody\n")
		inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
		if err != nil || len(inventory) != 1 {
			t.Fatalf("supported block aliases: %+v, %v", inventory, err)
		}
		got := inventory[0].Aliases
		if len(got) != 3 || got[0] != "advisor" || got[1] != "second" || got[2] != "third" {
			t.Fatalf("block alias parsing mismatch: %v", got)
		}
	})

	for _, tc := range []struct {
		name, metadata string
	}{
		{name: "nested mapping", metadata: "aliases:\n  child:\n    - nested\n"},
		{name: "inconsistent indentation", metadata: "aliases:\n  - first\n    - nested\n"},
		{name: "non-list block", metadata: "aliases:\n  advisor\n"},
		{name: "empty list item", metadata: "aliases:\n  -\n"},
	} {
		t.Run("malformed block "+tc.name, func(t *testing.T) {
			root := t.TempDir()
			dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\n"+tc.metadata+"description: Fixture\n---\nBody\n")
			if _, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}}); err == nil {
				t.Fatalf("malformed alias block admitted: %q", tc.metadata)
			}
		})
	}

	for _, tc := range []struct {
		name, metadata string
		want           []string
	}{
		{name: "duplicate singular last wins", metadata: "alias: first\nalias: second\n", want: []string{"second"}},
		{name: "duplicate plural last wins", metadata: "aliases: first\naliases: second, third\n", want: []string{"second", "third"}},
		{name: "plural wins when declared last", metadata: "alias: singular\naliases: plural\n", want: []string{"plural"}},
		{name: "plural wins when declared first", metadata: "aliases: plural\nalias: singular\n", want: []string{"plural"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dp02Write(t, filepath.Join(root, "one.md"), "---\nname: one\n"+tc.metadata+"description: Fixture\n---\nBody\n")
			inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
			if err != nil || len(inventory) != 1 {
				t.Fatalf("alias assignment semantics: %+v, %v", inventory, err)
			}
			got := inventory[0].Aliases
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("aliases=%v want %v", got, tc.want)
			}
		})
	}

	t.Run("duplicate unrelated metadata preserves parser behavior", func(t *testing.T) {
		root := t.TempDir()
		dp02Write(t, filepath.Join(root, "one.md"), "---\nname: prior\nname: one\ndescription: Prior\ndescription: Fixture\n---\nBody\n")
		inventory, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: root, Origin: "fixture"}})
		if err != nil || len(inventory) != 1 || inventory[0].Name != "one" {
			t.Fatalf("unrelated duplicate metadata semantics changed: %+v, %v", inventory, err)
		}
	})
}

func TestPiDiscoveryNpmAndBuiltinSources(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	agent := filepath.Join(home, "agent")
	dp02Write(t, filepath.Join(project, ".pi/settings.json"), "{}")
	pkg := filepath.Join(agent, "npm/node_modules/fixture-kit")
	dp02Write(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-kit","pi":{"skills":["skills"]},"pi-subagents":{"agents":["roles"]}}`)
	dp02Write(t, filepath.Join(pkg, "skills/fixture-skill/SKILL.md"), "---\nname: fixture-skill\ndescription: Fixture\n---\nBody\n")
	dp02Write(t, filepath.Join(pkg, "roles/package-role.md"), "---\nname: package-role\ndescription: Fixture\n---\nBody\n")
	dp02Write(t, filepath.Join(agent, "settings.json"), `{"packages":["npm:fixture-kit@1.2.3","git:fixture.invalid/team/cached-kit@v1"]}`)
	gitRoot := filepath.Join(agent, "git/fixture.invalid/team/cached-kit")
	dp02Write(t, filepath.Join(gitRoot, "package.json"), `{"name":"cached-kit","pi":{"subagents":{"agents":["roles"]}}}`)
	dp02Write(t, filepath.Join(gitRoot, "roles/cached-role.md"), "---\nname: cached-role\ndescription: Fixture\n---\nBody\n")
	// Invented consumer metadata exercises the native builtin-directory interface;
	// these are not acquired package bytes or a catalog recipe/profile assertion.
	builtin := filepath.Join(agent, "npm/node_modules/native-consumer")
	dp02Write(t, filepath.Join(builtin, "package.json"), `{"name":"pi-subagents","version":"0.72.1","pi":{}}`)
	dp02Write(t, filepath.Join(builtin, "agents/builtin-role.md"), "---\nname: builtin-role\ndescription: Fixture\n---\nBody\n")
	roots, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": "YES"}))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := DiscoverPiResources(roots)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, r := range inventory {
		names[r.Name] = true
	}
	for _, name := range []string{"fixture-skill", "package-role", "builtin-role", "cached-role"} {
		if !names[name] {
			t.Errorf("static source %s missing: %+v", name, inventory)
		}
	}
	if inventory[0].Origin != "subagents builtin: "+filepath.Join(builtin, "package.json") {
		t.Fatalf("builtin tier not first: %+v", inventory)
	}
}

func TestPiDiscoveryIdenticalNpmSourcePrefersProject(t *testing.T) {
	root := t.TempDir()
	home, project := filepath.Join(root, "home"), filepath.Join(root, "project")
	agent := filepath.Join(home, "agent")
	local := filepath.Join(project, ".pi")
	for _, base := range []string{local, agent} {
		dp02Write(t, filepath.Join(base, "settings.json"), `{"packages":["npm:pi-subagents@0.72.1"]}`)
		pkg := filepath.Join(base, "npm/node_modules/pi-subagents")
		dp02Write(t, filepath.Join(pkg, "package.json"), `{"name":"pi-subagents","version":"0.72.1","pi":{"skills":["skills"]}}`)
		dp02Write(t, filepath.Join(pkg, "skills/consumer-guide/SKILL.md"), "---\nname: consumer-guide\ndescription: Fixture\n---\nBody\n")
		dp02Write(t, filepath.Join(pkg, "agents/builtin-role.md"), "---\nname: builtin-role\ndescription: Fixture\n---\nBody\n")
	}
	roots, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": "true"}))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := DiscoverPiResources(roots)
	if err != nil {
		t.Fatalf("identical global/project package falsely conflicts: %v", err)
	}
	if len(inventory) != 2 {
		t.Fatalf("want one selected builtin and skill, got %+v", inventory)
	}
	for _, resource := range inventory {
		if !strings.HasPrefix(resource.Path, local+string(filepath.Separator)) {
			t.Fatalf("inactive global copy selected: %+v", resource)
		}
	}
	// Native definitions still conflict with package resources; selecting one
	// npm copy must not relax the ordinary identity-shadowing boundary.
	dp02Write(t, filepath.Join(local, "agents/builtin-role.md"), "---\nname: builtin-role\ndescription: User role\n---\nBody\n")
	roots, err = PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": "true"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverPiResources(roots); err == nil || !strings.Contains(err.Error(), "builtin-role") {
		t.Fatalf("native shadowing accepted: %v", err)
	}
}

func TestPiMCPBranding(t *testing.T) {
	root := t.TempDir()
	host := filepath.Join(root, "host")
	agent := filepath.Join(root, "agent")
	project := filepath.Join(root, "project")
	env := dp02Env(map[string]string{"PI_PACKAGE_DIR": host})
	for _, body := range []string{`{"name":"fixture-host","piConfig":{"configDir":".pi"}}`, `{"piConfig":{"name":"pi","configDir":".pi"}}`, `{}`} {
		dp02Write(t, filepath.Join(host, "package.json"), body)
		if _, err := DiscoverPiMCP(root, project, agent, filepath.Join(agent, "mcp-adapter.json"), env, ReadPiFile); err != nil {
			t.Fatalf("unchanged default branding refused: %v", err)
		}
	}
	for _, body := range []string{`{"piConfig":{"name":"other"}}`, `{"piConfig":{"configDir":".other"}}`, `{"piConfig":[]}`, `not JSON`} {
		dp02Write(t, filepath.Join(host, "package.json"), body)
		if _, err := DiscoverPiMCP(root, project, agent, filepath.Join(agent, "mcp-adapter.json"), env, ReadPiFile); err == nil {
			t.Fatal("unknown path override admitted")
		}
	}
}

func TestPiDiscoveryUnknownSources(t *testing.T) {
	for _, settings := range []string{`{"packages":["npm:missing"]}`, `{"packages":["git:example.invalid/kit"]}`, `{"skills":["./**"]}`, `{"subagents":{"agentExcludeDirs":["x"]}}`, `{"extensions":42}`} {
		t.Run(settings, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			agent := filepath.Join(home, "agent")
			project := filepath.Join(root, "project")
			dp02Write(t, filepath.Join(project, ".pi/settings.json"), "{}")
			dp02Write(t, filepath.Join(agent, "settings.json"), settings)
			if _, err := PiResourceRoots(home, project, agent, true, dp02Env(map[string]string{"PI_OFFLINE": "true"})); err == nil {
				t.Fatal("unknown active discovery admitted")
			}
		})
	}
}

func TestPiPaths(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	dp02Write(t, filepath.Join(outside, "data.md"), "external")
	if err := os.Symlink(outside, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "alias/data.md"), filepath.Join(root, "alias/new.md"), "relative/file"} {
		if err := PiSafePath(path); err == nil {
			t.Errorf("unsafe path %s admitted", path)
		}
	}
	for _, rel := range []string{"../data.md", "/data.md", "a/../data.md", "a\\data.md"} {
		if _, err := PiSourcePath(root, rel); err == nil {
			t.Errorf("escaping source %s admitted", rel)
		}
	}
	if _, err := PiSourcePath(root, "new/data.md"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "unreadable.md")
	dp02Write(t, file, "prior")
	if err := os.Chmod(file, 0000); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadPiFile(file); err == nil {
		t.Fatal("unreadable candidate admitted")
	}
	if _, _, err := ReadPiFile(root); err == nil {
		t.Fatal("directory admitted as file")
	}
}

func TestPiMCP(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "work/project")
	agent := filepath.Join(home, "relocated")
	global := filepath.Join(agent, "mcp-adapter.json")
	local := filepath.Join(project, ".pi/mcp-adapter.json")
	paths := []string{filepath.Join(home, ".config/mcp/mcp.json"), filepath.Join(home, ".agents/mcp.json"), filepath.Join(home, ".agents/mcp/mcp.json"), global, filepath.Join(project, ".mcp.json"), local}
	for i, path := range paths {
		dp02Write(t, path, fmt.Sprintf(`{"mcpServers":{"fixture-%d":{"url":"http://127.0.0.1/mcp"}}}`, i))
	}
	sources, err := DiscoverPiMCP(home, project, agent, local, dp02Env(nil), ReadPiFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != len(paths) {
		t.Fatalf("sources: %+v", sources)
	}
	for i, source := range sources {
		if source.Path != paths[i] {
			t.Fatalf("precedence %d: %s", i, source.Path)
		}
	}
	dp02Write(t, local, `{"mcpServers":{"fixture_0":{"url":"http://127.0.0.1/mcp"}}}`)
	if _, err := DiscoverPiMCP(home, project, agent, local, dp02Env(nil), ReadPiFile); err == nil || !strings.Contains(err.Error(), paths[0]) || !strings.Contains(err.Error(), local) {
		t.Fatalf("normalized duplicate not refused: %v", err)
	}
	dp02Write(t, local, `{"mcpServers":{}}`)
	for _, body := range []string{`{"imports":["codex"]}`, `{"settings":{"agentPluginPaths":["/tmp/plugin"]}}`, `{"claudePlugins":["fixture"]}`, `{"settings":{"hostConfigDiscovery":"on"}}`, `{"mcpServers":{"x":{},"x":{}}}`, `not json`} {
		dp02Write(t, global, body)
		if _, err := DiscoverPiMCP(home, project, agent, local, dp02Env(nil), ReadPiFile); err == nil {
			t.Errorf("active or malformed source admitted: %s", body)
		}
	}
	dp02Write(t, global, `{"settings":{"hostConfigDiscovery":"prompt"},"mcpServers":{}}`)
	dp02Write(t, filepath.Join(home, ".cursor/mcp.json"), "not active; not JSON")
	if _, err := DiscoverPiMCP(home, project, agent, local, dp02Env(nil), ReadPiFile); err != nil {
		t.Fatalf("dormant host source read: %v", err)
	}
	if _, err := DiscoverPiMCP(home, project, agent, local, dp02Env(map[string]string{"PI_MCP_CONFIG_MODE": "exclusive"}), ReadPiFile); err == nil {
		t.Fatal("exclusive destination mismatch admitted")
	}
	sources, err = DiscoverPiMCP(home, project, agent, global, dp02Env(map[string]string{"PI_MCP_CONFIG_MODE": "exclusive"}), ReadPiFile)
	if err != nil || len(sources) != 1 {
		t.Fatalf("exclusive: %+v %v", sources, err)
	}
	dp02Write(t, global, fmt.Sprintf(`{"settings":{"ancestorConfigRoots":[%q]}}`, home))
	ancestor := filepath.Join(home, "work/.mcp.json")
	dp02Write(t, ancestor, `{"mcpServers":{"ancestor":{}}}`)
	sources, err = DiscoverPiMCP(home, project, agent, local, dp02Env(nil), ReadPiFile)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range sources {
		found = found || s.Path == ancestor
	}
	if !found {
		t.Fatal("opted-in ancestor omitted")
	}
}

func TestPiDiscoveryExtensionDirectories(t *testing.T) {
	for _, kind := range []string{"explicit", "package", "native", "conventional package"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			project := filepath.Join(home, "project")
			agent := filepath.Join(home, "agent")
			dir := filepath.Join(home, "extensions")
			switch kind {
			case "explicit":
				dp02Write(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"extensions":[%q]}`, dir))
			case "native":
				dir = filepath.Join(agent, "extensions")
			default:
				pkg := filepath.Join(home, "package")
				dir = filepath.Join(pkg, "extensions")
				dp02Write(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"packages":[%q]}`, pkg))
				manifest := `{"name":"fixture-package","pi":{"extensions":["extensions"]}}`
				if kind == "conventional package" {
					manifest = `{"name":"fixture-package"}`
				}
				dp02Write(t, filepath.Join(pkg, "package.json"), manifest)
			}
			entry := filepath.Join(dir, "entry.js")
			dp02Write(t, entry, "throw new Error('never execute');")
			var warnings []string
			if _, err := PiResourceRoots(home, project, agent, false, dp02Env(nil), func(message string) { warnings = append(warnings, message) }); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(warnings, "\n"), entry) {
				t.Fatalf("entrypoint boundary omitted: %v", warnings)
			}
			dp02Write(t, filepath.Join(dir, "package.json"), `{"pi":{"extensions":["."]}}`)
			if _, err := PiResourceRoots(home, project, agent, false, dp02Env(nil)); err == nil || !strings.Contains(err.Error(), "cyclic or overly nested") {
				t.Fatalf("cyclic static extension declaration admitted: %v", err)
			}
		})
	}
}

func TestPiMCPNoExecution(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "project")
	agent := filepath.Join(home, "agent")
	config := filepath.Join(agent, "mcp-adapter.json")
	marker := filepath.Join(home, "executed")
	helper := filepath.Join(home, "fixture-helper")
	dp02Write(t, helper, fmt.Sprintf("#!/bin/sh\nprintf executed > %q\n", marker))
	if err := os.Chmod(helper, 0700); err != nil {
		t.Fatal(err)
	}
	dp02Write(t, config, `{"imports":["cursor"]}`)
	dp02Write(t, filepath.Join(home, ".cursor/mcp.json"), fmt.Sprintf(`{"mcpServers":{"fixture-import":{"command":%q,"env":{"TOKEN":"invented-secret"}}}}`, helper))
	pkg := filepath.Join(home, "package")
	dp02Write(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"packages":[%q]}`, pkg))
	dp02Write(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-package","pi":{"mcp":"defaults.json"}}`)
	dp02Write(t, filepath.Join(pkg, "defaults.json"), fmt.Sprintf(`{"mcpServers":{"fixture-default":{"command":%q}}}`, helper))
	sources, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile)
	if err != nil || len(sources) != 3 {
		t.Fatalf("static source inspection: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("MCP command/helper executed: %v", err)
	}
}

func TestPiMCPStructuredImports(t *testing.T) {
	for _, tc := range []struct{ name, path, body string }{
		{"cursor", ".cursor/mcp.json", `{/* fixture comment */ "mcpServers":{"fixture-host":{"command":"never-run-fixture","env":{"TOKEN":"invented-secret"}}},}`},
		{"claude-code", ".claude/mcp.json", `{"mcpServers":{"fixture-host":{"url":"https://example.invalid/mcp"}}}`},
		{"claude-desktop", "Library/Application Support/Claude/claude_desktop_config.json", `{"mcpServers":{"fixture-host":{}}}`},
		{"windsurf", ".windsurf/mcp.json", `{"mcp-servers":{"fixture-host":{}}}`},
		{"vscode", "project/.vscode/mcp.json", `{"mcpServers":{"fixture-host":{}}}`},
		{"codex", ".codex/config.toml", "[mcp_servers.fixture-host]\ncommand = 'never-run-fixture'\n"},
		{"opencode", ".config/opencode/opencode.json", `{"mcp":{"servers":{"fixture-host":{"type":"local","command":["never-run-fixture"],"environment":{"TOKEN":"invented-secret"}}}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			project := filepath.Join(home, "project")
			agent := filepath.Join(home, "agent")
			config := filepath.Join(agent, "mcp-adapter.json")
			source := filepath.Join(home, tc.path)
			dp02Write(t, source, tc.body)
			dp02Write(t, config, fmt.Sprintf(`{"imports":[%q],"mcpServers":{"fixture-native":{}}}`, tc.name))
			sources, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile)
			if err != nil {
				t.Fatal(err)
			}
			if len(sources) != 2 || sources[0].Path != source || sources[0].Servers["fixture-host"] == nil || sources[1].Path != config {
				t.Fatalf("import inventory: %+v", sources)
			}
			dp02Write(t, config, fmt.Sprintf(`{"imports":[%q],"mcpServers":{"fixture_host":{}}}`, tc.name))
			if _, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile); err == nil || !strings.Contains(err.Error(), source) || !strings.Contains(err.Error(), config) || strings.Contains(err.Error(), "invented-secret") {
				t.Fatalf("import collision diagnostic: %v", err)
			}
		})
	}
}

func TestPiMCPStaticSourceSafety(t *testing.T) {
	cases := []struct{ name, file, body, action, want string }{
		{"missing import", ".cursor/mcp.json", "", "remove", "no readable static candidate"},
		{"malformed import", ".cursor/mcp.json", `{"mcpServers":{"x":{"token":"invented-secret"}}`, "", ".cursor/mcp.json"},
		{"unreadable import", ".cursor/mcp.json", "", "chmod", "unreadable"},
		{"symlink import", ".cursor/mcp.json", "", "symlink", "symlink"},
		{"nonregular import", ".cursor/mcp.json", "", "directory", "regular"},
		{"import object", "agent/mcp-adapter.json", `{"imports":{}}`, "", "imports must be a list"},
		{"import null", "agent/mcp-adapter.json", `{"imports":null}`, "", "imports must be a list"},
		{"import unknown", "agent/mcp-adapter.json", `{"imports":["fixture-unknown"]}`, "", "unsupported static import"},
		{"import servers list", ".cursor/mcp.json", `{"mcpServers":[]}`, "", "must be an object"},
		{"import invalid entry", ".cursor/mcp.json", `{"mcpServers":{"fixture":42}}`, "", "must be an object"},
		{"import duplicate member", ".cursor/mcp.json", `{"mcpServers":{"fixture":{},"fixture":{}}}`, "", "duplicate JSON"},
		{"import normalized duplicate", ".cursor/mcp.json", `{"mcpServers":{"fixture-name":{},"fixture_name":{}}}`, "", "duplicate normalized"},
		{"package missing", "pkg/defaults.json", "", "remove", "missing pi.mcp"},
		{"package unreadable", "pkg/defaults.json", "", "chmod", "unreadable"},
		{"package symlink", "pkg/defaults.json", "", "symlink", "symlink"},
		{"package nonregular", "pkg/defaults.json", "", "directory", "regular"},
		{"package malformed", "pkg/defaults.json", `{"secret":"invented-secret"`, "", "pkg/defaults.json"},
		{"package missing servers", "pkg/defaults.json", `{}`, "", "missing mcpServers"},
		{"package invalid entry", "pkg/defaults.json", `{"mcpServers":{"fixture":[]}}`, "", "must be an object"},
		{"package escaped path", "pkg/package.json", `{"name":"fixture-kit","pi":{"mcp":"../outside.json"}}`, "", "escapes"},
		{"package invalid declaration", "pkg/package.json", `{"name":"fixture-kit","pi":{"mcp":true}}`, "", "pi.mcp must be"},
		{"package normalized duplicate", "pkg/defaults.json", `{"mcpServers":{"fixture.name":{},"fixture/name":{}}}`, "", "duplicate normalized"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			project := filepath.Join(home, "project")
			agent := filepath.Join(home, "agent")
			config := filepath.Join(agent, "mcp-adapter.json")
			dp02Write(t, config, `{"imports":["cursor"]}`)
			dp02Write(t, filepath.Join(home, ".cursor/mcp.json"), `{"mcpServers":{"fixture-host":{}}}`)
			dp02Write(t, filepath.Join(agent, "settings.json"), `{"packages":["../pkg"]}`)
			dp02Write(t, filepath.Join(home, "pkg/package.json"), `{"name":"fixture-kit","pi":{"mcp":"defaults.json"}}`)
			dp02Write(t, filepath.Join(home, "pkg/defaults.json"), `{"mcpServers":{"fixture-package":{}}}`)
			outside := filepath.Join(home, "outside.json")
			dp02Write(t, outside, `{"mcpServers":{}}`)
			path := filepath.Join(home, tc.file)
			switch tc.action {
			case "remove":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "chmod":
				if err := os.Chmod(path, 0); err != nil {
					t.Fatal(err)
				}
			case "symlink", "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if tc.action == "symlink" {
					if err := os.Symlink(outside, path); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			default:
				dp02Write(t, path, tc.body)
			}
			_, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile)
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "invented-secret") {
				t.Fatalf("unsafe static source diagnostic: %v (want %q)", err, tc.want)
			}
		})
	}
}

func TestPiMCPImportActivation(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "work/project")
	agent := filepath.Join(home, "agent")
	config := filepath.Join(agent, "mcp-adapter.json")
	dp02Write(t, config, `{"imports":["codex"],"settings":{"agentPluginPaths":["dormant"],"hostConfigDiscovery":"on"}}`)
	tomlPath := filepath.Join(home, ".codex/config.toml")
	dp02Write(t, tomlPath, "[mcp_servers.fixture]\ncommand = 'never-run-fixture'\n")
	for _, path := range []string{filepath.Join(home, ".codex/config.json"), filepath.Join(home, ".agents/mcp.json"), filepath.Join(project, ".pi/mcp-adapter.json"), filepath.Join(agent, "settings.json")} {
		dp02Write(t, path, "malformed dormant source")
	}
	sources, err := DiscoverPiMCP(home, project, agent, config, dp02Env(map[string]string{"PI_MCP_CONFIG_MODE": "exclusive"}), ReadPiFile)
	if err != nil || len(sources) != 2 || sources[0].Path != tomlPath {
		t.Fatalf("exclusive/import fallback activation: %+v %v", sources, err)
	}
	dp02Write(t, tomlPath, "[mcp_servers.fixture]\ncommand = invented-secret\n")
	if _, err := DiscoverPiMCP(home, project, agent, config, dp02Env(map[string]string{"PI_MCP_CONFIG_MODE": "exclusive"}), ReadPiFile); err == nil || !strings.Contains(err.Error(), "malformed TOML") || strings.Contains(err.Error(), "invented-secret") {
		t.Fatalf("earlier malformed import must not use fallback/leak values: %v", err)
	}
}

func TestPiMCPOpenCodeAncestor(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "work")
	project := filepath.Join(root, "sub/project")
	agent := filepath.Join(home, "agent")
	config := filepath.Join(agent, "mcp-adapter.json")
	dp02Write(t, filepath.Join(root, ".git"), "fixture git boundary")
	dp02Write(t, config, `{"imports":["opencode"]}`)
	global := filepath.Join(home, ".config/opencode/opencode.json")
	local := filepath.Join(root, "sub/opencode.json")
	dp02Write(t, global, `{"mcp":{"global-fixture":{"type":"remote","url":"https://example.invalid/mcp"}}}`)
	dp02Write(t, local, `{"mcp":{"servers":{"project-fixture":{"type":"local","command":["never-run-fixture"]},"disabled-fixture":{"enabled":false}},"timeout":100}}`)
	dp02Write(t, filepath.Join(root, "opencode.json"), "dormant more distant file")
	sources, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile)
	if err != nil || len(sources) != 3 || sources[0].Path != global || sources[1].Path != local || len(sources[1].Servers) != 1 {
		t.Fatalf("OpenCode nearest/merged inventory: %+v %v", sources, err)
	}
	dp02Write(t, local, `{"mcp":{"global_fixture":{"type":"remote","url":"https://example.invalid/mcp"}}}`)
	if _, err := DiscoverPiMCP(home, project, agent, config, dp02Env(nil), ReadPiFile); err == nil || !strings.Contains(err.Error(), global) || !strings.Contains(err.Error(), local) {
		t.Fatalf("OpenCode known override collision: %v", err)
	}
}

func TestPiMCPJSONC(t *testing.T) {
	for _, raw := range []string{`{"url":"https://example.invalid/*literal*/",/* comment */"list":["comma,]",],}`, "{//comment\n\"mcpServers\":{},}", `{"escaped":"quote\"//literal","x":1}`} {
		if _, err := piMCPJSONObject([]byte(raw)); err != nil {
			t.Fatalf("supported JSONC refused: %v", err)
		}
	}
	for _, raw := range []string{`{"x":1,"x":2}`, `{"x":/* unterminated}`, `{"x":1,,}`, `[]`, `{"x":1} {}`} {
		if _, err := piMCPJSONObject([]byte(raw)); err == nil {
			t.Fatalf("malformed JSONC admitted: %s", raw)
		}
	}
}

func TestPiMCPPackageDefaults(t *testing.T) {
	root := t.TempDir()
	agent := filepath.Join(root, "agent")
	project := filepath.Join(root, "project")
	pkg := filepath.Join(root, "pkg")
	dp02Write(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"packages":[%q]}`, pkg))
	dp02Write(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-package","pi":{"mcp":["defaults.json"]}}`)
	defaults := filepath.Join(pkg, "defaults.json")
	config := filepath.Join(agent, "mcp-adapter.json")
	if _, err := DiscoverPiMCP(root, project, agent, config, dp02Env(nil), ReadPiFile); err == nil || !strings.Contains(err.Error(), "pi.mcp") {
		t.Fatalf("missing static defaults admitted: %v", err)
	}
	dp02Write(t, defaults, `{"mcpServers":{"fixture-server":{"command":"never-run-fixture"}}}`)
	sources, err := DiscoverPiMCP(root, project, agent, config, dp02Env(nil), ReadPiFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Path != defaults || sources[0].Servers["fixture-package__fixture-server"] == nil {
		t.Fatalf("package inventory: %+v", sources)
	}
	dp02Write(t, config, `{"mcpServers":{"fixture_package__fixture_server":{"command":"never-run-fixture"}}}`)
	if _, err := DiscoverPiMCP(root, project, agent, config, dp02Env(nil), ReadPiFile); err == nil || !strings.Contains(err.Error(), "duplicate normalized") {
		t.Fatalf("package collision omitted: %v", err)
	}
}

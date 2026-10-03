package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Generic inverse coverage with invented content, not a shipped overlay test.
// File sentinels and the fake runner cannot prove live service survival.
func TestPiRemovalPreservesUnownedResources(t *testing.T) {
	for _, scope := range []string{"global", "local"} {
		t.Run(scope, func(t *testing.T) {
			f := dp02Setup(t) // isolates PI_CODING_AGENT_DIR before any CLI call
			dest := os.Getenv("PI_CODING_AGENT_DIR")
			if scope == "local" {
				dest = filepath.Join(f.root, ".pi")
			}
			dp02Artifact(t, f.root, "atlas-notes", "skill", "---\nname: atlas-notes\ndescription: Invented notes\n---\nFixture notes.\n")
			dp06Write(t, filepath.Join(f.root, "recipes/atlas-relay.yaml"), "apiVersion: patronus/v2\nfamily: recipe\nname: atlas-relay\nversion: 1.0.0\nrole: tools\ndescription: Invented wire\nwire:\n  method: merge\n  actor: patronus\n  tools: [pi]\n  mcp:\n    transport: http\n    url: https://atlas.invalid/mcp\n")
			settingsPath := filepath.Join(dest, "settings.json")
			settings := "{\n  \"subagents\": {\"agentOverrides\": {\"fixture-observer\": {\"tools\": [\"read\"], \"skills\": []}}},\n  \"userTheme\": \"amber\"\n}\n"
			dp06Write(t, settingsPath, settings)
			mcpPath := filepath.Join(dest, "mcp-adapter.json")
			dp06Write(t, mcpPath, `{"mcpServers":{"neighbor-relay":{"url":"https://neighbor.invalid/mcp"}},"allowInstall":false}`)
			priorMCP := dp07JSON(t, mcpPath)
			sentinels := map[string]string{
				filepath.Join(dest, "cache", "fixture-snapshot.bin"):            "unowned cache\x00\xff",
				filepath.Join(dest, "spill", "fixture-output.txt"):              "unowned spill",
				filepath.Join(f.home, "external-service", "binary.fixture"):     "inert binary sentinel, never executed",
				filepath.Join(f.home, "external-service", "auth.fixture"):       "invented credential sentinel, not a secret",
				filepath.Join(f.root, "outputs", "fixture-result.md"):           "unowned output",
				filepath.Join(f.root, "worktrees", "fixture", "unfinished.txt"): "directory sentinel, not an allocated git worktree",
			}
			for path, content := range sentinels {
				dp06Write(t, path, content)
			}
			for _, name := range []string{"atlas-notes", "atlas-relay"} {
				if _, _, err := runInstall(t, name, "--target", "pi", "--"+scope, "--deploy"); err != nil {
					t.Fatal(err)
				}
			}
			ownedSkill := filepath.Join(dest, "skills", "atlas-notes", "SKILL.md")
			if len(mustRead(t, ownedSkill)) == 0 {
				t.Fatal("setup did not install the owned skill")
			}
			servers, ok := dp07JSON(t, mcpPath)["mcpServers"].(map[string]any)
			if !ok || !reflect.DeepEqual(servers["atlas-relay"], map[string]any{"url": "https://atlas.invalid/mcp"}) {
				t.Fatalf("setup did not install the owned wire: %v", servers)
			}
			for _, name := range []string{"atlas-notes", "atlas-relay"} {
				if _, _, err := execRemove(t, name, "--target", "pi", "--"+scope, "--deploy"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(ownedSkill); !os.IsNotExist(err) {
				t.Fatalf("owned skill survived removal: %v", err)
			}
			if !reflect.DeepEqual(dp07JSON(t, mcpPath), priorMCP) {
				t.Fatal("inverse retained owned wire or changed unrelated MCP policy/entry")
			}
			if got := string(mustRead(t, settingsPath)); got != settings {
				t.Fatal("item removal changed manual override/settings bytes")
			}
			for path, want := range sentinels {
				if got := string(mustRead(t, path)); got != want {
					t.Fatalf("unowned bytes changed: %s", path)
				}
			}
			runner, ok := runnerForCommands.(*fakeRunner)
			if !ok || len(runner.ran) != 0 || f.fetcher.calls != 0 {
				t.Fatal("static install/remove executed a command or acquired a payload")
			}
		})
	}
}

package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// TestCodexMCPWiringOwnsTransportLeaves pins the generation policy: Codex owns
// each transport-supplied leaf with its own prior, user children stay unowned,
// and the other harnesses keep their whole-map edit.
func TestCodexMCPWiringOwnsTransportLeaves(t *testing.T) {
	res, home, _ := testEnv(t)
	config := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o755); err != nil {
		t.Fatal(err)
	}
	prior := []byte("model = 'user-model'\n[mcp_servers.demo-memory]\ncommand = 'user-old-command'\n[mcp_servers.demo-memory.env]\nINVENTED_KEY_REF = 'INVENTED_ENV'\n")
	if err := os.WriteFile(config, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	diffs, err := Compute(Request{Recipe: engramRecipe(), Adapters: loadAdapters(t), Resolver: res, Tool: "all", Scope: "global", GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string][]diff.FileDiff{}
	for _, d := range diffs {
		if d.Action == diff.Merge {
			byTool[d.Tool] = append(byTool[d.Tool], d)
		}
	}
	for _, tool := range []string{"claude", "opencode"} {
		if len(byTool[tool]) != 1 || byTool[tool][0].Setting.Dotted == "" {
			t.Fatalf("%s whole-map wiring changed: %+v", tool, byTool[tool])
		}
	}
	codex := byTool["codex"]
	if len(codex) != 2 || codex[0].Setting.Dotted != "mcp_servers.demo-memory.command" || codex[1].Setting.Dotted != "mcp_servers.demo-memory.args" {
		t.Fatalf("codex leaves = %+v", codex)
	}
	if !codex[0].Setting.PriorPresent || codex[0].Setting.PriorValue != "user-old-command" || codex[1].Setting.PriorPresent {
		t.Fatalf("per-leaf priors wrong: %+v %+v", codex[0].Setting, codex[1].Setting)
	}
	target := manifest.FileTarget{File: "~/.codex/config.toml", Format: "toml"}
	for _, d := range codex {
		got, present, err := adapter.ReadDotted(d.After, target, "mcp_servers.demo-memory.env.INVENTED_KEY_REF")
		if err != nil || !present || got != "INVENTED_ENV" {
			t.Fatalf("user env child lost from %s: %s", d.Setting.Dotted, d.After)
		}
	}
}

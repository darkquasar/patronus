package recipe

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// Default wiring is a recipe mechanism, not a property of any shipped server.
// Empty/all here are Compute inputs; the CLI's required-target gate is separate.
func TestComputePiWireDefaultRouting(t *testing.T) {
	for _, scope := range []string{"global", "local"} {
		for _, tc := range []struct{ name, target, wantTool string }{
			{"empty-default", "", "pi"},
			{"all-default", "all", "pi"},
			{"explicit-pi", "pi", "pi"},
			{"explicit-other-overrides-default", "claude", "claude"},
		} {
			t.Run(scope+"/"+tc.name, func(t *testing.T) {
				resolver, home, project := testEnv(t)
				adapters := loadAdapters(t)
				pi, err := manifest.LoadAdapter(filepath.Join(repoRoot(t), "adapters/pi.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				adapters["pi"] = pi
				rec := &manifest.Recipe{
					Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "invented-relay", Role: manifest.RoleTools},
					Wire: manifest.Wire{
						Method: manifest.WireMerge, Actor: manifest.ActorPatronus,
						Tools: []string{"pi"},
						Mcp:   &manifest.WireMcp{Transport: "http", URL: "https://relay.invalid/mcp"},
					},
				}
				rows, err := Compute(Request{Recipe: rec, Adapters: adapters, Resolver: resolver, Tool: tc.target, Scope: scope})
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != 1 || rows[0].Action != diff.Merge {
					t.Fatalf("want exactly one MERGE, no delivery/EXEC: %+v", rows)
				}
				row := rows[0]
				paths := map[string]map[string]string{
					"global": {"pi": filepath.Join(home, ".pi/agent/mcp-adapter.json"), "claude": filepath.Join(home, ".claude.json")},
					"local":  {"pi": filepath.Join(project, ".pi/mcp-adapter.json"), "claude": filepath.Join(project, ".mcp.json")},
				}
				if row.Tool != tc.wantTool || row.Scope != scope || row.Path != paths[scope][tc.wantTool] {
					t.Fatalf("wrong route for target %q: %+v", tc.target, row)
				}
				var config struct {
					Servers map[string]map[string]any `json:"mcpServers"`
				}
				if err := json.Unmarshal(row.After, &config); err != nil {
					t.Fatal(err)
				}
				if len(config.Servers) != 1 || config.Servers["invented-relay"]["url"] != "https://relay.invalid/mcp" {
					t.Fatalf("wrong HTTP leaf: %s", row.After)
				}
			})
		}
	}
}

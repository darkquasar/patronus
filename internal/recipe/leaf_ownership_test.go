package recipe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// toyJSONAdapter is an invented jsonc tool whose mcp layout matches the
// OpenCode shape (type/url leaves under mcp.<name>); ownership is a parameter.
func toyJSONAdapter(t *testing.T, ownership string) map[string]*manifest.Adapter {
	t.Helper()
	ownershipLine := ""
	if ownership != "" {
		ownershipLine = "    ownership: " + ownership + "\n"
	}
	ad, err := manifest.DecodeAdapter([]byte(`apiVersion: patronus/v2
family: adapter
tool: toy
layout:
  mcp:
    global: { file: "~/.toy/toy.json", format: jsonc, path: "mcp.{name}" }
    project: { file: "toy.json", format: jsonc, path: "mcp.{name}" }
` + ownershipLine + `    transports:
      http:
        keys: { type: "remote", url: "{url}", headers: "{headers}" }
`))
	if err != nil {
		t.Fatal(err)
	}
	return map[string]*manifest.Adapter{"toy": ad}
}

func fixtureHTTPRecipe() *manifest.Recipe {
	return &manifest.Recipe{
		Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "fixture", Role: manifest.RoleTools},
		Wire: manifest.Wire{
			Method: manifest.WireMerge,
			Actor:  manifest.ActorPatronus,
			Mcp:    &manifest.WireMcp{Transport: "http", URL: "https://invented.example/mcp"},
			Tools:  []string{"toy"},
		},
	}
}

func mergeDiffs(t *testing.T, diffs []diff.FileDiff) []diff.FileDiff {
	t.Helper()
	var out []diff.FileDiff
	for _, d := range diffs {
		if d.Action == diff.Merge {
			out = append(out, d)
		}
	}
	return out
}

// TestLeafOwnershipSelectedByLayoutFlag: an http recipe on an invented jsonc
// adapter with ownership: leaf yields one MERGE per transport leaf, each
// flagged PruneEmptyParent; without the flag the whole server object is one
// scalar edit with no prune flag (today's claude behaviour).
func TestLeafOwnershipSelectedByLayoutFlag(t *testing.T) {
	res, _, _ := testEnv(t)
	for _, tc := range []struct {
		name      string
		ownership string
		dotted    []string
		prune     bool
	}{
		{"leaf", manifest.OwnershipLeaf, []string{"mcp.fixture.type", "mcp.fixture.url"}, true},
		{"whole map", "", []string{"mcp.fixture"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diffs, err := Compute(Request{Recipe: fixtureHTTPRecipe(), Adapters: toyJSONAdapter(t, tc.ownership), Resolver: res, Tool: "toy", Scope: "local"})
			if err != nil {
				t.Fatal(err)
			}
			merges := mergeDiffs(t, diffs)
			var got []string
			for _, d := range merges {
				got = append(got, d.Setting.Dotted)
				if d.Setting.PruneEmptyParent != tc.prune {
					t.Fatalf("%s PruneEmptyParent = %v, want %v", d.Setting.Dotted, d.Setting.PruneEmptyParent, tc.prune)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.dotted, ",") {
				t.Fatalf("leaves = %v, want %v", got, tc.dotted)
			}
		})
	}
}

// applyAll folds every MERGE edit onto current in order (what plan.Finalize does
// when it composes same-path diffs).
func applyAll(t *testing.T, current []byte, merges []diff.FileDiff) []byte {
	t.Helper()
	for _, d := range merges {
		var err error
		current, err = adapter.ApplySettingEdit(current, d.Setting)
		if err != nil {
			t.Fatal(err)
		}
	}
	return current
}

func removeAll(t *testing.T, current []byte, merges []diff.FileDiff) []byte {
	t.Helper()
	for i := len(merges) - 1; i >= 0; i-- {
		var err error
		current, _, err = adapter.RemoveSettingEdit(current, merges[i].Setting)
		if err != nil {
			t.Fatal(err)
		}
	}
	return current
}

func serverOf(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	mcp, _ := root["mcp"].(map[string]any)
	server, _ := mcp["fixture"].(map[string]any)
	return server
}

// TestOpenCodeLeafDiffsPreserveUserChildren: with a user-authored server that
// already carries headers and oauth, wiring writes the type and url leaves only;
// headers and oauth survive install, reinstall and removal, and the server
// object remains after removal because user children keep it alive.
func TestOpenCodeLeafDiffsPreserveUserChildren(t *testing.T) {
	res, _, proj := testEnv(t)
	existing := []byte(`{"mcp":{"fixture":{"headers":{"X-Invented":"fixture-not-a-secret"},"oauth":{"clientId":"invented"}}},"user":1}` + "\n")
	path := filepath.Join(proj, "toy.json")
	if err := os.WriteFile(path, existing, 0o644); err != nil {
		t.Fatal(err)
	}
	compute := func() []diff.FileDiff {
		diffs, err := Compute(Request{Recipe: fixtureHTTPRecipe(), Adapters: toyJSONAdapter(t, manifest.OwnershipLeaf), Resolver: res, Tool: "toy", Scope: "local"})
		if err != nil {
			t.Fatal(err)
		}
		return mergeDiffs(t, diffs)
	}
	merges := compute()
	installed := applyAll(t, existing, merges)
	if err := os.WriteFile(path, installed, 0o644); err != nil {
		t.Fatal(err)
	}
	check := func(stage string, raw []byte, wantURL bool) {
		t.Helper()
		server := serverOf(t, raw)
		headers, _ := server["headers"].(map[string]any)
		oauth, _ := server["oauth"].(map[string]any)
		if headers["X-Invented"] != "fixture-not-a-secret" || oauth["clientId"] != "invented" {
			t.Fatalf("%s: user children changed: %s", stage, raw)
		}
		_, hasURL := server["url"]
		if hasURL != wantURL || (wantURL && (server["type"] != "remote" || server["url"] != "https://invented.example/mcp")) {
			t.Fatalf("%s: owned leaves wrong: %s", stage, raw)
		}
	}
	check("install", installed, true)
	reinstalled := applyAll(t, installed, compute())
	check("reinstall", reinstalled, true)
	if string(reinstalled) != string(installed) {
		t.Fatalf("reinstall changed bytes:\n%s\n%s", installed, reinstalled)
	}
	removed := removeAll(t, reinstalled, merges)
	check("remove", removed, false)
}

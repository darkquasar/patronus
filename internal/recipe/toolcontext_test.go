package recipe

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// These cover the {toolContext} wiring token added for per-tool stdio MCP flavours
// (e.g. Serena's `--context claude-code` vs `codex` vs `ide`). Two levels:
//   - the pure helpers (toolContext / substTokens) in isolation, and
//   - the real end-to-end path through Compute, asserting each tool's MERGEd MCP
//     config carries ITS OWN resolved context (proving the flavour diverges).

func TestToolContextMapping(t *testing.T) {
	for _, tc := range []struct {
		tool, want string
	}{
		{"claude", "claude-code"}, // differs from the bare tool id
		{"codex", "codex"},        // happens to equal the tool id
		{"opencode", "ide"},       // differs from the bare tool id
		{"unknown", "unknown"},    // unmapped tool falls back to its own id
		{"", ""},                  // empty falls back to itself (no panic)
	} {
		if got := toolContext(tc.tool); got != tc.want {
			t.Errorf("toolContext(%q) = %q, want %q", tc.tool, got, tc.want)
		}
	}
}

func TestSubstTokens(t *testing.T) {
	for _, tc := range []struct {
		name, in, installPath, tool, want string
	}{
		{"installPath only", "{installPath}", "/opt/bin/x", "claude", "/opt/bin/x"},
		{"toolContext only", "--context={toolContext}", "", "claude", "--context=claude-code"},
		{"both tokens", "{installPath} --context {toolContext}", "/b/x", "opencode", "/b/x --context ide"},
		{"no tokens passthrough", "start-mcp-server", "/b/x", "codex", "start-mcp-server"},
		{"repeated token", "{toolContext}-{toolContext}", "", "codex", "codex-codex"},
		{"unmapped tool", "--context {toolContext}", "", "zed", "--context zed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := substTokens(tc.in, tc.installPath, tc.tool); got != tc.want {
				t.Errorf("substTokens(%q, %q, %q) = %q, want %q", tc.in, tc.installPath, tc.tool, got, tc.want)
			}
		})
	}
}

// serenaLikeRecipe is a uvx-launched stdio MCP whose args carry {toolContext} —
// the canonical per-tool-flavoured recipe (no delivery; uvx fetches on demand).
func serenaLikeRecipe() *manifest.Recipe {
	return &manifest.Recipe{
		Meta: manifest.Meta{
			APIVersion: manifest.APIVersion,
			Family:     manifest.FamilyRecipe,
			Name:       "serena",
			Role:       manifest.RoleContext,
		},
		Wire: manifest.Wire{
			Mode: manifest.WireModeMcp,
			Mcp: &manifest.WireMcp{
				Transport: "stdio",
				Command:   "uvx",
				Args: []string{
					"--from", "git+https://github.com/oraios/serena",
					"serena", "start-mcp-server",
					"--context", "{toolContext}", "--project-from-cwd",
				},
			},
			Tools: []string{"claude", "codex", "opencode"},
		},
	}
}

// TestServerSpecStdioSubstitutesToolContext is the unit-level proof that the
// stdio ServerSpec resolves {toolContext} in BOTH the command and the args (and
// in the commandArray that OpenCode consumes).
func TestServerSpecStdioSubstitutesToolContext(t *testing.T) {
	wm := &manifest.WireMcp{
		Transport: "stdio",
		Command:   "{installPath}",
		Args:      []string{"--context", "{toolContext}", "--project-from-cwd"},
	}
	spec := serverSpec("serena", wm, "/opt/serena", "claude")

	if got := spec.Values["command"]; got != "/opt/serena" {
		t.Errorf("command = %v, want /opt/serena", got)
	}
	args, ok := spec.Values["args"].([]any)
	if !ok {
		t.Fatalf("args not []any: %T", spec.Values["args"])
	}
	if len(args) != 3 || args[1] != "claude-code" {
		t.Errorf("args = %v, want --context claude-code at [1]", args)
	}
	// commandArray (OpenCode) must carry the same resolved context.
	arr, ok := spec.Values["commandArray"].([]any)
	if !ok {
		t.Fatalf("commandArray not []any: %T", spec.Values["commandArray"])
	}
	joined := strings.TrimSpace(strings.Join(toStrs(arr), " "))
	if !strings.Contains(joined, "--context claude-code") {
		t.Errorf("commandArray missing resolved context: %q", joined)
	}
}

// TestComputeFlavoursToolContextPerTool drives the full Compute path against the
// real adapters and asserts each tool's MERGEd config carries ITS context —
// claude-code for claude, codex for codex, ide for opencode — proving one recipe
// flavours per tool with no per-tool authoring.
func TestComputeFlavoursToolContextPerTool(t *testing.T) {
	res, _, _ := testEnv(t)
	diffs, err := Compute(Request{
		Recipe:   serenaLikeRecipe(),
		Adapters: loadAdapters(t),
		Resolver: res,
		Tool:     "all",
		Scope:    "global",
	})
	if err != nil {
		t.Fatal(err)
	}

	merges := map[string]diff.FileDiff{}
	for _, d := range diffs {
		if d.Action == diff.Merge {
			merges[d.Tool] = d
		}
		if d.Action == diff.Fetch {
			t.Error("uvx-launched MCP should produce no FETCH (wire-only)")
		}
	}
	if len(merges) != 3 {
		t.Fatalf("expected 3 MERGE diffs, got %d", len(merges))
	}

	for tool, wantCtx := range map[string]string{
		"claude":   "claude-code",
		"codex":    "codex",
		"opencode": "ide",
	} {
		body := string(merges[tool].After)
		if !strings.Contains(body, wantCtx) {
			t.Errorf("%s config missing context %q:\n%s", tool, wantCtx, body)
		}
		// Negative: a tool must NOT carry another tool's distinct context. (codex
		// is a substring-safe check only against the two that differ from it.)
		for otherTool, otherCtx := range map[string]string{"claude": "claude-code", "opencode": "ide"} {
			if otherTool != tool && wantCtx != otherCtx && strings.Contains(body, otherCtx) {
				t.Errorf("%s config leaked %s's context %q:\n%s", tool, otherTool, otherCtx, body)
			}
		}
	}
}

// toStrs flattens a []any of strings for substring assertions.
func toStrs(in []any) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// TestComputeUnmappedToolFallsBackToOwnID guards the fallback branch end-to-end:
// a tool with no toolContexts entry wires with its own id as the context, never a
// blank. (Uses claude's adapter under a synthetic tool key so the MERGE resolves.)
func TestComputeUnmappedToolFallsBackToOwnID(t *testing.T) {
	res, _, _ := testEnv(t)
	ads := loadAdapters(t)
	// Alias the claude adapter under a tool id absent from toolContexts.
	ads["zed"] = ads["claude"]

	rec := serenaLikeRecipe()
	rec.Wire.Tools = []string{"zed"}
	diffs, err := Compute(Request{
		Recipe: rec, Adapters: ads, Resolver: res, Tool: "zed", Scope: "global",
	})
	if err != nil {
		t.Fatal(err)
	}
	var merged string
	for _, d := range diffs {
		if d.Action == diff.Merge {
			merged = string(d.After)
		}
	}
	if !strings.Contains(merged, "--context") || !strings.Contains(merged, "zed") {
		t.Errorf("unmapped tool should wire its own id as context:\n%s", merged)
	}
	if strings.Contains(merged, "{toolContext}") {
		t.Errorf("unresolved token leaked into config:\n%s", merged)
	}
}

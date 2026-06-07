package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// testEnv builds a Resolver with a fixed HOME and project dir.
func testEnv(t *testing.T) (toolpath.Resolver, string, string) {
	t.Helper()
	home := t.TempDir()
	proj := t.TempDir()
	env := func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}
	return toolpath.New(env, home, proj), home, proj
}

// loadAdapters loads the real repo adapters so transport templates are exercised.
func loadAdapters(t *testing.T) map[string]*manifest.Adapter {
	t.Helper()
	root := repoRoot(t)
	out := map[string]*manifest.Adapter{}
	for _, tool := range []string{"claude", "codex", "opencode"} {
		ad, err := manifest.LoadAdapter(filepath.Join(root, "adapters", tool+".yaml"))
		if err != nil {
			t.Fatalf("load adapter %s: %v", tool, err)
		}
		out[tool] = ad
	}
	return out
}

// repoRoot walks up to the repo root (the dir holding adapters/).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "adapters")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

// engramRecipe is a stdio github-release recipe (the floor), with a pinned asset.
func engramRecipe() *manifest.Recipe {
	return &manifest.Recipe{
		APIVersion: manifest.APIVersion,
		Kind:       manifest.KindRecipe,
		Name:       "memory-engram",
		Capability: "memory",
		Delivery: &manifest.Delivery{
			Primary:   "github-release",
			InstallTo: "~/.patronus/bin/",
			Binary:    "engram",
			Assets: []manifest.Asset{
				{OS: "linux", Arch: "amd64", URL: "https://x/engram-linux", SHA256: "abc"},
				{OS: "darwin", Arch: "arm64", URL: "https://x/engram-darwin", SHA256: "def"},
			},
		},
		Wire: manifest.Wire{
			Mcp:   &manifest.WireMcp{Transport: "stdio", Command: "{installPath}", Args: []string{"serve"}},
			Tools: []string{"claude", "codex", "opencode"},
		},
	}
}

func TestComputeFetchAndWire_Engram(t *testing.T) {
	res, home, _ := testEnv(t)
	diffs, err := Compute(Request{
		Recipe:   engramRecipe(),
		Adapters: loadAdapters(t),
		Resolver: res,
		Tool:     "all",
		Scope:    "global",
		GOOS:     "linux",
		GOARCH:   "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}

	var fetch *diff.FileDiff
	merges := map[string]diff.FileDiff{}
	for i := range diffs {
		d := diffs[i]
		switch d.Action {
		case diff.Fetch:
			fetch = &diffs[i]
		case diff.Merge:
			merges[d.Tool] = d
		}
	}

	// FETCH points at the linux asset and the ~/.patronus/bin dest.
	if fetch == nil {
		t.Fatal("expected a FETCH diff")
	}
	wantDest := filepath.Join(home, ".patronus", "bin", "engram")
	if fetch.Fetch.Dest != wantDest {
		t.Errorf("fetch dest = %q, want %q", fetch.Fetch.Dest, wantDest)
	}
	if fetch.Fetch.URL != "https://x/engram-linux" {
		t.Errorf("fetch url = %q", fetch.Fetch.URL)
	}

	// One MERGE per tool.
	if len(merges) != 3 {
		t.Fatalf("expected 3 MERGE diffs, got %d", len(merges))
	}

	// Claude global wiring must target ~/.claude.json (the user target), NOT a
	// global block (which Claude's adapter does not have).
	claude := merges["claude"]
	if !strings.HasSuffix(claude.Path, ".claude.json") {
		t.Errorf("claude global mcp path = %q, want ~/.claude.json", claude.Path)
	}
	// The merged config substitutes {installPath} with the resolved dest.
	if !strings.Contains(string(claude.After), wantDest) {
		t.Errorf("claude config does not reference installPath %q:\n%s", wantDest, claude.After)
	}

	// OpenCode stdio uses command:[...] (commandArray) — assert it's a JSON array.
	oc := merges["opencode"]
	var ocCfg map[string]any
	if err := json.Unmarshal(oc.After, &ocCfg); err != nil {
		t.Fatalf("opencode config not json: %v", err)
	}
	mcp := ocCfg["mcp"].(map[string]any)["memory-engram"].(map[string]any)
	if _, ok := mcp["command"].([]any); !ok {
		t.Errorf("opencode command should be a JSON array, got %T (%v)", mcp["command"], mcp["command"])
	}
	if mcp["type"] != "local" {
		t.Errorf("opencode type = %v, want local", mcp["type"])
	}
}

func TestComputeFetchSkipsWhenBinaryMatches(t *testing.T) {
	res, home, _ := testEnv(t)
	// Pre-place a binary whose sha matches the pinned digest.
	content := []byte("ENGRAM-BINARY")
	sha := sha256Hex(content)
	rec := engramRecipe()
	rec.Delivery.Assets = []manifest.Asset{{OS: "linux", Arch: "amd64", URL: "https://x/e", SHA256: sha}}
	dest := filepath.Join(home, ".patronus", "bin", "engram")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, content, 0o755); err != nil {
		t.Fatal(err)
	}

	diffs, err := Compute(Request{
		Recipe: rec, Adapters: loadAdapters(t), Resolver: res,
		Tool: "claude", Scope: "global", GOOS: "linux", GOARCH: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diffs {
		if d.Fetch != nil && d.Action != diff.Skip {
			t.Errorf("expected FETCH to classify as SKIP for matching binary, got %s", d.Action)
		}
	}
}

func TestComputeRemoteHttpMcp_NoFetch(t *testing.T) {
	res, _, _ := testEnv(t)
	rec := &manifest.Recipe{
		Name: "github", Capability: "tools",
		Wire: manifest.Wire{
			Mcp:   &manifest.WireMcp{Transport: "http", URL: "https://api.example/mcp/"},
			Tools: []string{"claude"},
		},
	}
	diffs, err := Compute(Request{Recipe: rec, Adapters: loadAdapters(t), Resolver: res, Tool: "claude", Scope: "local"})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diffs {
		if d.Action == diff.Fetch {
			t.Error("remote http MCP should produce no FETCH")
		}
	}
	if len(diffs) != 1 || diffs[0].Action != diff.Merge {
		t.Fatalf("expected 1 MERGE diff, got %+v", diffs)
	}
	if !strings.Contains(string(diffs[0].After), "https://api.example/mcp/") {
		t.Errorf("merged config missing url:\n%s", diffs[0].After)
	}
}

func TestComputeSelfWiring_EmitsExec(t *testing.T) {
	res, _, _ := testEnv(t)
	rec := &manifest.Recipe{
		Name: "memory-ai-memory", Capability: "memory",
		Delivery: &manifest.Delivery{Primary: "docker"}, // no fetch
		Wire: manifest.Wire{
			SelfWiring: true,
			PostInstall: []string{
				"ai-memory install-mcp --client {tool} --apply",
				"ai-memory install-hooks --agent {tool} --apply",
			},
			Tools: []string{"claude", "codex"},
		},
	}
	diffs, err := Compute(Request{Recipe: rec, Adapters: loadAdapters(t), Resolver: res, Tool: "all", Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	var execs []diff.FileDiff
	for _, d := range diffs {
		if d.Action == diff.Fetch {
			t.Error("docker self-wiring recipe should produce no FETCH")
		}
		if d.Action == diff.Exec {
			execs = append(execs, d)
		}
	}
	// 2 commands × 2 tools = 4 EXEC rows; {tool} substituted.
	if len(execs) != 4 {
		t.Fatalf("expected 4 EXEC diffs, got %d", len(execs))
	}
	found := false
	for _, e := range execs {
		if e.Exec.Display == "ai-memory install-mcp --client claude --apply" {
			found = true
			if got := e.Exec.Command[3]; got != "claude" {
				t.Errorf("argv tool = %q", got)
			}
		}
	}
	if !found {
		t.Error("expected an install-mcp --client claude command")
	}
}

func TestResolveAssetNoMatch(t *testing.T) {
	rec := engramRecipe()
	if _, err := rec.Delivery.ResolveAsset("windows", "arm64"); err == nil {
		t.Fatal("expected error for unpinned host")
	}
}

// sha256Hex returns the lowercase hex sha256 of b, matching classifyFetch.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

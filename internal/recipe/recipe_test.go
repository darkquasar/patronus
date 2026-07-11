package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		Meta: manifest.Meta{
			APIVersion: manifest.APIVersion,
			Family:     manifest.FamilyRecipe,
			Name:       "memory-engram",
			Role:       manifest.RoleMemory,
		},
		Delivery: &manifest.Delivery{
			Source:    manifest.SourceGithubRelease,
			InstallTo: "~/.patronus/bin/",
			Binary:    "engram",
			Assets: []manifest.Asset{
				{OS: "linux", Arch: "amd64", URL: "https://x/engram-linux", SHA256: "abc"},
				{OS: "darwin", Arch: "arm64", URL: "https://x/engram-darwin", SHA256: "def"},
			},
		},
		Wire: manifest.Wire{
			Mode:  manifest.WireModeMcp,
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

// tkRecipe is an install-only `url` recipe: a single pinned artifact, no per-OS
// asset matrix, gated to POSIX hosts (it is a bash script).
func tkRecipe(sha string) *manifest.Recipe {
	return &manifest.Recipe{
		Meta: manifest.Meta{
			APIVersion: manifest.APIVersion, Family: manifest.FamilyRecipe,
			Role: manifest.RoleOrchestration, Name: "tk", Version: "0.3.2",
		},
		Delivery: &manifest.Delivery{
			Source:    manifest.SourceURL,
			URL:       "https://example.test/ticket",
			SHA256:    sha,
			Platforms: []string{"linux", "darwin"},
		},
	}
}

// TestFetchDiffURLSource covers the three outcomes of a `url` delivery: a FETCH
// when the dest is absent, a SKIP when the dest's sha already matches the pin,
// and — on a host outside the Platforms allow-list — NO fetch plus a clear
// advisory. The advisory is asserted, not just its absence-of-FETCH consequence:
// core drops a working work-graph on native Windows, and an accepted regression
// that tells the user nothing is just a silent no-op.
func TestFetchDiffURLSource(t *testing.T) {
	content := []byte("#!/usr/bin/env bash\n# ticket\n")
	sha := sha256Hex(content)

	t.Run("absent dest fetches", func(t *testing.T) {
		res, _, _ := testEnv(t)
		dest, d := fetchDiff(Request{Recipe: tkRecipe(sha), Resolver: res}, "darwin", "arm64")
		if d == nil {
			t.Fatal("want a FETCH diff for a url delivery, got nil")
		}
		if d.Action != diff.Fetch {
			t.Errorf("Action = %s, want FETCH", d.Action)
		}
		if d.Fetch.URL != "https://example.test/ticket" || d.Fetch.SHA256 != sha {
			t.Errorf("FetchSpec = %+v, want the pinned url+sha", d.Fetch)
		}
		if d.Fetch.Archive != "" {
			t.Errorf("Archive = %q, want empty (a url artifact is a raw binary)", d.Fetch.Archive)
		}
		if filepath.Base(dest) != "tk" {
			t.Errorf("dest = %q, want it to end in tk", dest)
		}
	})

	t.Run("matching dest skips", func(t *testing.T) {
		res, home, _ := testEnv(t)
		dest := filepath.Join(home, ".patronus", "bin", "tk")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, content, 0o755); err != nil {
			t.Fatal(err)
		}
		_, d := fetchDiff(Request{Recipe: tkRecipe(sha), Resolver: res}, "darwin", "arm64")
		if d == nil {
			t.Fatal("want a diff, got nil")
		}
		if d.Action != diff.Skip {
			t.Errorf("Action = %s, want SKIP (dest sha already matches the pin)", d.Action)
		}
	})

	t.Run("unsupported platform warns and emits no fetch", func(t *testing.T) {
		res, _, _ := testEnv(t)
		var warnings []string
		dest, d := fetchDiff(Request{
			Recipe:   tkRecipe(sha),
			Resolver: res,
			Warnf:    func(f string, a ...any) { warnings = append(warnings, fmt.Sprintf(f, a...)) },
		}, "windows", "amd64")

		if d != nil {
			t.Errorf("want NO fetch diff on windows, got %+v", d)
		}
		if dest != "" {
			t.Errorf("dest = %q, want empty", dest)
		}
		if len(warnings) != 1 {
			t.Fatalf("want exactly 1 advisory, got %v", warnings)
		}
		if !strings.Contains(warnings[0], "windows") {
			t.Errorf("advisory %q should name the unsupported platform", warnings[0])
		}
	})
}

func TestComputeRemoteHttpMcp_NoFetch(t *testing.T) {
	res, _, _ := testEnv(t)
	rec := &manifest.Recipe{
		Meta: manifest.Meta{Family: manifest.FamilyRecipe, Name: "github", Role: manifest.RoleTools},
		Wire: manifest.Wire{
			Mode:  manifest.WireModeMcp,
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

func TestComputeSelfMode_EmitsExec(t *testing.T) {
	res, _, _ := testEnv(t)
	rec := &manifest.Recipe{
		Meta:     manifest.Meta{Family: manifest.FamilyRecipe, Name: "memory-ai-memory", Role: manifest.RoleMemory},
		Delivery: &manifest.Delivery{Source: manifest.SourceDocker}, // no fetch
		Wire: manifest.Wire{
			Mode: manifest.WireModeSelf,
			Run: []string{
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
		// A mode: self exec is ADVISORY (Patronus surfaces the self-wiring command
		// but never runs it — the recipe's own CLI may not be installed) AND
		// self-managed (provenance). This keeps a missing ai-memory binary from
		// failing the install.
		if !e.Exec.Advisory {
			t.Errorf("self-mode exec %q should be advisory (display-only)", e.Exec.Display)
		}
		if !e.Exec.SelfManaged {
			t.Errorf("self-mode exec %q should be self-managed", e.Exec.Display)
		}
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

// An install-only recipe (deliver: npm, no wire) emits exactly one display-only
// EXEC advisory carrying the package-install command, tool-agnostic and
// self-managed — Patronus never silently runs a global package install, and no
// FETCH/MERGE is produced (the wiring is a separate hook artifact's job).
func TestComputeInstallOnly_EmitsAdvisory(t *testing.T) {
	res, _, _ := testEnv(t)
	rec := &manifest.Recipe{
		Meta:     manifest.Meta{Family: manifest.FamilyRecipe, Name: "tdd-guard", Role: manifest.RoleEval},
		Delivery: &manifest.Delivery{Source: manifest.SourceNpm, Ref: "tdd-guard", Binary: "tdd-guard"},
		// no Wire — install-only
	}
	diffs, err := Compute(Request{Recipe: rec, Adapters: loadAdapters(t), Resolver: res, Tool: "all", Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("want exactly 1 advisory diff, got %d: %+v", len(diffs), diffs)
	}
	d := diffs[0]
	if d.Action != diff.Exec {
		t.Errorf("action = %s, want EXEC (advisory)", d.Action)
	}
	if d.Type != string(manifest.ShapeInstall) {
		t.Errorf("type = %s, want install-only", d.Type)
	}
	if d.Exec == nil || d.Exec.Display != "npm install -g tdd-guard" {
		t.Errorf("advisory command wrong: %+v", d.Exec)
	}
	if d.Exec == nil || !d.Exec.SelfManaged {
		t.Error("install advisory must be self-managed (Patronus does not auto-run global installs)")
	}
	if d.Tool != "-" {
		t.Errorf("tool = %q, want '-' (a global install is tool-agnostic)", d.Tool)
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

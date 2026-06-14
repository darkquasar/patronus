package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// claudeAdapter is a minimal adapter with global + project detect markers.
func claudeAdapter() *manifest.Adapter {
	return &manifest.Adapter{
		Tool: "claude",
		Detect: manifest.AdapterDetect{
			Global:  []string{"~/.claude/", "~/.claude.json"},
			Project: []string{".claude/", "CLAUDE.md"},
		},
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestScanDetectsGlobalAndLocal: a global marker dir and a local marker file
// both exist -> detected at both scopes, with the resolved paths reported.
func TestScanDetectsGlobalAndLocal(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	mustMkdir(t, filepath.Join(home, ".claude"))   // global marker "~/.claude/"
	mustWrite(t, filepath.Join(proj, "CLAUDE.md")) // local marker "CLAUDE.md"

	inv, err := Scan(Options{
		ProjectDir: proj,
		Adapters:   []*manifest.Adapter{claudeAdapter()},
		Env:        envFrom(map[string]string{"HOME": home}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Tools) != 1 || inv.Tools[0].Tool != "claude" {
		t.Fatalf("tools = %+v", inv.Tools)
	}
	g := inv.Tools[0].Global
	if !g.Detected || len(g.MatchedPaths) != 1 {
		t.Errorf("global: detected=%v matched=%v, want detected with 1 path", g.Detected, g.MatchedPaths)
	}
	l := inv.Tools[0].Local
	if !l.Detected || len(l.MatchedPaths) != 1 {
		t.Errorf("local: detected=%v matched=%v, want detected with 1 path", l.Detected, l.MatchedPaths)
	}
	if inv.Home != home || inv.ProjectDir != proj {
		t.Errorf("home/proj = %q/%q", inv.Home, inv.ProjectDir)
	}
}

// TestScanNoMarkers: nothing on disk -> not detected at either scope, empty
// (non-nil) matched-path slices.
func TestScanNoMarkers(t *testing.T) {
	inv, err := Scan(Options{
		ProjectDir: t.TempDir(),
		Adapters:   []*manifest.Adapter{claudeAdapter()},
		Env:        envFrom(map[string]string{"HOME": t.TempDir()}),
	})
	if err != nil {
		t.Fatal(err)
	}
	g := inv.Tools[0].Global
	if g.Detected {
		t.Errorf("global should not be detected: %+v", g)
	}
	if g.MatchedPaths == nil {
		t.Error("MatchedPaths should be non-nil empty slice (JSON stability)")
	}
}

// TestScanMultipleMarkersBothMatch: when several markers for a scope exist, all
// are reported (detection is OR, evidence accumulates).
func TestScanMultipleMarkersBothMatch(t *testing.T) {
	home := t.TempDir()
	mustMkdir(t, filepath.Join(home, ".claude"))
	mustWrite(t, filepath.Join(home, ".claude.json"))

	inv, _ := Scan(Options{
		ProjectDir: t.TempDir(),
		Adapters:   []*manifest.Adapter{claudeAdapter()},
		Env:        envFrom(map[string]string{"HOME": home}),
	})
	g := inv.Tools[0].Global
	if !g.Detected || len(g.MatchedPaths) != 2 {
		t.Errorf("expected both global markers matched, got %+v", g)
	}
}

// TestScanSortsToolsAndSnapshotsEnv: tools come back name-sorted and the env
// snapshot captures the overrides that influenced resolution.
func TestScanSortsToolsAndSnapshotsEnv(t *testing.T) {
	home := t.TempDir()
	codex := &manifest.Adapter{Tool: "codex", Detect: manifest.AdapterDetect{Global: []string{"~/.codex/"}}}
	opencode := &manifest.Adapter{Tool: "opencode", Detect: manifest.AdapterDetect{Global: []string{"~/.config/opencode/"}}}

	inv, _ := Scan(Options{
		ProjectDir: t.TempDir(),
		Adapters:   []*manifest.Adapter{opencode, codex, claudeAdapter()}, // unsorted input
		Env: envFrom(map[string]string{
			"HOME":                home,
			"CODEX_HOME":          "/custom/codex",
			"OPENCODE_CONFIG_DIR": "/oc",
			"XDG_CONFIG_HOME":     "/xdg",
		}),
	})
	got := []string{inv.Tools[0].Tool, inv.Tools[1].Tool, inv.Tools[2].Tool}
	want := []string{"claude", "codex", "opencode"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tool order = %v, want %v", got, want)
			break
		}
	}
	if inv.Env.CodexHome != "/custom/codex" || inv.Env.OpencodeConfigDir != "/oc" || inv.Env.XDGConfigHome != "/xdg" {
		t.Errorf("env snapshot = %+v", inv.Env)
	}
}

// TestScanDefaultsProjectDirToCwd: an empty ProjectDir falls back to cwd.
func TestScanDefaultsProjectDirToCwd(t *testing.T) {
	inv, err := Scan(Options{
		Adapters: []*manifest.Adapter{claudeAdapter()},
		Env:      envFrom(map[string]string{"HOME": t.TempDir()}),
	})
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if inv.ProjectDir != wd {
		t.Errorf("ProjectDir = %q, want cwd %q", inv.ProjectDir, wd)
	}
}

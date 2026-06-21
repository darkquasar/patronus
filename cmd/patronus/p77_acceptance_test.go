package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// P7.7 — the consolidated §6b acceptance gate. The per-sub-phase suites already
// prove the bulk of build → serve → install/lock/remove across all three tools
// (see core_profile_integration_test.go, hardened_*, l1_*, orchestration_*, …).
// This file closes the two coverage gaps the P7.7 audit found:
//
//  1. fetch+wire is the one recipe SHAPE never exercised through a real --deploy
//     (engram/sandbox were only dry-run): a deploy must FETCH the binary AND
//     MERGE an MCP entry into each tool's config. §6b.4 requires ≥1 recipe of
//     every shape to pass a real install.
//  2. a single test asserting the fetch+wire round-trip (install → idempotent
//     SKIP → remove restores the config) symmetrically on the MCP-merge path.

// findUnder reports whether any regular file under dir contains needle, skipping
// Patronus's own bookkeeping (.patronus/ holds state.json + the catalog index
// cache, which name every item regardless of install state — we want to assert on
// the TOOL's config files only).
func findUnder(t *testing.T, dir, needle string) bool {
	t.Helper()
	found := false
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		switch {
		case err != nil, info == nil:
			return err // surface a real walk error; nil entry means nothing to read
		case info.IsDir():
			if info.Name() == ".patronus" {
				return filepath.SkipDir
			}
		default:
			if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), needle) {
				found = true
			}
		}
		return nil
	})
	return found
}

// TestFetchWireRecipeDeploysAcrossTools is the §6b.4 fetch+wire proof: a real
// --deploy of the engram memory recipe (github-release binary + stdio MCP merge)
// on each tool FETCHes the binary (stubbed → SKIP offline) and MERGEs an MCP
// server entry into that tool's real config file — the apply path no prior deploy
// test exercised for this shape.
func TestFetchWireRecipeDeploysAcrossTools(t *testing.T) {
	for _, tool := range []string{"claude", "codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)
			stubBinary(t, home, "engram") // the github-release FETCH SKIPs offline

			if _, errOut, err := runInstall(t, "memory-engram", "--tool", tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("deploy memory-engram on %s: %v\n%s", tool, err, errOut)
			}

			// The MCP server entry was MERGEd into one of the tool's config files
			// under its config dir (~/.claude.json, ~/.codex/config.toml,
			// ~/.config/opencode/opencode.json — scope-mapping differs per tool, so
			// scan rather than hard-code which file).
			if !findUnder(t, home, "memory-engram") {
				t.Errorf("%s: no config under %s mentions the merged engram MCP server", tool, home)
			}

			// Idempotent re-run.
			out, _, err := runInstall(t, "memory-engram", "--tool", tool, "--global", "--dry-run")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "SKIP") {
				t.Errorf("%s: re-install should be idempotent (SKIP):\n%s", tool, out)
			}
		})
	}
}

// TestFetchWireRecipeRemoveRestoresConfig proves the fetch+wire MCP merge
// round-trips: after install the engram server entry is present; after remove the
// config no longer carries it (MERGE → RESTORE), closing the round-trip on the
// MCP-config path for this shape.
func TestFetchWireRecipeRemoveRestoresConfig(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)
	stubBinary(t, home, "engram")

	if _, e, err := runInstall(t, "memory-engram", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	if !findUnder(t, home, "memory-engram") {
		t.Fatal("precondition: engram MCP entry should be present after install")
	}

	if _, e, err := execRemove(t, "memory-engram", "--global", "--deploy"); err != nil {
		t.Fatalf("remove memory-engram: %v\n%s", err, e)
	}
	if findUnder(t, home, "memory-engram") {
		t.Errorf("engram MCP entry should be gone after remove (config under %s)", home)
	}
}

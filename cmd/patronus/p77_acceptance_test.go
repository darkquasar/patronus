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
// --deploy of a fetch+wire recipe (github-release binary + stdio MCP merge) on each
// tool FETCHes the binary AND MERGEs an MCP server entry into that tool's real
// config file — the apply path no prior deploy test exercised for this shape.
//
// CLASS A (mechanism): the recipe's identity is irrelevant to "a fetch+wire deploy
// places the binary and merges the MCP entry", so it is the fixture's fix-mcp-bin
// (the memory-engram shape, with bytes this suite invented).
//
// This is STRICTLY more coverage than before: the old form pre-placed a dummy
// `engram` with stubBinary so the github-release FETCH classified SKIP — meaning the
// download/verify/extract path, the very thing "fetch+wire" names, never ran. Here it
// runs for real, and the placed bytes are asserted to be the extracted member.
func TestFetchWireRecipeDeploysAcrossTools(t *testing.T) {
	for _, tool := range []string{"claude", "codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			f := fixtureRegistry(t)
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t, "fix-mcp-bin", "--tool", tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("deploy fix-mcp-bin on %s: %v\n%s", tool, err, errOut)
			}

			// The FETCH truly ran: the binary was downloaded, verified against its
			// invented pin, extracted, and placed.
			placed := mustRead(t, filepath.Join(home, ".patronus", "bin", "fix-mcp-bin"))
			if shaHex(placed) != shaHex(fixMcpBinary) {
				t.Errorf("%s: placed binary is not the tarball's extracted member", tool)
			}

			// The MCP server entry was MERGEd into one of the tool's config files under
			// its config dir (~/.claude.json, ~/.codex/config.toml,
			// ~/.config/opencode/opencode.json — scope-mapping differs per tool, so scan
			// rather than hard-code which file).
			if !findUnder(t, home, "fix-mcp-bin") {
				t.Errorf("%s: no config under %s mentions the merged MCP server", tool, home)
			}

			// Idempotent re-run.
			out, _, err := runInstall(t, "fix-mcp-bin", "--tool", tool, "--global", "--dry-run")
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
// round-trips: after install the server entry is present; after remove the config no
// longer carries it (MERGE → RESTORE), closing the round-trip on the MCP-config path
// for this shape. CLASS A, on the fixture.
func TestFetchWireRecipeRemoveRestoresConfig(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "fix-mcp-bin", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	if !findUnder(t, home, "fix-mcp-bin") {
		t.Fatal("precondition: the MCP entry should be present after install")
	}

	if _, e, err := execRemove(t, "fix-mcp-bin", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-mcp-bin: %v\n%s", err, e)
	}
	if findUnder(t, home, "fix-mcp-bin") {
		t.Errorf("the MCP entry should be gone after remove (config under %s)", home)
	}
}

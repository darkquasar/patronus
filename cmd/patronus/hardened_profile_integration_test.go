package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

// realCatalog loads the REAL checkout's catalog straight off disk — no build, no
// registry, no fetcher. It is how a CLASS-B test reads the catalog's SHAPE (which
// items a profile names) without ever touching its PINS: nothing is downloaded,
// hashed, or placed, so the archive-hashing fix cannot break it and CI never
// fetches a byte.
func realCatalog(t *testing.T) *registry.Catalog {
	t.Helper()
	root, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatalf("discover repo root: %v", err)
	}
	cat, err := registry.NewLocalRegistry(root).Catalog(context.Background())
	if err != nil {
		t.Fatalf("the real catalog does not load: %v", err)
	}
	return cat
}

// TestHardenedProfileSandboxFlavourDiverges is a CLASS-B test: it asserts the REAL
// catalog's CONTENTS — that the `hardened` profile really wires the L6 sandbox
// layer with a per-tool flavour, and that exactly one resolves per --tool:
//
//	claude   -> native-sandbox     (Claude's own settings `sandbox` switch)
//	codex    -> native-sandbox     (Codex's sandbox_mode = workspace-write)
//	opencode -> sandbox-runtime    (OpenCode has no native switch; srt wraps it)
//
// The item names ARE the assertion — renaming them to fixture names would produce
// a green tautology, so they stay real.
//
// It asserts against profile.Resolve rather than an install: the guarantee is a
// statement about what the CATALOG names, and resolution is where that lives. This
// keeps a real-catalog test entirely off the fetch path (`hardened` extends `core`,
// which wires the gitleaks + tk recipes whose pins are REAL upstream digests that
// no invented bytes can satisfy).
//
// It deliberately does NOT assert the settings.json/config.toml BYTES the switch
// produces. Doing that would need a --deploy of a binary-bearing profile, which is
// exactly what the archive-hashing fix makes impossible offline. The per-tool
// SETTINGS-MERGE mechanism is proven on invented items in internal/profile and
// internal/config; what is left here — and what only the real catalog can say — is
// that `hardened` names these three flavours.
func TestHardenedProfileSandboxFlavourDiverges(t *testing.T) {
	cat := realCatalog(t)
	for _, tc := range []struct {
		tool, want string
	}{
		{"claude", "native-sandbox"},
		{"codex", "native-sandbox"},
		{"opencode", "sandbox-runtime"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			r, err := profile.Resolve(cat, "hardened", tc.tool)
			if err != nil {
				t.Fatalf("resolve hardened/%s: %v", tc.tool, err)
			}
			var got []string
			for _, it := range r.Items {
				if it.Slot == "sandbox" {
					got = append(got, it.Name)
				}
			}
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("hardened/%s sandbox = %v, want exactly [%s]", tc.tool, got, tc.want)
			}
		})
	}
}

// TestHardIsolationAddsMicrosandbox is CLASS B: the hard-isolation overlay (extends
// hardened) really adds the microsandbox MCP — the microVM tier on top of the
// per-tool OS sandboxes — while KEEPING the native sandbox it inherits.
func TestHardIsolationAddsMicrosandbox(t *testing.T) {
	cat := realCatalog(t)
	r, err := profile.Resolve(cat, "hard-isolation", "claude")
	if err != nil {
		t.Fatalf("resolve hard-isolation: %v", err)
	}
	names := strings.Join(r.Names(), " ")
	if !strings.Contains(names, "microsandbox") {
		t.Errorf("hard-isolation should wire microsandbox, got: %s", names)
	}
	if !strings.Contains(names, "native-sandbox") {
		t.Errorf("hard-isolation should keep hardened's native-sandbox, got: %s", names)
	}
}

// TestFixtureHookFoldsIntoSettings is the CLASS-A counterpart, on the FIXTURE: a
// hook artifact folds into the tool's settings.json, and its `requires:` edge pulls
// the binary it invokes into the closure — the gitleaks-guard -> gitleaks shape,
// proven with bytes this test invented, so download -> verify -> extract -> place
// actually RUNS (the path stubBinary skipped entirely).
func TestFixtureHookFoldsIntoSettings(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "fix-hook", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}
	settings := string(mustRead(t, filepath.Join(home, ".claude", "settings.json")))
	if !strings.Contains(settings, "PreToolUse") || !strings.Contains(settings, "fix-archive-bin --check") {
		t.Errorf("fix-hook should fold a PreToolUse entry into settings.json:\n%s", settings)
	}
	// requires: [fix-archive-bin] pulled the binary into the closure and PLACED it.
	placed, err := os.ReadFile(filepath.Join(home, ".patronus", "bin", "fix-archive-bin"))
	if err != nil {
		t.Fatalf("the hook's required binary was not placed: %v", err)
	}
	if shaHex(placed) != shaHex(fixArchivedBinary) {
		t.Errorf("placed binary is not the tarball's extracted member")
	}
}

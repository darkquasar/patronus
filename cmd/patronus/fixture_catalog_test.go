package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
)

// fixtureCatalog writes a SELF-CONTAINED Patronus repo into t.TempDir() and
// returns its root. This is the keystone of the test surface: every binary pin in
// it is sha256(bytes this file invents), so NOTHING upstream can drift, no
// third-party byte enters the repo, and a test asserts Patronus's BEHAVIOR rather
// than the real catalog's CONTENTS.
//
// registry.DiscoverRoot walks up from cwd and returns the first dir holding BOTH
// artifacts/ and adapters/ (internal/registry/local.go), so a test that t.Chdir's
// here and runs `build` builds THIS catalog, not the repo's.
//
// Do not "simplify" this by importing a real recipe's sha. The invented pin is the
// whole point (see docs/specs/01-lifecycle-and-test-surface/test-surface-spec.md).

// The two fixture payloads are INERT TEXT, deliberately not programs.
//
// Patronus's delivery path is download -> hash against the pin -> write to
// ~/.patronus/bin/<name>. It never EXECUTES what it places, and neither does any
// test: the assertions are on the bytes and their digest. So the payload only has
// to be bytes we invented, whose sha256 the recipe pins. Making them plainly
// non-executable keeps that honest — there is no shebang here to tempt a later
// reader (or a later test) into running one.

// fixRawBinary is the `fix-bin` raw-delivery payload. Its sha256 IS the pin.
var fixRawBinary = []byte("fixture payload: fix-bin (raw delivery). Not a program.\n")

// fixArchivedBinary is the member inside `fix-archive-bin`'s tarball. The recipe
// pins the sha256 of the TARBALL (that is what a github-release delivery verifies);
// this is the digest of what actually lands on disk after extraction.
var fixArchivedBinary = []byte("fixture payload: fix-archive-bin (extracted from a tar.gz). Not a program.\n")

// fixMcpBinary is the member inside `fix-mcp-bin`'s tarball — the FETCH+WIRE shape
// (a delivered binary that is ALSO merged into each tool's MCP config).
var fixMcpBinary = []byte("fixture payload: fix-mcp-bin (fetch+wire). Not a program.\n")

const (
	fixRawURL     = testRegistryBase + "/bin/fix-bin"
	fixArchiveURL = testRegistryBase + "/bin/fix-archive-bin.tar.gz"
	fixMcpURL     = testRegistryBase + "/bin/fix-mcp-bin.tar.gz"
)

// shaHex is sha256(b) as lowercase hex — the form a recipe pin takes.
func shaHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// fixArchiveTarGz is the tarball `fix-archive-bin` delivers, holding the binary at
// member path "fix-archive-bin".
func fixArchiveTarGz(t *testing.T) []byte {
	t.Helper()
	return mustTarGz(t, map[string][]byte{"fix-archive-bin": fixArchivedBinary})
}

// fixMcpTarGz is the tarball `fix-mcp-bin` delivers, holding its binary at member
// path "fix-mcp-bin".
func fixMcpTarGz(t *testing.T) []byte {
	t.Helper()
	return mustTarGz(t, map[string][]byte{"fix-mcp-bin": fixMcpBinary})
}

func fixtureCatalog(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// adapters/ must EXIST for DiscoverRoot. Copy the REAL adapters: they are OUR
	// code — it is the pins and upstream BYTES we must not inherit.
	realRoot, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatalf("discover repo root (to copy adapters/): %v", err)
	}
	copyDir(t, filepath.Join(realRoot, "adapters"), filepath.Join(root, "adapters"))

	// --- recipes -----------------------------------------------------------
	// fix-bin: a RAW (source: url) delivery. Patronus hashes the download against
	// this pin AND re-hashes the placed file on every run (classifyFetch).
	write("recipes/fix-bin.yaml", `apiVersion: patronus/v2
family: recipe
name: fix-bin
role: orchestration
summary: "Fixture raw-delivery binary. Pin = sha256 of bytes this test invented."
deliver:
  source: url
  installTo: "~/.patronus/bin/"
  binary: fix-bin
  url: "`+fixRawURL+`"
  sha256: "`+shaHex(fixRawBinary)+`"
  platforms: [linux, darwin, windows]
`)

	// fix-archive-bin: a github-release (tar.gz) delivery. The pin is the
	// TARBALL's sha; the extracted member is what lands on disk. Every os/arch
	// points at the same invented tarball so the fixture works on any host.
	tgzSum := shaHex(fixArchiveTarGz(t))
	var assets strings.Builder
	for _, p := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	} {
		fmt.Fprintf(&assets, `    - os: %s
      arch: %s
      url: "%s"
      sha256: "%s"
      archive: tar.gz
      binaryPath: fix-archive-bin
`, p.goos, p.goarch, fixArchiveURL, tgzSum)
	}
	write("recipes/fix-archive-bin.yaml", `apiVersion: patronus/v2
family: recipe
name: fix-archive-bin
role: guardrail
summary: "Fixture archive-delivery binary. Pin = sha256 of a tarball this test built."
deliver:
  source: github-release
  installTo: "~/.patronus/bin/"
  binary: fix-archive-bin
  assets:
`+assets.String())

	// The same asset matrix for the fetch+wire recipe, at its own URL and member
	// name. It carries its own tarball so its extracted member is named after it.
	mcpTgzSum := shaHex(fixMcpTarGz(t))
	var mcpAssets strings.Builder
	for _, p := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	} {
		fmt.Fprintf(&mcpAssets, `    - os: %s
      arch: %s
      url: "%s"
      sha256: "%s"
      archive: tar.gz
      binaryPath: fix-mcp-bin
`, p.goos, p.goarch, fixMcpURL, mcpTgzSum)
	}

	// fix-mcp-bin: the FETCH+WIRE shape — a github-release delivery AND an MCP
	// config merge (the memory-engram shape). Deploying it must BOTH place the
	// binary and merge a stdio MCP server entry into each tool's own config. It
	// serves the same invented tarball as fix-archive-bin, under its own name.
	write("recipes/fix-mcp-bin.yaml", `apiVersion: patronus/v2
family: recipe
name: fix-mcp-bin
role: memory
summary: "Fixture fetch+wire recipe: an archive-delivered binary that is ALSO merged into each tool's MCP config."
deliver:
  source: github-release
  installTo: "~/.patronus/bin/"
  binary: fix-mcp-bin
  assets:
`+mcpAssets.String()+`
wire:
  mode: mcp
  mcp:
    transport: stdio
    command: "{installPath}"
    args: ["mcp"]
  tools: [claude, codex, opencode]
`)

	// --- artifacts ---------------------------------------------------------
	// fix-instruction requires: [fix-bin] — this edge is what the requires-closure
	// tests assert. It is the fixture's stand-in for ticket -> tk.
	write("artifacts/instructions/fix-instruction/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: instruction
role: orchestration
name: fix-instruction
description: "Fixture instruction. Requires fix-bin, so installing it pulls the binary into the closure."
version: 1.0.0
entry: INSTRUCTIONS.md
targets: [claude, codex, opencode]
defaults:
  scope: global
requires: [fix-bin]
`)
	write("artifacts/instructions/fix-instruction/INSTRUCTIONS.md",
		"# Fixture instruction\n\nDrive `fix-bin`.\n")

	// fix-instruction-2 is a SECOND instruction APPENDing to the SAME CLAUDE.md.
	// Two contributors to one composed file is what the composed-APPEND state
	// recording and the selective-remove round-trip need: removing one must strip
	// exactly its fenced section and leave the other's standing.
	write("artifacts/instructions/fix-instruction-2/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: instruction
role: instruction
name: fix-instruction-2
description: "Second fixture instruction, sharing one CLAUDE.md with fix-instruction (composed APPEND)."
version: 1.0.0
entry: INSTRUCTIONS.md
targets: [claude, codex, opencode]
defaults:
  scope: global
`)
	write("artifacts/instructions/fix-instruction-2/INSTRUCTIONS.md",
		"# Second fixture instruction\n\nA distinctive fixture rule: always fix the fixture.\n")

	write("artifacts/skills/fix-skill/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: skill
role: capability
name: fix-skill
description: "Fixture skill artifact."
version: 1.0.0
entry: SKILL.md
targets: [claude, codex, opencode]
defaults:
  scope: global
`)
	write("artifacts/skills/fix-skill/SKILL.md",
		"---\nname: fix-skill\ndescription: fixture skill\n---\nDo the fixture thing.\n")

	// The @tool flavour pair: one profile slot resolves a DIFFERENT item per tool.
	for _, tool := range []string{"claude", "codex"} {
		write("artifacts/skills/fix-skill-"+tool+"/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: skill
role: capability
name: fix-skill-`+tool+`
description: "Fixture skill, `+tool+` flavour."
version: 1.0.0
entry: SKILL.md
targets: [`+tool+`]
defaults:
  scope: global
`)
		write("artifacts/skills/fix-skill-"+tool+"/SKILL.md",
			"---\nname: fix-skill-"+tool+"\ndescription: fixture skill ("+tool+")\n---\nFlavoured for "+tool+".\n")
	}

	// fix-hook: a settings.json MERGE. requires the archive binary it invokes, so
	// the hook -> its-binary edge (gitleaks-guard -> gitleaks) is exercised too.
	write("artifacts/hooks/fix-hook/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: hook
role: eval
name: fix-hook
description: "Fixture PreToolUse hook; requires the archive-delivered binary it runs."
version: 1.0.0
targets: [claude, codex, opencode]
defaults:
  scope: global
requires: [fix-archive-bin]
hook:
  event: PreToolUse
  matcher: Edit|Write
  command: fix-archive-bin --check
`)

	// fix-hook-2: a SECOND PreToolUse hook, script-bearing. Two hooks on one event
	// is what proves the settings.json compose-FOLD (both land in ONE array) and the
	// selective-remove round-trip (removing one strips its element + its script, and
	// the sibling survives). It carries no requires:, so it also proves a hook need
	// not drag a binary.
	write("artifacts/hooks/fix-hook-2/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: hook
role: guardrail
name: fix-hook-2
description: "Second fixture PreToolUse hook, script-bearing, so two hooks fold into one settings array."
version: 1.0.0
entry: ""
files: [fix-hook-2.sh]
targets: [claude, codex, opencode]
defaults:
  scope: global
hook:
  event: PreToolUse
  matcher: Edit|Write
  command: "{script}"
  script: fix-hook-2.sh
`)
	write("artifacts/hooks/fix-hook-2/fix-hook-2.sh",
		"#!/bin/sh\n# fixture hook script — never executed by the tests; only placed and removed.\nexit 0\n")

	// fix-hook-claude: a CLAUDE-ONLY hook (targets: [claude]). Wired into a profile
	// as a @claude flavour, it must land on claude and be silently SKIPPED on
	// codex/opencode — the shape of the real re-grounding hooks, which have no hook
	// surface on the other tools.
	write("artifacts/hooks/fix-hook-claude/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: hook
role: capability
name: fix-hook-claude
description: "Claude-only fixture hook: lands on claude, silently skipped on codex/opencode."
version: 1.0.0
entry: ""
files: [fix-hook-claude.sh]
targets: [claude]
defaults:
  scope: global
hook:
  event: UserPromptSubmit
  command: "{script}"
  script: fix-hook-claude.sh
`)
	// The script is PLACED and its bytes asserted; it is executed only by the test
	// that proves a placed hook script runs (mirroring the real skills-heartbeat).
	// It enumerates the installed skills dir, exactly as the real heartbeat does.
	write("artifacts/hooks/fix-hook-claude/fix-hook-claude.sh", `#!/usr/bin/env bash
# Fixture hook: lists the installed skills, the way skills-heartbeat does.
set -euo pipefail
names=""
if [ -d "${HOME}/.claude/skills" ]; then
  for d in "${HOME}/.claude/skills"/*/; do
    [ -d "$d" ] || continue
    names="${names:+$names, }$(basename "$d")"
  done
fi
printf '{"installedSkills":"%s"}\n' "$names"
`)

	// fix-setting: a SCALAR settings MERGE (the ccusage-statusline shape). Installing
	// it adds one key to settings.json; removing it takes exactly that key away and
	// leaves the sibling hooks in the same file standing.
	write("artifacts/settings/fix-setting/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: setting
role: observability
name: fix-setting
description: "Fixture scalar setting: one key merged into settings.json, removed cleanly."
version: 1.0.0
entry: ""
targets: [claude]
defaults:
  scope: global
setting:
  path: fixtureLine
  value:
    command: fixture statusline
`)

	write("artifacts/output-styles/fix-style/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: output-style
role: instruction
name: fix-style
description: "Fixture output-style: a CREATE on claude, an APPEND on codex/opencode."
version: 1.0.0
entry: STYLE.md
targets: [claude, codex, opencode]
defaults:
  scope: global
`)
	write("artifacts/output-styles/fix-style/STYLE.md",
		"---\nname: fix-style\ndescription: fixture output style\nkeep-coding-instructions: true\n---\nAlways draw a fixture.\n")

	// --- profiles ----------------------------------------------------------
	write("profiles/fix-all.yaml", `apiVersion: patronus/v2
family: profile
role: lifecycle
name: fix-all
summary: "Fixture profile filling every layer slot."
layers:
  instructions:
    - fix-instruction
    - fix-instruction-2
    - fix-style
  capabilities:
    - fix-skill
    - fix-hook-claude@claude
  eval:
    - fix-hook
  guardrails:
    - fix-hook-2
`)

	write("profiles/fix-extends.yaml", `apiVersion: patronus/v2
family: profile
role: lifecycle
name: fix-extends
summary: "Fixture profile proving extends: composes."
extends: fix-all
layers:
  capabilities:
    - fix-skill-claude
`)

	write("profiles/fix-flavoured.yaml", `apiVersion: patronus/v2
family: profile
role: lifecycle
name: fix-flavoured
summary: "Fixture profile proving @tool flavours diverge per --tool."
layers:
  capabilities:
    - fix-skill-claude@claude
    - fix-skill-codex@codex
`)

	return root
}

// copyDir recursively copies src into dst. Used to lift the repo's OWN adapters/
// into the fixture (they are our code; only pins and upstream bytes are forbidden).
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

// TestFixtureCatalogBuilds is the fixture's own self-test: the fixture tree is a
// valid Patronus repo root, `build` produces a loadable index from it, and the
// index carries the fixture's items — including the requires-closure edge and a
// pin that is, by construction, the sha256 of bytes this test invented.
func TestFixtureCatalogBuilds(t *testing.T) {
	root := fixtureCatalog(t)

	// DiscoverRoot must find the fixture, not the real repo above it.
	got, err := registry.DiscoverRoot(root)
	if err != nil {
		t.Fatalf("DiscoverRoot(%s): %v", root, err)
	}
	if got != root {
		t.Fatalf("DiscoverRoot = %s, want the fixture root %s", got, root)
	}

	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	ix, err := registry.LoadIndex(mustRead(t, filepath.Join(outDir, "catalog", "index.json")))
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, a := range ix.Artifacts {
		names[a.Manifest.Name] = true
	}
	for _, want := range []string{"fix-instruction", "fix-skill", "fix-hook", "fix-style"} {
		if !names[want] {
			t.Errorf("index missing fixture artifact %q", want)
		}
	}

	// The keystone: the raw recipe's pin IS the sha256 of the bytes we invented.
	// Nothing upstream exists to drift from.
	rawYAML, err := os.ReadFile(filepath.Join(root, "recipes", "fix-bin.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := shaHex(fixRawBinary); !strings.Contains(string(rawYAML), want) {
		t.Errorf("fix-bin.yaml does not pin sha256(fixRawBinary)=%s:\n%s", want, rawYAML)
	}
}

// TestFixtureRegistryServesBothDeliveryShapes proves the fetcher can serve BOTH
// fixture binaries — the raw payload and the tarball — at the URLs their recipes
// pin. This is what lets an install drive download -> verify -> extract -> place
// for real, which stubBinary never did.
func TestFixtureRegistryServesBothDeliveryShapes(t *testing.T) {
	f := fixtureRegistry(t)

	raw, ok := f.bodies[fixRawURL]
	if !ok {
		t.Fatalf("fetcher does not serve the raw binary at %s", fixRawURL)
	}
	if shaHex(raw) != shaHex(fixRawBinary) {
		t.Errorf("served raw bytes do not match fixRawBinary")
	}

	tgz, ok := f.bodies[fixArchiveURL]
	if !ok {
		t.Fatalf("fetcher does not serve the archive at %s", fixArchiveURL)
	}
	if shaHex(tgz) != shaHex(fixArchiveTarGz(t)) {
		t.Errorf("served tarball does not match the fixture tarball")
	}

	// And the catalog index it serves is the FIXTURE's, not the real one.
	idx, ok := f.bodies[testRegistryBase+"/catalog/index.json"]
	if !ok {
		t.Fatal("fetcher does not serve a catalog index")
	}
	if strings.Contains(string(idx), "\"grilling\"") {
		t.Error("fixture registry served the REAL catalog — DiscoverRoot did not pick the fixture root")
	}
}

// fixtureRegistry builds the fixture catalog and serves it from memory: the
// index, every artifact tarball, AND both fixture binaries at the URLs their
// recipes pin. The drop-in replacement for builtRegistry at every test that
// asserts Patronus's BEHAVIOR (Class A) rather than the real catalog's CONTENTS.
//
// ORDERING (do not reorder): build runs while cwd is the fixture root, BEFORE any
// caller invokes withRemoteEnv — withRemoteEnv t.Chdir's into a dir where
// DiscoverRoot fails by design (that is what selects the Remote registry).
func fixtureRegistry(t *testing.T) *servingFetcher {
	t.Helper()
	root := fixtureCatalog(t)
	outDir := t.TempDir()

	t.Chdir(root) // build the FIXTURE, not the repo
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture registry: %v", err)
	}

	f := serveTree(t, outDir)
	// Serve the binaries the fixture's recipes pin. The bytes and the pins come
	// from the same place (this file), so they cannot drift.
	f.bodies[fixRawURL] = fixRawBinary
	f.bodies[fixArchiveURL] = fixArchiveTarGz(t)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	return f
}

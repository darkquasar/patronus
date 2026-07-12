package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// This is the §6b acceptance gate for P7.1: it drives the REAL CLI (build → serve
// → install → idempotent re-install → lock → remove) in a temp dir against a
// served catalog, exercising BOTH new mechanisms at once —
//   - the output-style artifact type, which DIVERGES per tool: a CREATE file on
//     Claude (output-styles/<name>.md) but an AGENTS.md APPEND on Codex/OpenCode;
//   - profile flavours (`@tool` suffix), so one `smoke` profile resolves a
//     different item per --tool.
// The fixture is throwaway (injected into the served index + a hand-built tarball,
// the same technique TestProfileInstallFollowsPerItemLock uses) — no repo artifact
// is authored, per Build Constraint §0.

const styleBody = "---\nname: smoke-style\ndescription: smoke output style\nkeep-coding-instructions: true\n---\nAlways draw an ASCII diagram.\n"

// serveSmokeFixture builds the real baseline registry, then injects a throwaway
// output-style artifact `smoke-style` and a flavoured profile `smoke` into the
// served index (+ the artifact's tarball), and refreshes the client cache so the
// commands see them. Returns the fetcher + temp HOME.
// The baseline is the FIXTURE catalog, not the real one. This file already INVENTS
// its item (smoke-style) — it only ever needed a valid catalog to inject into, and
// building the real one dragged every real recipe, and every real upstream pin, in
// for no reason.
func serveSmokeFixture(t *testing.T) (*servingFetcher, string) {
	t.Helper()
	outDir := t.TempDir()
	t.Chdir(fixtureCatalog(t)) // baseline = the fixture; no real pins inherited
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	home := withRemoteEnv(t, f)

	// Build the output-style artifact's portable-source tarball.
	tgz := mustTarGz(t, map[string][]byte{
		"patronus.yaml": []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: output-style\nrole: instruction\nname: smoke-style\ndescription: smoke output style\nversion: 1.0.0\nentry: STYLE.md\ntargets: [claude, codex, opencode]\ndefaults:\n  scope: global\n"),
		"STYLE.md":      []byte(styleBody),
	})
	url := testRegistryBase + "/catalog/smoke-style/1.0.0/smoke-style-1.0.0.tar.gz"

	// Inject the artifact + a flavoured profile into the served index.
	idx := mustRead(t, filepath.Join(outDir, "catalog", "index.json"))
	ix, err := registry.LoadIndex(idx)
	if err != nil {
		t.Fatal(err)
	}
	ix.Artifacts = append(ix.Artifacts, registry.IndexArtifact{
		Manifest: &manifest.Artifact{
			Meta:     manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyArtifact, Name: "smoke-style", Description: "smoke output style", Version: "1.0.0", Role: manifest.RoleInstruction},
			Type:     manifest.TypeOutputStyle,
			Entry:    "STYLE.md",
			Targets:  []string{"claude", "codex", "opencode"},
			Defaults: manifest.ArtifactDefaults{Scope: "global"},
		},
		Tarball: registry.Tarball{URL: url, SHA256: shaOf(tgz)},
	})
	ix.Profiles = append(ix.Profiles, registry.IndexProfile{
		Manifest: &manifest.Profile{
			Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyProfile, Name: "smoke", Role: manifest.RoleLifecycle},
			Layers: manifest.ProfileLayers{
				// Same intent, flavoured per tool — each tool selects exactly one.
				Instructions: manifest.StringList{"smoke-style@claude", "smoke-style@codex", "smoke-style@opencode"},
			},
		},
	})
	mutated, err := ix.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	f.bodies[testRegistryBase+"/catalog/index.json"] = mutated
	f.bodies[testRegistryBase+"/catalog/index.json.sha256"] = []byte(shaOf(mutated) + "\n")
	f.bodies[url] = tgz

	if _, _, err := runUpdate(t); err != nil {
		t.Fatalf("update: %v", err)
	}
	return f, home
}

// TestOutputStyleFlavourInstallClaude proves the Claude flavour: a CREATE file at
// output-styles/smoke-style.md (carrying the keep-coding-instructions frontmatter),
// idempotent on re-run, recorded in state, lockable, and cleanly removable.
func TestOutputStyleFlavourInstallClaude(t *testing.T) {
	_, home := serveSmokeFixture(t)

	// list shows the new item with its Type.
	if out, _, err := runList(t); err != nil {
		t.Fatalf("list: %v", err)
	} else if !strings.Contains(out, "smoke-style") || !strings.Contains(out, "output-style") {
		t.Errorf("list missing smoke-style/output-style:\n%s", out)
	}

	// Install the flavoured profile for claude.
	if _, errOut, err := runInstall(t, "--profile", "smoke", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	stylePath := filepath.Join(home, ".claude", "output-styles", "smoke-style.md")
	got, err := os.ReadFile(stylePath)
	if err != nil {
		t.Fatalf("output-style not created at %s: %v", stylePath, err)
	}
	if string(got) != styleBody {
		t.Errorf("output-style body not passed through verbatim:\n%s", got)
	}
	if !strings.Contains(string(got), "keep-coding-instructions: true") {
		t.Errorf("strict default frontmatter missing:\n%s", got)
	}
	// Claude must NOT have written an AGENTS.md block for this.
	if _, err := os.Stat(filepath.Join(home, ".claude", "AGENTS.md")); err == nil {
		t.Error("claude flavour should not write AGENTS.md")
	}

	// state.json recorded it.
	st := mustRead(t, filepath.Join(home, ".patronus", "state.json"))
	if !strings.Contains(string(st), "smoke-style") {
		t.Errorf("state missing smoke-style:\n%s", st)
	}

	// Idempotent re-run → SKIP.
	out, _, err := runInstall(t, "--profile", "smoke", "--tool", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("re-install should be idempotent (SKIP):\n%s", out)
	}

	// lock pins the claude-flavoured resolution.
	if _, _, err := runLock(t, "--profile", "smoke", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	if l := string(mustRead(t, filepath.Join(wd, "patronus.lock"))); !strings.Contains(l, "smoke-style") {
		t.Errorf("lock missing flavoured item smoke-style:\n%s", l)
	}

	// remove round-trips: the CREATEd file is deleted.
	if _, errOut, err := execRemove(t, "smoke-style", "--global", "--deploy"); err != nil {
		t.Fatalf("remove: %v\n%s", err, errOut)
	}
	if _, err := os.Stat(stylePath); !os.IsNotExist(err) {
		t.Errorf("output-style should be deleted after remove, stat err = %v", err)
	}
}

// TestOutputStyleFlavourAppendsForCodexAndOpencode proves the APPEND flavour: on
// Codex/OpenCode the same profile lands the style as a fenced AGENTS.md section
// (NOT an output-styles file), idempotent, and UNAPPENDs cleanly on remove leaving
// the user's prose intact.
func TestOutputStyleFlavourAppendsForCodexAndOpencode(t *testing.T) {
	for _, tc := range []struct {
		tool      string
		agentsRel string // AGENTS.md location for a --global install of this tool
	}{
		{"codex", filepath.Join(".codex", "AGENTS.md")},
		{"opencode", filepath.Join(".config", "opencode", "AGENTS.md")},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			_, home := serveSmokeFixture(t)

			if _, errOut, err := runInstall(t, "--profile", "smoke", "--tool", tc.tool, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install: %v\n%s", err, errOut)
			}

			// No output-styles file for these tools.
			if _, err := os.Stat(filepath.Join(home, ".claude", "output-styles", "smoke-style.md")); err == nil {
				t.Errorf("%s flavour must not write a Claude output-styles file", tc.tool)
			}

			agents := filepath.Join(home, tc.agentsRel)
			body, err := os.ReadFile(agents)
			if err != nil {
				t.Fatalf("%s: AGENTS.md not written at %s: %v", tc.tool, agents, err)
			}
			if !strings.Contains(string(body), "<!-- patronus:start smoke-style -->") {
				t.Errorf("%s: AGENTS.md missing fenced output-style section:\n%s", tc.tool, body)
			}
			if !strings.Contains(string(body), "Always draw an ASCII diagram.") {
				t.Errorf("%s: AGENTS.md missing style content:\n%s", tc.tool, body)
			}

			// Idempotent re-run → SKIP.
			out, _, err := runInstall(t, "--profile", "smoke", "--tool", tc.tool, "--global", "--dry-run")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "SKIP") {
				t.Errorf("%s: re-install should be idempotent (SKIP):\n%s", tc.tool, out)
			}

			// remove UNAPPENDs: the fenced section is stripped.
			if _, errOut, err := execRemove(t, "smoke-style", "--global", "--deploy"); err != nil {
				t.Fatalf("%s remove: %v\n%s", tc.tool, err, errOut)
			}
			rest, err := os.ReadFile(agents)
			// The file may be deleted (if it became empty) or just stripped — both fine.
			if err == nil && strings.Contains(string(rest), "smoke-style") {
				t.Errorf("%s: output-style section not removed:\n%s", tc.tool, rest)
			}
		})
	}
}

package registry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// repoRoot walks up from the test's working directory to the Patronus repo root
// (the dir holding artifacts/ + adapters/), so the integrity test reads the real
// shipped catalog rather than a fixture.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if isDir(filepath.Join(dir, "artifacts")) && isDir(filepath.Join(dir, "adapters")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found (no artifacts/+adapters/ above cwd)")
		}
		dir = parent
	}
}

// TestRealCatalogLoadsAndMatchesOntology is the canary against catalog<->code
// drift. It loads EVERY shipped manifest through the real loaders and asserts
// each item's three axes (family/type/role) and computed recipe Shape against
// the schema, without freezing catalog membership. If a future change desyncs
// a manifest from the schema —
// a bad enum value, a renamed field, a recipe whose deliver×wire no longer
// computes the documented shape — this fails loudly instead of shipping broken.
func TestRealCatalogLoadsAndMatchesOntology(t *testing.T) {
	root := repoRoot(t)
	reg := NewLocalRegistry(root)
	cat, err := reg.Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	if len(cat.Artifacts) == 0 || len(cat.Recipes) == 0 || len(cat.Profiles) == 0 {
		t.Fatal("real catalog must contain artifacts, recipes and profiles")
	}
	for _, entry := range cat.Artifacts {
		if err := entry.Manifest.Validate(); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
		if err := cp00ValidateMeta(entry.Manifest.Header()); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
	}

	// Vendored content must carry complete attribution (§3) so the catalog records
	// upstream provenance and the build packs a NOTICE.
	for _, name := range []string{
		"agents-spine", "ddd-distilled", "refactoring-distilled", "diagram-explain",
		"skills-dispatch", "plan-writing", "executing-plans",
		"plan-execute", // authored, but derived from superpowers: attribution + NOTICE required
		"grilling", "diagnosing-bugs", "tdd",
		"codebase-design", "domain-modeling",
		"go-style-uber",
		"tdd-guard-hook", "verification-before-completion",
		"git-guardrails",           // block-secrets + gitleaks-guard are authored (no attribution)
		"skills-dispatch-activate", // ccusage-statusline is authored (no attribution)
		// L10 orchestration: ticket (authored-but-attributed instruction) + 2 vendored superpowers skills.
		"ticket", "subagent-driven-development", "dispatching-parallel-agents",
		// The remaining vendored superpowers workflow skills.
		"spec-brainstorming", "using-git-worktrees", "finishing-a-development-branch",
		"writing-skills", "requesting-code-review", "receiving-code-review",
		// Vendored ai-memory lifecycle hooks.
		"ai-memory-session-start", "ai-memory-user-prompt", "ai-memory-pre-tool-use",
		"ai-memory-post-tool-use", "ai-memory-pre-compact", "ai-memory-stop",
	} {
		var found *manifest.Artifact
		for i := range cat.Artifacts {
			if cat.Artifacts[i].Manifest.Name == name {
				found = cat.Artifacts[i].Manifest
			}
		}
		if found == nil {
			t.Errorf("vendored artifact %q not in catalog", name)
			continue
		}
		at := found.Attribution
		if at == nil || at.Upstream == "" || at.License == "" || at.Copyright == "" {
			t.Errorf("%s: incomplete attribution: %+v", name, at)
		}
	}

	for _, entry := range cat.Recipes {
		if err := cp00ValidateRecipe(entry.Manifest); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
	}
	// Keep this substantive regression: without the MCP extra the installed
	// graphify server fails to import mcp, even though the manifest is valid.
	var graphify *manifest.Recipe
	for _, entry := range cat.Recipes {
		if entry.Manifest.Name == "graphify" {
			graphify = entry.Manifest
		}
	}
	if graphify == nil || graphify.Delivery == nil || len(graphify.Delivery.Install) == 0 {
		t.Error("graphify: missing install candidate")
	} else if cmd := graphify.Delivery.Install[0].InstallCommand(graphify.Name); !strings.Contains(cmd, "graphifyy[mcp]") {
		t.Errorf("graphify install command %q must contain graphifyy[mcp]", cmd)
	}

	// §6b.4 invariant: the catalog must carry at least one recipe of EVERY shape,
	// so the acceptance suite always has a real example of each delivery×wire path
	// to install. If a future change drops the last recipe of a shape (e.g. removes
	// every fetch+wire recipe), this fails — flagging that the corresponding deploy
	// proof in the P7.7 suite no longer has anything to exercise.
	shapeSeen := map[manifest.RecipeShape]bool{}
	for _, e := range cat.Recipes {
		shapeSeen[e.Manifest.Shape()] = true
	}
	for _, sh := range []manifest.RecipeShape{
		manifest.ShapeWireOnly, manifest.ShapeFetchWire,
		manifest.ShapeFetchRun, manifest.ShapeInstall,
	} {
		if !shapeSeen[sh] {
			t.Errorf("no recipe of shape %q in the catalog (§6b.4 wants ≥1 of each shape)", sh)
		}
	}

	for _, entry := range cat.Profiles {
		if err := cp00ValidateMeta(entry.Manifest.Header()); err != nil {
			t.Errorf("%s: %v", entry.Manifest.Name, err)
		}
	}
}

// TestRealAdaptersLoad discovers every shipped adapter, including newly added
// tools. Unsupported surfaces may be omitted; declared surfaces must be usable.
func TestRealAdaptersLoad(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(repoRoot(t), "adapters", "*.yaml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("discover adapters: paths=%v, err=%v", paths, err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			ad, err := manifest.LoadAdapter(path)
			if err != nil {
				t.Fatal(err)
			}
			if ad.Tool != strings.TrimSuffix(filepath.Base(path), ".yaml") {
				t.Errorf("adapter tool %q disagrees with filename", ad.Tool)
			}
			if err := cp00ValidateAdapter(ad); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestSkillsHeartbeatScriptCarriesLoadBearingText asserts the real
// skills-heartbeat hook script — read THROUGH the catalog, not via a relative
// path — still emits the two things that are its entire purpose: the 1% skill-
// dispatch rule it re-injects on every turn, and the enumeration of the installed
// skills directory. A hook whose behavior (emit JSON) is proven on a fixture does
// NOT prove the real artifact's TEXT; if a refactor drops the "1% chance" wording
// or stops reading ~/.claude/skills, this fails loudly instead of shipping a hook
// that no longer does its job.
func TestSkillsHeartbeatScriptCarriesLoadBearingText(t *testing.T) {
	root := repoRoot(t)
	reg := NewLocalRegistry(root)
	cat, err := reg.Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	var entry *ArtifactEntry
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest.Name == "skills-heartbeat" {
			entry = &cat.Artifacts[i]
			break
		}
	}
	if entry == nil {
		t.Fatal("skills-heartbeat artifact not found in the catalog")
	}
	if entry.Manifest.Hook == nil || entry.Manifest.Hook.Script == "" {
		t.Fatal("skills-heartbeat has no hook.script — cannot locate its bundled script")
	}

	scriptPath := filepath.Join(entry.Source.LocalDir, entry.Manifest.Hook.Script)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("reading hook script via catalog (%s): %v", scriptPath, err)
	}
	script := string(data)

	// The load-bearing pieces, each with what breaks if it goes.
	wants := []struct {
		substr, why string
	}{
		{
			"if any installed skill might apply (even a 1% chance)",
			"the 1% skill-dispatch rule is the hook's entire purpose",
		},
		{
			"${HOME}/.claude/skills",
			"the hook must enumerate the installed skills directory",
		},
	}
	for _, w := range wants {
		if !strings.Contains(script, w.substr) {
			t.Errorf("skills-heartbeat.sh no longer contains %q — %s", w.substr, w.why)
		}
	}
}

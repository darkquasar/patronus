package registry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

func agentFrontmatterList(t *testing.T, entry ArtifactEntry, field string) []string {
	t.Helper()
	path := filepath.Join(entry.Source.LocalDir, entry.Manifest.Entry)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	prefix := field + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		var values []string
		for _, value := range strings.Split(strings.TrimSpace(strings.TrimPrefix(line, prefix)), ",") {
			if value = strings.TrimSpace(value); value != "" {
				values = append(values, value)
			}
		}
		return values
	}
	t.Fatalf("%s: missing %s frontmatter", path, field)
	return nil
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

// TestCodeIntelPiProfileDeliversSharedRuntime verifies the two-profile split and
// the complete field-level overlays on every inherited core role.
func TestCodeIntelPiProfileDeliversSharedRuntime(t *testing.T) {
	cat, err := NewLocalRegistry(repoRoot(t)).Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	var profile, runtimeProfile *manifest.Profile
	for _, entry := range cat.Profiles {
		switch entry.Manifest.Name {
		case "code-intel-pi":
			profile = entry.Manifest
		case "code-intel-pi-runtime":
			runtimeProfile = entry.Manifest
		}
	}
	if profile == nil || runtimeProfile == nil {
		t.Fatalf("code-intel Pi profiles missing: overlay=%t runtime=%t", profile != nil, runtimeProfile != nil)
	}
	if profile.Extends != "core-profile-pi" {
		t.Fatalf("code-intel-pi extends %q, want core-profile-pi", profile.Extends)
	}
	if !slices.Contains(profile.Layers.Tools, "pi-mcp-adapter") {
		t.Error("code-intel-pi tools missing pi-mcp-adapter")
	}
	for _, name := range []string{"serena-shared-pi", "graphify-shared-pi"} {
		if !slices.Contains(profile.Layers.Context, name) {
			t.Errorf("code-intel-pi context missing %q", name)
		}
	}
	for _, name := range []string{"serena-runtime-pi", "graphify-runtime-pi"} {
		if !slices.Contains(runtimeProfile.Layers.Context, name) {
			t.Errorf("code-intel-pi-runtime context missing %q", name)
		}
		if slices.Contains(profile.Layers.Context, name) {
			t.Errorf("code-intel-pi must keep provisioning recipe %q out of the static Pi selection", name)
		}
	}

	recipes := make(map[string]*manifest.Recipe, len(cat.Recipes))
	for _, entry := range cat.Recipes {
		recipes[entry.Manifest.Name] = entry.Manifest
	}
	wantInstall := map[string]struct {
		manager manifest.PackageManager
		ref     string
		binary  string
	}{
		"pi-mcp-adapter":      {manager: manifest.PMPi, ref: "npm:pi-mcp-adapter@3.0.0"},
		"serena-runtime-pi":   {manager: manifest.PMUv, ref: "git+https://github.com/oraios/serena@7a2968335f2198b966864de1ce3655c8e485a653", binary: "serena"},
		"graphify-runtime-pi": {manager: manifest.PMUv, ref: "graphifyy[mcp]==0.9.31", binary: "graphify-mcp"},
	}
	for name, want := range wantInstall {
		recipe := recipes[name]
		if recipe == nil || recipe.Delivery == nil || len(recipe.Delivery.Install) != 1 {
			t.Errorf("%s: want exactly one install candidate", name)
			continue
		}
		got := recipe.Delivery.Install[0]
		if got.Manager != want.manager || got.Ref != want.ref {
			t.Errorf("%s install = (%s, %q), want (%s, %q)", name, got.Manager, got.Ref, want.manager, want.ref)
		}
		if recipe.Delivery.Binary != want.binary {
			t.Errorf("%s binary = %q, want %q", name, recipe.Delivery.Binary, want.binary)
		}
	}

	artifacts := make(map[string]*manifest.Artifact, len(cat.Artifacts))
	artifactEntries := make(map[string]ArtifactEntry, len(cat.Artifacts))
	for _, entry := range cat.Artifacts {
		artifacts[entry.Manifest.Name] = entry.Manifest
		artifactEntries[entry.Manifest.Name] = entry
	}
	roles := map[string]string{
		"plan-author":                "patronus-plan-author-pi",
		"plan-reviewer":              "patronus-plan-reviewer-pi",
		"researcher":                 "patronus-researcher-pi",
		"spec-author":                "patronus-spec-author-pi",
		"technical-reviewer":         "patronus-technical-reviewer-pi",
		"web-researcher":             "patronus-web-researcher-pi",
		"workflow-security-reviewer": "patronus-workflow-security-reviewer-pi",
		"writer":                     "patronus-writer-pi",
	}
	for short, role := range roles {
		for _, field := range []string{"tools", "skills"} {
			name := "code-intel-pi-" + short + "-" + field
			if !slices.Contains(profile.Layers.Orchestration, name) {
				t.Errorf("code-intel-pi orchestration missing %q", name)
			}
			artifact := artifacts[name]
			if artifact == nil || artifact.Setting == nil {
				t.Errorf("%s: missing setting artifact", name)
				continue
			}
			wantPath := "subagents.agentOverrides." + role + "." + field
			if artifact.Setting.Path != wantPath {
				t.Errorf("%s path = %q, want %q", name, artifact.Setting.Path, wantPath)
			}
			value, err := json.Marshal(artifact.Setting.Value)
			if err != nil {
				t.Errorf("%s value: %v", name, err)
				continue
			}
			var got []string
			if err := json.Unmarshal(value, &got); err != nil {
				t.Errorf("%s string list: %v", name, err)
				continue
			}
			want := agentFrontmatterList(t, artifactEntries[role], field)
			if field == "tools" {
				want = append(want, "mcp")
			} else {
				want = append(want, "pattern-mcp-pi", "graphify-pi", "code-intel-operations-pi")
			}
			if !slices.Equal(got, want) {
				t.Errorf("%s value = %q, want complete core list plus overlay %q", name, got, want)
			}
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

package registry

import (
	"context"
	"os"
	"path/filepath"
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
// the §6 mapping table. If a future change desyncs a manifest from the schema —
// a bad enum value, a renamed field, a recipe whose deliver×wire no longer
// computes the documented shape — this fails loudly instead of shipping broken.
func TestRealCatalogLoadsAndMatchesOntology(t *testing.T) {
	root := repoRoot(t)
	reg := NewLocalRegistry(root)
	cat, err := reg.Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	// --- Artifacts: family=artifact, declared type, declared role (§6). -------
	wantArtifacts := map[string]struct {
		typ  manifest.ArtifactType
		role manifest.Role
	}{
		"agent-principles":   {manifest.TypeInstruction, manifest.RoleInstruction},
		"team-research":      {manifest.TypeSkill, manifest.RoleCapability},
		"team-implement":     {manifest.TypeSkill, manifest.RoleCapability},
		"pattern-cloudflare": {manifest.TypeSkill, manifest.RoleContext}, // was role: pattern
		"pattern-mcp":        {manifest.TypeSkill, manifest.RoleContext},
		// P7.2-L1 vendored/authored instructions + the diagram-explain output-style.
		"agents-spine":    {manifest.TypeInstruction, manifest.RoleInstruction},
		"agent-rules":     {manifest.TypeInstruction, manifest.RoleInstruction},
		"diagram-explain": {manifest.TypeOutputStyle, manifest.RoleInstruction},
		// P7.2-L2 vendored capability skills (superpowers + mattpocock subset).
		"skills-dispatch": {manifest.TypeSkill, manifest.RoleCapability},
		"writing-plans":   {manifest.TypeSkill, manifest.RoleCapability},
		"executing-plans": {manifest.TypeSkill, manifest.RoleCapability},
		"grilling":        {manifest.TypeSkill, manifest.RoleCapability},
		"diagnosing-bugs": {manifest.TypeSkill, manifest.RoleCapability},
		"tdd":             {manifest.TypeSkill, manifest.RoleCapability},
		// P7.2-L4 vendored context/design-vocabulary skills (mattpocock).
		"codebase-design": {manifest.TypeSkill, manifest.RoleContext},
		"domain-modeling": {manifest.TypeSkill, manifest.RoleContext},
		// P7.3 distilled Go-idiomatic instruction (Uber Go Style Guide) — golang profile.
		"go-style-uber": {manifest.TypeInstruction, manifest.RoleInstruction},
		// P7.5.2 L8 eval: the test-first ENFORCEMENT hook + the verification skill (core's strict gate).
		"tdd-guard-hook":                 {manifest.TypeHook, manifest.RoleEval},
		"verification-before-completion": {manifest.TypeSkill, manifest.RoleEval},
		// P7.5.3 L9 guardrails: the default guard set (all type:hook).
		"git-guardrails": {manifest.TypeHook, manifest.RoleGuardrail},
		"block-secrets":  {manifest.TypeHook, manifest.RoleGuardrail},
		"gitleaks-guard": {manifest.TypeHook, manifest.RoleGuardrail},
		// P7.5.4: the keystone's SessionStart activation (L2) + the ccusage statusline setting (L7).
		"skills-dispatch-activate": {manifest.TypeHook, manifest.RoleCapability},
		"ccusage-statusline":       {manifest.TypeSetting, manifest.RoleObservability},
		// P7.5.5 L6 sandbox: the native-sandbox toggle (type:setting, flavoured @claude/@codex).
		"native-sandbox": {manifest.TypeSetting, manifest.RoleSandbox},
		// P7.6 L10 orchestration: the beads work-graph instruction (requires: [bd]) + 2 vendored superpowers skills.
		"beads":                       {manifest.TypeInstruction, manifest.RoleOrchestration},
		"subagent-driven-development": {manifest.TypeSkill, manifest.RoleOrchestration},
		"dispatching-parallel-agents": {manifest.TypeSkill, manifest.RoleOrchestration},
		// Remaining superpowers workflow skills (complete the vendored set).
		"brainstorming":                  {manifest.TypeSkill, manifest.RoleCapability},
		"using-git-worktrees":            {manifest.TypeSkill, manifest.RoleCapability},
		"finishing-a-development-branch": {manifest.TypeSkill, manifest.RoleCapability},
		"writing-skills":                 {manifest.TypeSkill, manifest.RoleCapability},
		"requesting-code-review":         {manifest.TypeSkill, manifest.RoleEval},
		"receiving-code-review":          {manifest.TypeSkill, manifest.RoleEval},
	}
	if len(cat.Artifacts) != len(wantArtifacts) {
		t.Errorf("artifact count = %d, want %d (did the catalog gain/lose an item without updating this guard?)",
			len(cat.Artifacts), len(wantArtifacts))
	}
	for _, e := range cat.Artifacts {
		m := e.Manifest
		want, ok := wantArtifacts[m.Name]
		if !ok {
			t.Errorf("unexpected artifact %q (add it to the ontology guard)", m.Name)
			continue
		}
		if m.Family != manifest.FamilyArtifact {
			t.Errorf("%s: family = %q, want artifact", m.Name, m.Family)
		}
		if m.Type != want.typ {
			t.Errorf("%s: type = %q, want %q", m.Name, m.Type, want.typ)
		}
		if m.Role != want.role {
			t.Errorf("%s: role = %q, want %q", m.Name, m.Role, want.role)
		}
		if m.APIVersion != manifest.APIVersion {
			t.Errorf("%s: apiVersion = %q, want %q", m.Name, m.APIVersion, manifest.APIVersion)
		}
	}

	// Vendored content must carry complete attribution (§3) so the catalog records
	// upstream provenance and the build packs a NOTICE.
	for _, name := range []string{
		"agents-spine", "agent-rules", "diagram-explain",
		"skills-dispatch", "writing-plans", "executing-plans",
		"grilling", "diagnosing-bugs", "tdd",
		"codebase-design", "domain-modeling",
		"go-style-uber",
		"tdd-guard-hook", "verification-before-completion",
		"git-guardrails",           // block-secrets + gitleaks-guard are authored (no attribution)
		"skills-dispatch-activate", // ccusage-statusline is authored (no attribution)
		// P7.6 orchestration: beads (authored-but-attributed instruction) + 2 vendored superpowers skills.
		"beads", "subagent-driven-development", "dispatching-parallel-agents",
		// The remaining vendored superpowers workflow skills.
		"brainstorming", "using-git-worktrees", "finishing-a-development-branch",
		"writing-skills", "requesting-code-review", "receiving-code-review",
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

	// --- Recipes: family=recipe, declared role, COMPUTED Shape (§6). ----------
	wantRecipes := map[string]struct {
		role  manifest.Role
		shape manifest.RecipeShape
		mode  manifest.WireMode
	}{
		"github":           {manifest.RoleTools, manifest.ShapeWireOnly, manifest.WireModeMcp},
		"memory-engram":    {manifest.RoleMemory, manifest.ShapeFetchWire, manifest.WireModeMcp},
		"memory-ai-memory": {manifest.RoleMemory, manifest.ShapeFetchRun, manifest.WireModeSelf},
		"sandbox":          {manifest.RoleSandbox, manifest.ShapeFetchWire, manifest.WireModeMcp},
		// P7.3 L4 context recipes (live docs + local semantic search) — wire-only MCP.
		"context7": {manifest.RoleContext, manifest.ShapeWireOnly, manifest.WireModeMcp},
		"serena":   {manifest.RoleContext, manifest.ShapeWireOnly, manifest.WireModeMcp},
		// P7.3 L5 tool recipes (opt-in) — all wire-only MCP (npx/uvx on demand, or hosted).
		"playwright":     {manifest.RoleTools, manifest.ShapeWireOnly, manifest.WireModeMcp},
		"postgres":       {manifest.RoleTools, manifest.ShapeWireOnly, manifest.WireModeMcp},
		"cloudflare-mcp": {manifest.RoleTools, manifest.ShapeWireOnly, manifest.WireModeMcp},
		// P7.5.2 L8 eval: install-only recipe (deliver: npm) for the tdd-guard CLI; no wire.
		"tdd-guard": {manifest.RoleEval, manifest.ShapeInstall, manifest.WireMode("")},
		// P7.5.3 L9 guardrails: install-only recipe (github-release fetch) for the gitleaks binary; no wire.
		"gitleaks": {manifest.RoleGuardrail, manifest.ShapeInstall, manifest.WireMode("")},
		// P7.5.4 L7 observability: install-only recipe (deliver: npm) for the ccusage CLI; no wire.
		"ccusage": {manifest.RoleObservability, manifest.ShapeInstall, manifest.WireMode("")},
		// P7.5.5 L6 sandbox: srt (install-only npm, @opencode) + microsandbox (wire-only MCP, hard-isolation).
		"sandbox-runtime": {manifest.RoleSandbox, manifest.ShapeInstall, manifest.WireMode("")},
		"microsandbox":    {manifest.RoleSandbox, manifest.ShapeWireOnly, manifest.WireModeMcp},
		// P7.5.6 L8 eval: promptfoo CI gate (install-only npm) — the eval profile.
		"promptfoo": {manifest.RoleEval, manifest.ShapeInstall, manifest.WireMode("")},
		// P7.6 L10 orchestration: the bd (Beads) work-graph binary — install-only github-release; the `beads` instruction (requires: [bd]) wires it.
		"bd": {manifest.RoleOrchestration, manifest.ShapeInstall, manifest.WireMode("")},
	}
	if len(cat.Recipes) != len(wantRecipes) {
		t.Errorf("recipe count = %d, want %d", len(cat.Recipes), len(wantRecipes))
	}
	for _, e := range cat.Recipes {
		m := e.Manifest
		want, ok := wantRecipes[m.Name]
		if !ok {
			t.Errorf("unexpected recipe %q (add it to the ontology guard)", m.Name)
			continue
		}
		if m.Family != manifest.FamilyRecipe {
			t.Errorf("%s: family = %q, want recipe", m.Name, m.Family)
		}
		if m.Role != want.role {
			t.Errorf("%s: role = %q, want %q", m.Name, m.Role, want.role)
		}
		if got := m.Shape(); got != want.shape {
			t.Errorf("%s: Shape() = %q, want %q", m.Name, got, want.shape)
		}
		if m.Wire.Mode != want.mode {
			t.Errorf("%s: wire.mode = %q, want %q", m.Name, m.Wire.Mode, want.mode)
		}
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

	// --- Profiles: family=profile, role=lifecycle (§6). -----------------------
	wantProfiles := []string{"cloudflare", "core", "data", "eval", "golang", "hard-isolation", "hardened", "lean-code", "no-tdd-guard", "python", "quiet", "terse", "visual", "web-dev"}
	if len(cat.Profiles) != len(wantProfiles) {
		t.Errorf("profile count = %d, want %d", len(cat.Profiles), len(wantProfiles))
	}
	for _, e := range cat.Profiles {
		m := e.Manifest
		if m.Family != manifest.FamilyProfile {
			t.Errorf("%s: family = %q, want profile", m.Name, m.Family)
		}
		if m.Role != manifest.RoleLifecycle {
			t.Errorf("%s: role = %q, want lifecycle", m.Name, m.Role)
		}
	}
}

// TestRealAdaptersLoad loads the three shipped adapters through LoadAdapter and
// asserts family=adapter plus a parsed layout keyed by the lowercase type axis
// (so the engine's type->layout identity lookup keeps working).
func TestRealAdaptersLoad(t *testing.T) {
	root := repoRoot(t)
	for _, tool := range []string{"claude", "codex", "opencode"} {
		ad, err := manifest.LoadAdapter(filepath.Join(root, "adapters", tool+".yaml"))
		if err != nil {
			t.Fatalf("load adapter %s: %v", tool, err)
		}
		if ad.Family != manifest.FamilyAdapter {
			t.Errorf("%s: family = %q, want adapter", tool, ad.Family)
		}
		if ad.Tool != tool {
			t.Errorf("%s: tool = %q", tool, ad.Tool)
		}
		// Every tool must declare the artifact-type layouts the engine dispatches
		// to (skill/agent/command/instruction); mcp is the recipe MERGE primitive.
		if ad.Layout.Skill == nil {
			t.Errorf("%s: missing skill layout", tool)
		}
		if ad.Layout.Instruction == nil {
			t.Errorf("%s: missing instruction layout", tool)
		}
		if ad.Layout.OutputStyle == nil {
			t.Errorf("%s: missing output-style layout", tool)
		}
		if ad.Layout.Mcp == nil {
			t.Errorf("%s: missing mcp layout", tool)
		}
	}
}

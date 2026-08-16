package registry

import (
	"context"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// TestPlanExecuteManifest pins the shape of the plan-execute router artifact:
// the requires edge that pulls the reviewer skill its whole-branch review needs,
// the three targets, and the sidecar files that must be packed alongside the
// entry. The gate's PROSE is not testable here; its structural contract is.
func TestPlanExecuteManifest(t *testing.T) {
	root := repoRoot(t)
	reg := NewLocalRegistry(root)
	cat, err := reg.Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	var found *manifest.Artifact
	for _, e := range cat.Artifacts {
		if e.Manifest.Name == "plan-execute" {
			found = e.Manifest
		}
	}
	if found == nil {
		t.Fatal("plan-execute not in the catalog")
	}

	if found.Type != manifest.TypeSkill {
		t.Errorf("type = %q, want skill", found.Type)
	}
	if found.Role != manifest.RoleCapability {
		t.Errorf("role = %q, want capability", found.Role)
	}
	if found.Entry != "SKILL.md" {
		t.Errorf("entry = %q, want SKILL.md", found.Entry)
	}

	wantRequires := map[string]bool{"requesting-code-review": true}
	for _, r := range found.Requires {
		delete(wantRequires, r)
	}
	if len(wantRequires) != 0 {
		t.Errorf("requires missing %v (got %v)", wantRequires, found.Requires)
	}

	wantFiles := map[string]bool{
		"solo.md": true, "sdd.md": true,
		"implementer-prompt.md": true, "task-reviewer-prompt.md": true,
		"scripts":  true,
		"fixtures": true,
	}
	for _, f := range found.Files {
		delete(wantFiles, f)
	}
	if len(wantFiles) != 0 {
		t.Errorf("files missing %v (got %v)", wantFiles, found.Files)
	}

	wantTargets := map[string]bool{"claude": true, "codex": true, "opencode": true}
	for _, tg := range found.Targets {
		delete(wantTargets, tg)
	}
	if len(wantTargets) != 0 {
		t.Errorf("targets missing %v (got %v)", wantTargets, found.Targets)
	}
}

// TestSupersededSkillsPointAtTheRouter guards the only mitigation available for
// trigger competition today. Catalog metadata that could express this
// structurally (supersededBy, recommended: false) is a manifest-schema change
// and does not exist, so the description field carries it: each original names
// plan-execute as the default and says what it is FOR instead.
func TestSupersededSkillsPointAtTheRouter(t *testing.T) {
	root := repoRoot(t)
	reg := NewLocalRegistry(root)
	cat, err := reg.Catalog(context.Background())
	if err != nil {
		t.Fatalf("loading real catalog: %v", err)
	}

	byName := map[string]*manifest.Artifact{}
	for _, e := range cat.Artifacts {
		byName[e.Manifest.Name] = e.Manifest
	}

	for _, name := range []string{"executing-plans", "subagent-driven-development"} {
		a, ok := byName[name]
		if !ok {
			t.Errorf("%s must stay in the catalog, installable and opt-in", name)
			continue
		}
		if !strings.Contains(a.Description, "plan-execute") {
			t.Errorf("%s: description must name plan-execute as the default:\n%s", name, a.Description)
		}
	}
}

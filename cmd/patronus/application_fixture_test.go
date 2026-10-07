package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
)

// fixtureApplicationCatalog loads only invented catalog data. Actual adapters
// remain application contracts, including on copies without artifacts/profiles.
func fixtureApplicationCatalog(t *testing.T) *registry.Catalog {
	t.Helper()
	cat, err := registry.NewLocalRegistry(fixtureCatalog(t)).Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// fixtureSkillBundle adds inert entry/sidecar bytes and a dependency to the
// existing fixture, exercising delivery without importing a shipped skill.
func fixtureSkillBundle(t *testing.T) string {
	t.Helper()
	root := fixtureCatalog(t)
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("artifacts/skills/fix-router/patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fix-router\ndescription: Invented route fixture\nversion: 1.0.0\nentry: SKILL.md\nfiles: [mode.md, scripts/helper, NOTICE]\ntargets: [claude, codex, opencode]\nrequires: [fix-review]\n")
	write("artifacts/skills/fix-router/SKILL.md", "---\nname: fix-router\ndescription: Invented router\n---\nRead `{skillDir}/mode.md` and `{skillsDir}/fix-review/SKILL.md`.\n")
	write("artifacts/skills/fix-router/mode.md", "Use `{skillDir}/scripts/helper`.\n")
	write("artifacts/skills/fix-router/scripts/helper", "Inert helper bytes, never executed.\n")
	write("artifacts/skills/fix-router/NOTICE", "Invented provenance bytes.\n")
	write("artifacts/skills/fix-review/patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fix-review\ndescription: Invented dependency\nversion: 1.0.0\nentry: SKILL.md\nfiles: [review.md]\ntargets: [claude, codex, opencode]\n")
	write("artifacts/skills/fix-review/SKILL.md", "---\nname: fix-review\ndescription: Invented companion\n---\nRead `{skillDir}/review.md`.\n")
	write("artifacts/skills/fix-review/review.md", "Invented review template.\n")
	write("profiles/fix-routing.yaml", "apiVersion: patronus/v2\nfamily: profile\nname: fix-routing\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fix-router]\n")
	return root
}

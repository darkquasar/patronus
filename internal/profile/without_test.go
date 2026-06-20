package profile

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// prof is a small builder for a profile manifest in these tests.
func prof(name, extends string, without manifest.StringList, layers manifest.ProfileLayers) *manifest.Profile {
	return &manifest.Profile{
		Meta:    manifest.Meta{Family: manifest.FamilyProfile, Name: name, Role: manifest.RoleLifecycle},
		Extends: extends,
		Without: without,
		Layers:  layers,
	}
}

// `without` subtracts an item from the composed (extends-merged) layers, leaving
// every sibling intact — the relaxation-overlay operator.
func TestWithoutSubtractsFromComposedLayers(t *testing.T) {
	base := prof("base", "", nil, manifest.ProfileLayers{
		Eval:         manifest.StringList{"tdd-guard", "tdd-guard-hook", "verification"},
		Capabilities: manifest.StringList{"tdd"},
	})
	overlay := prof("relaxed", "base", manifest.StringList{"tdd-guard", "tdd-guard-hook"}, manifest.ProfileLayers{})
	cat := fakeCatalog(
		[]string{"tdd-guard-hook", "verification", "tdd"},
		[]string{"tdd-guard"},
		base, overlay,
	)

	r, err := Resolve(cat, "relaxed", "all")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(r.Names(), ",")
	// The enforcement (recipe + hook) is gone; the tdd SKILL and verification stay.
	for _, gone := range []string{"tdd-guard", "tdd-guard-hook"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q should have been subtracted; items = %s", gone, got)
		}
	}
	for _, kept := range []string{"tdd", "verification"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q should survive the overlay; items = %s", kept, got)
		}
	}
}

// `without` matches on the BASE name, so it strips a bare name and all its
// `@tool` flavours alike (you subtract the item, not one tool's flavour).
func TestWithoutStripsFlavours(t *testing.T) {
	base := prof("base", "", nil, manifest.ProfileLayers{
		Sandbox: manifest.StringList{"native-sandbox@claude", "native-sandbox@codex"},
		Tools:   manifest.StringList{"github"},
	})
	overlay := prof("nosandbox", "base", manifest.StringList{"native-sandbox"}, manifest.ProfileLayers{})
	cat := fakeCatalog([]string{"github"}, []string{"native-sandbox"}, base, overlay)

	r, err := Resolve(cat, "nosandbox", "claude")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(r.Names(), ",")
	if strings.Contains(got, "native-sandbox") {
		t.Errorf("base-name without should strip every flavour; items = %s", got)
	}
	if !strings.Contains(got, "github") {
		t.Errorf("github should survive; items = %s", got)
	}
}

// A `without` entry that matches nothing is a no-op warning (so a stale exclusion
// surfaces as the overlay drifts from its parent, rather than failing silently).
func TestWithoutUnmatchedWarns(t *testing.T) {
	base := prof("base", "", nil, manifest.ProfileLayers{Tools: manifest.StringList{"github"}})
	overlay := prof("o", "base", manifest.StringList{"does-not-exist"}, manifest.ProfileLayers{})
	cat := fakeCatalog([]string{"github"}, nil, base, overlay)

	r, err := Resolve(cat, "o", "all")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "without") && strings.Contains(w, "does-not-exist") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an unmatched-without warning, got %v", r.Warnings)
	}
	// github still resolves despite the stale exclusion.
	if len(r.Items) != 1 || r.Items[0].Name != "github" {
		t.Errorf("github should still resolve; items = %+v", r.Items)
	}
}

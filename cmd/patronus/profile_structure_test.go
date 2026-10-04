package main

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

// These are STRUCTURAL property tests over the whole profile set. They assert a
// Patronus invariant — "a profile a user opts into resolves and is not silently
// hollow" — as a general property, NOT by enumerating which items a given profile
// ships. Which artifacts populate `core` or `code-intel` is CATALOG CONTENT: it is
// decided in the profile YAML and reviewed in the PR that changes it, the way npm
// does not unit-test that a given package is in its registry. What Patronus code
// must guarantee is that resolution WORKS and leaves no declared layer empty —
// which holds for every profile at once, with zero per-item test maintenance when
// the catalog changes. (Supersedes the per-profile name-mirroring tests; see
// .github/decisions/0002 and tasks/lessons.md L5.)

// declaredLayers reports the identities the YAML names in each §1A layer.
// Memory is a scalar; the rest are lists.
func declaredLayers(p *manifest.Profile) map[string][]string {
	l := p.Layers
	declared := map[string][]string{
		"instructions":  l.Instructions,
		"capabilities":  l.Capabilities,
		"context":       l.Context,
		"tools":         l.Tools,
		"sandbox":       l.Sandbox,
		"observability": l.Observability,
		"eval":          l.Eval,
		"guardrails":    l.Guardrails,
		"orchestration": l.Orchestration,
	}
	if l.Memory != "" {
		declared["memory"] = []string{l.Memory}
	}
	return declared
}

// TestEveryProfileResolves is the base structural guarantee: every profile in the
// real catalog resolves without error for every tool. A dangling item name, a
// broken extends:, or an unresolved slot surfaces here — for ALL profiles — instead
// of only where a hand-written per-profile test happened to look.
func TestEveryProfileResolves(t *testing.T) {
	cat := realCatalog(t)
	for _, pe := range cat.Profiles {
		name := pe.Manifest.Name
		for _, tool := range []string{"claude", "codex", "opencode", "pi", "all"} {
			if _, err := profile.Resolve(cat, name, tool); err != nil {
				t.Errorf("profile %q does not resolve for tool %q: %v", name, tool, err)
			}
		}
	}
}

// TestNoProfileLayerResolvesEmpty is the anti-hollow guarantee: when a profile
// DECLARES items in a layer, that layer must resolve to at least one item for at
// least one target tool. This catches the real regression the old per-profile name
// tests guarded against — a profile silently losing the tooling it promises — as a
// PROPERTY, without naming any specific item.
//
// "For at least one tool" is deliberate: a layer may be entirely @tool-flavoured
// (e.g. a claude-only statusline), which resolves empty under the other tools by
// design. Bare declarations may be reached anywhere; flavoured declarations
// must be reached for their named target, not rescued by another target.
func TestNoProfileLayerResolvesEmpty(t *testing.T) {
	cat := realCatalog(t)
	for _, pe := range cat.Profiles {
		name := pe.Manifest.Name
		if pe.Manifest.Status == "stub" {
			continue // a stub profile declares intent, not yet items
		}
		declared := declaredLayers(pe.Manifest)

		resolved := map[string][]profile.ResolvedItem{}
		for _, tool := range []string{"claude", "codex", "opencode", "pi"} {
			r, err := profile.Resolve(cat, name, tool)
			if err != nil {
				t.Errorf("profile %q: resolve for %q: %v", name, tool, err)
				continue
			}
			resolved[tool] = r.Items
		}

		for slot, names := range declared {
			if len(names) > 0 && !cp03LayerReachable(names, resolved) {
				t.Errorf("profile %q declares %d item(s) in layer %q but it resolves to none for any tool — a hollow layer",
					name, len(names), slot)
			}
		}
	}
}

func TestProfileDeclaredLayerDependencyDedup(t *testing.T) {
	p := &manifest.Profile{
		Meta: manifest.Meta{Name: "invented-bundle"},
		Layers: manifest.ProfileLayers{
			Instructions: []string{"invented-pointer"},
			Tools:        []string{"invented-helper"},
		},
	}
	cat := &registry.Catalog{
		Profiles: []registry.ProfileEntry{{Manifest: p}},
		Artifacts: []registry.ArtifactEntry{
			{Manifest: &manifest.Artifact{Meta: manifest.Meta{Name: "invented-pointer", Requires: []string{"invented-helper"}}}},
			{Manifest: &manifest.Artifact{Meta: manifest.Meta{Name: "invented-helper"}}},
		},
	}
	r, err := profile.Resolve(cat, p.Name, "pi")
	if err != nil {
		t.Fatal(err)
	}
	resolved := map[string][]profile.ResolvedItem{"pi": r.Items}
	if !cp03LayerReachable(declaredLayers(p)["tools"], resolved) {
		t.Fatal("dependency-first resolution lost the explicitly declared tools layer")
	}
	p.Layers.Tools = []string{"missing-helper"}
	r, err = profile.Resolve(cat, p.Name, "pi")
	if err != nil {
		t.Fatal(err)
	}
	resolved["pi"] = r.Items
	if cp03LayerReachable(declaredLayers(p)["tools"], resolved) {
		t.Fatal("unrelated resolved dependency concealed a missing declared layer")
	}
}

func TestProfileDeclaredLayerReachability(t *testing.T) {
	for _, tc := range []struct {
		name, declaration, target string
		want                      bool
	}{
		{"dependency-first", "invented-helper", "pi", true},
		{"missing-despite-dependency", "missing-helper", "pi", false},
		{"matching-flavour", "invented-helper@pi", "pi", true},
		{"other-target-only", "invented-helper@pi", "claude", false},
		{"unsupported-flavour", "invented-helper@future", "pi", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A dependency reached in instructions must still satisfy its later
			// declared tools layer, but only at a declaration's actual target.
			resolved := map[string][]profile.ResolvedItem{
				tc.target: {{Name: "invented-helper", Slot: "instructions"}},
			}
			if got := cp03LayerReachable([]string{tc.declaration}, resolved); got != tc.want {
				t.Fatalf("reachable = %v, want %v", got, tc.want)
			}
		})
	}
}

// Slot records first resolution provenance: a dependency can be deduplicated
// before its explicit layer is visited. Check declared identities, not that slot.
func cp03LayerReachable(declared []string, resolved map[string][]profile.ResolvedItem) bool {
	for _, declaration := range declared {
		name, target := declaration, ""
		if i := strings.LastIndexByte(declaration, '@'); i >= 0 {
			switch declaration[i+1:] {
			case "claude", "codex", "opencode", "pi":
				name, target = declaration[:i], declaration[i+1:]
			}
		}
		for tool, items := range resolved {
			if target != "" && target != tool {
				continue
			}
			for _, item := range items {
				if item.Name == name {
					return true
				}
			}
		}
	}
	return false
}

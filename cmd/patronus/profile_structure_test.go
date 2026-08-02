package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
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
// docs/adr/0002 and tasks/lessons.md L5.)

// declaredLayers reports, per profile, how many items the YAML names in each §1A
// layer slot. Memory is a scalar; the rest are lists.
func declaredLayers(p *manifest.Profile) map[string]int {
	l := p.Layers
	counts := map[string]int{
		"instructions":  len(l.Instructions),
		"capabilities":  len(l.Capabilities),
		"context":       len(l.Context),
		"tools":         len(l.Tools),
		"sandbox":       len(l.Sandbox),
		"observability": len(l.Observability),
		"eval":          len(l.Eval),
		"guardrails":    len(l.Guardrails),
		"orchestration": len(l.Orchestration),
	}
	if l.Memory != "" {
		counts["memory"] = 1
	}
	return counts
}

// TestEveryProfileResolves is the base structural guarantee: every profile in the
// real catalog resolves without error for every tool. A dangling item name, a
// broken extends:, or an unresolved slot surfaces here — for ALL profiles — instead
// of only where a hand-written per-profile test happened to look.
func TestEveryProfileResolves(t *testing.T) {
	cat := realCatalog(t)
	for _, pe := range cat.Profiles {
		name := pe.Manifest.Name
		for _, tool := range []string{"claude", "codex", "opencode", "all"} {
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
// design. Unioning the filled slots across every concrete tool asks only that the
// declared layer is reachable SOMEWHERE, which is exactly the hollow-layer check.
func TestNoProfileLayerResolvesEmpty(t *testing.T) {
	cat := realCatalog(t)
	for _, pe := range cat.Profiles {
		name := pe.Manifest.Name
		if pe.Manifest.Status == "stub" {
			continue // a stub profile declares intent, not yet items
		}
		declared := declaredLayers(pe.Manifest)

		filled := map[string]bool{}
		for _, tool := range []string{"claude", "codex", "opencode"} {
			r, err := profile.Resolve(cat, name, tool)
			if err != nil {
				t.Errorf("profile %q: resolve for %q: %v", name, tool, err)
				continue
			}
			for _, it := range r.Items {
				filled[it.Slot] = true
			}
		}

		for slot, n := range declared {
			if n > 0 && !filled[slot] {
				t.Errorf("profile %q declares %d item(s) in layer %q but it resolves to none for any tool — a hollow layer",
					name, n, slot)
			}
		}
	}
}

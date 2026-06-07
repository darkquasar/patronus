// Package profile resolves a named profile (a curated bundle across §1A layers,
// DESIGN §5d) into a flat, ordered list of install items. It is deliberately a
// *resolver*, not a *producer*: it reads only the in-memory catalog and returns
// the names to install (with provenance), which the install command then
// dispatches through the SAME artifact-vs-recipe path a plain `install <name>...`
// uses. That keeps the "one spine, two producers" invariant — profiles add no
// parallel install path, they just expand to a longer name list.
//
// Resolution handles `extends:` composition (append list slots, replace scalar
// slots), warns on a `status: stub` profile, and — per the Phase-5 decision —
// WARNS AND SKIPS any slot name absent from the catalog so a partly-sourced
// profile still plans its real items rather than hard-failing.
package profile

import (
	"fmt"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/source"
)

// ItemKind records whether a resolved name dispatches to the artifact path or the
// recipe path.
type ItemKind int

const (
	KindUnknown ItemKind = iota // not found in the catalog (becomes a warning, dropped)
	KindArtifact
	KindRecipe
)

func (k ItemKind) String() string {
	switch k {
	case KindArtifact:
		return "artifact"
	case KindRecipe:
		return "recipe"
	default:
		return "unknown"
	}
}

// ResolvedItem is one install target a profile expanded to.
type ResolvedItem struct {
	Name   string   // the catalog lookup key / install name
	Slot   string   // §1A layer slot it filled (informational)
	Kind   ItemKind // artifact | recipe
	Source string   // canonical provenance for the lock ("registry" for in-tree)
}

// Resolved is the full resolution of a profile against a catalog.
type Resolved struct {
	Profile  *manifest.Profile
	Items    []ResolvedItem // flat, ordered (by §1A layer, then author order), deduped
	Warnings []string       // stub status, unresolved names
}

// slotEntry is an intermediate (name, slot) pair from flattening the layers map.
type slotEntry struct {
	name string
	slot string
}

// Resolve expands a named profile into its install items. Pure: it reads only
// cat (no filesystem, no network), so it is trivially unit-testable with a fake
// catalog.
func Resolve(cat *registry.Catalog, name string) (*Resolved, error) {
	layers, prof, err := resolveLayers(cat, name, map[string]bool{})
	if err != nil {
		return nil, err
	}

	out := &Resolved{Profile: prof}
	if prof.Status == "stub" {
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("profile %q is a stub: layers marked TODO are not yet populated", name))
	}

	seen := map[string]bool{}
	for _, e := range flattenLayers(layers) {
		if seen[e.name] {
			continue // a name appearing in two slots installs once (first slot wins)
		}
		seen[e.name] = true

		ref, err := source.Parse(e.name)
		if err != nil {
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("profile %q slot %q: %v — skipped", name, e.slot, err))
			continue
		}

		kind := classify(cat, ref)
		if kind == KindUnknown {
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("profile %q slot %q: %q not found in catalog — skipped (not yet sourced?)", name, e.slot, e.name))
			continue
		}

		out.Items = append(out.Items, ResolvedItem{
			Name:   e.name,
			Slot:   e.slot,
			Kind:   kind,
			Source: ref.LockSource(),
		})
	}
	return out, nil
}

// resolveLayers returns the fully composed layers for a profile, applying
// `extends` parent-first. visiting guards against extends cycles. It also returns
// the leaf profile manifest (for status/name).
func resolveLayers(cat *registry.Catalog, name string, visiting map[string]bool) (manifest.ProfileLayers, *manifest.Profile, error) {
	if visiting[name] {
		return manifest.ProfileLayers{}, nil, fmt.Errorf("profile extends cycle detected at %q", name)
	}
	visiting[name] = true

	prof := findProfile(cat, name)
	if prof == nil {
		return manifest.ProfileLayers{}, nil, fmt.Errorf("unknown profile %q", name)
	}

	layers := prof.Layers
	if prof.Extends != "" {
		parent, _, err := resolveLayers(cat, prof.Extends, visiting)
		if err != nil {
			return manifest.ProfileLayers{}, nil, err
		}
		layers = mergeLayers(parent, prof.Layers)
	}
	return layers, prof, nil
}

// mergeLayers composes a child's layers onto its parent's: list slots APPEND
// (child onto parent, deduped, parent order first), scalar slots REPLACE when the
// child sets them (a scalar holds one value, so it can only be overridden). This
// makes `extends` genuinely extend list bundles while still letting a child swap
// a single-valued layer (e.g. a different memory recipe).
func mergeLayers(parent, child manifest.ProfileLayers) manifest.ProfileLayers {
	return manifest.ProfileLayers{
		Instructions:  appendDedup(parent.Instructions, child.Instructions),
		Capabilities:  appendDedup(parent.Capabilities, child.Capabilities),
		Context:       appendDedup(parent.Context, child.Context),
		Tools:         appendDedup(parent.Tools, child.Tools),
		Observability: appendDedup(parent.Observability, child.Observability),
		Harness:       appendDedup(parent.Harness, child.Harness),
		Guardrails:    appendDedup(parent.Guardrails, child.Guardrails),
		Memory:        replaceScalar(parent.Memory, child.Memory),
		Sandbox:       replaceScalar(parent.Sandbox, child.Sandbox),
	}
}

// appendDedup returns parent followed by the child entries not already present.
func appendDedup(parent, child manifest.StringList) manifest.StringList {
	out := append(manifest.StringList{}, parent...)
	have := map[string]bool{}
	for _, s := range parent {
		have[s] = true
	}
	for _, s := range child {
		if !have[s] {
			out = append(out, s)
			have[s] = true
		}
	}
	return out
}

// replaceScalar returns the child value if set, else the parent's.
func replaceScalar(parent, child string) string {
	if child != "" {
		return child
	}
	return parent
}

// flattenLayers walks the layers map in a FIXED §1A layer order and emits one
// entry per item in author order. The fixed order makes the resolved plan (and
// therefore the lock) deterministic regardless of YAML key order.
func flattenLayers(l manifest.ProfileLayers) []slotEntry {
	var out []slotEntry
	add := func(slot string, names manifest.StringList) {
		for _, n := range names {
			out = append(out, slotEntry{name: n, slot: slot})
		}
	}
	addScalar := func(slot, name string) {
		if name != "" {
			out = append(out, slotEntry{name: name, slot: slot})
		}
	}

	add("instructions", l.Instructions)
	add("capabilities", l.Capabilities)
	add("context", l.Context)
	add("tools", l.Tools)
	addScalar("memory", l.Memory)
	addScalar("sandbox", l.Sandbox)
	add("observability", l.Observability)
	add("harness", l.Harness)
	add("guardrails", l.Guardrails)
	return out
}

// classify dispatches a parsed reference to the artifact or recipe path. For a
// registry (bare-name) ref it looks the name up in the catalog; a non-registry
// source (file:) is assumed resolvable by the caller and classified by which
// manifest kind its location holds — but Phase 5 profiles use only bare names, so
// only the registry branch is exercised by shipped profiles. (file: items in a
// profile slot still resolve via the install-arg path; here a bare-name lookup
// covers the catalog cases.)
func classify(cat *registry.Catalog, ref *source.Ref) ItemKind {
	if ref.Scheme != source.Registry {
		// Non-registry sources are resolved+merged into the catalog by the install
		// command before dispatch; treat as unknown here only if not yet merged.
		if findRecipe(cat, ref.Raw) != nil {
			return KindRecipe
		}
		if findArtifact(cat, ref.Raw) != nil {
			return KindArtifact
		}
		return KindUnknown
	}
	if findRecipe(cat, ref.Name) != nil {
		return KindRecipe
	}
	if findArtifact(cat, ref.Name) != nil {
		return KindArtifact
	}
	return KindUnknown
}

// Names returns the resolved item names in order — the flat list the install
// dispatch consumes.
func (r *Resolved) Names() []string {
	out := make([]string, len(r.Items))
	for i, it := range r.Items {
		out[i] = it.Name
	}
	return out
}

func findProfile(cat *registry.Catalog, name string) *manifest.Profile {
	for i := range cat.Profiles {
		if cat.Profiles[i].Manifest.Name == name {
			return cat.Profiles[i].Manifest
		}
	}
	return nil
}

func findRecipe(cat *registry.Catalog, name string) *registry.RecipeEntry {
	for i := range cat.Recipes {
		if cat.Recipes[i].Manifest.Name == name {
			return &cat.Recipes[i]
		}
	}
	return nil
}

func findArtifact(cat *registry.Catalog, name string) *registry.ArtifactEntry {
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest.Name == name {
			return &cat.Artifacts[i]
		}
	}
	return nil
}

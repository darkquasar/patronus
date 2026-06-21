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
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/requires"
	"github.com/darkquasar/patronus/internal/source"
)

// familyUnknown is the not-found sentinel: classify returns it for a name absent
// from the catalog (which becomes a warning and is dropped). The real families
// live in manifest.Family (FamilyArtifact | FamilyRecipe).
const familyUnknown manifest.Family = ""

// ResolvedItem is one install target a profile expanded to.
type ResolvedItem struct {
	Name   string          // the catalog lookup key / install name
	Slot   string          // §1A layer slot it filled (informational)
	Family manifest.Family // artifact | recipe — the dispatch discriminator
	Source string          // canonical provenance for the lock ("registry" for in-tree)
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

// flavourTools is the closed set of target suffixes a `name@tool` item may carry.
// Anything else after an `@` is left as part of the base name (so it simply fails
// catalog lookup and falls into the existing warn-and-skip path, not silently
// dropped as a mistyped flavour).
var flavourTools = map[string]bool{"claude": true, "codex": true, "opencode": true}

// splitFlavour separates a slot item into its base name and optional `@tool`
// flavour. Only a trailing `@<tool>` where <tool> is a known tool counts; every
// other `@` is part of the base. "sandbox@opencode" → ("sandbox","opencode");
// "team-research" → ("team-research",""); "a@b.com" → ("a@b.com","").
func splitFlavour(item string) (base, flavour string) {
	i := strings.LastIndexByte(item, '@')
	if i < 0 {
		return item, ""
	}
	suffix := item[i+1:]
	if !flavourTools[suffix] {
		return item, ""
	}
	return item[:i], suffix
}

// Resolve expands a named profile into its install items for a target tool. Pure:
// it reads only cat (no filesystem, no network), so it is trivially unit-testable
// with a fake catalog.
//
// tool selects per-tool FLAVOURS (§4): a slot item may be a bare name (installed
// for every tool) or `name@tool` (installed only when its suffix matches). When
// tool is a concrete agent ("claude"|"codex"|"opencode"), bare names plus that
// tool's flavours resolve; when tool is "" or "all" (the tool-agnostic baseline
// `lock` uses by default), only bare names resolve and all `@tool` flavours drop.
// The base name is what the install path dispatches on; the `@tool` suffix never
// reaches the catalog or source.Parse.
func Resolve(cat *registry.Catalog, name, tool string) (*Resolved, error) {
	layers, prof, err := resolveLayers(cat, name, map[string]bool{})
	if err != nil {
		return nil, err
	}

	out := &Resolved{Profile: prof}
	if prof.Status == "stub" {
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("profile %q is a stub: layers marked TODO are not yet populated", name))
	}

	// `without` SUBTRACTS base names from the composed layers — the relaxation-
	// overlay operator (e.g. no-tdd-guard = core without the enforcement hook),
	// symmetric to the extends-append above. It matches on the BASE name, so it
	// strips a bare name and all its `@tool` flavours alike. An entry that matches
	// nothing is a no-op warning, so an overlay stays robust as `core` evolves.
	excluded, unmatched := excludeSet(prof.Without)
	for _, e := range flattenLayers(layers) {
		base, _ := splitFlavour(e.name)
		delete(unmatched, base)
	}
	for dropped := range unmatched {
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("profile %q: without %q matched nothing", name, dropped))
	}

	seen := map[string]bool{}
	for _, e := range flattenLayers(layers) {
		base, flavour := splitFlavour(e.name)
		if excluded[base] {
			continue // subtracted by `without`
		}
		if flavour != "" && flavour != tool {
			continue // a flavour for a different tool: silently not selected
		}
		if seen[base] {
			continue // dedup on the BASE name (a name in two slots, or bare + @tool, installs once)
		}
		seen[base] = true

		// Pull in the `requires` closure of this slot item BEFORE the item itself,
		// so a hook's binary recipe (or an instruction's binary) is resolved ahead
		// of the dependent. A pulled dep inherits the dependent's slot for
		// provenance and is itself deduped via `seen`. The closure is catalog-pure;
		// `without` still wins (an explicitly-subtracted item is never pulled back).
		for _, dep := range requires.Expand([]string{base}, cat.Deps) {
			if dep == base || seen[dep] || excluded[dep] {
				continue
			}
			seen[dep] = true
			if it, ok := resolveItem(cat, dep, e.slot); ok {
				out.Items = append(out.Items, it)
			}
		}

		if it, ok := resolveItem(cat, base, e.slot); ok {
			out.Items = append(out.Items, it)
		} else {
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("profile %q slot %q: %q not resolvable — skipped (not yet sourced?)", name, e.slot, base))
		}
	}
	return out, nil
}

// resolveItem builds one ResolvedItem for a base name in a slot, or reports
// !ok when the name does not parse or is absent from the catalog. It is the
// shared item-construction step used for both directly-listed slot items and
// items pulled in by a `requires` edge.
func resolveItem(cat *registry.Catalog, base, slot string) (ResolvedItem, bool) {
	ref, err := source.Parse(base)
	if err != nil {
		return ResolvedItem{}, false
	}
	fam := classify(cat, ref)
	if fam == familyUnknown {
		return ResolvedItem{}, false
	}
	return ResolvedItem{
		Name:   base,
		Slot:   slot,
		Family: fam,
		Source: ref.LockSource(),
	}, true
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
		Eval:          appendDedup(parent.Eval, child.Eval),
		Guardrails:    appendDedup(parent.Guardrails, child.Guardrails),
		Orchestration: appendDedup(parent.Orchestration, child.Orchestration),
		Sandbox:       appendDedup(parent.Sandbox, child.Sandbox),
		Memory:        replaceScalar(parent.Memory, child.Memory),
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
	add("sandbox", l.Sandbox)
	add("observability", l.Observability)
	add("eval", l.Eval)
	add("guardrails", l.Guardrails)
	add("orchestration", l.Orchestration)
	return out
}

// classify dispatches a parsed reference to the artifact or recipe path. For a
// registry (bare-name) ref it looks the name up in the catalog; a non-registry
// source (file:) is assumed resolvable by the caller and classified by which
// manifest kind its location holds — but Phase 5 profiles use only bare names, so
// only the registry branch is exercised by shipped profiles. (file: items in a
// profile slot still resolve via the install-arg path; here a bare-name lookup
// covers the catalog cases.)
func classify(cat *registry.Catalog, ref *source.Ref) manifest.Family {
	if ref.Scheme != source.Registry {
		// Non-registry sources are resolved+merged into the catalog by the install
		// command before dispatch; treat as unknown here only if not yet merged.
		if findRecipe(cat, ref.Raw) != nil {
			return manifest.FamilyRecipe
		}
		if findArtifact(cat, ref.Raw) != nil {
			return manifest.FamilyArtifact
		}
		return familyUnknown
	}
	if findRecipe(cat, ref.Name) != nil {
		return manifest.FamilyRecipe
	}
	if findArtifact(cat, ref.Name) != nil {
		return manifest.FamilyArtifact
	}
	return familyUnknown
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

// excludeSet builds the lookup of base names a profile's `without` subtracts,
// plus a parallel set used to detect entries that matched nothing (so a stale
// exclusion surfaces as a warning rather than a silent no-op). A `without` entry
// is matched on its base name, so a flavoured `name@tool` exclusion is normalized
// to its base — you subtract the item, not one tool's flavour of it.
func excludeSet(without manifest.StringList) (excluded, unmatched map[string]bool) {
	excluded = make(map[string]bool, len(without))
	unmatched = make(map[string]bool, len(without))
	for _, w := range without {
		base, _ := splitFlavour(w)
		excluded[base] = true
		unmatched[base] = true
	}
	return excluded, unmatched
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

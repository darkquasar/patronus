// Package requires computes the transitive closure of catalog `requires` edges
// and validates the dependency graph (no dangling edge, no cycle).
//
// A `requires` edge (manifest.Meta.Requires) is a directed "needs" relation
// between two catalog items by name: installing or locking an item also pulls in
// everything it requires. The relation is type-agnostic — a hook requires the
// recipe that delivers its binary; an instruction requires the binary it
// documents — so this package speaks only in names and a Deps lookup, never in
// families or on-disk shapes.
//
// The package is a deep module over one small interface (Deps): Expand turns a
// seed name list into its dependency-closed, topologically ordered (dependency
// before dependent), deduped superset; Validate proves the whole graph is sound
// at catalog-load time. Both are pure functions of the Deps lookup — no
// filesystem, no registry import, trivially unit-testable with a map.
package requires

import (
	"fmt"
	"sort"
)

// Deps reports the direct `requires` edges of name and whether name is a known
// catalog item. It is the single seam this package needs over a catalog: the
// caller adapts its own item store (the registry.Catalog, a test map) to it.
//
//   - (deps, true)  — name is a known item; deps are its direct requires (may be empty).
//   - (nil, false)  — name is not in the catalog (a dangling edge, or an unknown seed).
type Deps func(name string) (deps []string, ok bool)

// Expand returns the transitive `requires` closure of seeds: every seed plus
// everything reachable through requires edges, ordered dependency-before-dependent
// and deduped. Within that constraint the original seed order is preserved, so a
// profile's resolved order is stable; a required item not already listed is
// inserted immediately before the first item that needs it.
//
// A seed (or required name) absent from deps is kept in the output as-is rather
// than dropped: Expand is the closure step, not the existence gate. Catalog-known
// seeds get their deps pulled in; an unknown name simply contributes no edges and
// passes through (the install/profile path already warn-and-skips unknown names,
// and Validate is the load-time gate that rejects a dangling edge outright). This
// keeps Expand total and side-effect-free.
//
// Expand assumes an acyclic graph (guaranteed by Validate at catalog load). A
// cycle that slips past validation is still handled safely — the visited guard
// prevents infinite recursion — but the resulting order is then merely best-effort.
func Expand(seeds []string, deps Deps) []string {
	var out []string
	onStack := map[string]bool{} // cycle guard for the current DFS path
	placed := map[string]bool{}  // already appended to out (dedup)

	var visit func(name string)
	visit = func(name string) {
		if placed[name] || onStack[name] {
			return
		}
		onStack[name] = true
		if reqs, ok := deps(name); ok {
			for _, r := range reqs {
				visit(r)
			}
		}
		onStack[name] = false
		if !placed[name] {
			placed[name] = true
			out = append(out, name)
		}
	}

	for _, s := range seeds {
		visit(s)
	}
	return out
}

// Pulled reports the names Expand added that were NOT in seeds — the dependencies
// auto-pulled by the closure. Callers use it to emit the "also installing X
// (required by …)" notice. Order matches Expand's output.
func Pulled(seeds, expanded []string) []string {
	seed := make(map[string]bool, len(seeds))
	for _, s := range seeds {
		seed[s] = true
	}
	var extra []string
	for _, e := range expanded {
		if !seed[e] {
			extra = append(extra, e)
		}
	}
	return extra
}

// Validate proves the requires graph over the given item names is sound: every
// requires edge points at a known item (no dangling edge), and the graph has no
// cycle. It is called once at catalog load — the single place a malformed graph
// is rejected — so Expand downstream can assume soundness.
//
// names is the full set of catalog item names; deps returns one item's direct
// requires. Errors name the offending item(s) so a manifest author can fix the
// edge. Dangling edges are reported sorted for a deterministic message.
func Validate(names []string, deps Deps) error {
	// Dangling: every required name must be a known item.
	var dangling []string
	for _, n := range names {
		reqs, _ := deps(n)
		for _, r := range reqs {
			if _, ok := deps(r); !ok {
				dangling = append(dangling, fmt.Sprintf("%q requires unknown item %q", n, r))
			}
		}
	}
	if len(dangling) > 0 {
		sort.Strings(dangling)
		return fmt.Errorf("requires: dangling edge(s): %v", dangling)
	}

	// Cycle: a DFS over the edges; a back-edge to a node on the current stack is
	// a cycle. Report the offending node for an actionable message.
	color := map[string]int{} // 0 unvisited, 1 on-stack, 2 done
	var stack []string
	var dfs func(n string) error
	dfs = func(n string) error {
		color[n] = 1
		stack = append(stack, n)
		reqs, _ := deps(n)
		for _, r := range reqs {
			switch color[r] {
			case 0:
				if err := dfs(r); err != nil {
					return err
				}
			case 1:
				return fmt.Errorf("requires: cycle detected: %v -> %s", stack, r)
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = 2
		return nil
	}
	for _, n := range names {
		if color[n] == 0 {
			if err := dfs(n); err != nil {
				return err
			}
		}
	}
	return nil
}

package requires

import (
	"reflect"
	"strings"
	"testing"
)

// mapDeps adapts a name->requires map to the Deps lookup; a name present in the
// map (even with no edges) is "known", an absent name is unknown.
func mapDeps(m map[string][]string) Deps {
	return func(name string) ([]string, bool) {
		d, ok := m[name]
		return d, ok
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name  string
		graph map[string][]string
		seeds []string
		want  []string
	}{
		{
			name:  "no deps passes through in order",
			graph: map[string][]string{"a": nil, "b": nil},
			seeds: []string{"a", "b"},
			want:  []string{"a", "b"},
		},
		{
			name:  "single dep inserted before its dependent",
			graph: map[string][]string{"hook": {"bin"}, "bin": nil},
			seeds: []string{"hook"},
			want:  []string{"bin", "hook"},
		},
		{
			name:  "dep already listed is not duplicated",
			graph: map[string][]string{"hook": {"bin"}, "bin": nil},
			seeds: []string{"bin", "hook"},
			want:  []string{"bin", "hook"},
		},
		{
			name: "transitive chain a->b->c",
			graph: map[string][]string{
				"a": {"b"}, "b": {"c"}, "c": nil,
			},
			seeds: []string{"a"},
			want:  []string{"c", "b", "a"},
		},
		{
			name: "diamond pulls shared dep once",
			graph: map[string][]string{
				"top": {"l", "r"}, "l": {"base"}, "r": {"base"}, "base": nil,
			},
			seeds: []string{"top"},
			want:  []string{"base", "l", "r", "top"},
		},
		{
			name:  "unknown seed passes through (no edges, not dropped)",
			graph: map[string][]string{"known": nil},
			seeds: []string{"sourced-ref", "known"},
			want:  []string{"sourced-ref", "known"},
		},
		{
			name:  "two independent seeds keep order, each pulls its own dep",
			graph: map[string][]string{"h1": {"b1"}, "h2": {"b2"}, "b1": nil, "b2": nil},
			seeds: []string{"h1", "h2"},
			want:  []string{"b1", "h1", "b2", "h2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Expand(tc.seeds, mapDeps(tc.graph))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Expand(%v) = %v, want %v", tc.seeds, got, tc.want)
			}
		})
	}
}

func TestExpandCycleIsSafe(t *testing.T) {
	// A cycle would be rejected by Validate at load; Expand must still terminate
	// (the on-stack guard) rather than recurse forever.
	g := map[string][]string{"a": {"b"}, "b": {"a"}}
	got := Expand([]string{"a"}, mapDeps(g))
	// Both appear exactly once; order is best-effort under a cycle.
	if len(got) != 2 {
		t.Fatalf("Expand on a cycle = %v, want 2 distinct items", got)
	}
}

func TestPulled(t *testing.T) {
	seeds := []string{"hook"}
	expanded := []string{"bin", "hook"}
	if got := Pulled(seeds, expanded); !reflect.DeepEqual(got, []string{"bin"}) {
		t.Errorf("Pulled = %v, want [bin]", got)
	}
	// Nothing pulled when seeds already cover the closure.
	if got := Pulled([]string{"a", "b"}, []string{"a", "b"}); got != nil {
		t.Errorf("Pulled (no extras) = %v, want nil", got)
	}
}

func TestValidateDangling(t *testing.T) {
	// hook requires bin, but bin is not a known item.
	g := map[string][]string{"hook": {"bin"}}
	err := Validate([]string{"hook"}, mapDeps(g))
	if err == nil || !strings.Contains(err.Error(), "dangling") {
		t.Fatalf("Validate = %v, want a dangling-edge error", err)
	}
	if !strings.Contains(err.Error(), "bin") {
		t.Errorf("dangling error should name the missing item: %v", err)
	}
}

func TestValidateCycle(t *testing.T) {
	g := map[string][]string{"a": {"b"}, "b": {"a"}}
	err := Validate([]string{"a", "b"}, mapDeps(g))
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("Validate = %v, want a cycle error", err)
	}
}

func TestValidateClean(t *testing.T) {
	// The shipped shape: an instruction requires a recipe; both known; acyclic.
	g := map[string][]string{
		"beads": {"bd"}, "bd": nil,
		"ticket": {"tk"}, "tk": nil,
		"tdd-guard-hook": {"tdd-guard"}, "tdd-guard": nil,
	}
	if err := Validate([]string{"beads", "bd", "ticket", "tk", "tdd-guard-hook", "tdd-guard"}, mapDeps(g)); err != nil {
		t.Fatalf("Validate on a clean graph = %v, want nil", err)
	}
}

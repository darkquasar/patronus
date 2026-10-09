package remove

import (
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

func TestSectionPolicyZeroValue(t *testing.T) {
	var zero SectionPolicy
	for _, tc := range []struct {
		tool               string
		structural, strict bool
		prefix             string
	}{
		{"codex", true, true, ""},
		{"pi", true, false, "pi:"},
		{"claude", false, false, ""},
		{"opencode", false, false, ""},
		{"toy", false, false, ""},
	} {
		if zero.Structural(tc.tool) != tc.structural || zero.Strict(tc.tool) != tc.strict || zero.Prefix(tc.tool) != tc.prefix {
			t.Fatalf("zero policy %s = %v/%v/%q", tc.tool, zero.Structural(tc.tool), zero.Strict(tc.tool), zero.Prefix(tc.tool))
		}
	}
	rules := []SectionRule{{Tool: "toy", Prefix: "toy:", Structural: true, Strict: true}}
	p := NewSectionPolicy(rules...)
	rules[0].Tool = "mutated"
	if !p.Structural("toy") || p.Structural("codex") || NewSectionPolicy().Structural("pi") {
		t.Fatal("explicit policy wrong or shares the caller's slice")
	}
}

// An invented strict tool reproduces the four strict refusals' messages and the
// composed inverse; the same tool under a lenient rule follows the Pi shape.
func TestComposeSectionsWithPolicy(t *testing.T) {
	const path = "/fixture/AGENTS.md"
	strict := NewSectionPolicy(SectionRule{Tool: "toy", Prefix: "toy:", Structural: true, Strict: true})
	lenient := NewSectionPolicy(SectionRule{Tool: "toy", Prefix: "toy:", Structural: true})
	for _, tc := range []struct {
		name       string
		policy     SectionPolicy
		mode       string
		force      bool
		wantAction diff.Action
		wantNote   string
	}{
		{"strict clean", strict, "clean", false, diff.Unappend, "remove toy sections: toy:fixture-item"},
		{"strict drift even with force", strict, "drift", true, diff.Skip, "context changed or lacks checksum evidence — retained"},
		{"strict duplicate sibling", strict, "duplicate-sibling", false, diff.Skip, "ambiguous recorded sibling ownership — retained"},
		{"strict sibling identity", strict, "sibling-identity", false, diff.Skip, "incompatible recorded section identity — retained"},
		{"strict unknown marker", strict, "unknown-marker", false, diff.Skip, "unknown, duplicate or nested section \"other\" — retained"},
		{"lenient drift with force", lenient, "drift", true, diff.Unappend, "remove toy sections: toy:fixture-item"},
		{"lenient duplicate sibling", lenient, "duplicate-sibling", false, diff.Unappend, "remove toy sections: toy:fixture-item"},
		{"lenient unknown marker", lenient, "unknown-marker", false, diff.Unappend, "remove toy sections: toy:fixture-item"},
		{"unknown tool is not composed", NewSectionPolicy(), "clean", true, diff.Unappend, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := adapter.AppendSection([]byte("Invented user prose\n"), "toy:fixture-item", []byte("Body"))
			d := diff.FileDiff{Artifact: "fixture-item", Tool: "toy", Scope: "global", Path: path, Action: diff.Append, After: current, Section: &diff.SectionEdit{Name: "toy:fixture-item"}}
			items := state.FromChangeSet([]diff.FileDiff{d}, "t")
			occ := Occupancy{path: {{Artifact: "fixture-item", Tool: "toy", Scope: "global", Section: "toy:fixture-item"}}}
			switch tc.mode {
			case "drift":
				current = append(current, []byte("edit\n")...)
			case "duplicate-sibling":
				other := Contributor{Artifact: "other", Tool: "toy", Scope: "global", Section: "toy:other"}
				occ[path] = append(occ[path], other, other)
			case "sibling-identity":
				occ[path] = append(occ[path], Contributor{Artifact: "other", Tool: "toy", Scope: "global", Section: "renamed"})
			case "unknown-marker":
				current = append(current, []byte("<!-- patronus:start other -->\nx\n<!-- patronus:end other -->\n")...)
				items[0].Files[0].Checksum = sum(current)
			}
			got, err := ComputeWithPolicy(items, readerFrom(map[string][]byte{path: current}), occ, tc.force, tc.policy)
			if err != nil {
				t.Fatal(err)
			}
			dd := got.ChangeSet.Diffs[0]
			if dd.Action != tc.wantAction || dd.Note != tc.wantNote {
				t.Fatalf("action=%s note=%q want %s/%q", dd.Action, dd.Note, tc.wantAction, tc.wantNote)
			}
		})
	}
}

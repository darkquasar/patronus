package remove

import (
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

func TestCodexSectionsRefuseUnsafeContextEvenForce(t *testing.T) {
	for _, mode := range []string{"drift", "unknown", "malformed-sibling", "orphan-end", "duplicate", "nested", "mixed-tool", "mixed-scope", "duplicate-owner", "duplicate-sibling"} {
		t.Run(mode, func(t *testing.T) {
			path := "/fixture/AGENTS.md"
			current := adapter.AppendSection([]byte("Invented user prose\n"), "first-cx", []byte("First body"))
			d := diff.FileDiff{Artifact: "first-cx", Tool: "codex", Scope: "global", Path: path, Action: diff.Append, After: current, Section: &diff.SectionEdit{Name: "first-cx"}}
			items := state.FromChangeSet([]diff.FileDiff{d}, "first")
			occ := Occupancy{path: {{Artifact: "first-cx", Tool: "codex", Scope: "global", Section: "first-cx"}}}
			switch mode {
			case "drift":
				current = append(current, []byte("Invented user edit\n")...)
			case "unknown":
				current = adapter.AppendSection(current, "unknown-cx", []byte("Unowned"))
			case "malformed-sibling":
				current = append(current, []byte("\n<!-- patronus:start second-cx -->\nUnclosed\n")...)
				occ[path] = append(occ[path], Contributor{Artifact: "second-cx", Tool: "codex", Scope: "global", Section: "second-cx"})
			case "orphan-end":
				current = append(current, []byte("<!-- patronus:end first-cx -->\n")...)
			case "duplicate":
				current = append(current, current...)
			case "nested":
				current = adapter.AppendSection(current, "first-cx", []byte("<!-- patronus:start unknown-cx -->\nNested\n<!-- patronus:end unknown-cx -->"))
			case "mixed-tool":
				occ[path] = append(occ[path], Contributor{Artifact: "other", Tool: "pi", Scope: "global", Section: "pi:other"})
			case "mixed-scope":
				occ[path] = append(occ[path], Contributor{Artifact: "other", Tool: "codex", Scope: "local", Section: "other"})
			case "duplicate-owner":
				occ[path] = append(occ[path], occ[path][0])
			case "duplicate-sibling":
				current = adapter.AppendSection(current, "second-cx", []byte("Sibling body"))
				other := Contributor{Artifact: "second-cx", Tool: "codex", Scope: "global", Section: "second-cx"}
				occ[path] = append(occ[path], other, other)
			}
			if mode != "drift" {
				items[0].Files[0].Checksum = sum(current)
			}
			for _, force := range []bool{false, true} {
				got, err := ComputeWithForce(items, readerFrom(map[string][]byte{path: current}), occ, force)
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range got.Ledger {
					if e.Outcome.Complete() {
						t.Fatalf("unsafe %s was credited (force=%v): %+v", mode, force, got)
					}
				}
				for _, d := range got.ChangeSet.Diffs {
					if d.Action != diff.Skip || d.Intended != "" {
						t.Fatalf("unsafe context promotable: %+v", d)
					}
				}
			}
		})
	}
}

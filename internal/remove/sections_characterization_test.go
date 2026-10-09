package remove

import (
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// Characterization of the tool allowlists in section composition and the Pi
// insufficient-evidence skip (compute.go:169, 547, 821-869) for the tools codex,
// pi, claude and opencode at the base revision. Each row records the action,
// intended action, note and ledger outcome of the one inverse diff produced.

type sectionOutcome struct {
	action, intended, note string
	outcome                Outcome
}

func (o sectionOutcome) String() string {
	return o.action + "|" + o.intended + "|" + o.note + "|" + string(o.outcome)
}

func sectionComposeOutcome(t *testing.T, tool, mode string, force bool) sectionOutcome {
	t.Helper()
	const path = "/fixture/AGENTS.md"
	section := "fixture-item"
	if tool == "pi" {
		section = "pi:" + section
	}
	current := adapter.AppendSection([]byte("Invented user prose\n"), section, []byte("Body"))
	d := diff.FileDiff{Artifact: "fixture-item", Tool: tool, Scope: "global", Path: path, Action: diff.Append, After: current, Section: &diff.SectionEdit{Name: section}}
	items := state.FromChangeSet([]diff.FileDiff{d}, "t")
	occ := Occupancy{path: {{Artifact: "fixture-item", Tool: tool, Scope: "global", Section: section}}}
	switch mode {
	case "drift":
		current = append(current, []byte("edit\n")...)
	case "no-checksum":
		items[0].Files[0].Checksum = ""
	case "duplicate-sibling":
		other := Contributor{Artifact: "other", Tool: tool, Scope: "global", Section: "other"}
		occ[path] = append(occ[path], other, other)
	case "sibling-identity":
		occ[path] = append(occ[path], Contributor{Artifact: "other", Tool: tool, Scope: "global", Section: "renamed"})
	case "unknown-marker":
		current = append(current, []byte("<!-- patronus:start other -->\nx\n<!-- patronus:end other -->\n")...)
		items[0].Files[0].Checksum = sum(current)
	}
	got, err := ComputeWithForce(items, readerFrom(map[string][]byte{path: current}), occ, force)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ChangeSet.Diffs) != 1 || len(got.Ledger) != 1 {
		t.Fatalf("expected one inverse diff and ledger row: %+v", got)
	}
	dd := got.ChangeSet.Diffs[0]
	return sectionOutcome{string(dd.Action), string(dd.Intended), dd.Note, got.Ledger[0].Outcome}
}

func TestHarnessSectionCompositionAllowlist(t *testing.T) {
	const (
		codexApplied   = "UNAPPEND|||applied"
		piApplied      = "UNAPPEND|||applied"
		legacySkipped  = "SKIP|UNAPPEND|file changed since install — skipped (use --force)|drift-skipped"
		legacyForced   = "UNAPPEND|UNAPPEND||applied"
		noEvidence     = "SKIP||context changed or lacks checksum evidence — retained|ambiguous-skipped"
		ambiguousSib   = "SKIP||ambiguous recorded sibling ownership — retained|ambiguous-skipped"
		identitySib    = "SKIP||incompatible recorded section identity — retained|ambiguous-skipped"
		unknownMarkers = "SKIP||unknown, duplicate or nested section \"other\" — retained|ambiguous-skipped"
	)
	for _, tc := range []struct {
		tool, mode    string
		plain, forced string
		composedNote  string
	}{
		{tool: "codex", mode: "clean", plain: codexApplied, forced: codexApplied, composedNote: "remove codex sections: fixture-item"},
		{tool: "codex", mode: "drift", plain: noEvidence, forced: noEvidence},
		{tool: "codex", mode: "no-checksum", plain: noEvidence, forced: noEvidence},
		{tool: "codex", mode: "duplicate-sibling", plain: ambiguousSib, forced: ambiguousSib},
		{tool: "codex", mode: "sibling-identity", plain: identitySib, forced: identitySib},
		{tool: "codex", mode: "unknown-marker", plain: unknownMarkers, forced: unknownMarkers},
		{tool: "pi", mode: "clean", plain: piApplied, forced: piApplied, composedNote: "remove pi sections: pi:fixture-item"},
		{tool: "pi", mode: "drift", plain: noEvidence, forced: piApplied, composedNote: "remove pi sections: pi:fixture-item"},
		{tool: "pi", mode: "no-checksum", plain: noEvidence, forced: noEvidence},
		{tool: "pi", mode: "duplicate-sibling", plain: piApplied, forced: piApplied, composedNote: "remove pi sections: pi:fixture-item"},
		{tool: "pi", mode: "sibling-identity", plain: piApplied, forced: piApplied, composedNote: "remove pi sections: pi:fixture-item"},
		{tool: "pi", mode: "unknown-marker", plain: piApplied, forced: piApplied, composedNote: "remove pi sections: pi:fixture-item"},
		{tool: "claude", mode: "clean", plain: legacySkipped, forced: legacyForced},
		{tool: "claude", mode: "drift", plain: legacySkipped, forced: legacyForced},
		{tool: "claude", mode: "no-checksum", plain: legacySkipped, forced: legacyForced},
		{tool: "claude", mode: "duplicate-sibling", plain: legacySkipped, forced: legacyForced},
		{tool: "claude", mode: "sibling-identity", plain: legacySkipped, forced: legacyForced},
		{tool: "claude", mode: "unknown-marker", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "clean", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "drift", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "no-checksum", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "duplicate-sibling", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "sibling-identity", plain: legacySkipped, forced: legacyForced},
		{tool: "opencode", mode: "unknown-marker", plain: legacySkipped, forced: legacyForced},
	} {
		t.Run(tc.tool+"/"+tc.mode, func(t *testing.T) {
			plain := sectionComposeOutcome(t, tc.tool, tc.mode, false)
			forced := sectionComposeOutcome(t, tc.tool, tc.mode, true)
			// The composed note names the tool; strip it so the table carries it once.
			if got := stripNote(plain, tc.composedNote); got != tc.plain {
				t.Fatalf("plain outcome\n got: %s\nwant: %s", got, tc.plain)
			}
			if got := stripNote(forced, tc.composedNote); got != tc.forced {
				t.Fatalf("forced outcome\n got: %s\nwant: %s", got, tc.forced)
			}
		})
	}
}

func stripNote(o sectionOutcome, composedNote string) string {
	if composedNote != "" && o.note == composedNote {
		o.note = ""
	}
	return o.String()
}

// compute.go:547: a Pi CREATE row without checksum evidence is retained even
// under --force; every other tool deletes it.
func TestHarnessInsufficientEvidenceSkipIsPiOnly(t *testing.T) {
	const path = "/fixture/file.md"
	for _, tc := range []struct {
		tool string
		want string
	}{
		{"codex", "DELETE|||applied"},
		{"pi", "SKIP||insufficient Pi ownership evidence — retained|unsafe-legacy-skipped"},
		{"claude", "DELETE|||applied"},
		{"opencode", "DELETE|||applied"},
	} {
		for _, force := range []bool{false, true} {
			items := []state.Item{{Artifact: "fixture-item", Tool: tc.tool, Scope: "global", Files: []state.FileState{{Path: path, Action: "CREATE"}}}}
			got, err := ComputeWithForce(items, readerFrom(map[string][]byte{path: []byte("x")}), nil, force)
			if err != nil {
				t.Fatal(err)
			}
			dd := got.ChangeSet.Diffs[0]
			o := sectionOutcome{string(dd.Action), string(dd.Intended), dd.Note, got.Ledger[0].Outcome}
			if o.String() != tc.want {
				t.Fatalf("%s force=%v\n got: %s\nwant: %s", tc.tool, force, o, tc.want)
			}
		}
	}
}

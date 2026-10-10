package remove

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

const runnerPath = "/invented/.toy/plugins/runner.ts"

func runnerItem(artifact string, body []byte) state.Item {
	return state.Item{
		Artifact: artifact, Tool: "toy", Scope: "local", ItemVersion: "1.0.0",
		Files: []state.FileState{{Path: runnerPath, Action: "CREATE", Checksum: sum(body)}},
	}
}

func runnerOccupancy(artifacts ...string) Occupancy {
	occ := Occupancy{}
	for _, a := range artifacts {
		occ[runnerPath] = append(occ[runnerPath], Contributor{Artifact: a, Tool: "toy", Scope: "local", Created: true})
	}
	return occ
}

func outcomes(r Result) []string {
	var out []string
	for _, e := range r.Ledger {
		out = append(out, e.Artifact+":"+string(e.Outcome))
	}
	return out
}

func actions(r Result) []string {
	var out []string
	for _, d := range r.ChangeSet.Diffs {
		out = append(out, d.Artifact+":"+string(d.Action))
	}
	return out
}

// TestRemoveSharedCreateLifecycle: a placed file recorded under several
// contributors is released, not deleted, while another recorded contributor
// remains; it is deleted once with its last contributor; drift and absence keep
// today's verdicts for every contributor; --force never promotes a release.
func TestRemoveSharedCreateLifecycle(t *testing.T) {
	body := []byte("runner\n")
	alpha, beta := runnerItem("alpha", body), runnerItem("beta", body)
	present := readerFrom(map[string][]byte{runnerPath: body})
	for _, tc := range []struct {
		name     string
		items    []state.Item
		read     ReadExisting
		occ      Occupancy
		force    bool
		outcomes string
		actions  string
		note     string
	}{
		{"non-last contributor released", []state.Item{alpha}, present, runnerOccupancy("alpha", "beta"), false, "alpha:shared-retained", "alpha:SKIP", "shared placed file still owned by beta (toy local); this contribution released"},
		{"force does not promote a release", []state.Item{alpha}, present, runnerOccupancy("alpha", "beta"), true, "alpha:shared-retained", "alpha:SKIP", "shared placed file still owned by beta (toy local); this contribution released"},
		{"last contributor deletes", []state.Item{beta}, present, runnerOccupancy("beta"), false, "beta:applied", "beta:DELETE", ""},
		{"both selected: one delete, other settled", []state.Item{alpha, beta}, present, runnerOccupancy("alpha", "beta"), false, "alpha:applied,beta:applied", "alpha:DELETE,beta:SKIP", "shared placed file deleted with alpha"},
		{"drift keeps every contributor open", []state.Item{alpha, beta}, readerFrom(map[string][]byte{runnerPath: []byte("edited\n")}), runnerOccupancy("alpha", "beta"), false, "alpha:drift-skipped,beta:drift-skipped", "alpha:SKIP,beta:SKIP", "user-edited since install — skipped (use --force)"},
		{"user-deleted file with surviving rows", []state.Item{alpha}, readerFrom(map[string][]byte{}), runnerOccupancy("alpha", "beta"), false, "alpha:already-absent", "alpha:SKIP", "already absent"},
		{"sole owner without occupancy view", []state.Item{alpha}, present, nil, false, "alpha:applied", "alpha:DELETE", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ComputeWithForce(tc.items, tc.read, tc.occ, tc.force)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(outcomes(r), ","); got != tc.outcomes {
				t.Fatalf("outcomes = %s, want %s", got, tc.outcomes)
			}
			if got := strings.Join(actions(r), ","); got != tc.actions {
				t.Fatalf("actions = %s, want %s", got, tc.actions)
			}
			last := r.ChangeSet.Diffs[len(r.ChangeSet.Diffs)-1]
			if last.Note != tc.note {
				t.Fatalf("note = %q, want %q", last.Note, tc.note)
			}
		})
	}
	if !SharedRetained.Complete() {
		t.Fatal("SharedRetained must settle the released row")
	}
}

// TestSharedRetainedSkipIsVerifiable: the released contributor's SKIP carries
// the current bytes so the applier's unchanged check passes and records it in
// Skipped, which is what retires the row.
func TestSharedRetainedSkipIsVerifiable(t *testing.T) {
	body := []byte("runner\n")
	r, err := Compute([]state.Item{runnerItem("alpha", body)}, readerFrom(map[string][]byte{runnerPath: body}), runnerOccupancy("alpha", "beta"))
	if err != nil {
		t.Fatal(err)
	}
	d := r.ChangeSet.Diffs[0]
	if d.Action != diff.Skip || string(d.Before) != "runner\n" || d.Intended != "" {
		t.Fatalf("released diff = %+v", d)
	}
}

// TestCreatedContributorsDoNotGateLegacyMerge: a Created contributor on a path
// is invisible to the pre-compose whole-file MERGE gate, which judges config
// writers only.
func TestCreatedContributorsDoNotGateLegacyMerge(t *testing.T) {
	prior := []byte(`{"user":1}` + "\n")
	current := []byte(`{"user":1,"owned":2}` + "\n")
	items := []state.Item{{Artifact: "legacy", Tool: "claude", Scope: "global", Files: []state.FileState{{Path: "/c/settings.json", Action: "MERGE", Checksum: sum(current), Prior: prior}}}}
	occ := Occupancy{"/c/settings.json": {Contributor{Artifact: "placer", Tool: "claude", Scope: "global", Created: true}}}
	r, err := Compute(items, readerFrom(map[string][]byte{"/c/settings.json": current}), occ)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(outcomes(r), ","); got != "legacy:applied" {
		t.Fatalf("outcomes = %s", got)
	}
}

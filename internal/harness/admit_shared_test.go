package harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// TestAdmitStrictAcceptsIdenticalCoOwnedCreate: a strict descriptor admits a
// placed file another artifact of the same target owns when the recorded
// checksum equals the selected bytes, marking the SKIP as SharedAcquire; a
// different checksum keeps the exact incompatible-owner and drift refusals.
func TestAdmitStrictAcceptsIdenticalCoOwnedCreate(t *testing.T) {
	f := newAdmitFixture(t)
	path := filepath.Join(f.project, ".toy/plugins/runner.ts")
	body := []byte("runner\n")
	owner := func(tool, artifact string, checksumOf []byte) []state.Item {
		return []state.Item{{Artifact: artifact, Tool: tool, Scope: "local", Files: []state.FileState{{Path: path, Checksum: state.Checksum(checksumOf), Action: "CREATE"}}}}
	}
	for _, tc := range []struct {
		name    string
		fd      diff.FileDiff
		exists  bool
		owners  []state.Item
		want    string
		acquire bool
	}{
		{"identical co-owned skip admitted and marked", diff.FileDiff{Action: diff.Skip, Before: body, After: body}, true, owner("toy", "alpha", body), "", true},
		{"identical co-owned create (file missing) admitted, not marked", diff.FileDiff{Action: diff.Create, After: body}, false, owner("toy", "alpha", body), "", false},
		{"own unchanged skip is not a shared acquire", diff.FileDiff{Action: diff.Skip, Before: body, After: body}, true, owner("toy", "beta", body), "", false},
		{"co-owner with different recorded bytes stays incompatible", diff.FileDiff{Action: diff.Conflict, Before: []byte("other\n"), After: body}, true, owner("toy", "alpha", []byte("other\n")), "toy destination " + path + " owned by alpha (toy), selected beta", false},
		{"co-owner of another target stays incompatible", diff.FileDiff{Action: diff.Skip, Before: body, After: body}, true, owner("other", "alpha", body), "toy destination " + path + " owned by alpha (other), selected beta", false},
		{"own drifted file keeps the drift refusal", diff.FileDiff{Action: diff.Conflict, Before: []byte("edited\n"), After: body}, true, owner("toy", "beta", body), "toy ownership drift at " + path + "; preserve edits, reconcile before mutation (no --force bypass)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fd := tc.fd
			fd.Tool, fd.Type, fd.Artifact, fd.Path, fd.Scope = "toy", "hook", "beta", path, "local"
			if tc.exists {
				f.write(t, path, fd.Before)
			} else {
				_ = os.Remove(path)
			}
			cs := &diff.ChangeSet{Diffs: []diff.FileDiff{fd}}
			err := Admit(toyDescriptor(nil), f.admission(cs, tc.owners))
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("admission\n got: %s\nwant: %s", got, tc.want)
			}
			if cs.Diffs[0].SharedAcquire != tc.acquire {
				t.Fatalf("SharedAcquire = %v, want %v", cs.Diffs[0].SharedAcquire, tc.acquire)
			}
		})
	}
}

// TestAdmitLenientMarksSharedAcquire: the mark is shared behaviour, not a
// strict-only rule; a lenient descriptor marks the same SKIP without judging
// ownership of a non-resource file.
func TestAdmitLenientMarksSharedAcquire(t *testing.T) {
	f := newAdmitFixture(t)
	path := filepath.Join(f.project, ".lax/plugins/runner.ts")
	body := []byte("runner\n")
	f.write(t, path, body)
	owners := []state.Item{{Artifact: "alpha", Tool: "lax", Scope: "local", Files: []state.FileState{{Path: path, Checksum: state.Checksum(body), Action: "CREATE"}}}}
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "lax", Type: "hook", Artifact: "beta", Path: path, Scope: "local", Action: diff.Skip, Before: body, After: body}}}
	if err := Admit(laxDescriptor(nil), f.admission(cs, owners)); err != nil {
		t.Fatal(err)
	}
	if !cs.Diffs[0].SharedAcquire {
		t.Fatal("lenient descriptor did not mark the shared acquire")
	}
}

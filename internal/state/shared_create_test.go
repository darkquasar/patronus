package state

import (
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
)

const sharedRunnerPath = "/invented/.toy/plugins/runner.ts"

func sharedRunnerDiff(body []byte, contribs ...diff.CreateContrib) diff.FileDiff {
	return diff.FileDiff{
		Path:          sharedRunnerPath,
		Action:        diff.Create,
		After:         body,
		Artifact:      "alpha",
		Version:       "1.0.0",
		Type:          "hook",
		Tool:          "toy",
		Scope:         "local",
		CreateContrib: contribs,
	}
}

// TestFromChangeSetRecordsCreateContributors: a composed CREATE carrying
// CreateContrib yields one CREATE row per contributor, each with the shared
// file's checksum and the contributor's own version and type.
func TestFromChangeSetRecordsCreateContributors(t *testing.T) {
	body := []byte("runner\n")
	items := FromChangeSet([]diff.FileDiff{sharedRunnerDiff(body, diff.CreateContrib{Artifact: "beta", Version: "2.0.0", Type: "hook"})}, "now")
	if len(items) != 2 {
		t.Fatalf("want alpha and beta items, got %+v", items)
	}
	for _, it := range items {
		if it.Tool != "toy" || it.Scope != "local" || len(it.Files) != 1 {
			t.Fatalf("item %s shape: %+v", it.Artifact, it)
		}
		f := it.Files[0]
		if f.Path != sharedRunnerPath || f.Action != string(diff.Create) || f.Checksum != Checksum(body) || f.Section != "" || f.Setting != nil || f.Prior != nil {
			t.Fatalf("item %s row: %+v", it.Artifact, f)
		}
	}
	if items[1].Artifact != "beta" || items[1].ItemVersion != "2.0.0" || items[1].Type != "hook" {
		t.Fatalf("contributor item: %+v", items[1])
	}
}

// TestReconcileAdoptsSharedAcquireOnly: a SKIP over an identical placed file is
// adopted as a CREATE row for the new contributor only when admission marked it
// SharedAcquire and a recorded owner's checksum matches the bytes on disk; an
// ordinary equal SKIP stays unmanaged ("equality is not adoption authority").
func TestReconcileAdoptsSharedAcquireOnly(t *testing.T) {
	body := []byte("runner\n")
	old := FromChangeSet([]diff.FileDiff{sharedRunnerDiff(body)}, "first")
	beta := sharedRunnerDiff(body)
	beta.Artifact, beta.Action, beta.Before = "beta", diff.Skip, body
	for _, tc := range []struct {
		name    string
		acquire bool
		items   int
	}{
		{"plain equal skip is not adopted", false, 1},
		{"shared acquire adopts a create row", true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := beta
			d.SharedAcquire = tc.acquire
			got := Reconcile(ReconcileInput{Old: old, Desired: []diff.FileDiff{d}, Result: install.Result{Skipped: []diff.FileDiff{d}}, Now: "second"})
			if len(got.Unresolved) != 0 {
				t.Fatalf("unresolved: %v", got.Unresolved)
			}
			if len(got.Items) != tc.items {
				t.Fatalf("items = %+v", got.Items)
			}
			if tc.items == 1 {
				return
			}
			it := got.Items[1]
			if it.Artifact != "beta" || it.ItemVersion != "1.0.0" || len(it.Files) != 1 || it.Files[0].Action != string(diff.Create) || it.Files[0].Checksum != Checksum(body) {
				t.Fatalf("adopted item: %+v", it)
			}
		})
	}
	// A SharedAcquire mark over a path whose recorded owner's checksum does not
	// match the bytes on disk is a problem, never an adoption.
	drifted := beta
	drifted.SharedAcquire = true
	drifted.Before = []byte("edited\n")
	got := Reconcile(ReconcileInput{Old: old, Desired: []diff.FileDiff{drifted}, Result: install.Result{Skipped: []diff.FileDiff{drifted}}, Now: "second"})
	if len(got.Items) != 1 || len(got.Unresolved) != 1 {
		t.Fatalf("drifted shared acquire: items=%+v unresolved=%v", got.Items, got.Unresolved)
	}
}

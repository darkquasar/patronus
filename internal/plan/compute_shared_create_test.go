package plan

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
)

func absentFile() adapter.ReadExisting {
	return func(string) ([]byte, bool, error) { return nil, false, nil }
}

// sharedCreate is an invented sidecar two artifacts both place byte for byte.
func sharedCreate(artifact string, body []byte) diff.FileDiff {
	return diff.FileDiff{
		Path:     "/invented/.toy/plugins/runner.ts",
		Action:   diff.Create,
		After:    body,
		Artifact: artifact,
		Version:  "1.0.0",
		Type:     "hook",
		Role:     "guardrail",
		Tool:     "toy",
		Scope:    "local",
	}
}

// TestComposeIdenticalCreateRecordsContributors: two artifacts creating one
// path with identical bytes compose into one CREATE whose CreateContrib names
// every additional contributor, so state records a row per artifact.
func TestComposeIdenticalCreateRecordsContributors(t *testing.T) {
	body := []byte("export const server = () => ({});\n")
	cs, err := Finalize([]diff.FileDiff{sharedCreate("alpha", body), sharedCreate("beta", body)}, absentFile())
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want 1 composed diff, got %d", len(cs.Diffs))
	}
	d := cs.Diffs[0]
	if d.Action != diff.Create || d.Artifact != "alpha" {
		t.Fatalf("composed diff = %s %s", d.Action, d.Artifact)
	}
	if len(d.CreateContrib) != 1 || d.CreateContrib[0] != (diff.CreateContrib{Artifact: "beta", Version: "1.0.0", Type: "hook", Role: "guardrail"}) {
		t.Fatalf("CreateContrib = %+v", d.CreateContrib)
	}
	// The same artifact placing the same file for a second reason is not a
	// second contributor.
	cs, err = Finalize([]diff.FileDiff{sharedCreate("alpha", body), sharedCreate("alpha", body)}, absentFile())
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 || len(cs.Diffs[0].CreateContrib) != 0 {
		t.Fatalf("same-artifact repeat recorded a contributor: %+v", cs.Diffs[0].CreateContrib)
	}
}

// TestFinalizeRefusesConflictingPlacedBytes: two fresh CREATE intents on one
// path with different bytes are refused, naming both artifacts, instead of
// silently keeping the first payload.
func TestFinalizeRefusesConflictingPlacedBytes(t *testing.T) {
	_, err := Finalize([]diff.FileDiff{sharedCreate("alpha", []byte("one\n")), sharedCreate("beta", []byte("two\n"))}, absentFile())
	if err == nil {
		t.Fatal("conflicting placed bytes composed silently")
	}
	want := "compose /invented/.toy/plugins/runner.ts: conflicting placed bytes between alpha and beta"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

package plugin

import (
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
)

func TestReconcileFlipsStatuses(t *testing.T) {
	entries := []lock.Entry{
		{Name: "superpowers", Kind: "plugin", Status: lock.StatusUnverified},
		{Name: "ghost", Kind: "plugin", Status: lock.StatusVerified},
		{Name: "some-artifact", Kind: "artifact"},
	}
	// What each tool reports installed, keyed by tool then by lock-entry name.
	idsByTool := map[string]map[string]bool{
		"claude": {"superpowers": true}, // ghost absent
	}
	got := Reconcile(entries, idsByTool)

	byName := map[string]lock.Entry{}
	for _, e := range got {
		byName[e.Name] = e
	}
	if byName["superpowers"].Status != lock.StatusVerified {
		t.Errorf("superpowers should be verified, got %q", byName["superpowers"].Status)
	}
	if byName["ghost"].Status != lock.StatusMissing {
		t.Errorf("ghost should be missing, got %q", byName["ghost"].Status)
	}
	if byName["some-artifact"].Status != "" {
		t.Errorf("non-plugin entry must be untouched, got %q", byName["some-artifact"].Status)
	}
}

func TestReconcileUnreachableToolLeavesUnverified(t *testing.T) {
	entries := []lock.Entry{{Name: "superpowers", Kind: "plugin", Status: lock.StatusUnverified}}
	// No tool reported anything (all unreachable) -> stays unverified, not missing.
	got := Reconcile(entries, map[string]map[string]bool{})
	if got[0].Status != lock.StatusUnverified {
		t.Errorf("unreachable tool should leave unverified, got %q", got[0].Status)
	}
}

package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// testRegistry is the production registry for tests; every command constructor
// and request/options literal in the suite receives it.
func testRegistry(t *testing.T) *harness.Registry {
	t.Helper()
	reg, err := defaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// codexPreflightPlan and piPreflightPlan are the historical admission entry
// points, kept for the characterization suite as thin test wrappers over the
// shared admission of the named descriptor.
func codexPreflightPlan(reg *harness.Registry, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) error {
	d, ok := reg.Lookup("codex")
	if !ok {
		return harness.ErrUnknownTarget
	}
	_, err := admitTarget(reg, d, cs, res, home, project, nil)
	return err
}

func piPreflightPlan(reg *harness.Registry, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) ([]piContextReview, error) {
	d, ok := reg.Lookup("pi")
	if !ok {
		return nil, harness.ErrUnknownTarget
	}
	return admitTarget(reg, d, cs, res, home, project, nil)
}

func codexChecksum(raw []byte) string { return state.Checksum(raw) }

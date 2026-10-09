package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/harness"
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

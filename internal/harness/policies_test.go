package harness

import "testing"

func TestRegistryOwnershipPolicy(t *testing.T) {
	r := toyRegistry(t)
	p := r.OwnershipPolicy()
	for tool, want := range map[string]bool{"toy": true, "other": true, "lax": false, "plain": false, "codex": false, "pi": false} {
		if got := p.Structural(tool); got != want {
			t.Fatalf("Structural(%s) = %v want %v", tool, got, want)
		}
	}
	var nilReg *Registry
	if nilReg.OwnershipPolicy().Structural("codex") {
		t.Fatal("nil registry policy names a tool")
	}
}

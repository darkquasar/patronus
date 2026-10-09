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

func TestRegistryTargetPolicy(t *testing.T) {
	r := toyRegistry(t)
	p := r.TargetPolicy()
	for tool, want := range map[string]bool{"toy": true, "lax": true, "other": false, "plain": false, "pi": false} {
		if got := p.Targeted(tool); got != want {
			t.Fatalf("Targeted(%s) = %v want %v", tool, got, want)
		}
	}
	var nilReg *Registry
	if nilReg.TargetPolicy().Targeted("pi") {
		t.Fatal("nil registry policy stamps a target")
	}
}

func TestRegistrySectionPolicy(t *testing.T) {
	r := toyRegistry(t)
	p := r.SectionPolicy()
	if !p.Structural("toy") || !p.Strict("toy") || p.Prefix("toy") != "toy:" {
		t.Fatalf("toy rule: %v %v %q", p.Structural("toy"), p.Strict("toy"), p.Prefix("toy"))
	}
	if !p.Structural("other") || p.Strict("other") || p.Prefix("other") != "" {
		t.Fatalf("other rule: %v %v %q", p.Structural("other"), p.Strict("other"), p.Prefix("other"))
	}
	if p.Structural("lax") || p.Structural("codex") {
		t.Fatal("non-owning descriptors compose sections")
	}
	var nilReg *Registry
	if nilReg.SectionPolicy().Structural("pi") {
		t.Fatal("nil registry policy composes a tool")
	}
}

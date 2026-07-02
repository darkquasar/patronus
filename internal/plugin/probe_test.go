package plugin

import "testing"

// fakeProbe lets tests decide CLI presence without spawning a process.
type fakeProbe struct{ present map[string]bool }

func (f fakeProbe) HasPluginCLI(tool string) bool { return f.present[tool] }

func TestBin(t *testing.T) {
	if bin, ok := Bin("claude"); !ok || bin != "claude" {
		t.Errorf("Bin(claude) = %q,%v", bin, ok)
	}
	if bin, ok := Bin("codex"); !ok || bin != "codex" {
		t.Errorf("Bin(codex) = %q,%v", bin, ok)
	}
	if _, ok := Bin("opencode"); ok {
		t.Error("Bin(opencode) should be ok=false")
	}
}

func TestFakeProbeSatisfiesInterface(t *testing.T) {
	var p CLIProbe = fakeProbe{present: map[string]bool{"claude": true}}
	if !p.HasPluginCLI("claude") || p.HasPluginCLI("codex") {
		t.Error("fakeProbe wiring wrong")
	}
}

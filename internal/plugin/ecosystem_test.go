package plugin

import "testing"

func TestEcosystemFor(t *testing.T) {
	cases := []struct {
		tool    string
		wantEco string
		wantOK  bool
	}{
		{"claude", "claude-code", true},
		{"codex", "codex", true},
		{"opencode", "", false},
		{"unknown", "", false},
	}
	for _, c := range cases {
		eco, ok := EcosystemFor(c.tool)
		if eco != c.wantEco || ok != c.wantOK {
			t.Errorf("EcosystemFor(%q) = (%q,%v), want (%q,%v)", c.tool, eco, ok, c.wantEco, c.wantOK)
		}
	}
}

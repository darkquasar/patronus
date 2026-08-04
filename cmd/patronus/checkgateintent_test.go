package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func TestCheckGateIntent(t *testing.T) {
	tests := []struct {
		name          string
		hook          gateHook
		wantViolation bool
		wantAdvisory  bool
	}{
		{
			name:          "script exits 2 without gate intent fails",
			hook:          gateHook{Name: "block-secrets", Intent: "", ScriptExits2: true, HasScript: true, Command: "{script}"},
			wantViolation: true,
		},
		{
			name:          "script exits 2 WITH gate intent passes",
			hook:          gateHook{Name: "block-secrets", Intent: manifest.HookGate, ScriptExits2: true, HasScript: true, Command: "{script}"},
			wantViolation: false,
		},
		{
			name:          "blocking binary without gate intent fails",
			hook:          gateHook{Name: "tdd-guard-hook", Intent: "", Command: "tdd-guard", HasScript: false},
			wantViolation: true,
		},
		{
			name:          "blocking binary WITH gate intent passes",
			hook:          gateHook{Name: "tdd-guard-hook", Intent: manifest.HookGate, Command: "tdd-guard", HasScript: false},
			wantViolation: false,
		},
		{
			name:          "nudge script (no exit 2) does not fail",
			hook:          gateHook{Name: "graphify-hint", Intent: manifest.HookNudge, ScriptExits2: false, HasScript: false, Command: "sh -c 'echo hint; exit 0'"},
			wantViolation: false,
		},
		{
			name:          "nudge with empty intent and no blocking signal is clean",
			hook:          gateHook{Name: "some-nudge", Intent: "", ScriptExits2: false, HasScript: true, Command: "{script}"},
			wantViolation: false,
		},
		{
			name:         "unclassifiable bare binary passes with an advisory",
			hook:         gateHook{Name: "mystery-hook", Intent: "", Command: "some-tool --check", HasScript: false},
			wantAdvisory: true,
		},
		{
			name: "unclassifiable bare binary WITH gate intent: neither violation nor advisory",
			hook: gateHook{Name: "mystery-hook", Intent: manifest.HookGate, Command: "some-tool --check", HasScript: false},
			// declared gate short-circuits before classification
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, advisories := checkGateIntent([]gateHook{tt.hook})
			if got := len(violations) == 1; got != tt.wantViolation {
				t.Errorf("violation = %v, want %v (%+v)", got, tt.wantViolation, violations)
			}
			if got := len(advisories) == 1; got != tt.wantAdvisory {
				t.Errorf("advisory = %v, want %v (%+v)", got, tt.wantAdvisory, advisories)
			}
		})
	}
}

func TestScriptExits2(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"plain exit 2", "#!/bin/sh\nif bad; then\n  exit 2\nfi\n", true},
		{"exit 0 only", "#!/bin/sh\necho ok\nexit 0\n", false},
		{"exit 20 is not exit 2", "#!/bin/sh\nexit 20\n", false},
		{"no exit at all", "#!/bin/sh\necho hi\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scriptExits2([]byte(tt.body)); got != tt.want {
				t.Errorf("scriptExits2 = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFirstToken(t *testing.T) {
	cases := map[string]string{
		"tdd-guard":         "tdd-guard",
		"tdd-guard --check": "tdd-guard",
		"  spaced   arg":    "spaced",
		"":                  "",
		"{script}":          "{script}",
	}
	for in, want := range cases {
		if got := firstToken(in); got != want {
			t.Errorf("firstToken(%q) = %q, want %q", in, got, want)
		}
	}
}

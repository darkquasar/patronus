package main

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plugin"
)

type fakeProbe struct{ present map[string]bool }

func (f fakeProbe) HasPluginCLI(tool string) bool { return f.present[tool] }

func sp() *manifest.Plugin {
	return &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers", Version: "2.1.0"},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers", Ref: "v2.1.0"}},
		Targets: []string{"claude", "codex", "opencode"},
	}
}

func execDisplays(ds []diff.FileDiff) []string {
	var out []string
	for _, d := range ds {
		if d.Action == diff.Exec && d.Exec != nil {
			adv := "RUN"
			if d.Exec.Advisory {
				adv = "ADV"
			}
			out = append(out, adv+": "+d.Exec.Display)
		}
	}
	return out
}

func TestPluginInstallDiffsExecutedWhenCLIPresent(t *testing.T) {
	probe := fakeProbe{present: map[string]bool{"claude": true}} // codex CLI absent
	ds := pluginInstallDiffs(sp(), []string{"claude", "codex", "opencode"}, "user", probe)
	got := strings.Join(execDisplays(ds), "\n")

	// Claude CLI present -> RUN; its commands are non-advisory.
	if !strings.Contains(got, "RUN: claude plugin marketplace add claude-plugins-official") {
		t.Errorf("expected claude marketplace add as RUN, got:\n%s", got)
	}
	if !strings.Contains(got, "RUN: claude plugin install superpowers@claude-plugins-official --scope user") {
		t.Errorf("expected claude install as RUN, got:\n%s", got)
	}
	// Codex is plugin-capable but no codex source AND CLI absent -> advisory note,
	// never a RUN line.
	if strings.Contains(got, "RUN: codex") {
		t.Errorf("codex must not be RUN when its CLI is absent:\n%s", got)
	}
	// opencode has no plugin construct -> a skip/advisory line, never RUN.
	for _, line := range execDisplays(ds) {
		if strings.HasPrefix(line, "RUN: ") && strings.Contains(line, "opencode") {
			t.Errorf("opencode must never produce a RUN line:\n%s", got)
		}
	}
}

func TestPluginInstallDiffsAdvisoryWhenCLIAbsent(t *testing.T) {
	probe := fakeProbe{present: map[string]bool{}} // no CLI anywhere
	ds := pluginInstallDiffs(sp(), []string{"claude"}, "user", probe)
	for _, d := range ds {
		if d.Action == diff.Exec && d.Exec != nil && !d.Exec.Advisory {
			t.Errorf("with no CLI present, every exec must be advisory; got non-advisory %q", d.Exec.Display)
		}
	}
}

var _ plugin.CLIProbe = fakeProbe{}

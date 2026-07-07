package main

import (
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/state"
)

func TestPluginUninstallDiffsEmitsToolRemove(t *testing.T) {
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers"},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers"}},
	}
	items := []state.Item{{Artifact: "superpowers", Tool: "claude", Scope: "user"}}
	ds := pluginUninstallDiffs(p, items, fakeProbe{present: map[string]bool{"claude": true}})
	var found bool
	for _, d := range ds {
		if d.Action == diff.Exec && d.Exec != nil && !d.Exec.Advisory &&
			strings.Contains(d.Exec.Display, "claude plugin uninstall superpowers@claude-plugins-official") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a non-advisory claude uninstall exec, got %+v", ds)
	}
}

func TestPluginUninstallDiffsSkipsIncapableTool(t *testing.T) {
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers"},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers"}},
	}
	// opencode has no plugin construct -> nothing to undo, no diffs.
	items := []state.Item{{Artifact: "superpowers", Tool: "opencode", Scope: "user"}}
	if ds := pluginUninstallDiffs(p, items, fakeProbe{}); len(ds) != 0 {
		t.Errorf("opencode item must yield no uninstall diffs, got %+v", ds)
	}
}

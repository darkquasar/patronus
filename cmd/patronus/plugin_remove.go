package main

import (
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/state"
)

// pluginUninstallDiffs builds the symmetric teardown for v2-installed plugins:
// one uninstall EXEC per recorded (tool,scope), advisory when the tool's plugin
// CLI is absent. It is the inverse of pluginInstallDiffs and reuses runExecs.
func pluginUninstallDiffs(p *manifest.Plugin, items []state.Item, probe plugin.CLIProbe) []diff.FileDiff {
	var out []diff.FileDiff
	for _, it := range items {
		eco, capable := plugin.EcosystemFor(it.Tool)
		if !capable {
			continue // never installed there; nothing to undo
		}
		cmds, ok := plugin.UninstallCommands(p, it.Tool, eco, it.Scope)
		if !ok || len(cmds) == 0 {
			continue
		}
		advisory := !probe.HasPluginCLI(it.Tool)
		for _, c := range cmds {
			out = append(out, diff.FileDiff{
				Action:   diff.Exec,
				Type:     "plugin",
				Role:     string(manifest.RoleLifecycle),
				Tool:     it.Tool,
				Scope:    it.Scope,
				Artifact: p.Name,
				Path:     c.Display,
				Exec:     &diff.ExecSpec{Command: c.Argv, Display: c.Display, Advisory: advisory},
			})
		}
	}
	return out
}

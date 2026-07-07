package main

import (
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plugin"
)

// pluginInstallDiffs turns one plugin's per-tool install plan into EXEC diffs that
// ride the existing apply spine. Each tool-CLI command becomes a diff.Exec diff:
//   - tool plugin-capable AND its CLI present  -> Advisory=false (Patronus runs it)
//   - tool plugin-capable but CLI absent / no
//     source for this ecosystem                -> Advisory=true  (shown, not run)
//   - tool has no plugin construct (opencode)  -> one advisory "skipped" line
//
// The diff carries Type="plugin" + the plugin name as Artifact so state records it
// under its own identity (remove can then find + revert it).
func pluginInstallDiffs(p *manifest.Plugin, tools []string, scope string, probe plugin.CLIProbe) []diff.FileDiff {
	var out []diff.FileDiff
	for _, tool := range tools {
		eco, capable := plugin.EcosystemFor(tool)
		if !capable {
			out = append(out, pluginExec(p, tool, scope,
				"skipped: "+tool+" has no plugin system", nil, true))
			continue
		}
		cmds, ok := plugin.InstallCommands(p, tool, eco, scope)
		if !ok || len(cmds) == 0 {
			out = append(out, pluginExec(p, tool, scope,
				"skipped: no "+eco+" source for "+p.Name, nil, true))
			continue
		}
		advisory := !probe.HasPluginCLI(tool)
		for _, c := range cmds {
			out = append(out, pluginExec(p, tool, scope, c.Display, c.Argv, advisory))
		}
	}
	return out
}

// pluginExec builds one plugin EXEC diff. argv==nil yields a display-only note
// (always advisory). The Type/Role/Tool/Scope/Artifact/Version fields make the row
// render and record under the plugin's own identity.
func pluginExec(p *manifest.Plugin, tool, scope, display string, argv []string, advisory bool) diff.FileDiff {
	return diff.FileDiff{
		Action:   diff.Exec,
		Type:     "plugin",
		Role:     string(manifest.RoleLifecycle),
		Tool:     tool,
		Scope:    scope,
		Artifact: p.Name,
		Version:  p.Version,
		Path:     display, // EXEC rows key on display text in the summary table
		Exec: &diff.ExecSpec{
			Command:  argv,
			Display:  display,
			Advisory: advisory || argv == nil,
		},
	}
}

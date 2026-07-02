package plugin

import (
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
)

// Command is one tool-CLI invocation as argv (never a shell string) plus a
// human-readable Display for the dry run.
type Command struct {
	Argv    []string
	Display string
}

// toolCLI captures the per-tool plugin-CLI dialect. The install/uninstall VERBS
// differ between tools (Claude: install/uninstall; Codex: add/remove) — this
// table is the single home for that divergence.
type toolCLI struct {
	bin          string // "claude" | "codex"
	installVerb  string // "install" | "add"
	uninstallVrb string // "uninstall" | "remove"
}

var cliByTool = map[string]toolCLI{
	"claude": {bin: "claude", installVerb: "install", uninstallVrb: "uninstall"},
	"codex":  {bin: "codex", installVerb: "add", uninstallVrb: "remove"},
}

// pluginRef builds the "<plugin>@<marketplace>" identity used by both tools. It
// prefers the source's explicit Plugin name (the upstream id) over the manifest
// name, and falls back to the manifest name when Plugin is empty.
func pluginRef(p *manifest.Plugin, src manifest.PluginSource) string {
	name := src.Plugin
	if name == "" {
		name = p.Name
	}
	return name + "@" + src.Marketplace
}

func cmd(argv ...string) Command {
	return Command{Argv: argv, Display: strings.Join(argv, " ")}
}

// InstallCommands returns the ordered tool-CLI commands that register and install
// one plugin on one tool, plus ok=false when the tool has no plugin CLI dialect.
// For a marketplace source it emits `marketplace add <marketplace>` first
// (idempotent — the tool no-ops an already-registered marketplace; even the
// official marketplace needs this in a fresh config), then the install command
// using the tool's verb.
func InstallCommands(p *manifest.Plugin, tool, eco, scope string) ([]Command, bool) {
	cli, ok := cliByTool[tool]
	if !ok {
		return nil, false
	}
	src, has := p.Sources[eco]
	if !has {
		return nil, true // capable tool, but nothing to install from this ecosystem
	}
	var out []Command
	if src.Kind == "marketplace" && src.Marketplace != "" {
		out = append(out, cmd(cli.bin, "plugin", "marketplace", "add", src.Marketplace))
	}
	out = append(out, cmd(cli.bin, "plugin", cli.installVerb, pluginRef(p, src), "--scope", scope))
	return out, true
}

// UninstallCommands returns the tool-CLI commands that remove one plugin, plus
// ok=false when the tool has no plugin CLI dialect.
func UninstallCommands(p *manifest.Plugin, tool, eco, scope string) ([]Command, bool) {
	cli, ok := cliByTool[tool]
	if !ok {
		return nil, false
	}
	src, has := p.Sources[eco]
	if !has {
		return nil, true
	}
	return []Command{cmd(cli.bin, "plugin", cli.uninstallVrb, pluginRef(p, src))}, true
}

package plugin

import (
	"context"
	"os/exec"
)

// Bin returns the executable name that hosts a tool's plugin subcommand, and
// ok=false for tools without a plugin CLI dialect.
func Bin(tool string) (string, bool) {
	cli, ok := cliByTool[tool]
	if !ok {
		return "", false
	}
	return cli.bin, true
}

// CLIProbe answers whether a tool's `plugin` subcommand exists on this machine —
// the runtime fact that decides whether install is EXECUTED or merely ADVISED.
type CLIProbe interface {
	HasPluginCLI(tool string) bool
}

// ExecProbe is the production CLIProbe: it runs `<bin> plugin --help` and treats
// a zero exit as "present". A missing binary or a binary without the plugin
// subcommand (e.g. older Codex) yields false, which degrades install to advisory.
type ExecProbe struct{}

func (ExecProbe) HasPluginCLI(tool string) bool {
	bin, ok := Bin(tool)
	if !ok {
		return false
	}
	if _, err := exec.LookPath(bin); err != nil {
		return false
	}
	c := exec.CommandContext(context.Background(), bin, "plugin", "--help")
	return c.Run() == nil
}

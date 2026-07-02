package plugin

import (
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func claudePlugin() *manifest.Plugin {
	return &manifest.Plugin{
		Meta: manifest.Meta{Name: "superpowers", Version: "2.1.0"},
		Sources: map[string]manifest.PluginSource{
			"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers", Ref: "v2.1.0"},
		},
		Targets: []string{"claude", "codex"},
	}
}

func TestInstallCommandsClaude(t *testing.T) {
	cmds, ok := InstallCommands(claudePlugin(), "claude", "claude-code", "user")
	if !ok {
		t.Fatal("claude should have a CLI table entry")
	}
	want := [][]string{
		{"claude", "plugin", "marketplace", "add", "claude-plugins-official"},
		{"claude", "plugin", "install", "superpowers@claude-plugins-official", "--scope", "user"},
	}
	if len(cmds) != len(want) {
		t.Fatalf("got %d commands, want %d: %+v", len(cmds), len(want), cmds)
	}
	for i := range want {
		if !reflect.DeepEqual(cmds[i].Argv, want[i]) {
			t.Errorf("cmd[%d].Argv = %v, want %v", i, cmds[i].Argv, want[i])
		}
	}
}

func TestInstallCommandsCodexUsesAddVerb(t *testing.T) {
	p := &manifest.Plugin{
		Meta: manifest.Meta{Name: "superpowers", Version: "2.1.0"},
		Sources: map[string]manifest.PluginSource{
			"codex": {Kind: "marketplace", Marketplace: "openai-curated", Plugin: "superpowers", Ref: "v1"},
		},
		Targets: []string{"codex"},
	}
	cmds, ok := InstallCommands(p, "codex", "codex", "user")
	if !ok {
		t.Fatal("codex should have a CLI table entry")
	}
	// Codex's install verb is "add", not "install".
	got := cmds[len(cmds)-1].Argv
	want := []string{"codex", "plugin", "add", "superpowers@openai-curated", "--scope", "user"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("codex install cmd = %v, want %v", got, want)
	}
}

func TestInstallCommandsNoCLITable(t *testing.T) {
	if _, ok := InstallCommands(claudePlugin(), "opencode", "", "user"); ok {
		t.Error("opencode must have no CLI table entry (ok=false)")
	}
}

func TestUninstallCommandsVerbsDiffer(t *testing.T) {
	cc, _ := UninstallCommands(claudePlugin(), "claude", "claude-code", "user")
	if got := cc[0].Argv; !reflect.DeepEqual(got, []string{"claude", "plugin", "uninstall", "superpowers@claude-plugins-official"}) {
		t.Errorf("claude uninstall = %v", got)
	}
	p := &manifest.Plugin{
		Meta:    manifest.Meta{Name: "superpowers"},
		Sources: map[string]manifest.PluginSource{"codex": {Kind: "marketplace", Marketplace: "openai-curated", Plugin: "superpowers"}},
	}
	cx, _ := UninstallCommands(p, "codex", "codex", "user")
	if got := cx[0].Argv; !reflect.DeepEqual(got, []string{"codex", "plugin", "remove", "superpowers@openai-curated"}) {
		t.Errorf("codex uninstall = %v", got)
	}
}

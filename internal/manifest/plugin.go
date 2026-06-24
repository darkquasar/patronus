package manifest

import "fmt"

// Plugin is an upstream plugin package (Claude Code / Codex) tracked by Patronus
// as a first-class lifecycle unit. Unlike an artifact it is not decomposed into a
// shape; its identity (name + ecosystem + source) is preserved. Sources is keyed
// by ecosystem ("claude-code" | "codex") so one upstream that ships a native
// build in more than one ecosystem installs natively on each.
type Plugin struct {
	Meta     `yaml:",inline" json:",inline"`
	Sources  map[string]PluginSource `yaml:"sources" json:"sources"`
	Targets  []string                `yaml:"targets,omitempty" json:"targets,omitempty"`
	Defaults PluginDefaults          `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

// PluginSource is one ecosystem's native build of the plugin.
type PluginSource struct {
	Kind        string `yaml:"kind" json:"kind"` // marketplace | git | local
	Marketplace string `yaml:"marketplace,omitempty" json:"marketplace,omitempty"`
	Plugin      string `yaml:"plugin,omitempty" json:"plugin,omitempty"`
	Ref         string `yaml:"ref,omitempty" json:"ref,omitempty"`
	SHA         string `yaml:"sha,omitempty" json:"sha,omitempty"`
}

// PluginDefaults carries install defaults (scope).
type PluginDefaults struct {
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// LoadPlugin reads and validates a plugin manifest from disk.
func LoadPlugin(path string) (*Plugin, error) {
	var p Plugin
	if err := decodeFile(path, &p); err != nil {
		return nil, err
	}
	if err := validateMeta(p.Meta, FamilyPlugin); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &p, nil
}

// DecodePlugin validates a plugin manifest from raw bytes (https: source).
func DecodePlugin(data []byte) (*Plugin, error) {
	var p Plugin
	if err := decodeBytes(data, &p); err != nil {
		return nil, err
	}
	if err := validateMeta(p.Meta, FamilyPlugin); err != nil {
		return nil, err
	}
	return &p, nil
}

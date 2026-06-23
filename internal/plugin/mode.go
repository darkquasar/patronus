// Package plugin holds the family: plugin lifecycle logic that is independent of
// I/O — chiefly per-target install-mode resolution.
package plugin

import "github.com/darkquasar/patronus/internal/manifest"

// Mode is the per-target install disposition, derived (never authored).
type Mode string

const (
	ModeNative      Mode = "native"      // a source exists for this target's ecosystem
	ModeTranslate   Mode = "translate"   // no native source for this ecosystem; synthesize (flagged)
	ModeUnsupported Mode = "unsupported" // target has no plugin construct, or plugin has no sources
)

// EcosystemFor maps a Patronus tool name to its plugin ecosystem key. The second
// return is false for tools that have no plugin construct (opencode), which makes
// every plugin unsupported there.
func EcosystemFor(tool string) (string, bool) {
	switch tool {
	case "claude":
		return "claude-code", true
	case "codex":
		return "codex", true
	default:
		return "", false
	}
}

// ResolveMode computes the install mode for one plugin on one tool, and the
// source-ecosystem key it will install from ("" when unsupported).
func ResolveMode(p *manifest.Plugin, tool string) (Mode, string) {
	eco, ok := EcosystemFor(tool)
	if !ok {
		return ModeUnsupported, ""
	}
	if len(p.Sources) == 0 {
		return ModeUnsupported, ""
	}
	if _, has := p.Sources[eco]; has {
		return ModeNative, eco
	}
	// Cross-ecosystem: pick a deterministic source to translate from.
	// Prefer claude-code, then codex; if neither is present it is unsupported.
	for _, pref := range []string{"claude-code", "codex"} {
		if _, has := p.Sources[pref]; has {
			return ModeTranslate, pref
		}
	}
	return ModeUnsupported, ""
}

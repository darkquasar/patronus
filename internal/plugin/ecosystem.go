// Package plugin holds the plugin family's PURE logic: per-tool plugin-CLI
// command construction and installed-plugin detection. It performs no I/O and
// spawns no processes — the cmd layer injects execution and runtime probing.
package plugin

// EcosystemFor maps a Patronus tool name to its plugin ecosystem key. ok is false
// for tools with no plugin construct (opencode), which makes every plugin an
// honest skip there. This is the ONLY manifest-derived plugin distinction:
// plugin-capable (ok) vs not (skip).
func EcosystemFor(tool string) (eco string, ok bool) {
	switch tool {
	case "claude":
		return "claude-code", true
	case "codex":
		return "codex", true
	default:
		return "", false
	}
}

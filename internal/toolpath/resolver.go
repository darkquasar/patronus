// Package toolpath turns adapter detect:/layout: markers into absolute
// filesystem paths, honoring per-tool environment overrides (CODEX_HOME,
// OPENCODE_CONFIG_DIR, XDG_CONFIG_HOME) and ~ expansion. The scanner and the
// planner share this logic so detection and install target the same paths.
package toolpath

import (
	"path/filepath"
	"runtime"
	"strings"
)

// EnvLookup mirrors os.LookupEnv; injectable for tests.
type EnvLookup func(string) (string, bool)

// Scope constants used by ResolveMarker. Kept as plain strings so callers
// outside this package (and the scan package's own Scope type) interoperate
// without an import cycle.
const (
	ScopeGlobal = "global"
	ScopeLocal  = "local"
)

// Resolver resolves markers for a given environment, home, and project dir.
type Resolver struct {
	env        EnvLookup
	home       string
	projectDir string
}

// New constructs a Resolver.
func New(env EnvLookup, home, projectDir string) Resolver {
	return Resolver{env: env, home: home, projectDir: projectDir}
}

// HomeDir returns the user's home directory, preferring HOME then (on Windows)
// USERPROFILE.
func HomeDir(env EnvLookup) string {
	if h, ok := env("HOME"); ok && h != "" {
		return h
	}
	if runtime.GOOS == "windows" {
		if h, ok := env("USERPROFILE"); ok && h != "" {
			return h
		}
	}
	return ""
}

// ResolveMarker resolves a single adapter marker for the given tool/scope to an
// absolute path. Global markers may be redirected by env overrides; project
// markers are joined against the project directory. Scope is "global"|"local".
func (r Resolver) ResolveMarker(marker, tool, scope string) string {
	if scope == ScopeGlobal {
		if redirected, ok := r.redirectGlobal(marker, tool); ok {
			return redirected
		}
		return r.ExpandHome(marker)
	}
	// Project scope: relative to the project directory.
	return filepath.Join(r.projectDir, filepath.FromSlash(marker))
}

// redirectGlobal applies tool-specific base-directory env overrides to a global
// marker that lives under that tool's canonical default base.
func (r Resolver) redirectGlobal(marker, tool string) (string, bool) {
	switch tool {
	case "codex":
		if base, ok := r.env("CODEX_HOME"); ok && base != "" {
			if rest, found := underBase(marker, "~/.codex"); found {
				return filepath.Join(base, rest), true
			}
		}
	case "opencode":
		if base := r.opencodeBase(); base != "" {
			if rest, found := underBase(marker, "~/.config/opencode"); found {
				return filepath.Join(base, rest), true
			}
		}
	}
	return "", false
}

// opencodeBase returns the resolved OpenCode global config base, preferring
// OPENCODE_CONFIG_DIR, then $XDG_CONFIG_HOME/opencode.
func (r Resolver) opencodeBase() string {
	if base, ok := r.env("OPENCODE_CONFIG_DIR"); ok && base != "" {
		return base
	}
	if xdg, ok := r.env("XDG_CONFIG_HOME"); ok && xdg != "" {
		return filepath.Join(xdg, "opencode")
	}
	return ""
}

// underBase reports whether marker is the prefix path or a child of it, and if
// so returns the remainder after the prefix (relative).
func underBase(marker, prefix string) (string, bool) {
	marker = strings.TrimSuffix(marker, "/")
	prefix = strings.TrimSuffix(prefix, "/")
	if marker == prefix {
		return "", true
	}
	if strings.HasPrefix(marker, prefix+"/") {
		return filepath.FromSlash(strings.TrimPrefix(marker, prefix+"/")), true
	}
	return "", false
}

// ExpandHome replaces a leading ~/ with the resolved home directory.
func (r Resolver) ExpandHome(p string) string {
	if p == "~" {
		return r.home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(r.home, filepath.FromSlash(p[2:]))
	}
	return filepath.FromSlash(p)
}

// CollapseHome is the inverse of ExpandHome for display: an absolute path under
// the home directory is rewritten with a leading ~. Paths outside home are
// returned unchanged.
func (r Resolver) CollapseHome(p string) string {
	if r.home == "" {
		return p
	}
	if p == r.home {
		return "~"
	}
	prefix := r.home + string(filepath.Separator)
	if strings.HasPrefix(p, prefix) {
		return "~/" + filepath.ToSlash(strings.TrimPrefix(p, prefix))
	}
	return p
}

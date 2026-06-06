package scan

import (
	"path/filepath"
	"runtime"
	"strings"
)

// EnvLookup mirrors os.LookupEnv; injectable for tests.
type EnvLookup func(string) (string, bool)

// resolver turns adapter detect: markers into absolute filesystem paths,
// honoring per-tool environment overrides (CODEX_HOME, OPENCODE_CONFIG_DIR,
// XDG_CONFIG_HOME) and ~ expansion.
type resolver struct {
	env        EnvLookup
	home       string
	projectDir string
}

func newResolver(env EnvLookup, home, projectDir string) *resolver {
	return &resolver{env: env, home: home, projectDir: projectDir}
}

// homeDir returns the user's home directory, preferring HOME then (on Windows)
// USERPROFILE.
func homeDir(env EnvLookup) string {
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

// resolveMarker resolves a single adapter marker for the given tool/scope to an
// absolute path. Global markers may be redirected by env overrides; project
// markers are joined against the project directory.
func (r *resolver) resolveMarker(marker, tool string, scope Scope) string {
	if scope == ScopeGlobal {
		if redirected, ok := r.redirectGlobal(marker, tool); ok {
			return redirected
		}
		return r.expandHome(marker)
	}
	// Project scope: relative to the project directory.
	return filepath.Join(r.projectDir, filepath.FromSlash(marker))
}

// redirectGlobal applies tool-specific base-directory env overrides to a global
// marker that lives under that tool's canonical default base.
func (r *resolver) redirectGlobal(marker, tool string) (string, bool) {
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
func (r *resolver) opencodeBase() string {
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

// expandHome replaces a leading ~/ with the resolved home directory.
func (r *resolver) expandHome(p string) string {
	if p == "~" {
		return r.home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(r.home, filepath.FromSlash(p[2:]))
	}
	return filepath.FromSlash(p)
}

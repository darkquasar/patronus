// Package scan detects which AI coding tools (Claude Code, Codex, OpenCode) are
// present at global and local scope, driven by the adapters' detect: markers.
package scan

import (
	"os"
	"sort"

	"github.com/darkquasar/patronus/internal/manifest"
)

// Scope identifies global vs project-local detection.
type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeLocal  Scope = "local"
)

// Options configures a scan.
type Options struct {
	ProjectDir string              // defaults to cwd if empty
	Adapters   []*manifest.Adapter // tool definitions to detect
	Env        EnvLookup           // defaults to os.LookupEnv
}

// Inventory is the structured result of a scan.
type Inventory struct {
	ProjectDir string       `json:"projectDir"`
	Home       string       `json:"home"`
	Tools      []ToolStatus `json:"tools"`
	Env        EnvSnapshot  `json:"env"`
}

// ToolStatus reports detection for one tool across both scopes.
type ToolStatus struct {
	Tool   string     `json:"tool"`
	Global *Detection `json:"global,omitempty"`
	Local  *Detection `json:"local,omitempty"`
}

// Detection is the per-scope outcome for one tool.
type Detection struct {
	Scope        Scope    `json:"scope"`
	Detected     bool     `json:"detected"`
	MatchedPaths []string `json:"matchedPaths"`
}

// EnvSnapshot records the env overrides that influenced path resolution.
type EnvSnapshot struct {
	CodexHome         string `json:"codexHome,omitempty"`
	OpencodeConfigDir string `json:"opencodeConfigDir,omitempty"`
	XDGConfigHome     string `json:"xdgConfigHome,omitempty"`
}

// Scan detects each adapter's tool at global and local scope. Detection is
// positive when ANY of the tool's markers for that scope exists on disk.
//
// Phase 1 performs path-marker detection only.
// TODO(phase8): content-signature heuristics (stray config.toml with
// [mcp_servers.*], opencode.json with $schema: opencode.ai, non-standard
// skills/<x>/SKILL.md trees).
func Scan(opts Options) (*Inventory, error) {
	env := opts.Env
	if env == nil {
		env = os.LookupEnv
	}
	projectDir := opts.ProjectDir
	if projectDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectDir = wd
	}

	home := homeDir(env)
	res := newResolver(env, home, projectDir)

	inv := &Inventory{
		ProjectDir: projectDir,
		Home:       home,
		Env:        envSnapshot(env),
	}

	for _, ad := range opts.Adapters {
		inv.Tools = append(inv.Tools, ToolStatus{
			Tool:   ad.Tool,
			Global: detectScope(res, ad.Tool, ScopeGlobal, ad.Detect.Global),
			Local:  detectScope(res, ad.Tool, ScopeLocal, ad.Detect.Project),
		})
	}
	sort.Slice(inv.Tools, func(i, j int) bool { return inv.Tools[i].Tool < inv.Tools[j].Tool })
	return inv, nil
}

func envSnapshot(env EnvLookup) EnvSnapshot {
	get := func(k string) string { v, _ := env(k); return v }
	return EnvSnapshot{
		CodexHome:         get("CODEX_HOME"),
		OpencodeConfigDir: get("OPENCODE_CONFIG_DIR"),
		XDGConfigHome:     get("XDG_CONFIG_HOME"),
	}
}

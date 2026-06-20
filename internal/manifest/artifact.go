package manifest

import "fmt"

// Artifact is an authored-in-repo, portable installable (§5). It is the ONLY
// family with a declared file Type — the same files + role could be a skill or a
// command, so Type is the only signal and it drives the write action.
type Artifact struct {
	Meta        `yaml:",inline" json:",inline"`
	Type        ArtifactType                      `yaml:"type" json:"type"`                       // skill | agent | command | hook | instruction | output-style
	Entry       string                            `yaml:"entry,omitempty" json:"entry,omitempty"` // body file; omitted for Hook
	Files       []string                          `yaml:"files,omitempty" json:"files,omitempty"` // supporting dirs copied verbatim
	Targets     []string                          `yaml:"targets" json:"targets"`
	Defaults    ArtifactDefaults                  `yaml:"defaults" json:"defaults"`
	Overrides   map[string]map[string]interface{} `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Attribution *Attribution                      `yaml:"attribution,omitempty" json:"attribution,omitempty"` // set on vendored content
	Hook        *HookSpec                         `yaml:"hook,omitempty" json:"hook,omitempty"`               // required for Type==hook; the event/matcher/command to register
}

// HookSpec is the declarative definition of a hook artifact (Type==hook). Rather
// than parsing a hook out of a body file, the event/matcher/command are
// structured data the adapter merges into the agent's settings at the layout's
// hooks.{event} path, and the (matcher, command) pair is the identity the
// planner and remove path use to register exactly one array element and pull it
// back out — so two hooks on one event coexist and revert independently.
type HookSpec struct {
	Event   string `yaml:"event" json:"event"`                         // e.g. PreToolUse | SessionStart
	Matcher string `yaml:"matcher,omitempty" json:"matcher,omitempty"` // tool/glob filter; "" means "all" (omitted)
	Command string `yaml:"command" json:"command"`                     // the shell command the hook runs; may contain {script} when Script is set
	Script  string `yaml:"script,omitempty" json:"script,omitempty"`   // optional bundled helper script (a files: entry) PLACED in the tool's hook-script dir; {script} in Command resolves to its installed path
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`       // hook handler type; defaults to "command"
	Timeout int    `yaml:"timeout,omitempty" json:"timeout,omitempty"` // seconds; omitted when zero (tool default)
}

// Attribution records the upstream provenance of vendored (de-vendored) artifact
// content (§3). It rides along in the catalog metadata and the canonically
// re-marshalled patronus.yaml so a source/commit note is always discoverable; the
// human-readable license + copyright also ships as a NOTICE file in the artifact
// folder. Present only on artifacts whose body is sourced from a permissive
// upstream; absent for original in-repo content.
type Attribution struct {
	Upstream  string `yaml:"upstream" json:"upstream"`                 // e.g. github.com/ciembor/agent-rules-books
	License   string `yaml:"license" json:"license"`                   // SPDX id, e.g. MIT
	Copyright string `yaml:"copyright" json:"copyright"`               // e.g. "Copyright (c) 2026 Maciej Ciemborowicz"
	Commit    string `yaml:"commit,omitempty" json:"commit,omitempty"` // pinned upstream commit the content was taken at
	Note      string `yaml:"note,omitempty" json:"note,omitempty"`     // caveats (e.g. "inspired by, not reproductions of, the source books")
}

// Header returns the artifact's shared identity header (implements Installable).
func (a *Artifact) Header() Meta { return a.Meta }

// ArtifactDefaults holds install-time defaults the user may override.
type ArtifactDefaults struct {
	Scope string `yaml:"scope" json:"scope"` // project | global
}

// LoadArtifact reads and validates an artifact patronus.yaml.
func LoadArtifact(path string) (*Artifact, error) {
	var a Artifact
	if err := decodeFile(path, &a); err != nil {
		return nil, err
	}
	return finishArtifact(&a)
}

// DecodeArtifact parses+validates an artifact manifest from raw YAML bytes — used
// for an https: sourced manifest that never lands on a local path.
func DecodeArtifact(data []byte) (*Artifact, error) {
	var a Artifact
	if err := decodeBytes(data, &a); err != nil {
		return nil, err
	}
	return finishArtifact(&a)
}

func finishArtifact(a *Artifact) (*Artifact, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return a, nil
}

// Validate performs Phase-1-light checks: schema version, family, a valid
// artifact type, and the universally-required identity fields.
func (a *Artifact) Validate() error {
	if err := validateMeta(a.Meta, FamilyArtifact); err != nil {
		return err
	}
	if !artifactTypes[a.Type] {
		return fmt.Errorf("invalid artifact type %q", a.Type)
	}
	if a.Description == "" {
		return fmt.Errorf("missing description")
	}
	if a.Type == TypeHook {
		if a.Hook == nil {
			return fmt.Errorf("hook artifact %q: missing hook block", a.Name)
		}
		if a.Hook.Event == "" || a.Hook.Command == "" {
			return fmt.Errorf("hook artifact %q: hook requires event and command", a.Name)
		}
	}
	if a.Attribution != nil {
		if a.Attribution.Upstream == "" || a.Attribution.License == "" || a.Attribution.Copyright == "" {
			return fmt.Errorf("attribution requires upstream, license, and copyright")
		}
	}
	return nil
}

package manifest

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Profile is a curated bundle selecting items across §1A layers (§5). It expands
// into other installables; it carries NO type (computed "expansion") and no
// delivery/wire. Its role is lifecycle (L11) by nature.
type Profile struct {
	Meta    `yaml:",inline" json:",inline"`
	Summary string        `yaml:"summary,omitempty" json:"summary,omitempty"`
	Status  string        `yaml:"status,omitempty" json:"status,omitempty"` // stub | (populated)
	Extends string        `yaml:"extends,omitempty" json:"extends,omitempty"`
	Layers  ProfileLayers `yaml:"layers" json:"layers"`
	Todo    []string      `yaml:"todo,omitempty" json:"todo,omitempty"`
}

// Header returns the profile's shared identity header (implements Installable).
func (p *Profile) Header() Meta { return p.Meta }

// ProfileLayers maps each §1A layer to its selected item(s). Memory is the lone
// scalar (one memory recipe per environment); the rest are lists. Sandbox is a
// list too — not because a tool runs several sandboxes, but so ONE profile can
// carry per-tool flavours (native@claude/@codex + sandbox-runtime@opencode) and
// the resolver picks the one matching --tool.
type ProfileLayers struct {
	Instructions  StringList `yaml:"instructions,omitempty" json:"instructions,omitempty"`
	Capabilities  StringList `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Context       StringList `yaml:"context,omitempty" json:"context,omitempty"`
	Tools         StringList `yaml:"tools,omitempty" json:"tools,omitempty"`
	Memory        string     `yaml:"memory,omitempty" json:"memory,omitempty"`
	Sandbox       StringList `yaml:"sandbox,omitempty" json:"sandbox,omitempty"` // list so one profile can flavour the L6 recipe per tool (native@claude/@codex vs sandbox-runtime@opencode)
	Observability StringList `yaml:"observability,omitempty" json:"observability,omitempty"`
	Eval          StringList `yaml:"eval,omitempty" json:"eval,omitempty"`
	Guardrails    StringList `yaml:"guardrails,omitempty" json:"guardrails,omitempty"`
}

// StringList accepts either a single scalar or a YAML sequence, decoding both to
// a []string — robust against authoring slips like `capabilities: foo`.
type StringList []string

// UnmarshalYAML implements yaml.Unmarshaler for the scalar-or-sequence shape.
func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = StringList{value.Value}
		return nil
	case yaml.SequenceNode:
		var out []string
		if err := value.Decode(&out); err != nil {
			return err
		}
		*s = out
		return nil
	default:
		return fmt.Errorf("expected scalar or sequence for string list, got yaml kind %d", value.Kind)
	}
}

// LoadProfile reads and validates a profile manifest.
func LoadProfile(path string) (*Profile, error) {
	var p Profile
	if err := decodeFile(path, &p); err != nil {
		return nil, err
	}
	if err := validateMeta(p.Meta, FamilyProfile); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &p, nil
}

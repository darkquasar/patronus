package manifest

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Profile is a curated bundle selecting items across §1A layers (§5d).
type Profile struct {
	APIVersion string        `yaml:"apiVersion" json:"apiVersion"`
	Kind       Kind          `yaml:"kind" json:"kind"`
	Name       string        `yaml:"name" json:"name"`
	Summary    string        `yaml:"summary" json:"summary"`
	Status     string        `yaml:"status,omitempty" json:"status,omitempty"` // stub | (populated)
	Extends    string        `yaml:"extends,omitempty" json:"extends,omitempty"`
	Layers     ProfileLayers `yaml:"layers" json:"layers"`
	Todo       []string      `yaml:"todo,omitempty" json:"todo,omitempty"`
}

// ProfileLayers maps each §1A layer to its selected item(s). Single-recipe
// layers (memory, sandbox) are scalars; the rest are lists.
type ProfileLayers struct {
	Instructions  StringList `yaml:"instructions,omitempty" json:"instructions,omitempty"`
	Capabilities  StringList `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Context       StringList `yaml:"context,omitempty" json:"context,omitempty"`
	Tools         StringList `yaml:"tools,omitempty" json:"tools,omitempty"`
	Memory        string     `yaml:"memory,omitempty" json:"memory,omitempty"`
	Sandbox       string     `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`
	Observability StringList `yaml:"observability,omitempty" json:"observability,omitempty"`
	Harness       StringList `yaml:"harness,omitempty" json:"harness,omitempty"`
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
	if p.APIVersion != APIVersion {
		return nil, fmt.Errorf("%s: unexpected apiVersion %q (want %q)", path, p.APIVersion, APIVersion)
	}
	if p.Kind != KindProfile {
		return nil, fmt.Errorf("%s: expected kind Profile, got %q", path, p.Kind)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("%s: missing name", path)
	}
	return &p, nil
}

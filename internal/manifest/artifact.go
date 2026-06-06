package manifest

import "fmt"

// Artifact is an authored-in-repo, portable installable (§5a). Two orthogonal
// axes describe it: Kind (on-disk shape) and Role (job / §1A layer).
type Artifact struct {
	APIVersion  string                            `yaml:"apiVersion" json:"apiVersion"`
	Kind        Kind                              `yaml:"kind" json:"kind"`
	Role        Role                              `yaml:"role,omitempty" json:"role,omitempty"`
	Name        string                            `yaml:"name" json:"name"`
	Description string                            `yaml:"description" json:"description"`
	Version     string                            `yaml:"version" json:"version"`
	Entry       string                            `yaml:"entry,omitempty" json:"entry,omitempty"` // body file; omitted for Hook
	Files       []string                          `yaml:"files,omitempty" json:"files,omitempty"` // supporting dirs copied verbatim
	Targets     []string                          `yaml:"targets" json:"targets"`
	Defaults    ArtifactDefaults                  `yaml:"defaults" json:"defaults"`
	Overrides   map[string]map[string]interface{} `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

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
	if a.Role == "" {
		a.Role = DefaultRole(a.Kind)
	}
	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &a, nil
}

// Validate performs Phase-1-light checks: schema version, a valid artifact kind,
// and the universally-required identity fields.
func (a *Artifact) Validate() error {
	if a.APIVersion != APIVersion {
		return fmt.Errorf("unexpected apiVersion %q (want %q)", a.APIVersion, APIVersion)
	}
	if !artifactKinds[a.Kind] {
		return fmt.Errorf("invalid artifact kind %q", a.Kind)
	}
	if a.Name == "" {
		return fmt.Errorf("missing name")
	}
	if a.Description == "" {
		return fmt.Errorf("missing description")
	}
	return nil
}

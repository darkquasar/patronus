package manifest

import "fmt"

// Artifact is an authored-in-repo, portable installable (§5). It is the ONLY
// family with a declared file Type — the same files + role could be a skill or a
// command, so Type is the only signal and it drives the write action.
type Artifact struct {
	Meta      `yaml:",inline" json:",inline"`
	Type      ArtifactType                      `yaml:"type" json:"type"`                       // skill | agent | command | hook | instruction
	Entry     string                            `yaml:"entry,omitempty" json:"entry,omitempty"` // body file; omitted for Hook
	Files     []string                          `yaml:"files,omitempty" json:"files,omitempty"` // supporting dirs copied verbatim
	Targets   []string                          `yaml:"targets" json:"targets"`
	Defaults  ArtifactDefaults                  `yaml:"defaults" json:"defaults"`
	Overrides map[string]map[string]interface{} `yaml:"overrides,omitempty" json:"overrides,omitempty"`
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
	return nil
}

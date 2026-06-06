package manifest

import "fmt"

// Adapter declares how one tool wants artifacts laid out and how the scanner IDs
// it (§5b). Phase 1 consumes only Tool + Detect; Layout is held opaque for the
// Phase 2 transform engine.
type Adapter struct {
	APIVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       Kind                   `yaml:"kind,omitempty"`
	Tool       string                 `yaml:"tool"`
	Detect     AdapterDetect          `yaml:"detect"`
	Layout     map[string]interface{} `yaml:"layout,omitempty"`
}

// AdapterDetect holds the positive-ID markers for each scope.
type AdapterDetect struct {
	Global  []string `yaml:"global"`
	Project []string `yaml:"project"`
}

// LoadAdapter reads and validates an adapter definition.
func LoadAdapter(path string) (*Adapter, error) {
	var a Adapter
	if err := decodeFile(path, &a); err != nil {
		return nil, err
	}
	if a.Tool == "" {
		return nil, fmt.Errorf("%s: missing tool", path)
	}
	return &a, nil
}

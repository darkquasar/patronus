package manifest

import "fmt"

// Recipe is an external binary/tool to fetch, verify, and wire into each agent
// (§5c). Phase 1 parses recipes for display only; fetch/wire fields are held,
// not acted on.
type Recipe struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       Kind         `yaml:"kind" json:"kind"`
	Name       string       `yaml:"name" json:"name"`
	Capability string       `yaml:"capability" json:"capability"` // memory | tools | sandbox | observability | context | harness
	Summary    string       `yaml:"summary" json:"summary"`
	Upstream   string       `yaml:"upstream,omitempty" json:"upstream,omitempty"`
	License    string       `yaml:"license,omitempty" json:"license,omitempty"`
	Delivery   *Delivery    `yaml:"delivery,omitempty" json:"delivery,omitempty"` // nil for wire-only remote MCP
	Scope      *RecipeScope `yaml:"scope,omitempty" json:"scope,omitempty"`
	Wire       Wire         `yaml:"wire" json:"wire"`
	Seed       []string     `yaml:"seed,omitempty" json:"seed,omitempty"`
}

// Delivery describes how the recipe's binary is obtained (§2c).
type Delivery struct {
	Primary   string                 `yaml:"primary" json:"primary"`                         // github-release | docker | cargo | ...
	Fallbacks map[string]interface{} `yaml:"fallbacks,omitempty" json:"fallbacks,omitempty"` // values mix bool(false) and string refs
	InstallTo string                 `yaml:"installTo,omitempty" json:"installTo,omitempty"`
	Assets    []interface{}          `yaml:"assets,omitempty" json:"assets,omitempty"`
}

// RecipeScope captures per-repo isolation markers and the global store location.
type RecipeScope struct {
	Marker string `yaml:"marker,omitempty" json:"marker,omitempty"`
	Global string `yaml:"global,omitempty" json:"global,omitempty"`
}

// Wire describes how the recipe is bound to each agent.
type Wire struct {
	SelfWiring  bool     `yaml:"selfWiring,omitempty" json:"selfWiring,omitempty"`
	PostInstall []string `yaml:"postInstall,omitempty" json:"postInstall,omitempty"`
	Tools       []string `yaml:"tools,omitempty" json:"tools,omitempty"`
	Mcp         *WireMcp `yaml:"mcp,omitempty" json:"mcp,omitempty"`
}

// WireMcp is the MCP-config entry Patronus merges for a non-self-wiring recipe.
type WireMcp struct {
	Transport string   `yaml:"transport" json:"transport"` // http | stdio
	URL       string   `yaml:"url,omitempty" json:"url,omitempty"`
	Command   string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args      []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// LoadRecipe reads and validates a recipe manifest.
func LoadRecipe(path string) (*Recipe, error) {
	var r Recipe
	if err := decodeFile(path, &r); err != nil {
		return nil, err
	}
	if r.APIVersion != APIVersion {
		return nil, fmt.Errorf("%s: unexpected apiVersion %q (want %q)", path, r.APIVersion, APIVersion)
	}
	if r.Kind != KindRecipe {
		return nil, fmt.Errorf("%s: expected kind Recipe, got %q", path, r.Kind)
	}
	if r.Name == "" {
		return nil, fmt.Errorf("%s: missing name", path)
	}
	if r.Capability == "" {
		return nil, fmt.Errorf("%s: missing capability", path)
	}
	return &r, nil
}

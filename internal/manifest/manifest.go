// Package manifest parses Patronus manifest files: artifact patronus.yaml,
// recipe and profile manifests, and per-tool adapter definitions. It is
// parse-and-validate only — it performs no install/transform actions.
//
// Schema v2 describes every installable along three orthogonal axes:
//
//   - family — how it's delivered + installed (the dispatch discriminator):
//     artifact | recipe | profile (adapters carry family: adapter but are not
//     installable).
//   - type   — its on-disk shape: DECLARED for artifacts (skill | agent |
//     command | hook | instruction), COMPUTED for recipes (from deliver × wire)
//     and profiles (always "expansion"). Only Artifact carries a type field.
//   - role   — which layer it fills: universal, every installable declares one.
package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// APIVersion is the only manifest schema version Patronus understands today.
const APIVersion = "patronus/v2"

// Family is the dispatch discriminator: how an installable is delivered and
// installed. It is the single field the loader switches on to pick the concrete
// struct. Closed set of three installable families, plus adapter (which carries
// the field but is not an Installable).
type Family string

const (
	FamilyArtifact Family = "artifact" // authored in-repo; transformed per tool
	FamilyRecipe   Family = "recipe"   // external; delivered and/or wired
	FamilyProfile  Family = "profile"  // a bundle that expands into installables
	FamilyAdapter  Family = "adapter"  // per-tool layout def (not installable)
)

// ArtifactType is the on-disk shape of an artifact — a CLOSED set that drives
// the write action (skill→CREATE skills/{name}/, hook→MERGE settings,
// instruction→APPEND). Declared only on artifacts; recipes/profiles compute
// their shape from structure (see Recipe.Shape), so no type field can drift.
type ArtifactType string

const (
	TypeSkill       ArtifactType = "skill"
	TypeAgent       ArtifactType = "agent"
	TypeCommand     ArtifactType = "command"
	TypeHook        ArtifactType = "hook"
	TypeInstruction ArtifactType = "instruction"
)

// artifactTypes is the valid set for an artifact's `type:` field.
var artifactTypes = map[ArtifactType]bool{
	TypeSkill: true, TypeAgent: true, TypeCommand: true,
	TypeHook: true, TypeInstruction: true,
}

// Role is the layer an installable fills — universal across all three families,
// and role names ARE layer names so the summary table's column is one axis.
// Open set; the active values are declared in the catalog today, the rest are
// reserved.
type Role string

const (
	RoleInstruction   Role = "instruction"   // L1 — Instructions / Identity
	RoleCapability    Role = "capability"    // L2 — Capabilities
	RoleMemory        Role = "memory"        // L3 — Memory
	RoleContext       Role = "context"       // L4 — Context / Knowledge (was "pattern")
	RoleTools         Role = "tools"         // L5 — Tools / Integrations
	RoleSandbox       Role = "sandbox"       // L6 — Sandbox / Execution safety
	RoleObservability Role = "observability" // L7 — reserved
	RoleEval          Role = "eval"          // L8 — Evaluation (eval suites, test/lint/typecheck loops, CI gates); reserved
	RoleGuardrail     Role = "guardrail"     // L9 — reserved
	RoleOrchestration Role = "orchestration" // L10 — reserved
	RoleLifecycle     Role = "lifecycle"     // L11 — Lifecycle (profiles are L11 by nature)
)

// Meta is the shared identity header every installable embeds — the one and only
// definition of Family and Role. Artifacts add a declared Type; recipes/profiles
// carry no shape field (it is computed).
type Meta struct {
	APIVersion  string `yaml:"apiVersion" json:"apiVersion"`
	Family      Family `yaml:"family" json:"family"`
	Role        Role   `yaml:"role,omitempty" json:"role,omitempty"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Installable is implemented by all three installable families (Artifact,
// Recipe, Profile). The loader dispatches on Header().Family.
//
// Note: the accessor is Header() rather than Meta() because each family EMBEDS
// the Meta struct (so Family/Role/Name/… promote onto the value), and Go forbids
// a method whose name collides with an embedded field. Header() exposes the same
// struct through the interface.
type Installable interface {
	Header() Meta
}

// validateMeta checks the fields common to every installable: the schema
// version, the expected family, and the universally-required identity fields.
func validateMeta(m Meta, want Family) error {
	if m.APIVersion != APIVersion {
		return fmt.Errorf("unexpected apiVersion %q (want %q)", m.APIVersion, APIVersion)
	}
	if m.Family != want {
		return fmt.Errorf("expected family %q, got %q", want, m.Family)
	}
	if m.Name == "" {
		return fmt.Errorf("missing name")
	}
	return nil
}

// decodeFile reads path and YAML-decodes it into v leniently (unknown fields are
// ignored, keeping the parser forward-compatible with later schema additions).
func decodeFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := decodeBytes(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// decodeBytes YAML-decodes data into v with the same lenient rules as decodeFile.
// It is the seam embedded adapters and out-of-tree (https:) manifests parse
// through, where the bytes don't come from a local path.
func decodeBytes(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

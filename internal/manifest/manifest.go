// Package manifest parses Patronus manifest files: artifact patronus.yaml,
// recipe and profile manifests, and per-tool adapter definitions. It is
// parse-and-validate only — it performs no install/transform actions.
package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// APIVersion is the only manifest schema version Patronus understands today.
const APIVersion = "patronus/v1"

// Kind is the on-disk shape of an installable (for artifacts, a CLOSED set) plus
// the Recipe/Profile discriminators used by their respective manifest files.
type Kind string

const (
	// Artifact kinds (§5a closed set).
	KindSkill       Kind = "Skill"
	KindAgent       Kind = "Agent"
	KindCommand     Kind = "Command"
	KindHook        Kind = "Hook"
	KindInstruction Kind = "Instruction"

	// Manifest discriminators for the other two installable families.
	KindRecipe  Kind = "Recipe"
	KindProfile Kind = "Profile"
	KindAdapter Kind = "Adapter"
)

// artifactKinds is the valid set for an artifact's `kind:` field.
var artifactKinds = map[Kind]bool{
	KindSkill: true, KindAgent: true, KindCommand: true,
	KindHook: true, KindInstruction: true,
}

// Role is the job an artifact does — which §1A layer it fills and which profile
// slot it lands in. Open set; two roles are active today, the rest reserved.
type Role string

const (
	RoleCapability  Role = "capability"  // L2 — active
	RolePattern     Role = "pattern"     // L4 — active
	RoleGuardrail   Role = "guardrail"   // L9 — reserved
	RoleHarness     Role = "harness"     // L8 — reserved
	RoleInstruction Role = "instruction" // L1 — reserved
)

// DefaultRole encodes the §5a "Default for kind" mapping, used when an artifact
// omits its `role:`.
func DefaultRole(k Kind) Role {
	switch k {
	case KindSkill, KindAgent, KindCommand:
		return RoleCapability
	case KindHook:
		return RoleGuardrail
	case KindInstruction:
		return RoleInstruction
	default:
		return RoleCapability
	}
}

// Capability returns the human-facing "what does installing this add" label for
// an artifact kind/role, used in the dry-run summary table's capability column.
// A pattern-role Skill reads as "pattern" rather than the generic "skill".
func Capability(k Kind, r Role) string {
	if k == KindSkill && r == RolePattern {
		return "pattern"
	}
	switch k {
	case KindSkill:
		return "skill"
	case KindAgent:
		return "agent"
	case KindCommand:
		return "command"
	case KindHook:
		return "hook"
	case KindInstruction:
		return "instruction"
	default:
		return string(k)
	}
}

// decodeFile reads path and YAML-decodes it into v leniently (unknown fields are
// ignored, keeping the parser forward-compatible with later schema additions).
func decodeFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

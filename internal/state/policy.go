package state

// legacyStructuralTools is the zero-value default of OwnershipPolicy: the two
// targets that required structural ownership before the harness registry.
func legacyStructuralTools() []string { return []string{"codex", "pi"} }

// OwnershipPolicy says which tools own structural effects: their SettingEdit
// rows are validated on load and their section checksums refresh after a
// verified Unappend. The zero value reproduces today's behaviour; the registry
// derives the production value from descriptor flags (internal/harness cannot
// be imported here).
type OwnershipPolicy struct {
	structural []string
	explicit   bool
}

// NewOwnershipPolicy returns a policy naming exactly structuralTools (copied);
// with no arguments no tool is structural.
func NewOwnershipPolicy(structuralTools ...string) OwnershipPolicy {
	return OwnershipPolicy{structural: append([]string(nil), structuralTools...), explicit: true}
}

// Structural reports whether tool owns structural effects under p.
func (p OwnershipPolicy) Structural(tool string) bool {
	tools := p.structural
	if !p.explicit {
		tools = legacyStructuralTools()
	}
	for _, t := range tools {
		if t == tool {
			return true
		}
	}
	return false
}

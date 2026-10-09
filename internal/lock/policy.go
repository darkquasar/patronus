package lock

// legacyTargetedTools is the zero-value default of TargetPolicy: the one
// target FromResolved stamped before the harness registry (Codex received its
// stamp from the lock command instead).
func legacyTargetedTools() []string { return []string{"pi"} }

// TargetPolicy says which resolved targets stamp the target-bearing lock schema
// (TargetVersion plus Target). The zero value reproduces today's behaviour; the
// registry derives the production value from LockTargeted (internal/harness
// cannot be imported here).
type TargetPolicy struct {
	targeted []string
	explicit bool
}

// NewTargetPolicy returns a policy naming exactly targeted (copied); with no
// arguments no target is stamped.
func NewTargetPolicy(targeted ...string) TargetPolicy {
	return TargetPolicy{targeted: append([]string(nil), targeted...), explicit: true}
}

// Targeted reports whether target stamps the lock under p.
func (p TargetPolicy) Targeted(target string) bool {
	tools := p.targeted
	if !p.explicit {
		tools = legacyTargetedTools()
	}
	for _, t := range tools {
		if t == target {
			return true
		}
	}
	return false
}

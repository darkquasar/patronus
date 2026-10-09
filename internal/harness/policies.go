package harness

import (
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/state"
)

// OwnershipPolicy derives the structural-ownership policy from the descriptors
// that own effects (OwnsEffects), in registration order. A nil registry yields
// a policy naming no tool.
func (r *Registry) OwnershipPolicy() state.OwnershipPolicy {
	if r == nil {
		return state.NewOwnershipPolicy()
	}
	var tools []string
	for _, name := range r.order {
		if r.byName[name].OwnsEffects {
			tools = append(tools, name)
		}
	}
	return state.NewOwnershipPolicy(tools...)
}

// TargetPolicy derives the lock-stamping policy from the descriptors with
// LockTargeted, in registration order (stamping only; coexistence is
// LockCoexistence on LockTarget).
func (r *Registry) TargetPolicy() lock.TargetPolicy {
	if r == nil {
		return lock.NewTargetPolicy()
	}
	var tools []string
	for _, name := range r.order {
		if r.byName[name].LockTargeted {
			tools = append(tools, name)
		}
	}
	return lock.NewTargetPolicy(tools...)
}

// SectionPolicy derives the section-composition policy from the descriptors
// that own effects: one rule per OwnsEffects descriptor carrying its
// SectionPrefix and StrictSections.
func (r *Registry) SectionPolicy() remove.SectionPolicy {
	if r == nil {
		return remove.NewSectionPolicy()
	}
	var rules []remove.SectionRule
	for _, name := range r.order {
		d := r.byName[name]
		if d.OwnsEffects {
			rules = append(rules, remove.SectionRule{Tool: name, Prefix: d.SectionPrefix, Structural: true, Strict: d.StrictSections})
		}
	}
	return remove.NewSectionPolicy(rules...)
}

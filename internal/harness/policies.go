package harness

import "github.com/darkquasar/patronus/internal/state"

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

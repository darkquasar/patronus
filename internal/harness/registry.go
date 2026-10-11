package harness

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
)

// Registry holds the registered descriptors in registration order. Callers
// receive copies; the registry is never mutated after construction.
type Registry struct {
	byName map[string]Descriptor
	order  []string
}

// NewRegistry copies each descriptor's slices and rejects a zero Name, a
// repeated Name and a zero ResourcePhase on a descriptor with SkillRoots.
func NewRegistry(ds ...Descriptor) (*Registry, error) {
	r := &Registry{byName: make(map[string]Descriptor, len(ds))}
	for _, d := range ds {
		if d.Name == "" {
			return nil, ErrZeroName
		}
		if _, dup := r.byName[d.Name]; dup {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateDescriptor, d.Name)
		}
		if d.SkillRoots != nil && d.ResourcePhase == 0 {
			return nil, fmt.Errorf("%w: %s", ErrZeroResourcePhase, d.Name)
		}
		r.byName[d.Name] = copyDescriptor(d)
		r.order = append(r.order, d.Name)
	}
	return r, nil
}

func copyDescriptor(d Descriptor) Descriptor {
	d.Profiles = copyStrings(d.Profiles)
	d.ShadowFiles = copyStrings(d.ShadowFiles)
	return d
}

func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s...)
}

// Lookup returns a copy of the named descriptor.
func (r *Registry) Lookup(name string) (Descriptor, bool) {
	if r == nil {
		return Descriptor{}, false
	}
	d, ok := r.byName[name]
	if !ok {
		return Descriptor{}, false
	}
	return copyDescriptor(d), true
}

// ValidateTarget accepts the unset and all sentinels and every registered
// name; the message lists the sorted names and then all.
func (r *Registry) ValidateTarget(target string) error {
	if r == nil {
		return ErrNilRegistry
	}
	if target == TargetUnset || target == TargetAll {
		return nil
	}
	if _, ok := r.byName[target]; ok {
		return nil
	}
	return fmt.Errorf("unknown target %q; use %s|%s", target, strings.Join(r.sortedNames(), "|"), TargetAll)
}

func (r *Registry) sortedNames() []string {
	names := copyStrings(r.order)
	sort.Strings(names)
	return names
}

// ProfileTarget enforces the profile-to-target binding: the union of every
// descriptor that lists profile and the profile's own declared targets. An
// empty union admits any target; otherwise target must be a member.
func (r *Registry) ProfileTarget(profile string, declared []string, target string) error {
	if r == nil {
		return ErrNilRegistry
	}
	var admitted []string
	for _, name := range r.order {
		if contains(r.byName[name].Profiles, profile) && !contains(admitted, name) {
			admitted = append(admitted, name)
		}
	}
	for _, name := range declared {
		if !contains(admitted, name) {
			admitted = append(admitted, name)
		}
	}
	if len(admitted) == 0 || contains(admitted, target) {
		return nil
	}
	return fmt.Errorf("%s requires --target %s (got %q)", profile, strings.Join(admitted, "|"), target)
}

// LockTarget applies the lock coexistence rule only when the prior or the
// requested target names a LockCoexistence descriptor: a differing recorded
// target refuses, and a legacy lock without a target cannot authorize crossing.
func (r *Registry) LockTarget(prior *lock.Lock, target string) error {
	if r == nil {
		return ErrNilRegistry
	}
	if !r.coexistence(prior.Target) && !r.coexistence(target) {
		return nil
	}
	if prior.Target != "" && prior.Target != target {
		return fmt.Errorf("existing lock target %s conflicts with --target %s; use separate project roots or explicit migration", prior.Target, target)
	}
	if prior.Target == "" && (prior.Profile != "" || len(prior.Entries) > 0) {
		return fmt.Errorf("existing legacy lock target is unknown; preserve patronus.lock and reconcile provenance before Codex migration")
	}
	return nil
}

func (r *Registry) coexistence(name string) bool {
	d, ok := r.byName[name]
	return ok && d.LockCoexistence
}

// Descriptors returns independent descriptor copies in registration order.
func (r *Registry) Descriptors() []Descriptor {
	if r == nil {
		return nil
	}
	var out []Descriptor
	for _, name := range r.order {
		out = append(out, copyDescriptor(r.byName[name]))
	}
	return out
}

// Targeted returns the descriptors that stamp the lock or own structural
// effects, in registration order.
func (r *Registry) Targeted() []Descriptor {
	if r == nil {
		return nil
	}
	var out []Descriptor
	for _, name := range r.order {
		d := r.byName[name]
		if d.LockTargeted || d.OwnsEffects {
			out = append(out, copyDescriptor(d))
		}
	}
	return out
}

// Selected returns the descriptors named by at least one diff's tool
// (composite tools count each member), in registration order.
func (r *Registry) Selected(cs *diff.ChangeSet) []Descriptor {
	if r == nil || cs == nil {
		return nil
	}
	named := map[string]bool{}
	for _, d := range cs.Diffs {
		for _, tool := range strings.Split(d.Tool, "+") {
			named[tool] = true
		}
	}
	var out []Descriptor
	for _, name := range r.order {
		if named[name] {
			out = append(out, copyDescriptor(r.byName[name]))
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

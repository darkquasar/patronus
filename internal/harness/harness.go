// Package harness describes install targets as data consumed by the shared
// spine. A Registry is constructed explicitly in main and injected; there is
// no package-level registry, no init registration and no mutable global. The
// shared rules read descriptor flags and never branch on a target name.
package harness

import (
	"errors"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

const (
	// TargetUnset is the empty --target value every command accepts.
	TargetUnset = ""
	// TargetAll selects the tool-agnostic baseline.
	TargetAll = "all"
)

var (
	ErrDuplicateDescriptor = errors.New("harness: duplicate descriptor")
	ErrZeroName            = errors.New("harness: descriptor has no name")
	ErrZeroResourcePhase   = errors.New("harness: descriptor with skill roots has no resource phase")
	ErrUnknownTarget       = errors.New("harness: unknown target")
	ErrNilRegistry         = errors.New("harness: nil registry")
)

// ResourcePhase says where Admit runs the shared same-name and legacy resource
// rules for one diff. The zero value is invalid for a descriptor with roots.
type ResourcePhase int

const (
	// ResourceBeforeOwnership runs the resource rules before the ownership rules.
	ResourceBeforeOwnership ResourcePhase = iota + 1
	// ResourceAfterContext runs the resource rules after the context rules.
	ResourceAfterContext
)

// RootInputs is what a RootsFunc needs to resolve a target's discovery roots.
type RootInputs struct {
	Home, Project, Config string
	Env                   scan.EnvLookup
	Warn                  func(string)
}

// RootsFunc resolves a target's skill discovery roots, native and legacy.
type RootsFunc func(in RootInputs) ([]scan.SkillRoot, error)

// NameFunc derives a skill name from its file bytes; see scan.SkillRoot.Name.
type NameFunc func(raw []byte, path string) (string, error)

// SkillMask is the adapter-declared native mechanism that hides a skill name
// pattern from the harness. The zero value (Dotted == "") means no native mask.
type SkillMask struct {
	File   string
	Format string
	Dotted string
	Value  any
}

// IsZero reports whether the target declares no native mask.
func (m SkillMask) IsZero() bool { return m.Dotted == "" }

// Descriptor is one install target as data. Flags carry every divergence
// between targets; Validator carries the rules that are not data.
type Descriptor struct {
	Name          string
	ItemSuffix    string
	Profiles      []string
	ContextFile   string
	ShadowFiles   []string
	SectionPrefix string
	GlobalMarker  string
	SkillRoots    RootsFunc
	SkillName     NameFunc
	SkillMask     SkillMask
	ResourcePhase ResourcePhase
	// LockTargeted governs TargetVersion stamping of the lock only.
	LockTargeted bool
	// LockCoexistence governs the conflict body of Registry.LockTarget.
	LockCoexistence bool
	OwnsEffects     bool
	StrictUnowned   bool
	StrictSections  bool
	// SafePathCheck gates the mutation safe-path check; AdmissionPrefix wraps
	// its errors when nonempty.
	SafePathCheck       bool
	AdmissionPrefix     string
	MixedContextConsent bool
	ErrorPrefix         string
	// Validator runs the target's own rules at the phase hooks; nil means shared
	// rules only. Implementations are stateless.
	Validator Validator
}

// ContextFacts is what Admit learned about one context file in the context phase.
type ContextFacts struct {
	// Mixed reports that another target owns a section at this path.
	Mixed bool
	// OtherOwners lists "<artifact> (<tool>/<scope>) at <path>" for each such owner.
	OtherOwners []string
}

// ContextReview is a harness-neutral mixed-context review; the target wrapper
// converts it into its own consent type.
type ContextReview struct {
	Path        string
	Prior       []byte
	Result      []byte
	Sources     []string
	OtherOwners []string
}

// Validator hooks run at today's positions in the per-diff admission loop so a
// target's own rules keep their first-error order. A hook with nothing to do
// returns nil. Implementations must not retain state between calls.
type Validator interface {
	Prepare(d Descriptor, in Admission) error
	BeforeOwnership(d Descriptor, in Admission, fd diff.FileDiff) error
	AfterOwnership(d Descriptor, in Admission, fd diff.FileDiff) error
	AfterContext(d Descriptor, in Admission, fd diff.FileDiff, facts ContextFacts) error
}

// Admission is the input of Admit: the computed change set, the recorded
// owners of both scopes, path resolution and the channels a target may use.
type Admission struct {
	Changes  *diff.ChangeSet
	Owners   []state.Item
	Resolver toolpath.Resolver
	Home     string
	Project  string
	Env      scan.EnvLookup
	Read     func(path string) ([]byte, bool, error)
	// Warn is the placement-warning channel; nil discards.
	Warn func(message string)
	// Review receives mixed-context reviews; nil discards.
	Review func(review ContextReview)
	// Roots are other targets' roots for cross-target conflict detection.
	Roots []scan.SkillRoot
}

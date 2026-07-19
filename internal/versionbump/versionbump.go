// Package versionbump enforces one rule: an artifact whose CONTENT changed in a PR
// must also have its patronus.yaml version: bumped. It is the PR-review counterpart
// to the drift guard — the same "record diverged from reality, nothing reconciled
// them" defect, one layer up (the catalog). An un-bumped content change publishes
// nothing new to `patronus update`, and worse, collides with R2's write-once
// catalog/<name>/<version>/ key on the deploy — the worst place to learn of it.
//
// The logic here is pure: it takes, per touched artifact, whether its content
// changed and its version: on both sides of the diff, and reports the violations.
// The git plumbing that gathers those facts lives in the cmd wrapper, so this stays
// table-driven testable — mirroring how internal/drift holds the real check and
// cmd/patronus/scan.go is the thin shell.
package versionbump

import "fmt"

// Change is the reconciled state of one artifact directory across a PR diff. The
// cmd wrapper fills it from `git diff` (ContentChanged) and the version: line read
// on each side (BaseVersion at the merge-base, HeadVersion in the working tree).
type Change struct {
	// Name is the artifact directory's relative path, used to name it in a failure.
	Name string
	// ContentChanged reports whether any file OTHER than patronus.yaml changed.
	// patronus.yaml is never counted as content: a canonical re-marshal or a
	// version-only edit must not, by itself, demand a bump.
	ContentChanged bool
	// ExistedInBase is false for an artifact added in this PR — a new artifact has
	// no baseline version to compare against, so it can never violate the rule.
	ExistedInBase bool
	// BaseVersion is the version: at the merge-base (empty when unreadable/absent).
	BaseVersion string
	// HeadVersion is the version: in the working tree (empty when absent).
	HeadVersion string
}

// Violation is one artifact that changed content without bumping its version.
type Violation struct {
	Name    string
	Version string // the un-bumped version, shown so the author sees what to move past
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: content changed but version: is still %s — bump it", v.Name, v.Version)
}

// Check returns the artifacts that changed content without a version bump. It is
// order-preserving over the input and returns nil when everything is clean.
//
// An artifact is a violation exactly when: it existed at the base, its content
// changed, and its version: is unchanged from base to head. Every other shape is
// allowed — a new artifact (no base to compare), a deleted one (no head content), a
// version-only change, a content change WITH a bump, or a no-op (git lists no
// changed content, so ContentChanged is false).
func Check(changes []Change) []Violation {
	var out []Violation
	for _, c := range changes {
		if !c.ContentChanged || !c.ExistedInBase {
			continue
		}
		if c.HeadVersion != c.BaseVersion {
			continue // bumped (or cleared) — the rule is satisfied
		}
		out = append(out, Violation{Name: c.Name, Version: c.HeadVersion})
	}
	return out
}

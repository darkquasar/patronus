package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// admitTarget runs the shared admission for one selected descriptor with the
// command's owner loader, placement channel and review conversion. stderr
// receives a PlacementStderr descriptor's warning after a successful
// admission (nil discards it, as deployment does); every other descriptor's
// warning is recorded on the plan. Mixed-context reviews are converted to the
// operation-local consent shape as they arrive, so a refused Claude provenance
// lookup keeps its position.
func admitTarget(reg *harness.Registry, d harness.Descriptor, cs *diff.ChangeSet, res toolpath.Resolver, home, project string, stderr func(string)) ([]piContextReview, error) {
	paths := []string{filepath.Join(home, ".patronus/state.json"), filepath.Join(project, ".patronus/state.json")}
	var pre func(string) error
	var wrap func(int, string, error) error
	if d.AdmissionPrefix != "" {
		pre = func(path string) error { return admissionSafePath(d, path) }
		wrap = func(_ int, path string, err error) error {
			return fmt.Errorf("%s ownership %s: %w", d.ErrorPrefix, path, err)
		}
	}
	var deferred []string
	var reviews []piContextReview
	in := &harness.Admission{
		Changes:    cs,
		Resolver:   res,
		Home:       home,
		Project:    project,
		Env:        os.LookupEnv,
		Read:       scan.ReadPiFile,
		LoadOwners: func() ([]state.Item, error) { return loadOwners(paths, pre, wrap, reg.OwnershipPolicy()) },
		Warn: func(message string) {
			if d.PlacementStderr {
				deferred = append(deferred, message)
				return
			}
			appendPlacementWarning(cs, d.Name, message)
		},
		Review: func(r harness.ContextReview) error {
			review, err := contextReviewConsent(res, project, r)
			if err != nil {
				return err
			}
			reviews = append(reviews, review)
			return nil
		},
	}
	if err := harness.Admit(d, in); err != nil {
		return nil, err
	}
	if stderr != nil {
		for _, message := range deferred {
			stderr(message)
		}
	}
	return reviews, nil
}

// admissionSafePath is the descriptor's spelling of the shared safe-path guard.
func admissionSafePath(d harness.Descriptor, path string) error {
	err := scan.SafePath(path)
	if err != nil && d.AdmissionPrefix != "" {
		return fmt.Errorf("%s: %w", d.AdmissionPrefix, err)
	}
	return err
}

// contextReviewConsent binds a harness review to the displayed bytes and the
// effective sources the consent prompt names.
func contextReviewConsent(res toolpath.Resolver, project string, r harness.ContextReview) (piContextReview, error) {
	claudeSource, err := piClaudeContextSources(res, project, r.Scope)
	if err != nil {
		return piContextReview{}, err
	}
	source := r.Path
	if len(r.Sources) > 0 {
		source = r.Sources[0]
	}
	return piContextReview{Consent: piContextConsent{Path: r.Path, PiSource: source, ClaudeSource: claudeSource, PriorHash: sha256.Sum256(r.Prior), ResultHash: sha256.Sum256(r.Result)}, Proposed: append([]byte(nil), r.Result...), OtherOwners: r.OtherOwners}, nil
}

// declaredTargets returns the profile manifest's own targets: list when the
// catalog holds the profile (nil otherwise), for Registry.ProfileTarget.
func declaredTargets(cat *registry.Catalog, profile string) []string {
	if cat == nil || profile == "" {
		return nil
	}
	for i := range cat.Profiles {
		if m := cat.Profiles[i].Manifest; m != nil && m.Name == profile {
			return append([]string(nil), manifest.StringList(m.Targets)...)
		}
	}
	return nil
}

// checkLock refuses a request whose target conflicts with the recorded lock
// under the registry's coexistence rule.
func checkLock(reg *harness.Registry, project, target string) error {
	prior, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		return err
	}
	return reg.LockTarget(prior, target)
}

// checkComputedLock exempts a target-free request only after the complete
// dependency plan proves every operation is runtime-agnostic. A missing target
// for a runtime selection is not consent to cross an existing targeted lock.
func checkComputedLock(reg *harness.Registry, project, target string, cs *diff.ChangeSet) error {
	prior, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		return err
	}
	if target == "" && len(cs.Diffs) > 0 {
		agnostic := true
		for _, d := range cs.Diffs {
			agnostic = agnostic && d.Tool == recipe.TargetAgnostic
		}
		if agnostic {
			return nil
		}
	}
	return reg.LockTarget(prior, target)
}

// admitRemoval runs the admission of every selected descriptor that admits
// removal inverses (AdmitRemoval) over the inverse change set.
func admitRemoval(reg *harness.Registry, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) error {
	for _, d := range reg.Selected(cs) {
		if !d.AdmitRemoval {
			continue
		}
		if _, err := admitTarget(reg, d, cs, res, home, project, nil); err != nil {
			return err
		}
	}
	return nil
}

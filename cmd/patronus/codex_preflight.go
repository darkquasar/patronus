package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// codexPreflightPlan is admission over an already computed shared ChangeSet.
// It never plans, adopts, migrates or writes. Legacy ownership stays blocked
// here; the only relocation path is the separately consented, checksum-proven
// `patronus migrate codex-skills` command. The shared rules run in
// harness.Admit with the Codex descriptor; Codex-only rules live in
// codexValidator.
func codexPreflightPlan(reg *harness.Registry, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) error {
	d, ok := reg.Lookup("codex")
	if !ok {
		return harness.ErrUnknownTarget
	}
	in := &harness.Admission{
		Changes:  cs,
		Resolver: res,
		Home:     home,
		Project:  project,
		Env:      os.LookupEnv,
		Read:     scan.ReadPiFile,
		LoadOwners: func() ([]state.Item, error) {
			return loadOwners(
				[]string{filepath.Join(home, ".patronus/state.json"), filepath.Join(project, ".patronus/state.json")},
				scan.CodexSafePath,
				func(_ int, path string, err error) error { return fmt.Errorf("codex ownership %s: %w", path, err) },
			)
		},
	}
	return harness.Admit(d, in)
}

func codexChecksum(raw []byte) string { return state.Checksum(raw) }

// codexProfileTarget delegates the profile-to-target binding to the registry
// (no declared manifest targets at this call; text unchanged).
func codexProfileTarget(reg *harness.Registry, profile, target string) error {
	return reg.ProfileTarget(profile, nil, target)
}

// codexLockTarget delegates the lock coexistence rule to the registry, whose
// LockCoexistence flag keeps today's Codex-only trigger.
func codexLockTarget(reg *harness.Registry, prior *lock.Lock, target string) error {
	return reg.LockTarget(prior, target)
}

func codexCheckLock(reg *harness.Registry, project, target string) error {
	prior, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		return err
	}
	return codexLockTarget(reg, prior, target)
}

// Target-free requests are exempt only after the complete dependency plan
// proves every operation is runtime-agnostic. A missing target for a runtime
// selection is not consent to cross an existing Codex lock.
func codexCheckComputedLock(reg *harness.Registry, project, target string, cs *diff.ChangeSet) error {
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
	return codexLockTarget(reg, prior, target)
}

// Shared remove emits inverse diffs, with the structural evidence retained in
// state rather than a forward SettingEdit. Validate every selected contributor
// without comparing the whole config hash, so user sibling fields survive.
func codexInverseSettings(d diff.FileDiff, owners []state.Item, tool string) (bool, error) {
	if d.Action != diff.Restore && d.Action != diff.Skip {
		return false, nil
	}
	names := []string{d.Artifact}
	for _, c := range d.RestoreContrib {
		names = append(names, c.Artifact)
	}
	structured := false
	for _, name := range names {
		for _, owner := range owners {
			if owner.Tool != tool || owner.Scope != d.Scope || owner.Artifact != name {
				continue
			}
			for _, f := range owner.Files {
				if f.Path != d.Path || f.Setting == nil {
					continue
				}
				structured = true
				self := adapter.SettingOwner{Artifact: owner.Artifact, Tool: owner.Tool, Scope: owner.Scope}
				for _, other := range owners {
					otherOwner := adapter.SettingOwner{Artifact: other.Artifact, Tool: other.Tool, Scope: other.Scope}
					if otherOwner == self {
						continue
					}
					for _, sibling := range other.Files {
						if sibling.Path != d.Path {
							continue
						}
						if err := adapter.CheckSettingPair(f.Setting, self, sibling.Setting, otherOwner); err != nil {
							return false, fmt.Errorf("codex inverse ownership %s: %w", d.Path, err)
						}
						if adapter.SettingEditsOverlap(f.Setting, sibling.Setting) {
							return false, fmt.Errorf("codex inverse ownership overlap at %s with %s (%s); explicit migration required, no --force bypass", d.Path, other.Artifact, other.Tool)
						}
					}
				}
				present, equal, err := adapter.SettingStatus(d.Before, f.Setting)
				if err != nil {
					return false, fmt.Errorf("codex inverse setting %s: %w", d.Path, err)
				}
				if present && !equal {
					return false, fmt.Errorf("codex setting ownership drift at %s (%s); no --force bypass", d.Path, f.Setting.Dotted)
				}
			}
		}
	}
	return structured, nil
}

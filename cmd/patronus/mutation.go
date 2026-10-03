package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

// mutation owns one nonblocking home lock, from before planning reads through
// persistence. Locked callees receive it explicitly; they never reacquire.
// Editors, old binaries and different homes do not honor this lock. A final
// check/write race remains; shared project files require operator single-writer
// access. A busy/error result requires a fresh preview, never a queued retry.
type mutation struct {
	release func() error
	before  map[string][]byte
}

func beginMutation(home, project string) (*mutation, error) {
	return acquireMutation(home, filepath.Join(home, ".patronus/state.json"), filepath.Join(project, ".patronus/state.json"), filepath.Join(project, "patronus.lock"))
}

func acquireMutation(home string, paths ...string) (*mutation, error) {
	release, err := packagestate.Acquire(home)
	if err != nil {
		return nil, err
	}
	m := &mutation{release: release, before: map[string][]byte{}}
	for _, path := range paths {
		if err := m.watch(path); err != nil {
			return nil, errors.Join(err, release())
		}
	}
	return m, nil
}

func (m *mutation) close(err *error) { *err = errors.Join(*err, m.release()) }

func (m *mutation) watch(path string) error {
	if _, ok := m.before[path]; ok {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	m.before[path] = data
	return nil
}

func (m *mutation) check(path string) error {
	before, ok := m.before[path]
	if !ok {
		return fmt.Errorf("mutation: %s was not captured before reading; fresh preview required", path)
	}
	return install.CheckUnchanged(path, before)
}

// save checks the original bytes, then verifies the complete serialized result.
// Only our verified writes update the snapshot, including directory discovery
// repairs made earlier in the same outer operation.
func (m *mutation) save(path string, value any, write func() error) error {
	expected, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	expected = append(expected, '\n')
	if err := m.check(path); err != nil {
		return err
	}
	if err := write(); err != nil {
		return fmt.Errorf("persist %s: %w; write outcome uncertain; fresh preview required", path, err)
	}
	observed, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("persist %s committed but readback failed: %w; ownership uncertain; fresh preview required", path, err)
	}
	if !bytes.Equal(expected, observed) {
		return fmt.Errorf("persist %s committed but readback differs; ownership uncertain; fresh preview required", path)
	}
	m.before[path] = observed
	return nil
}

func (m *mutation) saveState(path string, s *state.State, save func(string, *state.State) error) error {
	state.RefreshProfiles(s)
	if save == nil {
		save = state.Save
	}
	return m.save(path, s, func() error { return save(path, s) })
}

func (m *mutation) saveLock(path string, l *lock.Lock) error {
	return m.save(path, l, func() error { return lock.Save(path, l) })
}

// checkPlan is the final whole-selection admission barrier after interactive
// confirmation. Later per-file checks still run, since acquisition/prompts and
// earlier writes can take time. A stale consent is never silently renewed.
func (m *mutation) checkPlan(cs *diff.ChangeSet, consents []piContextConsent) error {
	for path := range m.before {
		if err := m.check(path); err != nil {
			return err
		}
	}
	for _, d := range cs.Diffs {
		if d.IsDir || d.Directory != nil || d.Native != nil || d.Action == diff.Exec {
			continue
		}
		if err := m.checkFile(d, consents); err != nil {
			return err
		}
		if err := install.CheckUnchanged(d.Path, d.Before); err != nil {
			return err
		}
	}
	return nil
}

func (m *mutation) checkFile(d diff.FileDiff, consents []piContextConsent) error {
	for path := range m.before {
		if err := m.check(path); err != nil {
			return err
		}
	}
	if d.Tool == "pi" {
		if err := scan.PiSafePath(d.Path); err != nil {
			return err
		}
	}
	for _, c := range consents {
		if c.Path != d.Path {
			continue
		}
		prior, err := os.ReadFile(d.Path)
		if err != nil {
			return fmt.Errorf("pi mixed-context consent stale for %s: %w; new interactive preview required", d.Path, err)
		}
		if err := validatePiConsentBytes(c, d.Path, prior, d.After); err != nil {
			return err
		}
	}
	return nil
}

// Explicit selection survives even when delivery-only recipes emit no Pi rows.
func staticPiSelection(cs *diff.ChangeSet, piSelected bool) error {
	pi := piSelected
	for _, d := range cs.Diffs {
		if d.Native != nil {
			if d.Action != diff.Native || d.Tool != "pi" || d.Scope != d.Native.Identity.Scope {
				return fmt.Errorf("invalid typed native action")
			}
			if err := d.Native.Validate(); err != nil {
				return err
			}
		} else if d.Action == diff.Native {
			return fmt.Errorf("missing typed native action")
		}
		pi = pi || d.Tool == "pi"
	}
	if pi {
		for _, d := range cs.Diffs {
			if d.Action == diff.Exec || d.Exec != nil {
				return fmt.Errorf("pi static selection includes EXEC/provisioning intent for %s; no runtime execution is permitted", d.Artifact)
			}
		}
	}
	return nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/scan"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/nativepi"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/spf13/cobra"
)

var nativeRunner nativepi.Runner

type piRunner struct{}

func (piRunner) Run(ctx context.Context, in nativepi.Invocation) (string, error) {
	c := exec.CommandContext(ctx, in.Argv[0], in.Argv[1:]...)
	c.Dir, c.Env = in.Cwd, in.Env
	out, err := c.CombinedOutput()
	return string(out), err
}

func nativeItem(s *state.State, id nativepi.Identity) (int, error) {
	index := -1
	for i, it := range s.Items {
		if it.Native == nil {
			continue
		}
		other := it.Native.Operation.Identity
		// Project is execution context for global scope, not a second ownership
		// namespace for the same global npm identity.
		if other.Name == id.Name && other.Root == id.Root && other.Scope == id.Scope {
			if index >= 0 {
				return -1, errors.New("ambiguous duplicate native ownership")
			}
			index = i
		}
	}
	return index, nil
}

func nativeShared(s *state.State, id nativepi.Identity, selected map[string]bool) bool {
	for _, p := range s.Profiles {
		if selected[p.Name+"\x00"+p.Root] {
			continue
		}
		for _, m := range p.Members {
			if m.Native != nil {
				other := m.Native.Operation.Identity
				if other.Name == id.Name && other.Scope == id.Scope && other.Root == id.Root {
					return true
				}
			}
		}
	}
	return false
}

// inspectNative attaches honest read-only evidence to both render surfaces.
func inspectNative(cs *diff.ChangeSet, home, project string, force bool, selected map[string]bool) error {
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Native == nil {
			continue
		}
		o, err := nativepi.Observe(*d.Native)
		if err != nil {
			return err
		}
		d.NativeObservation = &o
		s, err := state.Load(removeStatePath(d.Scope, home, project))
		if err != nil {
			return err
		}
		index, err := nativeItem(s, d.Native.Identity)
		if err != nil {
			return err
		}
		provenance := "external (not enrolled)"
		if o.Status == "missing" {
			provenance = "not installed"
		}
		if index >= 0 {
			n := s.Items[index].Native
			provenance = n.Provenance
			if n.InstalledAt != "" {
				provenance += "; installed " + n.InstalledAt
			}
			if n.TrackedAt != "" {
				provenance += "; tracked " + n.TrackedAt + " (installation date unknown)"
			}
			prior, err := nativepi.Observe(n.Operation)
			if err != nil {
				return err
			}
			if prior.Status != "present" && prior.Status != "missing" {
				provenance += "; recorded declaration/version changed; use --force"
			}
		}
		var consumers []string
		for _, p := range s.Profiles {
			for _, member := range p.Members {
				if member.Native != nil && member.Native.Operation.Identity.Same(d.Native.Identity) {
					consumers = append(consumers, p.Name+"@"+p.Root)
					break
				}
			}
		}
		d.NativeDecision = d.Native.Kind
		if d.NativePrerequisite && index < 0 && o.Status == "present" {
			d.NativeDecision = "observe compatible external prerequisite (tracking requires --track-existing)"
		}
		if d.Native.Kind == "remove" && index >= 0 {
			n := s.Items[index].Native
			if n.Pending != nil || n.Provenance == "pending" {
				d.NativeDecision = "blocked; unresolved pending intent requires inspection/reconciliation"
			} else if o.Status == "missing" {
				d.NativeDecision = "settle verified absence"
			} else if !force && (n.Provenance == "tracked" || nativeShared(s, d.Native.Identity, selected) || o.Status != "present") {
				d.NativeDecision = "retain; use --force"
			} else if force {
				d.NativeDecision = "force-remove selected identity"
			}
		} else if o.Status == "missing" && d.Native.Kind == "remove" {
			d.NativeDecision = "settle verified absence"
		}
		d.Note = "Decision: " + d.NativeDecision + "; known profiles=" + strings.Join(consumers, ",") + "; " + strings.Join(d.Native.Argv(), " ") + "; root " + d.Native.Identity.Root + "; observed " + o.Status + "; " + provenance + ". " + nativepi.EditWarning + " Reload/restart required; runtime loading unverified. Single-writer access required."
	}
	return nil
}

// nativePolicy is shared by the pre-effect batch gate and the immediate recheck.
func nativePolicy(d diff.FileDiff, o nativepi.Observation, s *state.State, opts deployOptions, selected map[string]bool) (int, bool, error) {
	op := *d.Native
	index, err := nativeItem(s, op.Identity)
	if err != nil {
		return -1, false, err
	}
	if op.Kind == "remove" {
		if index < 0 {
			return index, false, errors.New("native removal has no controlled identity")
		}
		n := s.Items[index].Native
		if n.Pending != nil || n.Provenance == "pending" {
			return index, false, errors.New("pending intent alone grants no native deletion authority")
		}
		if o.Status == "missing" {
			return index, false, nil
		}
		if !opts.force && (n.Provenance == "tracked" || nativeShared(s, op.Identity, selected) || o.Status != "present") {
			return index, false, fmt.Errorf("native %s at %s retained (%s, %s, or shared); use --force", op.Identity.Name, op.Identity.Root, n.Provenance, o.Status)
		}
		return index, true, nil
	}
	if index < 0 && d.NativePrerequisite && o.Status == "present" && !opts.trackExisting {
		return index, false, nil
	}
	if index < 0 && o.Status != "missing" {
		if !opts.trackExisting {
			return index, false, fmt.Errorf("external native %s at %s requires --track-existing; force/yes do not enroll", op.Identity.Name, op.Identity.Root)
		}
		if o.Status != "present" && !opts.allowPkgInstalls {
			return index, false, errors.New("external native replacement requires --allow-package-installs")
		}
	}
	if index >= 0 {
		n := s.Items[index].Native
		prior, err := nativepi.Observe(n.Operation)
		if err != nil {
			return index, false, err
		}
		if n.Pending != nil && o.Status != "present" && o.Status != "missing" {
			return index, false, errors.New("unresolved native operation; inspect partial effects before retry")
		}
		if prior.Status != "present" && prior.Status != "missing" && !opts.force {
			return index, false, fmt.Errorf("native declaration/version changed at %s; use --force", op.Identity.Root)
		}
	}
	if o.Status == "present" && (index < 0 || s.Items[index].Native.Operation.Source == op.Source) {
		return index, false, nil
	}
	if !opts.allowPkgInstalls {
		return index, false, errors.New("required native install/update declined: --allow-package-installs required (neither --yes nor tracking authorizes installation)")
	}
	return index, true, nil
}

// applyNative persists exact intent before Pi, observes even after errors, and
// saves unresolved evidence. It never touches npm-owned files itself.
func applyNative(cmd *cobra.Command, cs *diff.ChangeSet, opts deployOptions, selected map[string]bool) error {
	runner := nativeRunner
	if runner == nil {
		runner = piRunner{}
	}
	// No mutation or probe until every selected native member passes consent.
	for _, d := range cs.Diffs {
		if d.Native != nil {
			op := *d.Native
			for _, p := range []string{op.Identity.SettingsPath(), op.Identity.MetadataPath()} {
				if err := opts.mutation.watch(p); err != nil {
					return err
				}
			}
			if op.Identity.Scope == "local" && !opts.allowPiProject {
				return fmt.Errorf("native local operation at cwd %s, root %s requires --allow-pi-project-config", op.Identity.Project, op.Identity.Root)
			}
			s, err := state.Load(statePath(d.Scope, opts))
			if err != nil {
				return err
			}
			o, err := nativepi.Observe(op)
			if err != nil {
				return err
			}
			if _, _, err = nativePolicy(d, o, s, opts, selected); err != nil {
				return err
			}
		}
	}
	// Probe all needed runtimes before any destructive effect.
	for _, d := range cs.Diffs {
		if d.Native != nil {
			s, err := state.Load(statePath(d.Scope, opts))
			if err != nil {
				return err
			}
			o, err := nativepi.Observe(*d.Native)
			if err != nil {
				return err
			}
			_, mutate, err := nativePolicy(d, o, s, opts, selected)
			if err != nil {
				return err
			}
			if mutate {
				if err := nativepi.Probe(cmd.Context(), runner, *d.Native); err != nil {
					return err
				}
			}
		}
	}
	for _, d := range cs.Diffs {
		if d.Native != nil {
			op := *d.Native
			path := statePath(d.Scope, opts)
			s, err := state.Load(path)
			if err != nil {
				return err
			}
			before, err := nativepi.Observe(op)
			if err != nil {
				return err
			}
			index, mutate, err := nativePolicy(d, before, s, opts, selected)
			if err != nil {
				return err
			}
			if index < 0 && !mutate && d.NativePrerequisite && !opts.trackExisting {
				fmt.Fprintf(cmd.OutOrStdout(), "Native prerequisite %s at %s: external observation only; not enrolled\n", op.Identity.Name, op.Identity.Root)
				continue
			}
			now := time.Now().UTC().Format(time.RFC3339)
			if index < 0 {
				provenance := "pending"
				tracked := ""
				if before.Status != "missing" {
					provenance = "tracked"
					tracked = now
				}
				s.Items = append(s.Items, state.Item{Artifact: d.Artifact, ItemVersion: d.Version, Type: d.Type, Tool: "pi", Scope: d.Scope, Root: op.Identity.Root, Native: &nativepi.Record{Operation: op, Provenance: provenance, TrackedAt: tracked, Observed: before}})
				index = len(s.Items) - 1
			}
			item := &s.Items[index]
			record := item.Native
			if mutate {
				record.Pending = &nativepi.Pending{Operation: op, Before: before}
				record.Outcome = "pending"
				state.EnrollProfile(s, opts.profile, "pi", d.Scope, op.Identity.Root, []diff.FileDiff{d})
				if err := opts.mutation.saveState(path, s, opts.saveState); err != nil {
					return err
				}
				// State persistence is not authorization for a stale filesystem snapshot.
				fresh, err := nativepi.Observe(op)
				if err != nil {
					return err
				}
				if err := opts.mutation.check(op.Identity.SettingsPath()); err != nil {
					return err
				}
				if err := opts.mutation.check(op.Identity.MetadataPath()); err != nil {
					return err
				}
				if fresh != before {
					return errors.New("native observation changed before execution; re-preview")
				}
				_, runErr := runner.Run(cmd.Context(), nativepi.InvocationFor(op, op.Argv()))
				after, observeErr := nativepi.Observe(op)
				for _, p := range []string{op.Identity.SettingsPath(), op.Identity.MetadataPath()} {
					b, e := os.ReadFile(p)
					if e != nil && !os.IsNotExist(e) {
						observeErr = errors.Join(observeErr, e)
					}
					opts.mutation.before[p] = b
				}
				record.Observed = after
				record.Outcome = "unresolved"
				confirmed := observeErr == nil && ((op.Kind == "install" && after.Status == "present") || (op.Kind == "remove" && after.Status == "missing"))
				if confirmed && runErr == nil {
					record.Pending = nil
					record.Outcome = "confirmed"
					if op.Kind == "install" {
						record.Operation = op
						record.Provenance = "installed"
						record.InstalledAt = now
						item.ItemVersion = d.Version
					}
				}
				if runErr != nil {
					record.Outcome = "manager-error: " + runErr.Error()
				}
				if op.Kind == "install" {
					state.EnrollProfile(s, opts.profile, "pi", d.Scope, op.Identity.Root, []diff.FileDiff{d})
				}
				if err := opts.mutation.saveState(path, s, opts.saveState); err != nil {
					return errors.Join(runErr, observeErr, err)
				}
				if runErr != nil || observeErr != nil || !confirmed {
					return errors.Join(runErr, observeErr, fmt.Errorf("native %s unresolved at %s; observed %s (exit zero is not success)", op.Kind, op.Identity.Root, after.Status))
				}
			} else {
				record.Observed = before
				record.Outcome = "confirmed"
				record.Pending = nil
			}
			if op.Kind == "remove" {
				s.Items = append(s.Items[:index], s.Items[index+1:]...)
				settleProfiles(s, selected)
			} else {
				// A matching externally tracked package is not assigned an installation date.
				if record.Provenance == "pending" {
					record.Provenance = "installed"
					record.InstalledAt = now
				}
				// Exact-source presence is sufficient to settle a recipe-only version
				// update without asking Pi to reinstall the same package.
				item.ItemVersion = d.Version
				state.EnrollProfile(s, opts.profile, "pi", d.Scope, op.Identity.Root, []diff.FileDiff{d})
			}
			if err := opts.mutation.saveState(path, s, opts.saveState); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Native %s: %s at %s; runtime loading unverified\n", op.Kind, record.Observed.Status, op.Identity.Root)
		}
	}
	return nil
}

// Pi changes its own packages array before authored fragments are applied.
// Re-fold only settings files touched by the selected native manager operation;
// never apply a stale whole-file image that could resurrect declarations.
func refreshNativeSettings(cs *diff.ChangeSet) error {
	paths := map[string]bool{}
	for _, d := range cs.Diffs {
		if d.Native != nil {
			paths[d.Native.Identity.SettingsPath()] = true
		}
	}
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Native != nil || !paths[d.Path] {
			continue
		}
		if d.Setting == nil {
			return fmt.Errorf("native settings file %s cannot mix whole-file authored writes", d.Path)
		}
		before, err := os.ReadFile(d.Path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		after, err := adapter.ApplySettingEdit(before, d.Setting)
		if err != nil {
			return err
		}
		for _, c := range d.SettingContrib {
			after, err = adapter.ApplySettingEdit(after, c.Edit)
			if err != nil {
				return err
			}
		}
		d.Before, d.After = before, after
		d.Action = diff.Classify(diff.Merge, before, after, before != nil)
	}
	return nil
}

func scanNativePackages(home, project string) ([]scan.NativeStatus, error) {
	var out []scan.NativeStatus
	seen := map[string]bool{}
	for _, root := range []string{home, project} {
		path, err := filepath.Abs(filepath.Join(root, ".patronus/state.json"))
		if err != nil {
			return nil, err
		}
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		s, err := state.Load(path)
		if err != nil {
			return nil, err
		}
		for _, it := range s.Items {
			if it.Native == nil {
				continue
			}
			n := it.Native
			o, err := nativepi.Observe(n.Operation)
			status := scan.NativeStatus{Artifact: it.Artifact, Identity: n.Operation.Identity, Provenance: n.Provenance, Observation: o}
			if err != nil {
				status.Error = err.Error()
			}
			out = append(out, status)
		}
	}
	return out, nil
}

package state

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
)

// ReconcileInput includes the complete desired set, not just successful writes.
// Desired must have passed ownership admission before apply; Applied contains
// only read-back-verified results. Omitted owned files are
// retained for explicit removal; manifest omission is not deletion authority.
type ReconcileInput struct {
	Old     []Item
	Desired []diff.FileDiff
	Result  install.Result
	Now     string
	// Policy selects the tools whose section checksums refresh after an
	// Unappend; the zero value is the legacy pair.
	Policy OwnershipPolicy
}

// ReconcileResult retains all ownership, including unresolved obsolete paths.
// Unresolved blocks command success and item-version advancement. It is an
// operation result, not a new persisted journal or ownership grant.
type ReconcileResult struct {
	Items      []Item
	Unresolved []error
}

type itemKey struct{ artifact, tool, scope, root string }

func keyOf(it Item) itemKey { return itemKey{it.Artifact, it.Tool, it.Scope, it.Root} }

// Reconcile accounts for every old and desired contribution. Equal external
// SKIPs remain unmanaged. A partial item keeps its previous version (or an empty
// version for a new partial install), while successful files retain their new
// checksums and immutable pre-management baselines.
func Reconcile(in ReconcileInput) ReconcileResult {
	in.Desired = retirementRows(in.Desired)
	in.Result.Applied = retirementRows(in.Result.Applied)
	in.Result.Skipped = retirementRows(in.Result.Skipped)
	out := ReconcileResult{Items: make([]Item, len(in.Old))}
	byKey := map[itemKey]int{}
	for i, it := range in.Old {
		out.Items[i] = it
		out.Items[i].Files = append([]FileState(nil), it.Files...)
		byKey[keyOf(it)] = i
	}
	selected := map[itemKey]bool{}
	versions := map[itemKey]string{}
	unresolved := map[itemKey]bool{}
	seen := map[itemKey][]FileState{}
	execs := map[itemKey]bool{}
	problem := func(k itemKey, path, reason string) {
		unresolved[k] = true
		out.Unresolved = append(out.Unresolved, fmt.Errorf("ownership unresolved: %s (%s/%s) %s: %s", k.artifact, k.tool, k.scope, path, reason))
	}
	// adopt records a newly owned file under the item for k, creating the item
	// (incomplete: no version until every desired outcome resolves) when the
	// key was not owned before. It returns the item index and ownership state.
	adopt := func(idx int, owned bool, k itemKey, row Item, next FileState) (int, bool) {
		if !owned {
			idx = len(out.Items)
			byKey[k] = idx
			fresh := row
			fresh.Files = nil
			fresh.ItemVersion = ""
			out.Items = append(out.Items, fresh)
		}
		out.Items[idx].Files = append(out.Items[idx].Files, next)
		return idx, true
	}
	for _, desired := range in.Desired {
		if desired.IsDir || desired.Directory != nil || desired.Native != nil {
			continue
		}
		actual, applied := resultDiff(in.Result.Applied, desired)
		skipped, wasSkipped := resultDiff(in.Result.Skipped, desired)
		d := desired
		if applied {
			d = actual
		}
		if desired.Action == diff.Skip && desired.Intended == diff.Restore && desired.Setting != nil && desired.Note == "computed leaf "+desired.Setting.Dotted+" was edited; not retired" {
			k := itemKey{desired.Artifact, desired.Tool, desired.Scope, desired.Root}
			selected[k], versions[k] = true, desired.Version
			idx, ok := byKey[k]
			found := false
			if ok {
				for _, old := range out.Items[idx].Files {
					if old.Path == desired.Path && adapter.SameSettingTarget(old.Setting, desired.Setting) {
						seen[k] = append(seen[k], old)
						found = true
						break
					}
				}
			}
			if !found || applied || !wasSkipped || skipped.Action != diff.Skip || !bytes.Equal(desired.Before, desired.After) {
				problem(k, desired.Path, "edited retirement was not preserved")
			}
			continue
		}
		if d.Action == diff.Restore && d.Setting != nil && strings.HasPrefix(d.Note, "computed leaf retired: ") {
			k := itemKey{d.Artifact, d.Tool, d.Scope, d.Root}
			selected[k], versions[k] = true, d.Version
			idx, ok := byKey[k]
			if !ok {
				problem(k, d.Path, "retirement has no recorded owner")
				continue
			}
			oldIndex := -1
			for j, f := range out.Items[idx].Files {
				if f.Path == d.Path && adapter.SameSettingTarget(f.Setting, d.Setting) {
					oldIndex = j
					break
				}
			}
			if oldIndex < 0 {
				problem(k, d.Path, "retirement has no recorded leaf")
				continue
			}
			old := out.Items[idx].Files[oldIndex]
			seen[k] = append(seen[k], old)
			if !applied {
				problem(k, d.Path, "retirement was not completed")
				continue
			}
			if err := verifyOwned(old, desired.Before); err != nil {
				problem(k, d.Path, err.Error())
				continue
			}
			ft := manifest.FileTarget{File: old.Setting.Target.File, Format: old.Setting.Target.Format}
			value, present, err := adapter.ReadDotted(d.After, ft, old.Setting.Dotted)
			if err != nil || present != old.Setting.PriorPresent || (present && !adapter.SettingValuesEqual(value, old.Setting.PriorValue)) {
				problem(k, d.Path, "retirement inverse was not verified")
				continue
			}
			out.Items[idx].Files = append(out.Items[idx].Files[:oldIndex], out.Items[idx].Files[oldIndex+1:]...)
			continue
		}
		rows := FromChangeSet([]diff.FileDiff{forwardDiff(d)}, in.Now)
		for _, row := range rows {
			k := keyOf(row)
			selected[k] = true
			if row.ItemVersion != "" {
				versions[k] = row.ItemVersion
			}
			idx, ownedItem := byKey[k]
			if d.Action == diff.Exec {
				if !applied {
					problem(k, d.Path, "execution was not completed")
					continue
				}
				if !ownedItem {
					idx = len(out.Items)
					byKey[k] = idx
					row.ItemVersion = "" // not complete until every desired outcome resolves
					out.Items = append(out.Items, row)
				} else {
					if !execs[k] {
						out.Items[idx].PostInstall = nil
					}
					out.Items[idx].SelfWired = row.SelfWired
					out.Items[idx].PostInstall = append(out.Items[idx].PostInstall, row.PostInstall...)
				}
				execs[k] = true
				continue
			}
			for _, next := range row.Files {
				seen[k] = append(seen[k], next)
				oldIndex := -1
				if ownedItem {
					for j, old := range out.Items[idx].Files {
						if sameFile(old, next) {
							oldIndex = j
							break
						}
					}
				}
				if !applied {
					if !wasSkipped || skipped.Action != diff.Skip {
						problem(k, next.Path, "failed, unattempted, or declined")
						continue
					}
					if oldIndex < 0 {
						if !desired.SharedAcquire {
							continue // equality is not adoption authority
						}
						// Admission marked this SKIP as a shared placed file: adopt a
						// CREATE row for the new contributor, but only against a recorded
						// owner whose checksum proves the bytes on disk.
						if err := verifySharedOwner(out.Items, k, next, skipped.Before); err != nil {
							problem(k, next.Path, err.Error())
							continue
						}
						idx, ownedItem = adopt(idx, ownedItem, k, row, next)
						continue
					}
					if err := verifyOwned(out.Items[idx].Files[oldIndex], skipped.Before); err != nil {
						problem(k, next.Path, err.Error())
					}
					continue
				}
				if oldIndex >= 0 {
					old := out.Items[idx].Files[oldIndex]
					if old.Setting != nil || next.Setting != nil {
						if !adapter.SameSettingTarget(old.Setting, next.Setting) {
							problem(k, next.Path, "structural path changed")
							continue
						}
						if err := verifyOwned(old, desired.Before); err != nil {
							problem(k, next.Path, err.Error())
							continue
						}
						edit := *next.Setting
						edit.PriorValue, edit.PriorPresent = old.Setting.PriorValue, old.Setting.PriorPresent
						next.Setting = &edit
					}
					if next.Action == string(diff.Append) || next.Action == string(diff.Merge) {
						next.Prior = old.Prior
					}
					out.Items[idx].Files[oldIndex] = next
				} else {
					idx, ownedItem = adopt(idx, ownedItem, k, row, next)
				}
			}
		}
	}
	for _, old := range in.Old {
		k := keyOf(old)
		if !selected[k] {
			continue
		}
		for _, f := range old.Files {
			found := false
			for _, desired := range seen[k] {
				if sameFile(f, desired) {
					found = true
					break
				}
			}
			if !found {
				problem(k, f.Path, "obsolete owned path retained; explicit removal required")
			}
		}
	}
	for k := range selected {
		idx, ok := byKey[k]
		if ok && !unresolved[k] {
			if versions[k] != "" {
				out.Items[idx].ItemVersion = versions[k]
			}
			out.Items[idx].InstalledAt = in.Now
		}
	}
	RefreshSectionChecksumsWithPolicy(out.Items, in.Result.Applied, in.Policy)
	return out
}

// retirementRows expands in-memory computed inverse evidence, leaving legacy rows alone.
func retirementRows(diffs []diff.FileDiff) []diff.FileDiff {
	var out []diff.FileDiff
	for _, d := range diffs {
		main := d
		main.RestoreContrib = nil
		out = append(out, main)
		for _, c := range d.RestoreContrib {
			if c.Edit == nil {
				continue
			}
			row := d
			row.Artifact, row.Version, row.Type, row.Role, row.Setting = c.Artifact, c.Version, c.Type, c.Role, c.Edit
			row.Action, row.Note = diff.Restore, "computed leaf retired: "+c.Edit.Dotted
			row.SettingContrib, row.RestoreContrib = nil, nil
			out = append(out, row)
		}
	}
	return out
}

// RefreshSectionChecksums refreshes surviving section checksums after verified
// shared-file appends or removals. Only same-runtime ownership whose checksum
// proves Before and whose section survived unchanged is refreshed. Identity,
// version, timestamps and pre-management baselines remain untouched.
// The return value reports whether ownership changed and needs persistence.
func RefreshSectionChecksums(items []Item, applied []diff.FileDiff) bool {
	return RefreshSectionChecksumsWithPolicy(items, applied, OwnershipPolicy{})
}

// RefreshSectionChecksumsWithPolicy is RefreshSectionChecksums with the tools
// whose Unappend refreshes siblings taken from p.
func RefreshSectionChecksumsWithPolicy(items []Item, applied []diff.FileDiff, p OwnershipPolicy) bool {
	changed := false
	for _, d := range applied {
		if d.Section == nil || (d.Action != diff.Append && d.Action != diff.Unappend) || (d.Action == diff.Unappend && !p.Structural(d.Tool)) {
			continue
		}
		for i := range items {
			it := &items[i]
			if it.Tool != d.Tool || it.Scope != d.Scope || it.Root != d.Root {
				continue
			}
			for j := range it.Files {
				f := &it.Files[j]
				if f.Path != d.Path || f.Section == "" || f.Setting != nil || verifyOwned(*f, d.Before) != nil {
					continue
				}
				before, wasPresent := adapter.SectionBody(d.Before, f.Section)
				after, isPresent := adapter.SectionBody(d.After, f.Section)
				if wasPresent && isPresent && bytes.Equal(before, after) {
					next := checksum(d.After)
					changed = changed || f.Checksum != next
					f.Checksum = next
				}
			}
		}
	}
	return changed
}

func resultDiff(diffs []diff.FileDiff, desired diff.FileDiff) (diff.FileDiff, bool) {
	for _, d := range diffs {
		if desired.Action == diff.Exec && d.Exec != desired.Exec {
			continue
		}
		if (d.Setting != nil || desired.Setting != nil) && !adapter.SameSettingTarget(d.Setting, desired.Setting) {
			continue
		}
		if (d.Section == nil) != (desired.Section == nil) {
			continue
		}
		if d.Section != nil && d.Section.Name != desired.Section.Name {
			continue
		}
		if d.Artifact == desired.Artifact && d.Tool == desired.Tool && d.Scope == desired.Scope && d.Path == desired.Path && (d.Action == diff.Exec) == (desired.Action == diff.Exec) {
			return d, true
		}
	}
	return diff.FileDiff{}, false
}

// Restore the intended forward action after classification (CONFLICT/SKIP).
func forwardDiff(d diff.FileDiff) diff.FileDiff {
	if d.Action != diff.Conflict && d.Action != diff.Skip {
		return d
	}
	switch {
	case d.Setting != nil:
		d.Action = diff.Merge
	case d.Section != nil:
		d.Action = diff.Append
	case d.Fetch != nil:
		d.Action = diff.Fetch
	default:
		d.Action = diff.Create
	}
	return d
}

// verifySharedOwner proves a shared-acquire adoption: some other item of the
// same tool and scope records a CREATE row at next.Path whose checksum matches
// both the bytes on disk and the adopting row.
func verifySharedOwner(items []Item, k itemKey, next FileState, current []byte) error {
	if next.Action != string(diff.Create) || next.Checksum != checksum(current) {
		return fmt.Errorf("shared placed file changed; not adopted")
	}
	for _, it := range items {
		if it.Tool != k.tool || it.Scope != k.scope || it.Artifact == k.artifact {
			continue
		}
		for _, f := range it.Files {
			if f.Path == next.Path && f.Action == string(diff.Create) && f.Checksum == next.Checksum {
				return nil
			}
		}
	}
	return fmt.Errorf("no recorded owner holds the shared placed file; not adopted")
}

func sameFile(a, b FileState) bool {
	if a.Path != b.Path || a.Section != b.Section {
		return false
	}
	if a.Setting != nil || b.Setting != nil {
		return adapter.SameSettingTarget(a.Setting, b.Setting)
	}
	return true
}

func verifyOwned(f FileState, current []byte) error {
	if !filepath.IsAbs(f.Path) {
		return fmt.Errorf("owned path is not absolute")
	}
	if f.Setting != nil {
		present, equal, err := adapter.SettingStatus(current, f.Setting)
		if err != nil {
			return err
		}
		if !present || !equal {
			return fmt.Errorf("recorded setting changed or missing")
		}
		return nil
	}
	if f.Checksum == "" || checksum(current) != f.Checksum {
		return fmt.Errorf("owned file changed or lacks checksum evidence")
	}
	return nil
}

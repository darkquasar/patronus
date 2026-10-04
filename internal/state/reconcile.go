package state

import (
	"fmt"
	"path/filepath"

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
						continue
					} // equality is not adoption authority
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
					if !ownedItem {
						idx = len(out.Items)
						byKey[k] = idx
						fresh := row
						fresh.Files = nil
						fresh.ItemVersion = ""
						out.Items = append(out.Items, fresh)
						ownedItem = true
					}
					out.Items[idx].Files = append(out.Items[idx].Files, next)
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
	return out
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

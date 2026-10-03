package state_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/state"
)

func TestReconcileRetainsOwnedSkipAndFailedPaths(t *testing.T) {
	a := diff.FileDiff{Artifact: "fixture", Tool: "pi", Scope: "global", Version: "1", Path: "/fixture/a", Action: diff.Create, After: []byte("a")}
	b := a
	b.Path = "/fixture/b"
	b.After = []byte("b")
	old := state.FromChangeSet([]diff.FileDiff{a, b}, "first")
	a.Version = "2"
	a.Action = diff.Skip
	a.Before = a.After
	b.Version = "2"
	b.Before = b.After
	b.After = []byte("new")
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{a, b}, Result: install.Result{Skipped: []diff.FileDiff{a}, Failed: &b}, Now: "second"})
	if len(got.Items) != 1 || len(got.Items[0].Files) != 2 || got.Items[0].ItemVersion != "1" || len(got.Unresolved) == 0 {
		t.Fatalf("lost unresolved ownership: %+v", got)
	}
}

// The initial present/null values model an already approved structural edit,
// not a CLI migration grant. The complete apply/record/update/inverse mechanism
// must keep that first baseline, even when each transform captures a fresh prior.
func TestReconcileImmutableBaseline(t *testing.T) {
	for _, shape := range []string{"scalar", "mcp"} {
		for _, tc := range []struct {
			name, initial string
			prior         any
			present       bool
		}{
			{"absent", `{"user":"keep"}`, nil, false},
			{"present", `{"owned":"original","user":"keep"}`, "original", true},
			{"null", `{"owned":null,"user":"keep"}`, nil, true},
		} {
			t.Run(shape+"/"+tc.name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "settings.json")
				initial := []byte(tc.initial)
				if shape == "mcp" {
					initial = bytes.ReplaceAll(initial, []byte(`"owned":`), []byte(`"mcpServers":{"fixture":`))
					if tc.present {
						initial = bytes.ReplaceAll(initial, []byte(`,"user"`), []byte(`},"user"`))
					}
				}
				if err := os.WriteFile(path, initial, 0600); err != nil {
					t.Fatal(err)
				}
				dotted := "owned"
				if shape == "mcp" {
					dotted = "mcpServers.fixture"
				}
				var owned []state.Item
				for _, version := range []string{"1", "2", "3", "3"} {
					current, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					ft := manifest.FileTarget{File: "settings.json", Format: "json"}
					prior, present, err := adapter.ReadDotted(current, ft, dotted)
					if err != nil {
						t.Fatal(err)
					}
					var value any = "installed-" + version
					if shape == "mcp" {
						value = map[string]any{"url": "https://fixture.invalid/" + version}
					}
					edit := &diff.SettingEdit{Target: diff.FileTargetRef{File: ft.File, Format: ft.Format}, Dotted: dotted, ScalarValue: value, PriorValue: prior, PriorPresent: present}
					after, err := adapter.ApplySettingEdit(current, edit)
					if err != nil {
						t.Fatal(err)
					}
					d := diff.FileDiff{Artifact: "fixture", Version: version, Tool: "pi", Scope: "global", Path: path, Action: diff.Classify(diff.Merge, current, after, true), Before: current, After: after, Setting: edit}
					result, err := (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: []diff.FileDiff{d}})
					if err != nil {
						t.Fatal(err)
					}
					reconciled := state.Reconcile(state.ReconcileInput{Old: owned, Desired: []diff.FileDiff{d}, Result: *result, Now: "fixture-time"})
					if len(reconciled.Unresolved) > 0 {
						t.Fatal(reconciled.Unresolved)
					}
					owned = reconciled.Items
					got := owned[0].Files[0].Setting
					if got.PriorPresent != tc.present || !adapter.SettingValuesEqual(got.PriorValue, tc.prior) {
						t.Fatalf("baseline changed at v%s: %+v", version, got)
					}
					if !adapter.SettingValuesEqual(edit.PriorValue, prior) || edit.PriorPresent != present {
						t.Fatal("mutated transform input")
					}
					statePath := filepath.Join(filepath.Dir(path), "state.json")
					if err := state.Save(statePath, &state.State{Version: state.Version, Items: owned}); err != nil {
						t.Fatal(err)
					}
					loaded, err := state.Load(statePath)
					if err != nil {
						t.Fatal(err)
					}
					owned = loaded.Items
				}
				inverse, err := remove.Compute(owned, dp04Read, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(inverse.Ledger) != 1 || inverse.Ledger[0].Outcome != remove.Applied {
					t.Fatalf("inverse: %+v", inverse)
				}
				if _, err := (&install.Applier{}).Apply(inverse.ChangeSet); err != nil {
					t.Fatal(err)
				}
				current, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				value, present, err := adapter.ReadDotted(current, manifest.FileTarget{File: "settings.json", Format: "json"}, dotted)
				if err != nil || present != tc.present || !adapter.SettingValuesEqual(value, tc.prior) {
					t.Fatalf("wrong inverse %s: %v", current, err)
				}
				if !bytes.Contains(current, []byte(`"user": "keep"`)) {
					t.Fatalf("lost sibling: %s", current)
				}
			})
		}
	}
}

func TestReconcileObsoleteSidecarRetainedUntilExplicitRemove(t *testing.T) {
	for _, edited := range []bool{false, true} {
		t.Run(fmt.Sprint("edited=", edited), func(t *testing.T) {
			dir := t.TempDir()
			a := diff.FileDiff{Artifact: "fixture", Type: "skill", Tool: "pi", Scope: "global", Version: "1", Path: filepath.Join(dir, "SKILL.md"), Action: diff.Create, After: []byte("main")}
			b := a
			b.Path = filepath.Join(dir, "sidecar")
			b.After = []byte("sidecar")
			res, err := (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: []diff.FileDiff{a, b}})
			if err != nil {
				t.Fatal(err)
			}
			old := state.FromChangeSet(res.Applied, "first")
			if edited {
				if err := os.WriteFile(b.Path, []byte("user edit"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.Version = "2"
			a.Action = diff.Skip
			a.Before = a.After
			reconciled := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{a}, Result: install.Result{Skipped: []diff.FileDiff{a}}, Now: "second"})
			if len(reconciled.Items[0].Files) != 2 || reconciled.Items[0].ItemVersion != "1" || len(reconciled.Unresolved) != 1 {
				t.Fatalf("obsolete ownership lost: %+v", reconciled)
			}
			inverse, err := remove.Compute(reconciled.Items, dp04Read, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := (&install.Applier{}).Apply(inverse.ChangeSet); err != nil {
				t.Fatal(err)
			}
			_, err = os.Stat(b.Path)
			if edited && err != nil {
				t.Fatal("edited sidecar removed")
			}
			if !edited && !os.IsNotExist(err) {
				t.Fatal("explicit remove left unchanged sidecar")
			}
		})
	}
}

func TestReconcileUnownedEqualityDoesNotAdopt(t *testing.T) {
	d := diff.FileDiff{Artifact: "external", Tool: "pi", Scope: "global", Version: "2", Path: "/fixture/equal", Action: diff.Skip, Before: []byte("equal"), After: []byte("equal")}
	got := state.Reconcile(state.ReconcileInput{Desired: []diff.FileDiff{d}, Result: install.Result{Skipped: []diff.FileDiff{d}}})
	if len(got.Items) != 0 || len(got.Unresolved) != 0 {
		t.Fatalf("equal bytes adopted: %+v", got)
	}
}

func TestReconcilePartialWriteKeepsSuccessfulAndUnattemptedOwnership(t *testing.T) {
	dir := t.TempDir()
	var oldDiffs []diff.FileDiff
	for _, name := range []string{"a", "b", "c"} {
		oldDiffs = append(oldDiffs, diff.FileDiff{Artifact: "fixture", Tool: "pi", Scope: "global", Version: "1", Path: filepath.Join(dir, name), Action: diff.Create, After: []byte(name)})
	}
	initial, err := (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: oldDiffs})
	if err != nil {
		t.Fatal(err)
	}
	old := state.FromChangeSet(initial.Applied, "first")
	desired := append([]diff.FileDiff(nil), oldDiffs...)
	for i := range desired {
		desired[i].Version = "2"
		desired[i].Before = desired[i].After
		desired[i].After = []byte("new-" + string(desired[i].After))
	}
	// A real filesystem obstruction fails the second atomic rename, after a lands.
	if err := os.Remove(desired[1].Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(desired[1].Path, 0700); err != nil {
		t.Fatal(err)
	}
	result, err := (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: desired})
	if err == nil {
		t.Fatal("expected second write failure")
	}
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: desired, Result: *result, Now: "second"})
	if len(got.Items[0].Files) != 3 || got.Items[0].ItemVersion != "1" || len(got.Unresolved) != 2 {
		t.Fatalf("lost partial ownership: %+v", got)
	}
	if got.Items[0].Files[0].Checksum == old[0].Files[0].Checksum || got.Items[0].Files[1].Checksum != old[0].Files[1].Checksum || got.Items[0].Files[2].Checksum != old[0].Files[2].Checksum {
		t.Fatal("wrong per-file result evidence")
	}
}

func dp04Read(path string) ([]byte, bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return b, err == nil, err
}

func TestReconcileDistinguishesEqualExternalLeafOnSameItemAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	oldEdit := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "owned", ScalarValue: "old"}
	d := diff.FileDiff{Artifact: "fixture", Tool: "pi", Scope: "global", Version: "1", Path: path, Action: diff.Merge, After: []byte(`{"owned":"old","external":true}`), Setting: oldEdit}
	old := state.FromChangeSet([]diff.FileDiff{d}, "first")
	d.Version = "2"
	d.Before = d.After
	d.After = []byte(`{"owned":"new","external":true}`)
	nextEdit := *oldEdit
	nextEdit.ScalarValue = "new"
	nextEdit.PriorPresent = true
	nextEdit.PriorValue = "old"
	d.Setting = &nextEdit
	// AdmitSettings emits an equal unmanaged leaf as a separate no-op without
	// structural ownership intent, even when this item also owns another leaf.
	noop := d
	noop.Action = diff.Skip
	noop.Setting = nil
	noop.After = noop.Before
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{d, noop}, Result: install.Result{Applied: []diff.FileDiff{d}, Skipped: []diff.FileDiff{noop}}})
	if len(got.Unresolved) != 0 || len(got.Items[0].Files) != 1 || got.Items[0].ItemVersion != "2" {
		t.Fatalf("confused separate outcomes at same path: %+v", got)
	}
}

func TestReconcileOwnedFetchSkipUsesObservedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture-binary")
	body := []byte("inert binary fixture")
	d := diff.FileDiff{Artifact: "fixture", Tool: "agnostic", Scope: "global", Version: "1", Path: path, Action: diff.Create, After: body}
	result, err := (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: []diff.FileDiff{d}})
	if err != nil {
		t.Fatal(err)
	}
	old := state.FromChangeSet(result.Applied, "first")
	old[0].Files[0].Action = string(diff.Fetch)
	// FETCH plans retain the observed destination snapshot, not downloaded After bytes.
	d.Action = diff.Skip
	d.Before = body
	d.After = nil
	d.Version = "2"
	d.Fetch = &diff.FetchSpec{Dest: path, SHA256: old[0].Files[0].Checksum}
	result, err = (&install.Applier{}).Apply(&diff.ChangeSet{Diffs: []diff.FileDiff{d}})
	if err != nil {
		t.Fatal(err)
	}
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{d}, Result: *result})
	if len(got.Unresolved) != 0 || len(got.Items[0].Files) != 1 || got.Items[0].ItemVersion != "2" {
		t.Fatalf("verified fetch SKIP lost ownership: %+v", got)
	}
}

func TestReconcileComposedLeavesAndDirectoryReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	a := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "first", ScalarValue: "one", PriorPresent: true, PriorValue: nil}
	b := &diff.SettingEdit{Target: a.Target, Dotted: "second", ScalarValue: "one"}
	initial := diff.FileDiff{Artifact: "first-fixture", Tool: "pi", Scope: "global", Version: "1", Path: path, Action: diff.Merge, After: []byte(`{"first":"one","second":"one"}`), Setting: a, SettingContrib: []diff.SettingContrib{{Artifact: "second-fixture", Version: "1", Edit: b}}}
	old := state.FromChangeSet([]diff.FileDiff{initial}, "first")
	receipt := state.Item{Artifact: "payload-fixture", Tool: "agnostic", Scope: "global", PackageReceipt: "payload-fixture", ItemVersion: "pinned", InstalledAt: "receipt-time"}
	old = append(old, receipt)
	next := initial
	next.Version = "2"
	next.Before = initial.After
	next.After = []byte(`{"first":"two","second":"two"}`)
	editA := *a
	editA.ScalarValue = "two"
	editA.PriorPresent = true
	editA.PriorValue = "one"
	next.Setting = &editA
	editB := *b
	editB.ScalarValue = "two"
	editB.PriorPresent = true
	editB.PriorValue = "one"
	next.SettingContrib = []diff.SettingContrib{{Artifact: "second-fixture", Version: "2", Edit: &editB}}
	desired := []diff.FileDiff{next, {Artifact: receipt.Artifact, Tool: receipt.Tool, Scope: receipt.Scope, Directory: &diff.DirectorySpec{Recipe: receipt.Artifact}}}
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: desired, Result: install.Result{Applied: []diff.FileDiff{next}}, Now: "second"})
	if len(got.Unresolved) != 0 || len(got.Items) != 3 {
		t.Fatalf("composed reconciliation: %+v", got)
	}
	for i := 0; i < 2; i++ {
		if got.Items[i].ItemVersion != "2" || got.Items[i].Files[0].Setting.PriorPresent != old[i].Files[0].Setting.PriorPresent || got.Items[i].Files[0].Setting.PriorValue != nil {
			t.Fatalf("composed baseline lost: %+v", got.Items[i])
		}
	}
	if got.Items[2].PackageReceipt != receipt.PackageReceipt || got.Items[2].ItemVersion != receipt.ItemVersion || got.Items[2].InstalledAt != receipt.InstalledAt {
		t.Fatal("file lifecycle altered directory authority")
	}
}

func TestReconcileFailedSettingKeepsOriginalBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	edit := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "fixture", ScalarValue: "installed", PriorPresent: true, PriorValue: nil}
	d := diff.FileDiff{Artifact: "fixture", Tool: "pi", Scope: "global", Version: "1", Path: path, Action: diff.Merge, After: []byte(`{"fixture":"installed"}`), Setting: edit}
	old := state.FromChangeSet([]diff.FileDiff{d}, "first")
	d.Version = "2"
	d.Before = d.After
	d.After = []byte(`{"fixture":"next"}`)
	next := *edit
	next.ScalarValue = "next"
	next.PriorValue = "installed"
	d.Setting = &next
	// A failed read-back is not Applied even if the expected bytes remain on disk.
	if err := os.WriteFile(path, d.After, 0600); err != nil {
		t.Fatal(err)
	}
	got := state.Reconcile(state.ReconcileInput{Old: old, Desired: []diff.FileDiff{d}, Result: install.Result{Failed: &d}, Now: "second"})
	if len(got.Unresolved) != 1 || got.Items[0].ItemVersion != "1" || got.Items[0].Files[0].Setting.PriorValue != nil || !got.Items[0].Files[0].Setting.PriorPresent {
		t.Fatalf("failure reset prior: %+v", got)
	}
	inverse, err := remove.Compute(got.Items, dp04Read, nil)
	if err != nil {
		t.Fatal(err)
	}
	if inverse.Ledger[0].Outcome.Complete() {
		t.Fatal("inverse accepted uncertain post-write setting")
	}
	if _, err := (&install.Applier{}).Apply(inverse.ChangeSet); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, d.After) {
		t.Fatal("uncertain bytes not preserved")
	}
}

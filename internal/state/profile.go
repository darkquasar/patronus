package state

import (
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/nativepi"
)

// Profile is historical membership, qualified by the selected runtime root.
// Effects name concrete owned files/fragments; they confer no authority over
// other paths in an item's row. A missing referenced effect remains unsatisfied.
type Profile struct {
	Name    string `json:"name"`
	Tool    string `json:"tool"`
	Scope   string `json:"scope"`
	Root    string `json:"root"`
	Members []Item `json:"members"`
}

// EnrollProfile snapshots only owned desired effects, including verified skips.
// Reinstalling a changed catalog never forgets historical membership.
func EnrollProfile(s *State, name, tool, scope, root string, desired []diff.FileDiff) {
	if name == "" {
		return
	}
	p := Profile{Name: name, Tool: tool, Scope: scope, Root: root}
	index := -1
	for i, old := range s.Profiles {
		if old.Name == name && old.Tool == tool && old.Scope == scope && old.Root == root {
			p = old
			index = i
			break
		}
	}
	members := &State{Items: p.Members}
	for _, want := range FromChangeSet(desired, "") {
		for _, owned := range s.Items {
			if keyOf(want) != keyOf(owned) {
				continue
			}
			member := owned
			member.Files = nil
			for _, f := range owned.Files {
				for _, w := range want.Files {
					if sameFile(f, w) {
						member.Files = append(member.Files, f)
						break
					}
				}
			}
			if len(member.Files) > 0 {
				Merge(members, []Item{profileReference(member)})
			}
		}
	}
	for _, d := range desired {
		if d.Directory != nil {
			for _, owned := range s.Items {
				if owned.PackageReceipt == d.Artifact && owned.ItemVersion == d.Version {
					Merge(members, []Item{profileReference(owned)})
				}
			}
		}
		if d.Native != nil {
			for _, owned := range s.Items {
				if owned.Native != nil && owned.Native.Operation.Identity.Same(d.Native.Identity) {
					Merge(members, []Item{profileReference(owned)})
				}
			}
		}
	}
	p.Members = members.Items
	if len(p.Members) == 0 {
		return
	}
	if index < 0 {
		s.Profiles = append(s.Profiles, p)
	} else {
		s.Profiles[index] = p
	}
}

// SameEffect compares a file/fragment identity, not its mutable checksum.
func SameEffect(a, b FileState) bool { return sameFile(a, b) }

// ForgetEffects retires only confirmed selected effects, preserving all other
// roots and fragments, including siblings in the same legacy item row.
func ForgetEffects(s *State, selected Item, files []FileState) {
	var items []Item
	for _, it := range s.Items {
		if keyOf(it) != keyOf(selected) {
			items = append(items, it)
			continue
		}
		var kept []FileState
		for _, f := range it.Files {
			removed := false
			for _, done := range files {
				if sameFile(f, done) {
					removed = true
					break
				}
			}
			if !removed {
				kept = append(kept, f)
			}
		}
		it.Files = kept
		if len(kept) > 0 || it.SelfWired || it.PackageReceipt != "" {
			items = append(items, it)
		}
	}
	s.Items = items
}

// Membership carries concrete effect identities, not duplicate checksums, prior
// file images, or a second native pending journal. Removal resolves current
// authoritative ownership before using a reference.
func profileReference(it Item) Item {
	ref := Item{MembershipStatus: "fulfilled", MembershipProvenance: "installed", Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope, Root: it.Root, Type: it.Type, PackageReceipt: it.PackageReceipt}
	for _, f := range it.Files {
		effect := FileState{Path: f.Path, Action: f.Action, Section: f.Section}
		if f.Setting != nil {
			edit := *f.Setting
			edit.Elem = nil
			edit.ScalarValue = nil
			edit.PriorValue = nil
			edit.PriorPresent = false
			effect.Setting = &edit
		}
		ref.Files = append(ref.Files, effect)
	}
	if it.Native != nil {
		ref.MembershipProvenance = it.Native.Provenance
		if it.Native.Pending != nil || it.Native.Provenance == "pending" {
			ref.MembershipStatus = "pending"
		}
		n := *it.Native
		n.Pending = nil
		n.Observed = nativepi.Observation{}
		n.Outcome = ""
		ref.Native = &n
	}
	return ref
}

// RefreshProfiles marks references to deleted effects unsatisfied without
// reinstalling them. Retained selected memberships remain actionable.
func RefreshProfiles(s *State) {
	for i := range s.Profiles {
		for j := range s.Profiles[i].Members {
			member := &s.Profiles[i].Members[j]
			status := "unsatisfied"
			for _, owned := range s.Items {
				if keyOf(*member) != keyOf(owned) {
					continue
				}
				if member.Native != nil && owned.Native != nil && member.Native.Operation.Identity.Same(owned.Native.Operation.Identity) {
					status = "fulfilled"
					if owned.Native.Pending != nil || owned.Native.Provenance == "pending" {
						status = "pending"
					}
					break
				}
				if member.PackageReceipt != "" && member.PackageReceipt == owned.PackageReceipt {
					status = "fulfilled"
					break
				}
				matched := 0
				for _, f := range member.Files {
					for _, of := range owned.Files {
						if sameFile(f, of) {
							matched++
							break
						}
					}
				}
				if matched == len(member.Files) && matched > 0 {
					status = "fulfilled"
				}
			}
			if status == "fulfilled" && member.MembershipStatus == "retained" {
				status = "retained"
			}
			member.MembershipStatus = status
		}
	}
}

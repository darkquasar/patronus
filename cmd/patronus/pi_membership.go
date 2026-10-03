package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func piSelectedRoot(scope, home, project string) string {
	r := toolpath.New(os.LookupEnv, home, project)
	if scope == "local" {
		return r.ResolveMarker(".pi", "pi", scope)
	}
	return r.ResolveMarker("~/.pi/agent", "pi", scope)
}

func withinPiRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func piProjectItem(it state.Item, root string) (state.Item, bool) {
	if it.Tool != "pi" {
		return it, true
	}
	if it.Root != "" && it.Root != root {
		return it, false
	}
	if it.Native != nil {
		return it, it.Root == root
	}
	var files []state.FileState
	for _, f := range it.Files {
		if withinPiRoot(f.Path, root) || (it.Scope == "local" && f.Section != "" && filepath.Dir(f.Path) == filepath.Dir(root)) {
			files = append(files, f)
		}
	}
	it.Files = files
	return it, len(files) > 0
}

func stampPiRoots(diffs []diff.FileDiff, home, project string) {
	for i := range diffs {
		if diffs[i].Tool == "pi" {
			diffs[i].Root = piSelectedRoot(diffs[i].Scope, home, project)
		}
	}
}

func sharedProfileEffect(s *state.State, it state.Item, f state.FileState, selected map[string]bool) bool {
	for _, p := range s.Profiles {
		if selected[p.Name+"\x00"+p.Root] {
			continue
		}
		for _, member := range p.Members {
			if member.Artifact != it.Artifact || member.Tool != it.Tool || member.Scope != it.Scope {
				continue
			}
			for _, effect := range member.Files {
				if state.SameEffect(effect, f) {
					return true
				}
			}
		}
	}
	return false
}

// Retire only selected memberships whose concrete effects are now absent from
// ownership. Other profiles retain their desired references as unsatisfied.
func settleProfiles(s *state.State, selected map[string]bool) {
	var profiles []state.Profile
	for _, p := range s.Profiles {
		if !selected[p.Name+"\x00"+p.Root] {
			profiles = append(profiles, p)
			continue
		}
		var members []state.Item
		for _, m := range p.Members {
			if m.Native != nil {
				for _, owned := range s.Items {
					if owned.Native != nil && owned.Native.Operation.Identity.Same(m.Native.Operation.Identity) {
						members = append(members, m)
						break
					}
				}
				continue
			}
			var files []state.FileState
			for _, f := range m.Files {
				found := false
				for _, owned := range s.Items {
					if owned.Artifact != m.Artifact || owned.Tool != m.Tool || owned.Scope != m.Scope {
						continue
					}
					for _, of := range owned.Files {
						if state.SameEffect(f, of) {
							found = true
						}
					}
				}
				if found {
					files = append(files, f)
				}
			}
			m.Files = files
			m.MembershipStatus = "retained"
			if len(files) > 0 {
				members = append(members, m)
			}
		}
		p.Members = members
		if len(members) > 0 {
			profiles = append(profiles, p)
		}
	}
	s.Profiles = profiles
	state.RefreshProfiles(s)
}

// Older authored records have no root field. Qualify only the selected root's
// existing concrete paths; this does not enroll any external content.
func qualifyPiOwnership(s *state.State, desired []diff.FileDiff) {
	roots := map[string]bool{}
	for _, d := range desired {
		if d.Tool == "pi" && d.Root != "" {
			roots[d.Root] = true
		}
	}
	var items []state.Item
	for _, it := range s.Items {
		if it.Tool != "pi" || it.Root != "" {
			items = append(items, it)
			continue
		}
		remaining := it.Files
		for root := range roots {
			row := it
			row.Root = root
			row.Files = nil
			var rest []state.FileState
			for _, f := range remaining {
				if withinPiRoot(f.Path, root) || (it.Scope == "local" && f.Section != "" && filepath.Dir(f.Path) == filepath.Dir(root)) {
					row.Files = append(row.Files, f)
				} else {
					rest = append(rest, f)
				}
			}
			if len(row.Files) > 0 {
				items = append(items, row)
			}
			remaining = rest
		}
		if len(remaining) > 0 || len(it.Files) == 0 {
			it.Files = remaining
			items = append(items, it)
		}
	}
	s.Items = items
}

func currentProfileMembers(s *state.State, p state.Profile) []state.Item {
	var members []state.Item
	for _, ref := range p.Members {
		for _, owned := range s.Items {
			if owned.Artifact != ref.Artifact || owned.Tool != ref.Tool || owned.Scope != ref.Scope || owned.Root != ref.Root {
				continue
			}
			row := owned
			row.Files = nil
			for _, f := range owned.Files {
				for _, effect := range ref.Files {
					if state.SameEffect(f, effect) {
						row.Files = append(row.Files, f)
						break
					}
				}
			}
			if len(row.Files) > 0 || owned.Native != nil || owned.PackageReceipt != "" {
				members = append(members, row)
			}
		}
	}
	return members
}

func deduplicateEffects(items []state.Item) []state.Item {
	var out []state.Item
	for _, it := range items {
		index := -1
		for i, old := range out {
			if old.Artifact == it.Artifact && old.Tool == it.Tool && old.Scope == it.Scope && old.Root == it.Root {
				index = i
				break
			}
		}
		if index < 0 {
			out = append(out, it)
			continue
		}
		for _, f := range it.Files {
			found := false
			for _, old := range out[index].Files {
				if state.SameEffect(f, old) {
					found = true
					break
				}
			}
			if !found {
				out[index].Files = append(out[index].Files, f)
			}
		}
	}
	return out
}

func removalEffectKey(artifact, tool, scope string, f state.FileState) string {
	key := artifact + "\x00" + tool + "\x00" + scope + "\x00" + f.Path + "\x00" + f.Section
	if f.Setting != nil {
		key += "\x00" + f.Setting.Dotted + "\x00" + f.Setting.IdentityKey + "\x00" + f.Setting.Identity
	}
	return key
}

func recordDirectoryProfile(cs *diff.ChangeSet, opts deployOptions) error {
	if opts.profile == "" || opts.target != "pi" {
		return nil
	}
	var directories []diff.FileDiff
	for _, d := range cs.Diffs {
		if d.Directory != nil {
			directories = append(directories, d)
		}
	}
	if len(directories) == 0 {
		return nil
	}
	path := statePath("global", opts)
	s, err := state.Load(path)
	if err != nil {
		return err
	}
	state.EnrollProfile(s, opts.profile, "pi", "global", piSelectedRoot("global", opts.home, opts.projectDir), directories)
	return opts.mutation.saveState(path, s, opts.saveState)
}

func sharedDirectoryProfile(s *state.State, name string, selected map[string]bool) bool {
	for _, p := range s.Profiles {
		if !selected[p.Name+"\x00"+p.Root] {
			for _, m := range p.Members {
				if m.PackageReceipt == name {
					return true
				}
			}
		}
	}
	return false
}

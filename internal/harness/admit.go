package harness

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

// Admit runs the shared admission rules for one descriptor over an already
// computed change set. It never plans, adopts, migrates or writes. Target rules
// run through the descriptor's Validator at the phase hooks so each target
// keeps today's first-error order:
//
//	selection (composite destinations refused), placement warning
//	Prepare hook
//	P1  safe paths on home, project and the global marker
//	owners loaded (in.LoadOwners), shared skill discovery (in.Inventory)
//	for each selected diff:
//	    P1  safe path (SafePathCheck, AdmissionPrefix)
//	    BeforeOwnership hook
//	    P4  resource rules when ResourcePhase is ResourceBeforeOwnership
//	    P2  ownership (different owner, drift and unknown under StrictUnowned,
//	        conflict-only refusal otherwise)
//	    AfterOwnership hook
//	    P3  context (shadow files when consent is not modelled, then the
//	        strict or lenient owner walk: relocation, mixed context, drift,
//	        unknown ownership), AfterContext hook, unknown selected section,
//	        review
//	    P4  resource rules when ResourcePhase is ResourceAfterContext
//
// Native diffs are skipped at selection; executions, directory payloads and
// directories are skipped per diff (no target's admission judges them: Codex
// skipped them explicitly and Pi never selects them, since Pi executions are
// refused earlier and Pi directory payloads are target-agnostic).
func Admit(d Descriptor, in *Admission) error {
	if in == nil || in.Changes == nil {
		return nil
	}
	var selected []diff.FileDiff
	skills := false
	for _, fd := range in.Changes.Diffs {
		if fd.Native != nil {
			continue
		}
		if !contains(strings.Split(fd.Tool, "+"), d.Name) {
			continue
		}
		if fd.Tool != d.Name {
			return crossTargetMessage(d.ErrorPrefix, fd.Path, fd.Tool)
		}
		selected = append(selected, fd)
		skills = skills || fd.Type == "skill"
	}
	if len(selected) == 0 {
		return nil
	}
	if in.Warn != nil && d.PlacementWarning != "" {
		in.Warn(d.PlacementWarning)
	}
	if d.Validator != nil {
		if err := d.Validator.Prepare(d, in); err != nil {
			return err
		}
	}
	config := in.Resolver.ResolveMarker(d.GlobalMarker, d.Name, "global")
	for _, path := range []string{in.Home, in.Project, config} {
		if err := safePath(d, path); err != nil {
			return err
		}
	}
	if in.LoadOwners != nil {
		owners, err := in.LoadOwners()
		if err != nil {
			return err
		}
		in.Owners = owners
	}
	verified := func(r scan.Resource) bool { return verifiedResource(d, in.Owners, r) }
	var roots []scan.SkillRoot
	if skills && d.SkillRoots != nil {
		var err error
		roots, err = d.SkillRoots(RootInputs{Home: in.Home, Project: in.Project, Config: config, Env: in.Env, Warn: in.Warn})
		if err != nil {
			return err
		}
		// Other targets' roots join the inventory after the target's own roots,
		// so a same-name resource another target can see is judged by the
		// selected root's policy.
		roots = append(roots, in.Roots...)
		if !in.InventoryResolved {
			in.Inventory, err = scan.DiscoverSkills(roots, verified)
			if err != nil {
				return err
			}
			in.InventoryResolved = true
		}
	}
	for _, fd := range selected {
		if fd.Action == diff.Exec || fd.Directory != nil || fd.IsDir {
			continue
		}
		if err := safePath(d, fd.Path); err != nil {
			return err
		}
		if d.Validator != nil {
			if err := d.Validator.BeforeOwnership(d, in, fd); err != nil {
				return err
			}
		}
		if d.ResourcePhase == ResourceBeforeOwnership {
			if err := admitResources(d, in, fd, roots, verified); err != nil {
				return err
			}
		}
		if err := admitOwnership(d, in, fd); err != nil {
			return err
		}
		var facts ContextFacts
		if d.Validator != nil {
			if err := d.Validator.AfterOwnership(d, in, fd, &facts); err != nil {
				return err
			}
		}
		if contextDiff(d, fd) {
			if err := admitContext(d, in, fd, &facts); err != nil {
				return err
			}
		} else if d.Validator != nil {
			// AfterContext runs once per diff: inside the context phase for a
			// context diff, here for every other diff.
			if err := d.Validator.AfterContext(d, in, fd, &facts); err != nil {
				return err
			}
		}
		if d.ResourcePhase == ResourceAfterContext {
			if err := admitResources(d, in, fd, roots, verified); err != nil {
				return err
			}
		}
	}
	return nil
}

func safePath(d Descriptor, path string) error {
	if !d.SafePathCheck {
		return nil
	}
	err := scan.SafePath(path)
	if err != nil && d.AdmissionPrefix != "" {
		return fmt.Errorf("%s: %w", d.AdmissionPrefix, err)
	}
	return err
}

func readExisting(in *Admission, path string) ([]byte, bool, error) {
	if in.Read != nil {
		return in.Read(path)
	}
	return scan.ReadPiFile(path)
}

// verifiedResource reports whether r is a managed, unchanged installation of
// this target: every owner file at its path names the same tool and artifact
// with a matching checksum.
func verifiedResource(d Descriptor, owners []state.Item, r scan.Resource) bool {
	owned := false
	for _, owner := range owners {
		for _, f := range owner.Files {
			if f.Path != r.Path {
				continue
			}
			if owner.Tool != d.Name || owner.Artifact != r.Name || f.Checksum != state.Checksum(r.Content) {
				return false
			}
			owned = true
		}
	}
	return owned
}

// structuredSettingEvidence reports whether fd carries structured setting
// ownership: a forward SettingEdit, setting contributions, or (for an inverse)
// a recorded SettingEdit of this target at the same path for any artifact the
// inverse restores.
func structuredSettingEvidence(d Descriptor, owners []state.Item, fd diff.FileDiff) bool {
	if fd.Setting != nil || len(fd.SettingContrib) > 0 {
		return true
	}
	if fd.Action != diff.Restore && fd.Action != diff.Skip {
		return false
	}
	names := []string{fd.Artifact}
	for _, c := range fd.RestoreContrib {
		names = append(names, c.Artifact)
	}
	for _, owner := range owners {
		if owner.Tool != d.Name || owner.Scope != fd.Scope || !contains(names, owner.Artifact) {
			continue
		}
		for _, f := range owner.Files {
			if f.Path == fd.Path && f.Setting != nil {
				return true
			}
		}
	}
	return false
}

func isResourceType(kind string) bool {
	return kind == "agent" || kind == "command" || kind == "skill"
}

// admitOwnership is P2: a different owner at the exact path is refused; a
// strict target also refuses drift and any existing unowned file, while a
// lenient target refuses only a conflict over a path it does not own unchanged.
func admitOwnership(d Descriptor, in *Admission, fd diff.FileDiff) error {
	if !ownershipDiff(d, in, fd) {
		return nil
	}
	owned, unchanged := false, false
	for _, owner := range in.Owners {
		for _, f := range owner.Files {
			if f.Path != fd.Path {
				continue
			}
			if owner.Tool != d.Name || owner.Artifact != fd.Artifact || owner.Scope != fd.Scope {
				return incompatibleOwnerMessage(d.ErrorPrefix, fd, owner)
			}
			matches := f.Checksum == state.Checksum(fd.Before)
			if d.StrictUnowned {
				alreadyAbsent := fd.Action == diff.Skip && len(fd.Before) == 0 && len(fd.After) == 0 && f.Checksum != ""
				if !alreadyAbsent && !matches {
					return ownershipDriftMessage(d.ErrorPrefix, fd.Path)
				}
			}
			unchanged = unchanged || matches
			owned = true
		}
	}
	if d.StrictUnowned {
		_, exists, err := readExisting(in, fd.Path)
		if err != nil {
			return err
		}
		if (exists || len(fd.Before) > 0) && !owned {
			return unknownOwnershipMessage(d.ErrorPrefix, fd.Path)
		}
		return nil
	}
	if fd.Action == diff.Conflict && !unchanged {
		return resourceConflictMessage(d.ErrorPrefix, fd)
	}
	return nil
}

// contextDiff says whether fd enters the context phase: any section diff, and
// for a strict-ownership target also a whole-file diff of its context file.
func contextDiff(d Descriptor, fd diff.FileDiff) bool {
	return fd.Section != nil || (d.StrictUnowned && filepath.Base(fd.Path) == d.ContextFile)
}

// ownershipDiff says whether fd enters the ownership phase: a strict-ownership
// target judges every file that is neither a section, its context file nor a
// structured setting; a lenient target judges resource files only.
func ownershipDiff(d Descriptor, in *Admission, fd diff.FileDiff) bool {
	if d.StrictUnowned {
		return fd.Section == nil && filepath.Base(fd.Path) != d.ContextFile && !structuredSettingEvidence(d, in.Owners, fd)
	}
	return isResourceType(fd.Type)
}

type selectedSection struct{ artifact, name string }

func selectedSections(fd diff.FileDiff) []selectedSection {
	if fd.Section == nil {
		return nil
	}
	out := []selectedSection{{fd.Artifact, fd.Section.Name}}
	for _, c := range fd.Contrib {
		out = append(out, selectedSection{c.Artifact, c.Section})
	}
	return out
}

// admitContext is P3 over one context diff. The owner walk has two shapes,
// selected by StrictSections: a strict target walks the recorded owners once
// (relocation of any section of the selected artifact, mixed-context refusal,
// drift of every same-target section at the path, ambiguous rows, composed
// inverse, unknown ownership); a lenient target walks the selected sections in
// order (relocation and drift of that section's owner, mixed facts, then the
// unknown-section refusal) exactly as before.
func admitContext(d Descriptor, in *Admission, fd diff.FileDiff, facts *ContextFacts) error {
	if !d.MixedContextConsent {
		for _, shadow := range d.ShadowFiles {
			path := filepath.Join(filepath.Dir(fd.Path), shadow)
			if _, exists, err := readExisting(in, path); err != nil {
				return err
			} else if exists {
				return shadowMessage(d.ErrorPrefix, fd.Path, path)
			}
		}
	}
	if d.StrictSections {
		return admitStrictContext(d, in, fd, facts)
	}
	return admitLenientContext(d, in, fd, facts)
}

func admitStrictContext(d Descriptor, in *Admission, fd diff.FileDiff, facts *ContextFacts) error {
	known := map[string]string{}
	for _, owner := range in.Owners {
		for _, f := range owner.Files {
			if owner.Tool == d.Name && owner.Artifact == fd.Artifact && owner.Scope == fd.Scope && f.Section != "" && f.Path != fd.Path {
				return relocationMessage(d.ErrorPrefix, f.Path, fd.Path)
			}
			if f.Path != fd.Path {
				continue
			}
			if owner.Tool != d.Name || owner.Scope != fd.Scope || f.Section == "" {
				if !d.MixedContextConsent {
					return mixedContextMessage(d.ErrorPrefix, fd.Path, owner)
				}
				if owner.Tool != d.Name {
					facts.Mixed = true
					facts.OtherOwners = append(facts.OtherOwners, otherOwner(owner, f.Path))
				}
				continue
			}
			if f.Checksum != state.Checksum(fd.Before) {
				return contextDriftMessage(d.ErrorPrefix, fd.Path, f.Section)
			}
			if f.Action != string(diff.Append) || f.Section != d.SectionPrefix+owner.Artifact || known[f.Section] != "" {
				return ambiguousSectionMessage(d.ErrorPrefix, fd.Path)
			}
			known[f.Section] = owner.Artifact
		}
	}
	if len(known) > 1 && (fd.Action == diff.Restore || fd.Intended == diff.Unappend || (fd.Action == diff.Skip && fd.Section == nil)) {
		return composedInverseMessage(d.ErrorPrefix, fd.Path)
	}
	if d.StrictUnowned {
		_, exists, err := readExisting(in, fd.Path)
		if err != nil {
			return err
		}
		if (exists || len(bytes.TrimSpace(fd.Before)) > 0) && len(known) == 0 {
			return unknownContextMessage(d.ErrorPrefix, fd.Path)
		}
	}
	facts.Known = known
	if d.Validator != nil {
		if err := d.Validator.AfterContext(d, in, fd, facts); err != nil {
			return err
		}
	}
	for _, s := range selectedSections(fd) {
		if _, exists := adapter.SectionBody(fd.Before, s.name); exists && known[s.name] != s.artifact {
			return unknownSectionMessage(d.ErrorPrefix, fd.Path, s.name)
		}
	}
	return finishContext(d, in, fd, facts)
}

func admitLenientContext(d Descriptor, in *Admission, fd diff.FileDiff, facts *ContextFacts) error {
	known := map[string]string{}
	for _, s := range selectedSections(fd) {
		owned := false
		for _, owner := range in.Owners {
			for _, f := range owner.Files {
				if owner.Tool == d.Name && owner.Artifact == s.artifact && owner.Scope == fd.Scope && f.Section == s.name {
					if f.Path != fd.Path {
						return relocationMessage(d.ErrorPrefix, f.Path, fd.Path)
					}
					if f.Checksum != state.Checksum(fd.Before) {
						return contextDriftMessage(d.ErrorPrefix, fd.Path, s.name)
					}
					owned = true
				}
				if f.Path == fd.Path && owner.Tool != d.Name {
					if !d.MixedContextConsent {
						return mixedContextMessage(d.ErrorPrefix, fd.Path, owner)
					}
					facts.Mixed = true
					facts.OtherOwners = append(facts.OtherOwners, otherOwner(owner, f.Path))
				}
			}
		}
		if owned {
			known[s.name] = s.artifact
		}
		if _, exists := adapter.SectionBody(fd.Before, s.name); exists && !owned {
			return unknownSectionMessage(d.ErrorPrefix, fd.Path, s.name)
		}
	}
	facts.Known = known
	if d.Validator != nil {
		if err := d.Validator.AfterContext(d, in, fd, facts); err != nil {
			return err
		}
	}
	return finishContext(d, in, fd, facts)
}

func otherOwner(owner state.Item, path string) string {
	return owner.Artifact + " (" + owner.Tool + "/" + owner.Scope + ") at " + path
}

// finishContext emits the mixed-context review for a consenting target.
func finishContext(d Descriptor, in *Admission, fd diff.FileDiff, facts *ContextFacts) error {
	if d.MixedContextConsent && facts.Mixed && in.Review != nil {
		return in.Review(ContextReview{Path: fd.Path, Scope: fd.Scope, Prior: fd.Before, Result: fd.After, Sources: facts.Sources, OtherOwners: facts.OtherOwners})
	}
	return nil
}

// admitResources is P4: legacy ownership and legacy discovered copies of a
// skill are refused; a same-name resource at another path is judged by the
// selected root's duplicate policy; the target's SkillName rule validates a
// skill entry file.
func admitResources(d Descriptor, in *Admission, fd diff.FileDiff, roots []scan.SkillRoot, verified func(scan.Resource) bool) error {
	entry := filepath.Base(fd.Path) == "SKILL.md"
	if fd.Type == "skill" {
		for _, owner := range in.Owners {
			if owner.Tool != d.Name || owner.Artifact != fd.Artifact {
				continue
			}
			for _, f := range owner.Files {
				for _, root := range roots {
					if !root.Legacy || !strings.HasPrefix(f.Path, root.Path+string(filepath.Separator)) {
						continue
					}
					raw, exists, err := readExisting(in, f.Path)
					if err != nil {
						return err
					}
					if exists && f.Checksum != state.Checksum(raw) {
						return legacyDriftMessage(d.ErrorPrefix, f.Path)
					}
					return legacyOwnershipMessage(d.ErrorPrefix, f.Path)
				}
			}
		}
	}
	if !isResourceType(fd.Type) {
		return nil
	}
	policy := selectedRootPolicy(roots, fd.Path)
	selected := scan.Resource{Kind: fd.Type, Name: fd.Artifact, Path: fd.Path, Origin: "selected " + fd.Scope, Content: fd.After}
	for _, r := range in.Inventory {
		if r.Kind != fd.Type || !sharesName(r, selected) {
			continue
		}
		if r.Legacy {
			return legacyDiscoveredMessage(d.ErrorPrefix, r.Path, fd.Artifact)
		}
		if fd.Type == "skill" && !entry {
			continue
		}
		if r.Path == fd.Path {
			continue
		}
		if policy == scan.RefuseDifferentBytesVerifiedException && (bytes.Equal(r.Content, fd.After) || verified(r)) {
			continue
		}
		return selectedConflictMessage(d.ErrorPrefix, matchedName(r, selected), r, selected)
	}
	if fd.Type == "skill" && entry && d.SkillName != nil && fd.Action != diff.Delete && fd.Intended != diff.Delete {
		if _, err := d.SkillName(fd.After, fd.Path); err != nil {
			return err
		}
	}
	return nil
}

// selectedRootPolicy is the duplicate policy of the root holding path (the
// first root otherwise); without roots every duplicate is refused.
func selectedRootPolicy(roots []scan.SkillRoot, path string) scan.DuplicatePolicy {
	if len(roots) == 0 {
		return scan.RefuseAlways
	}
	for _, root := range roots {
		if strings.HasPrefix(path, root.Path+string(filepath.Separator)) {
			return root.DuplicatePolicy
		}
	}
	return roots[0].DuplicatePolicy
}

func resourceNames(r scan.Resource) []string { return append([]string{r.Name}, r.Aliases...) }

func sharesName(a, b scan.Resource) bool { return matchedName(a, b) != "" }

func matchedName(a, b scan.Resource) string {
	for _, bn := range resourceNames(b) {
		for _, an := range resourceNames(a) {
			if an == bn {
				return bn
			}
		}
	}
	return ""
}

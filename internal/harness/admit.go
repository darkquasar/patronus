package harness

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	var selected []int
	skills := false
	for i, fd := range in.Changes.Diffs {
		if fd.Native != nil {
			continue
		}
		if !contains(strings.Split(fd.Tool, "+"), d.Name) {
			continue
		}
		if fd.Tool != d.Name {
			return crossTargetMessage(d.ErrorPrefix, fd.Path, fd.Tool)
		}
		selected = append(selected, i)
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
	references := func(a, b scan.Resource) bool { return managedReferences(in, a, b) }
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
			in.Inventory, err = scan.DiscoverSkills(roots, verified, references)
			if err != nil {
				return err
			}
			in.InventoryResolved = true
		} else if len(in.Roots) > 0 {
			// A Prepare hook resolved the target's own inventory; the foreign
			// roots are discovered separately and join it.
			foreign, err := scan.DiscoverSkills(in.Roots, verified, references)
			if err != nil {
				return err
			}
			in.Inventory = append(in.Inventory, foreign...)
		}
	}
	for _, i := range selected {
		fd := in.Changes.Diffs[i]
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
		if sharedAcquire(d, in.Owners, fd) {
			// Shared behaviour for strict and lenient targets alike: the SKIP over
			// an identical placed file every recorded owner holds becomes a
			// contributor acquisition that reconcile may adopt.
			in.Changes.Diffs[i].SharedAcquire = true
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

// managedReferences verifies the whole declared installation, not just its
// SKILL.md. State rows are the payload declaration; every row must match disk,
// and every file in the bundle must be owned. Only the two named references and
// the owner-approved callable writing-editorial skill can use this proof.
func managedReferences(in *Admission, a, b scan.Resource) bool {
	if !scan.ManagedSharedSkill(a) || !scan.ManagedSharedSkill(b) || a.Name != b.Name || a.Legacy || b.Legacy {
		return false
	}
	av, ah, ok := referencePayload(in, a)
	if !ok {
		return false
	}
	bv, bh, ok := referencePayload(in, b)
	return ok && av == bv && ah == bh
}

func referencePayload(in *Admission, r scan.Resource) (string, string, bool) {
	dir := filepath.Dir(r.Path)
	version, payload := "", ""
	owned := false
	for _, owner := range in.Owners {
		claims := false
		entryClaim := false
		for _, file := range owner.Files {
			if strings.HasPrefix(file.Path, dir+string(filepath.Separator)) {
				claims = true
			}
			if file.Path == r.Path {
				entryClaim = true
			}
		}
		if !claims {
			continue
		}
		if !entryClaim {
			return "", "", false
		}
		if owner.Artifact != r.Name || owner.Type != "skill" || owner.ItemVersion == "" || !scan.NeutralReferenceConsumer(owner.Tool) {
			return "", "", false
		}
		declared := map[string]string{}
		for _, file := range owner.Files {
			rel, err := filepath.Rel(dir, file.Path)
			if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || file.Action != string(diff.Create) || file.Checksum == "" {
				return "", "", false
			}
			if _, duplicate := declared[rel]; duplicate {
				return "", "", false
			}
			if scan.SafePath(file.Path) != nil {
				return "", "", false
			}
			raw, exists, err := readExisting(in, file.Path)
			if err != nil || !exists || state.Checksum(raw) != file.Checksum || (file.Path == r.Path && !bytes.Equal(raw, r.Content)) {
				return "", "", false
			}
			declared[rel] = file.Checksum
		}
		count := 0
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if scan.SafePath(path) != nil {
				return fmt.Errorf("unsafe reference payload")
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil || !entry.Type().IsRegular() || declared[rel] == "" {
				return fmt.Errorf("unowned reference payload")
			}
			count++
			return nil
		})
		if err != nil || count != len(declared) {
			return "", "", false
		}
		var paths []string
		for rel := range declared {
			paths = append(paths, rel)
		}
		sort.Strings(paths)
		var identity strings.Builder
		for _, rel := range paths {
			fmt.Fprintf(&identity, "%d:%s:%s\n", len(rel), rel, declared[rel])
		}
		hash := state.Checksum([]byte(identity.String()))
		if owned && (version != owner.ItemVersion || payload != hash) {
			return "", "", false
		}
		owned, version, payload = true, owner.ItemVersion, hash
	}
	return version, payload, owned
}

// plannedReferencePayload proves a fresh destination from the complete change
// set. It is intentionally limited to CREATEs into an otherwise empty bundle:
// existing destinations still require state-backed referencePayload proof.
func plannedReferencePayload(in *Admission, r scan.Resource, selected diff.FileDiff) (string, string, bool) {
	if in.Changes == nil || selected.Action != diff.Create || len(selected.Before) != 0 || selected.Version == "" || filepath.Base(selected.Path) != "SKILL.md" {
		return "", "", false
	}
	dir := filepath.Dir(selected.Path)
	declared := map[string]string{}
	entry := false
	for _, fd := range in.Changes.Diffs {
		if fd.Path != dir && !strings.HasPrefix(fd.Path, dir+string(filepath.Separator)) {
			continue
		}
		if fd.Artifact != selected.Artifact || fd.Version != selected.Version || fd.Tool != selected.Tool || fd.Scope != selected.Scope || fd.Type != "skill" || fd.Action != diff.Create || len(fd.Before) != 0 || fd.Native != nil || fd.Directory != nil || fd.IsDir {
			return "", "", false
		}
		rel, err := filepath.Rel(dir, fd.Path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || scan.SafePath(fd.Path) != nil {
			return "", "", false
		}
		if _, duplicate := declared[rel]; duplicate {
			return "", "", false
		}
		if _, exists, err := readExisting(in, fd.Path); err != nil || exists {
			return "", "", false
		}
		declared[rel] = state.Checksum(fd.After)
		if fd.Path == r.Path && bytes.Equal(fd.After, r.Content) {
			entry = true
		}
	}
	if !entry || len(declared) == 0 {
		return "", "", false
	}
	if _, err := os.Stat(dir); err == nil {
		err = filepath.WalkDir(dir, func(path string, file fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == dir || file.IsDir() {
				return nil
			}
			return fmt.Errorf("existing reference payload")
		})
		if err != nil {
			return "", "", false
		}
	} else if !os.IsNotExist(err) {
		return "", "", false
	}
	var paths []string
	for rel := range declared {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	var identity strings.Builder
	for _, rel := range paths {
		fmt.Fprintf(&identity, "%d:%s:%s\n", len(rel), rel, declared[rel])
	}
	return selected.Version, state.Checksum([]byte(identity.String())), true
}

// ManagedReferenceCoexistence retains the existing discovery callback API for
// the finite shared-skill set, including the explicitly named editorial workflow.
// It supplies Admit's state-backed proof without mutating installed payloads.
func ManagedReferenceCoexistence(owners []state.Item) func(scan.Resource, scan.Resource) bool {
	in := &Admission{Owners: owners}
	return func(a, b scan.Resource) bool { return managedReferences(in, a, b) }
}

func sharedReferenceOwner(in *Admission, fd diff.FileDiff, owner state.Item) bool {
	if fd.Type != "skill" || fd.Action != diff.Skip || !bytes.Equal(fd.Before, fd.After) || owner.Scope != fd.Scope {
		return false
	}
	entry := filepath.Join(filepath.Dir(fd.Path), "SKILL.md")
	if filepath.Base(fd.Path) == "SKILL.md" {
		entry = fd.Path
	}
	raw, exists, err := readExisting(in, entry)
	if err != nil || !exists {
		return false
	}
	r := scan.Resource{Kind: "skill", Name: fd.Artifact, Path: entry, Content: raw}
	if !scan.ManagedSharedSkill(r) {
		return false
	}
	version, _, ok := referencePayload(in, r)
	return ok && fd.Version != "" && version == fd.Version
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
				if sharedReferenceOwner(in, fd, owner) {
					owned, unchanged = true, true
					continue
				}
				if owner.Tool == d.Name && owner.Scope == fd.Scope && sharedPlacedOwner(f, fd) {
					// Another artifact of this target placed the identical file: a
					// shared placed file is co-owned, not an incompatible owner.
					owned, unchanged = true, true
					continue
				}
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

// sharedPlacedOwner reports whether the recorded row f is a CREATE of the very
// bytes fd places or releases: a fresh CREATE or identical SKIP carries them in
// After; a DELETE or a release SKIP (remove's SharedRetained) carries the
// current bytes in Before.
func sharedPlacedOwner(f state.FileState, fd diff.FileDiff) bool {
	if f.Action != string(diff.Create) || f.Checksum == "" {
		return false
	}
	switch fd.Action {
	case diff.Create:
		return f.Checksum == state.Checksum(fd.After)
	case diff.Skip:
		return f.Checksum == state.Checksum(fd.After) || (len(fd.Before) > 0 && f.Checksum == state.Checksum(fd.Before))
	case diff.Delete:
		return len(fd.Before) > 0 && f.Checksum == state.Checksum(fd.Before)
	default:
		return false
	}
}

// sharedAcquire reports whether fd is a SKIP over a placed file that every
// recorded owner of the path (same target and scope) holds with these exact
// bytes, while fd.Artifact is not yet among them.
func sharedAcquire(d Descriptor, owners []state.Item, fd diff.FileDiff) bool {
	if fd.Action != diff.Skip || fd.Section != nil || fd.Setting != nil || len(fd.SettingContrib) > 0 || !bytes.Equal(fd.Before, fd.After) {
		return false
	}
	others := 0
	for _, owner := range owners {
		for _, f := range owner.Files {
			if f.Path != fd.Path {
				continue
			}
			if owner.Tool != d.Name || owner.Scope != fd.Scope || !sharedPlacedOwner(f, fd) {
				return false
			}
			if owner.Artifact == fd.Artifact {
				return false
			}
			others++
		}
	}
	return others > 0
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
		if scan.ManagedSharedSkill(selected) || scan.ManagedSharedSkill(r) {
			version, payload, ok := referencePayload(in, selected)
			if !ok {
				version, payload, ok = plannedReferencePayload(in, selected, fd)
			}
			foreignVersion, foreignPayload, foreignOK := referencePayload(in, r)
			if !ok || !foreignOK || r.Name != selected.Name || version != fd.Version || version != foreignVersion || payload != foreignPayload {
				return selectedConflictMessage(d.ErrorPrefix, matchedName(r, selected), r, selected)
			}
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

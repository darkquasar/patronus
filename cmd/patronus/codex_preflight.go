package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// codexPreflightPlan is admission over an already computed shared ChangeSet.
// It never plans, adopts, migrates or writes. Legacy ownership stays blocked
// here; the only relocation path is the separately consented, checksum-proven
// `patronus migrate codex-skills` command.
func codexPreflightPlan(cs *diff.ChangeSet, res toolpath.Resolver, home, project string) error {
	var selected []diff.FileDiff
	skills := false
	for _, d := range cs.Diffs {
		if !contains(strings.Split(d.Tool, "+"), "codex") {
			continue
		}
		if d.Tool != "codex" {
			return fmt.Errorf("codex cross-target destination %s (%s): explicit mixed-context migration required", d.Path, d.Tool)
		}
		selected = append(selected, d)
		skills = skills || d.Type == "skill"
	}
	if len(selected) == 0 {
		return nil
	}
	config := res.ResolveMarker("~/.codex", "codex", "global")
	for _, path := range []string{home, project, config} {
		if err := scan.CodexSafePath(path); err != nil {
			return err
		}
	}
	var owners []state.Item
	for _, root := range []string{home, project} {
		path := filepath.Join(root, ".patronus/state.json")
		if err := scan.CodexSafePath(path); err != nil {
			return err
		}
		s, err := state.Load(path)
		if err != nil {
			return fmt.Errorf("codex ownership %s: %w", path, err)
		}
		owners = append(owners, s.Items...)
	}
	verifiedSkill := func(r scan.CodexSkill) bool {
		owned := false
		for _, owner := range owners {
			for _, f := range owner.Files {
				if f.Path != r.Path {
					continue
				}
				if owner.Tool != "codex" || owner.Artifact != r.Name || f.Checksum != codexChecksum(r.Content) {
					return false
				}
				owned = true
			}
		}
		return owned
	}
	var discovered []scan.CodexSkill
	if skills {
		var err error
		discovered, err = scan.DiscoverCodexSkillsWithOwnership(scan.CodexSkillRoots(home, project, config), verifiedSkill)
		if err != nil {
			return err
		}
	}
	for _, d := range selected {
		if d.Action == diff.Exec || d.Native != nil || d.Directory != nil || d.IsDir {
			continue
		}
		if err := scan.CodexSafePath(d.Path); err != nil {
			return err
		}
		inverseSettings, err := codexInverseSettings(d, owners)
		if err != nil {
			return err
		}
		if filepath.Base(d.Path) == "config.toml" && d.Setting == nil && len(d.SettingContrib) == 0 && !inverseSettings {
			return fmt.Errorf("codex config %s lacks structured SettingEdit ownership; raw TOML mutation refused", d.Path)
		}
		if d.Setting == nil && len(d.SettingContrib) == 0 && !inverseSettings && d.Section == nil && filepath.Base(d.Path) != "AGENTS.md" {
			owned := false
			for _, owner := range owners {
				for _, f := range owner.Files {
					if f.Path != d.Path {
						continue
					}
					if owner.Tool != "codex" || owner.Artifact != d.Artifact || owner.Scope != d.Scope {
						return fmt.Errorf("codex destination %s has incompatible owner %s (%s)", d.Path, owner.Artifact, owner.Tool)
					}
					alreadyAbsent := d.Action == diff.Skip && len(d.Before) == 0 && len(d.After) == 0 && f.Checksum != ""
					if !alreadyAbsent && f.Checksum != codexChecksum(d.Before) {
						return fmt.Errorf("codex ownership drift at %s; preserve edits, reconcile before mutation (no --force bypass)", d.Path)
					}
					owned = true
				}
			}
			_, exists, err := scan.ReadPiFile(d.Path)
			if err != nil {
				return err
			}
			if (exists || len(d.Before) > 0) && !owned {
				return fmt.Errorf("codex destination %s has unknown ownership; explicit adoption/migration required", d.Path)
			}
		}
		if d.Section != nil || filepath.Base(d.Path) == "AGENTS.md" {
			if err := codexContextAdmission(d, owners); err != nil {
				return err
			}
		}
		if d.Type == "skill" {
			for _, owner := range owners {
				if owner.Tool != "codex" || owner.Artifact != d.Artifact {
					continue
				}
				for _, f := range owner.Files {
					for _, root := range scan.CodexSkillRoots(home, project, config) {
						if !root.Legacy || !strings.HasPrefix(f.Path, root.Path+string(filepath.Separator)) {
							continue
						}
						raw, exists, err := scan.ReadPiFile(f.Path)
						if err != nil {
							return err
						}
						if exists && f.Checksum != codexChecksum(raw) {
							return fmt.Errorf("codex legacy ownership checksum drift at %s; preserve edits, explicit migration required (no --force bypass)", f.Path)
						}
						return fmt.Errorf("codex legacy ownership at %s requires explicit migration/retirement (patronus migrate codex-skills); automatic migration is Partial even if the old file is absent", f.Path)
					}
				}
			}
			for _, r := range discovered {
				if r.Name != d.Artifact {
					continue
				}
				if r.Legacy {
					return fmt.Errorf("codex legacy skill %s: automatic migration is Partial; preserve files and reconcile ownership/checksums and retire the old root explicitly before installing %s", r.Path, d.Artifact)
				}
				if filepath.Base(d.Path) == "SKILL.md" && r.Path != d.Path && !bytes.Equal(r.Content, d.After) {
					// A separately scoped, checksum-verified Codex install may
					// still be on the old version during the shared update flow.
					if !verifiedSkill(r) {
						return fmt.Errorf("codex skill %q conflicts with %s; incompatible same-name resource", d.Artifact, r.Path)
					}
				}
			}
			if filepath.Base(d.Path) == "SKILL.md" && d.Action != diff.Delete && d.Intended != diff.Delete {
				if _, err := scan.CodexSkillName(d.After, d.Path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func codexChecksum(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }

// Existing shared context needs positive Codex provenance. There is no implicit
// consent, including for --force, and no whole-file restore across owners.
func codexContextAdmission(d diff.FileDiff, owners []state.Item) error {
	override := filepath.Join(filepath.Dir(d.Path), "AGENTS.override.md")
	if _, exists, err := scan.ReadPiFile(override); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("codex context %s is shadowed by %s; reconcile before mutation", d.Path, override)
	}
	known := map[string]string{}
	for _, owner := range owners {
		for _, f := range owner.Files {
			if owner.Tool == "codex" && owner.Artifact == d.Artifact && owner.Scope == d.Scope && f.Section != "" && f.Path != d.Path {
				return fmt.Errorf("codex context ownership at %s cannot be silently relocated to %s", f.Path, d.Path)
			}
			if f.Path != d.Path {
				continue
			}
			if owner.Tool != "codex" || owner.Scope != d.Scope || f.Section == "" {
				return fmt.Errorf("codex mixed-context %s owned by %s (%s); explicit migration/consent required", d.Path, owner.Artifact, owner.Tool)
			}
			if f.Checksum != codexChecksum(d.Before) {
				return fmt.Errorf("codex context ownership drift at %s; preserve edits before mutation", d.Path)
			}
			if f.Action != string(diff.Append) || f.Section != owner.Artifact || known[f.Section] != "" {
				return fmt.Errorf("codex context %s has ambiguous section ownership", d.Path)
			}
			known[f.Section] = owner.Artifact
		}
	}
	if len(known) > 1 && (d.Action == diff.Restore || d.Intended == diff.Unappend || (d.Action == diff.Skip && d.Section == nil)) {
		return fmt.Errorf("codex context %s lacks a verified composed section inverse; retain sections and ownership", d.Path)
	}
	_, exists, err := scan.ReadPiFile(d.Path)
	if err != nil {
		return err
	}
	if (exists || len(bytes.TrimSpace(d.Before)) > 0) && len(known) == 0 {
		return fmt.Errorf("codex context %s has unknown ownership; preserve prose and obtain explicit migration/consent", d.Path)
	}
	allowed := map[string]bool{}
	for name := range known {
		allowed[name] = true
	}
	if err := adapter.ValidateSectionMarkers(d.Before, allowed); err != nil {
		return fmt.Errorf("codex context %s: %w", d.Path, err)
	}
	rest := string(d.Before)
	seen := map[string]bool{}
	for {
		_, tail, ok := strings.Cut(rest, "<!-- patronus:start ")
		if !ok {
			break
		}
		name, after, ok := strings.Cut(tail, " -->")
		if !ok {
			return fmt.Errorf("codex context %s: malformed section", d.Path)
		}
		if seen[name] {
			return fmt.Errorf("codex context %s: duplicate section %s requires reconciliation", d.Path, name)
		}
		seen[name] = true
		if strings.HasPrefix(name, "pi:") || known[name] == "" {
			return fmt.Errorf("codex context %s section %s has unknown/Pi ownership; explicit migration required", d.Path, name)
		}
		if _, ok := adapter.SectionBody(d.Before, name); !ok {
			return fmt.Errorf("codex context %s section %s malformed", d.Path, name)
		}
		rest = after
	}
	if d.Action == diff.Unappend && d.Section == nil {
		return fmt.Errorf("codex context %s lacks selected section identity", d.Path)
	}
	if d.Section != nil {
		expected := d.Before
		selected := map[string]bool{}
		sections := []struct{ name, artifact string }{{d.Section.Name, d.Artifact}}
		for _, c := range d.Contrib {
			sections = append(sections, struct{ name, artifact string }{c.Section, c.Artifact})
		}
		for _, s := range sections {
			if selected[s.name] || s.name != s.artifact {
				return fmt.Errorf("codex context %s has ambiguous selected section identity", d.Path)
			}
			selected[s.name] = true
			if _, exists := adapter.SectionBody(d.Before, s.name); exists && known[s.name] != s.artifact {
				return fmt.Errorf("codex context %s section %s has incompatible ownership", d.Path, s.name)
			}
			if d.Action == diff.Unappend {
				var found bool
				expected, found = adapter.RemoveSection(expected, s.name)
				if !found || known[s.name] != s.artifact {
					return fmt.Errorf("codex context %s lacks verified selected section %s", d.Path, s.name)
				}
			}
		}
		if d.Action == diff.Unappend && !bytes.Equal(expected, d.After) {
			return fmt.Errorf("codex context %s inverse changes unselected sections or prose", d.Path)
		}
	}
	return nil
}

func codexProfileTarget(profile, target string) error {
	if profile == "core-profile-cx" && target != "codex" {
		return fmt.Errorf("core-profile-cx requires --target codex (got %q)", target)
	}
	return nil
}

// Codex uses the existing target-bearing lock schema. An ambiguous legacy lock
// cannot authorize crossing into Codex, even when the desired profile differs.
func codexLockTarget(prior *lock.Lock, target string) error {
	if prior.Target != "codex" && target != "codex" {
		return nil
	}
	if prior.Target != "" && prior.Target != target {
		return fmt.Errorf("existing lock target %s conflicts with --target %s; use separate project roots or explicit migration", prior.Target, target)
	}
	if prior.Target == "" && (prior.Profile != "" || len(prior.Entries) > 0) {
		return fmt.Errorf("existing legacy lock target is unknown; preserve patronus.lock and reconcile provenance before Codex migration")
	}
	return nil
}

func codexCheckLock(project, target string) error {
	prior, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		return err
	}
	return codexLockTarget(prior, target)
}

// Target-free requests are exempt only after the complete dependency plan
// proves every operation is runtime-agnostic. A missing target for a runtime
// selection is not consent to cross an existing Codex lock.
func codexCheckComputedLock(project, target string, cs *diff.ChangeSet) error {
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
	return codexLockTarget(prior, target)
}

// Shared remove emits inverse diffs, with the structural evidence retained in
// state rather than a forward SettingEdit. Validate every selected contributor
// without comparing the whole config hash, so user sibling fields survive.
func codexInverseSettings(d diff.FileDiff, owners []state.Item) (bool, error) {
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
			if owner.Tool != "codex" || owner.Scope != d.Scope || owner.Artifact != name {
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

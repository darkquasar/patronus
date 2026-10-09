package main

import (
	"fmt"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// Shared remove emits inverse diffs, with the structural evidence retained in
// state rather than a forward SettingEdit. Validate every selected contributor
// without comparing the whole config hash, so user sibling fields survive.
func codexInverseSettings(d diff.FileDiff, owners []state.Item, tool string) (bool, error) {
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
			if owner.Tool != tool || owner.Scope != d.Scope || owner.Artifact != name {
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

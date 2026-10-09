package harness

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
)

// DescriptorRoots pairs a descriptor with the discovery roots the caller
// resolved for it.
type DescriptorRoots struct {
	Descriptor Descriptor
	Roots      []scan.SkillRoot
}

// SkillMaskEdits is the shared co-visibility rule. For every other descriptor
// with a nonempty ItemSuffix whose roots intersect the target's roots (by
// SkillRoot.Path), when at least one discovered skill is named *-<suffix>, it
// emits one mask edit through the target's SkillMask (the quoted dotted
// segment `."*-<suffix>"` names the pattern key exactly) and one human message
// naming the count, the suffix, the source lane and the file. A target without
// a SkillMask gets the message only. No target name appears in the rule.
func SkillMaskEdits(target Descriptor, targetRoots []scan.SkillRoot, discovered []scan.Resource, others []DescriptorRoots) ([]diff.SettingEdit, []string) {
	targetPaths := map[string]bool{}
	for _, r := range targetRoots {
		targetPaths[r.Path] = true
	}
	var edits []diff.SettingEdit
	var messages []string
	for _, other := range others {
		suffix := other.Descriptor.ItemSuffix
		if suffix == "" || other.Descriptor.Name == target.Name {
			continue
		}
		shared := sharedRoots(targetPaths, other.Roots)
		if len(shared) == 0 {
			continue
		}
		count := 0
		for _, r := range discovered {
			if strings.HasSuffix(r.Name, "-"+suffix) && len(r.Name) > len(suffix)+1 {
				count++
			}
		}
		if count == 0 {
			continue
		}
		if target.SkillMask.IsZero() {
			messages = append(messages, fmt.Sprintf("Found %d co-visible %s skills (*-%s) in roots %s read by %s; no native mask declared", count, other.Descriptor.Name, suffix, strings.Join(shared, ", "), target.Name))
			continue
		}
		mask := target.SkillMask
		edits = append(edits, diff.SettingEdit{
			Target:      diff.FileTargetRef{File: mask.File, Format: mask.Format},
			Dotted:      mask.Dotted + `."*-` + suffix + `"`,
			ScalarValue: mask.Value,
		})
		messages = append(messages, fmt.Sprintf("Masked %d co-visible %s skills (*-%s) from %s via %s in %s; re-run update after adding or removing %s skills", count, other.Descriptor.Name, suffix, target.Name, mask.Dotted, mask.File, other.Descriptor.Name))
	}
	return edits, messages
}

func sharedRoots(targetPaths map[string]bool, roots []scan.SkillRoot) []string {
	var shared []string
	seen := map[string]bool{}
	for _, r := range roots {
		if targetPaths[r.Path] && !seen[r.Path] {
			seen[r.Path] = true
			shared = append(shared, r.Path)
		}
	}
	sort.Strings(shared)
	return shared
}

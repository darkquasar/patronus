package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/state"
)

// opencodeValidator holds the OpenCode-only admission rules that are not
// descriptor data: the config env overrides as admission inputs (P-OC1), the
// harness-owned local-plugin files Patronus never claims, and the refusal to
// write into a commented opencode.json(c), keyed on the target file so Merge,
// Restore and a forced Delete are all covered (OP-09). It is stateless and
// never names its target: messages use the descriptor's ErrorPrefix.
type opencodeValidator struct{}

var _ harness.Validator = opencodeValidator{}

// harnessOwnedLocalPlugin reports whether rel (relative to the project) is a
// file OpenCode writes for local plugins under .opencode/.
func harnessOwnedLocalPlugin(rel string) bool {
	rel = filepath.ToSlash(rel)
	switch {
	case rel == ".opencode/package.json", rel == ".opencode/package-lock.json", rel == ".opencode/.gitignore":
		return true
	case strings.HasPrefix(rel, ".opencode/node_modules/"):
		return true
	}
	return false
}

// settingFiles resolves the descriptor's adapter setting and MCP layout files
// at both scopes: the files whose writes OPENCODE_CONFIG would hide and whose
// comments the structured writer cannot preserve. The .json and .jsonc
// spellings are both admitted.
func (opencodeValidator) settingFiles(d harness.Descriptor, in *harness.Admission) map[string]bool {
	files := map[string]bool{}
	for _, scope := range []string{"global", "local"} {
		for _, marker := range []string{"opencode.json", "opencode.jsonc"} {
			base := marker
			if scope == "global" {
				base = strings.TrimSuffix(d.GlobalMarker, "/") + "/" + marker
			}
			files[in.Resolver.ResolveMarker(base, d.Name, scope)] = true
		}
	}
	return files
}

func (v opencodeValidator) Prepare(d harness.Descriptor, in *harness.Admission) error {
	files := v.settingFiles(d, in)
	writesSettings := false
	for _, fd := range in.Changes.Diffs {
		if fd.Tool == d.Name && files[fd.Path] {
			writesSettings = true
		}
	}
	if value, ok := in.Env("OPENCODE_CONFIG"); ok && value != "" {
		if writesSettings {
			return fmt.Errorf("%s: OPENCODE_CONFIG is set; Patronus would merge into a file OpenCode does not read; unset it or point Patronus at the override", d.ErrorPrefix)
		}
		warn(in, fmt.Sprintf("%s: OPENCODE_CONFIG is set; placed settings are not read while it is in effect", d.ErrorPrefix))
	}
	if value, ok := in.Env("OPENCODE_CONFIG_CONTENT"); ok && value != "" {
		warn(in, fmt.Sprintf("%s: OPENCODE_CONFIG_CONTENT is set; inline config overrides placed settings", d.ErrorPrefix))
	}
	return nil
}

func (v opencodeValidator) BeforeOwnership(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff) error {
	if rel, err := filepath.Rel(in.Project, fd.Path); err == nil && harnessOwnedLocalPlugin(rel) {
		return fmt.Errorf("%s: %s is written by OpenCode for local plugins; Patronus never claims it", d.ErrorPrefix, fd.Path)
	}
	if v.settingFiles(d, in)[fd.Path] && len(fd.Before) > 0 && adapter.HasJSONComments(fd.Before) {
		return fmt.Errorf("%s: %s carries JSONC comments the structured writer cannot preserve; remove the comments or edit by hand", d.ErrorPrefix, fd.Path)
	}
	return nil
}

func (opencodeValidator) AfterOwnership(harness.Descriptor, *harness.Admission, diff.FileDiff, *harness.ContextFacts) error {
	return nil
}

// AfterContext proves provenance for every existing fence, including siblings
// the lenient selected-section walk does not collect. Plain prose needs no owner.
// Recorded bare artifact names remain valid legacy spellings (OC-CD5).
func (opencodeValidator) AfterContext(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff, _ *harness.ContextFacts) error {
	if fd.Section == nil && filepath.Base(fd.Path) != d.ContextFile {
		return nil
	}
	allowed := map[string]bool{}
	for _, owner := range in.Owners {
		for _, f := range owner.Files {
			if f.Path != fd.Path {
				continue
			}
			if owner.Tool != d.Name || owner.Scope != fd.Scope || owner.Artifact == "" ||
				f.Section == "" || f.Action != string(diff.Append) ||
				(f.Section != d.SectionPrefix+owner.Artifact && f.Section != owner.Artifact) ||
				allowed[f.Section] {
				return fmt.Errorf("%s context %s has incompatible or ambiguous section ownership", d.ErrorPrefix, fd.Path)
			}
			if f.Checksum != state.Checksum(fd.Before) {
				return fmt.Errorf("%s context %s section %s has drifted; explicit reconciliation required", d.ErrorPrefix, fd.Path, f.Section)
			}
			allowed[f.Section] = true
		}
	}
	if err := adapter.ValidateSectionMarkers(fd.Before, allowed); err != nil {
		return fmt.Errorf("%s context %s: %w", d.ErrorPrefix, fd.Path, err)
	}
	return nil
}

func warn(in *harness.Admission, message string) {
	if in.Warn != nil {
		in.Warn(message)
	}
}

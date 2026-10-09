package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
)

// codexValidator holds the Codex-only admission rules: the structured TOML
// inverse and raw-config refusal before ownership, and the strict context
// marker walk and selected-section identity rules after the shared context
// phase. It is stateless; everything it needs travels in the Admission.
type codexValidator struct{}

var _ harness.Validator = codexValidator{}

func (codexValidator) Prepare(harness.Descriptor, *harness.Admission) error { return nil }

func (codexValidator) BeforeOwnership(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff) error {
	inverseSettings, err := codexInverseSettings(fd, in.Owners, d.Name)
	if err != nil {
		return err
	}
	if filepath.Base(fd.Path) == "config.toml" && fd.Setting == nil && len(fd.SettingContrib) == 0 && !inverseSettings {
		return fmt.Errorf("%s config %s lacks structured SettingEdit ownership; raw TOML mutation refused", d.ErrorPrefix, fd.Path)
	}
	return nil
}

func (codexValidator) AfterOwnership(harness.Descriptor, *harness.Admission, diff.FileDiff, *harness.ContextFacts) error {
	return nil
}

// AfterContext validates every marker in the existing context against the
// known sections and proves the selected section inverse touches nothing
// else. Existing shared context needs positive provenance: no implicit
// consent, including for --force, and no whole-file restore across owners.
func (codexValidator) AfterContext(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff, facts *harness.ContextFacts) error {
	if fd.Section == nil && filepath.Base(fd.Path) != d.ContextFile {
		return nil
	}
	prefix := d.ErrorPrefix
	known := facts.Known
	allowed := map[string]bool{}
	for name := range known {
		allowed[name] = true
	}
	if err := adapter.ValidateSectionMarkers(fd.Before, allowed); err != nil {
		return fmt.Errorf("%s context %s: %w", prefix, fd.Path, err)
	}
	rest := string(fd.Before)
	seen := map[string]bool{}
	for {
		_, tail, ok := strings.Cut(rest, "<!-- patronus:start ")
		if !ok {
			break
		}
		name, after, ok := strings.Cut(tail, " -->")
		if !ok {
			return fmt.Errorf("%s context %s: malformed section", prefix, fd.Path)
		}
		if seen[name] {
			return fmt.Errorf("%s context %s: duplicate section %s requires reconciliation", prefix, fd.Path, name)
		}
		seen[name] = true
		if strings.HasPrefix(name, "pi:") || known[name] == "" {
			return fmt.Errorf("%s context %s section %s has unknown/Pi ownership; explicit migration required", prefix, fd.Path, name)
		}
		if _, ok := adapter.SectionBody(fd.Before, name); !ok {
			return fmt.Errorf("%s context %s section %s malformed", prefix, fd.Path, name)
		}
		rest = after
	}
	if fd.Action == diff.Unappend && fd.Section == nil {
		return fmt.Errorf("%s context %s lacks selected section identity", prefix, fd.Path)
	}
	if fd.Section == nil {
		return nil
	}
	expected := fd.Before
	selected := map[string]bool{}
	sections := []struct{ name, artifact string }{{fd.Section.Name, fd.Artifact}}
	for _, c := range fd.Contrib {
		sections = append(sections, struct{ name, artifact string }{c.Section, c.Artifact})
	}
	for _, s := range sections {
		if selected[s.name] || s.name != d.SectionPrefix+s.artifact {
			return fmt.Errorf("%s context %s has ambiguous selected section identity", prefix, fd.Path)
		}
		selected[s.name] = true
		if _, exists := adapter.SectionBody(fd.Before, s.name); exists && known[s.name] != s.artifact {
			return fmt.Errorf("%s context %s section %s has incompatible ownership", prefix, fd.Path, s.name)
		}
		if fd.Action == diff.Unappend {
			var found bool
			expected, found = adapter.RemoveSection(expected, s.name)
			if !found || known[s.name] != s.artifact {
				return fmt.Errorf("%s context %s lacks verified selected section %s", prefix, fd.Path, s.name)
			}
		}
	}
	if fd.Action == diff.Unappend && !bytes.Equal(expected, fd.After) {
		return fmt.Errorf("%s context %s inverse changes unselected sections or prose", prefix, fd.Path)
	}
	return nil
}

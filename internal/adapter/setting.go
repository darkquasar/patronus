package adapter

import (
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// transformSetting writes a setting artifact's value at its dotted path in the
// agent's settings file (a scalar MERGE — the twin of a hook's array-append). It
// rides the same config merger as MCP and hooks, but uses the wholesale-Prior
// remove path (state records the pre-install bytes; remove RESTOREs them) rather
// than the targeted element-strip, because a scalar key has one value to restore,
// not an array element to pull. A tool that models no setting surface at this
// scope is an honest no-op, so a @tool-flavoured setting installs cleanly only
// where it applies.
func (e *Engine) transformSetting(art *manifest.Artifact, ad *manifest.Adapter, scope string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	if ad.Layout.Setting == nil {
		return nil, nil // tool models no settings surface — honest skip
	}
	target := ad.Layout.Setting.ForScope(scope)
	if !target.OK() {
		return nil, nil
	}
	spec := art.Setting
	if spec == nil {
		return nil, fmt.Errorf("adapter: setting artifact %q missing setting block", art.Name)
	}

	path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)
	existing, _, err := readExisting(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read settings for %q: %w", art.Name, err)
	}
	after, err := MergeSettings(existing, target, spec.Path, spec.Value)
	if err != nil {
		return nil, fmt.Errorf("adapter: merge setting %q: %w", art.Name, err)
	}

	return []diff.FileDiff{{
		Path:   path,
		Action: diff.Merge,
		Before: existing,
		After:  after,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
		Note:   "setting " + strings.ReplaceAll(spec.Path, ".", "/") + ": " + art.Name,
		// Carry the scalar edit (IdentityKey "") so the planner can re-fold this set
		// onto an accumulated settings file without clobbering hooks merged before
		// it. Remove uses the wholesale-Prior path for a scalar (see fileUndo).
		Setting: &diff.SettingEdit{
			Target:      diff.FileTargetRef{File: target.File, Format: target.Format},
			Dotted:      spec.Path,
			ScalarValue: spec.Value,
		},
	}}, nil
}

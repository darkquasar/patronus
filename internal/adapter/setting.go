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
	if art.Setting == nil {
		return nil, fmt.Errorf("adapter: setting artifact %q missing setting block", art.Name)
	}
	// A setting may diverge per tool — same artifact, different native key/value
	// (native-sandbox: Claude's `sandbox` object vs Codex's `sandbox_mode` string).
	// Per-tool overrides (overrides.{tool}.path / .value) refine the base spec; the
	// settings FILE itself is picked by the adapter's per-tool setting layout above.
	dotted, value := settingFor(art, ad.Tool)

	path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)
	existing, _, err := readExisting(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read settings for %q: %w", art.Name, err)
	}
	after, err := MergeSettings(existing, target, dotted, value)
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
		Note:   "setting " + strings.ReplaceAll(dotted, ".", "/") + ": " + art.Name,
		// Carry the scalar edit (IdentityKey "") so the planner re-folds this set
		// onto an accumulated settings file without clobbering hooks merged before
		// it, and remove deletes exactly this key (see RemoveSettingScalar).
		Setting: &diff.SettingEdit{
			Target:      diff.FileTargetRef{File: target.File, Format: target.Format},
			Dotted:      dotted,
			ScalarValue: value,
		},
	}}, nil
}

// settingFor resolves a setting artifact's dotted path + value for one tool,
// applying any per-tool overrides (overrides.{tool}.path / .value) over the base
// setting block. This is how ONE native-sandbox artifact carries Claude's
// `sandbox` object and Codex's `sandbox_mode` string under a single name.
func settingFor(art *manifest.Artifact, tool string) (dotted string, value any) {
	dotted, value = art.Setting.Path, art.Setting.Value
	ov := art.Overrides[tool]
	if ov == nil {
		return dotted, value
	}
	if p, ok := ov["path"].(string); ok && p != "" {
		dotted = p
	}
	if v, ok := ov["value"]; ok {
		value = v
	}
	return dotted, value
}

package plugin

import (
	"fmt"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// Request is the input to Compute: one plugin onto one tool at one scope.
type Request struct {
	Plugin *manifest.Plugin
	Tool   string
	Scope  string
}

// Contribution is the plan-facing summary of what happened for this target: the
// resolved install mode and the source ecosystem it installs from. The dry-run
// uses it to print an honest per-target disposition (including the no-op skip an
// unsupported target produces).
type Contribution struct {
	Tool      string
	Mode      Mode
	Ecosystem string
}

// Compute returns the registration diffs (zero for an unsupported target) and a
// Contribution describing the per-target disposition for the dry-run.
//
// ResolveMode runs first. An unsupported target (a tool with no plugin construct,
// or a plugin with no usable source) is an honest no-op: no diffs, no error. A
// native or translate target writes ONE registration MERGE into the tool's plugin
// config at dotted path "plugins.<name>".
func Compute(req Request) ([]diff.FileDiff, Contribution, error) {
	mode, eco := ResolveMode(req.Plugin, req.Tool)
	contrib := Contribution{Tool: req.Tool, Mode: mode, Ecosystem: eco}
	if mode == ModeUnsupported {
		return nil, contrib, nil
	}
	d, err := buildRegistrationDiff(req, eco)
	if err != nil {
		return nil, contrib, fmt.Errorf("plugin %q on %s: %w", req.Plugin.Name, req.Tool, err)
	}
	return []diff.FileDiff{d}, contrib, nil
}

// buildRegistrationDiff constructs the registration MERGE for one plugin on one
// tool. It mirrors the scalar-setting MERGE in internal/adapter/transformSetting:
// a diff.Merge with the edit carried as a diff.SettingEdit whose IdentityKey is ""
// (the scalar-set form, re-foldable + restorable via wholesale Prior), here keyed
// to dotted path "plugins.<name>" with the registration value as the scalar.
//
// The on-disk registration path/format per tool is a known open question (the v1
// stand-in is the single dotted path "plugins.<name>"); the construction is kept
// tool-agnostic, exactly like the setting MERGE, so it does not depend on an
// adapter layout this package cannot see.
func buildRegistrationDiff(req Request, eco string) (diff.FileDiff, error) {
	name := req.Plugin.Name
	src, has := req.Plugin.Sources[eco]
	if !has {
		return diff.FileDiff{}, fmt.Errorf("no source for ecosystem %q", eco)
	}

	dotted := "plugins." + name
	value := map[string]any{
		"enabled":   true,
		"ecosystem": eco,
	}
	if src.Marketplace != "" {
		value["marketplace"] = src.Marketplace
	}
	if src.Ref != "" {
		value["ref"] = src.Ref
	}

	return diff.FileDiff{
		Action:   diff.Merge,
		Artifact: name,
		Type:     "plugin",
		Tool:     req.Tool,
		Scope:    req.Scope,
		Note:     "register plugin " + name + " (" + eco + ")",
		// Scalar-set form (IdentityKey ""): the registration value lives at one
		// dotted key, restored wholesale on remove — the twin of a setting MERGE.
		Setting: &diff.SettingEdit{
			Dotted:      dotted,
			ScalarValue: value,
		},
	}, nil
}

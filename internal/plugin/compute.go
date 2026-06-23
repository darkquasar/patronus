package plugin

import (
	"fmt"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// Request is the input to Compute: one plugin onto one tool at one scope, plus
// the adapter dependencies the registration MERGE is routed through. The engine,
// adapter, and existing-reader are injected (not hand-built) so the registration
// rides the SAME setting-merge transform every other setting artifact uses,
// rather than duplicating its diff construction. For an unsupported target the
// engine/adapter/reader are never touched and may be nil.
type Request struct {
	Plugin       *manifest.Plugin
	Tool         string
	Scope        string
	Engine       *adapter.Engine      // adapter.New(resolver)
	Adapter      *manifest.Adapter    // the per-tool adapter
	ReadExisting adapter.ReadExisting // closure over the filesystem (or a test stub)
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
// native or translate target registers the plugin by routing a synthetic setting
// artifact (dotted path "plugins.<name>") through the adapter setting transform,
// which produces an APPLICABLE settings MERGE — the same code path, and the same
// diff shape, as every other setting artifact. A tool that models no settings
// surface yields zero diffs (an honest skip from the transform) while still
// counting as a native/translate contribution.
func Compute(req Request) ([]diff.FileDiff, Contribution, error) {
	mode, eco := ResolveMode(req.Plugin, req.Tool)
	contrib := Contribution{Tool: req.Tool, Mode: mode, Ecosystem: eco}
	if mode == ModeUnsupported {
		return nil, contrib, nil
	}

	art := registrationArtifact(req.Plugin, eco, req.Tool)
	diffs, err := req.Engine.Transform(art, req.Adapter, req.Scope, "", req.ReadExisting)
	if err != nil {
		return nil, contrib, fmt.Errorf("plugin %q on %s: %w", req.Plugin.Name, req.Tool, err)
	}
	return diffs, contrib, nil
}

// registrationArtifact builds the synthetic setting artifact that registers one
// plugin on one tool. The registration is modeled as a setting MERGE at dotted
// path "plugins.<name>" so it rides the adapter's existing setting transform
// (an applicable scalar/object MERGE) instead of a hand-built diff. The engine
// stamps Type="setting" and Artifact=<name> on the resulting diff; that is the
// honest shape (a settings edit) and is left as-is.
func registrationArtifact(p *manifest.Plugin, eco, tool string) *manifest.Artifact {
	value := map[string]any{
		"enabled":   true,
		"ecosystem": eco,
	}
	if src, has := p.Sources[eco]; has {
		if src.Marketplace != "" {
			value["marketplace"] = src.Marketplace
		}
		if src.Ref != "" {
			value["ref"] = src.Ref
		}
	}

	return &manifest.Artifact{
		Meta: manifest.Meta{
			APIVersion: manifest.APIVersion,
			Family:     manifest.FamilyArtifact,
			Role:       manifest.RoleLifecycle,
			Name:       p.Name,
			Version:    p.Version,
		},
		Type:    manifest.TypeSetting,
		Targets: []string{tool},
		Setting: &manifest.SettingSpec{
			Path:  "plugins." + p.Name,
			Value: value,
		},
	}
}

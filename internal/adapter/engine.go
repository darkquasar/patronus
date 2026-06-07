// Package adapter transforms portable artifacts into per-tool, on-disk file
// changes. It is pure: it reads an artifact's source files and computes the
// would-be result bytes for each target path, emitting diff.FileDiffs. It never
// writes to disk — the planner classifies the diffs against the real filesystem
// and the (Phase 3) applier realizes them.
package adapter

import (
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// ReadExisting returns the current bytes of a target path, whether it exists,
// and any error. The planner injects a real filesystem reader; tests inject a
// map. Keeping it injected leaves the engine pure and unit-testable.
type ReadExisting func(path string) ([]byte, bool, error)

// Engine resolves layout path templates and computes per-tool file changes.
type Engine struct {
	resolver toolpath.Resolver
}

// New constructs an Engine with the given path resolver.
func New(resolver toolpath.Resolver) *Engine {
	return &Engine{resolver: resolver}
}

// Transform computes the file changes for installing one artifact onto one tool
// at one scope ("global"|"local"). srcDir is the artifact's on-disk directory
// (registry ArtifactEntry.Source.LocalDir). readExisting supplies current target
// content for APPEND/MERGE folding and is also stored as FileDiff.Before.
func (e *Engine) Transform(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	var (
		diffs []diff.FileDiff
		err   error
	)
	switch art.Kind {
	case manifest.KindSkill:
		diffs, err = e.transformSkill(art, ad, scope, srcDir)
	case manifest.KindInstruction:
		diffs, err = e.transformInstruction(art, ad, scope, srcDir, readExisting)
	case manifest.KindAgent:
		diffs, err = e.transformAgent(art, ad, scope, srcDir)
	case manifest.KindCommand:
		diffs, err = e.transformCommand(art, ad, scope, srcDir)
	default:
		return nil, fmt.Errorf("adapter: kind %q not supported for tool %q", art.Kind, ad.Tool)
	}
	if err != nil {
		return nil, err
	}
	// Stamp source-artifact identity + capability on every emitted diff so the
	// dry-run summary can group by artifact and label the added capability.
	cap := manifest.Capability(art.Kind, art.Role)
	for i := range diffs {
		diffs[i].Artifact = art.Name
		diffs[i].Capability = cap
	}
	return diffs, nil
}

// resolvePath substitutes {name} in a layout path template and expands it to an
// absolute path for the given tool/scope.
func (e *Engine) resolvePath(template, name, tool, scope string) string {
	marker := strings.ReplaceAll(template, "{name}", name)
	return e.resolver.ResolveMarker(marker, tool, scope)
}

package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// transformOutputStyle installs an output-style, with the action chosen by the
// adapter layout rather than a per-tool Go branch:
//   - a CREATE target (no action) writes a standalone output-styles/{name}.md
//     file with the body passed through verbatim (Claude's native surface).
//   - an appendSection target folds the body into AGENTS.md as an idempotent
//     fenced section (Codex/OpenCode, which have no output-style concept) — the
//     same APPEND machinery the instruction transform uses.
func (e *Engine) transformOutputStyle(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	if ad.Layout.OutputStyle == nil {
		return nil, fmt.Errorf("adapter %q: no output-style layout", ad.Tool)
	}
	target := ad.Layout.OutputStyle.ForScope(scope)
	if !target.OK() {
		return nil, fmt.Errorf("adapter %q: output-style has no %s target", ad.Tool, scope)
	}

	entry := art.Entry
	if entry == "" {
		return nil, fmt.Errorf("adapter: output-style %q missing entry", art.Name)
	}
	body, err := os.ReadFile(filepath.Join(srcDir, entry))
	if err != nil {
		return nil, fmt.Errorf("adapter: read output-style entry: %w", err)
	}

	// APPEND flavour (AGENTS.md): delegate to the shared section helper.
	if target.Action == "appendSection" {
		path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)
		d, err := e.appendSectionDiff(path, ad.Tool, scope, string(art.Role), art.Name, body, readExisting)
		if err != nil {
			return nil, err
		}
		return []diff.FileDiff{d}, nil
	}

	// CREATE flavour (Claude output-styles/{name}.md): passthrough body, like a
	// skill's SKILL.md. The authored body already carries any required frontmatter
	// (e.g. keep-coding-instructions: true) — passthrough never reshapes it.
	path := e.resolvePath(target.File, art.Name, ad.Tool, scope)
	return []diff.FileDiff{{
		Path:   path,
		Action: diff.Create,
		After:  body,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
	}}, nil
}

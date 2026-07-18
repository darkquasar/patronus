package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// transformCommand produces a CREATE diff for a Command: a single markdown file
// copied verbatim to the tool's commands directory.
func (e *Engine) transformCommand(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string) ([]diff.FileDiff, error) {
	if ad.Layout.Command == nil {
		return nil, fmt.Errorf("adapter %q: no Command layout", ad.Tool)
	}
	target := ad.Layout.Command.ForScope(scope)
	if !target.OK() {
		return nil, fmt.Errorf("adapter %q: Command has no %s target", ad.Tool, scope)
	}

	// Resolve the destination BEFORE reading the source: the path depends only on
	// the name + layout, never on the body, so a caller that needs to know where a
	// command WOULD land (drift's shadow hunt) can get it without the source present.
	path := e.resolvePath(target.Path, art.Name, ad.Tool, scope)

	entry := art.Entry
	if entry == "" {
		entry = art.Name + ".md"
	}
	body, err := os.ReadFile(filepath.Join(srcDir, entry))
	if err != nil {
		return nil, fmt.Errorf("adapter: read command entry: %w", err)
	}
	return []diff.FileDiff{{
		Path:   path,
		Action: diff.Create,
		After:  body,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
	}}, nil
}

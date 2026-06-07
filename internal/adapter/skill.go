package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// transformSkill produces CREATE diffs for a Skill: the SKILL.md body
// (passthrough — Claude/Codex/OpenCode all read it natively) plus every file in
// the artifact's supporting Files directories, copied verbatim into the skill
// directory.
func (e *Engine) transformSkill(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string) ([]diff.FileDiff, error) {
	if ad.Layout.Skill == nil {
		return nil, fmt.Errorf("adapter %q: no Skill layout", ad.Tool)
	}
	target := ad.Layout.Skill.ForScope(scope)
	if !target.OK() {
		return nil, fmt.Errorf("adapter %q: Skill has no %s target", ad.Tool, scope)
	}

	// The resolved SKILL.md path; its parent is the skill directory root.
	skillMd := e.resolvePath(target.Path, art.Name, ad.Tool, scope)
	skillDir := filepath.Dir(skillMd)

	var diffs []diff.FileDiff

	// 1. The entry body (SKILL.md), passthrough bytes.
	entry := art.Entry
	if entry == "" {
		entry = "SKILL.md"
	}
	body, err := os.ReadFile(filepath.Join(srcDir, entry))
	if err != nil {
		return nil, fmt.Errorf("adapter: read skill entry: %w", err)
	}
	diffs = append(diffs, diff.FileDiff{
		Path:   skillMd,
		Action: diff.Create,
		After:  body,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
	})

	// 2. Supporting Files directories, copied verbatim under the skill dir.
	for _, rel := range art.Files {
		rel = filepath.Clean(rel)
		ops, err := e.copyTree(filepath.Join(srcDir, rel), filepath.Join(skillDir, rel), ad.Tool, scope, string(art.Role))
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, ops...)
	}

	return diffs, nil
}

// copyTree enumerates every regular file under srcRoot and emits a CREATE diff
// mapping it to the corresponding path under dstRoot, content verbatim.
func (e *Engine) copyTree(srcRoot, dstRoot, tool, scope, role string) ([]diff.FileDiff, error) {
	var diffs []diff.FileDiff
	err := filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		diffs = append(diffs, diff.FileDiff{
			Path:   filepath.Join(dstRoot, rel),
			Action: diff.Create,
			After:  content,
			Tool:   tool,
			Scope:  scope,
			Role:   role,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("adapter: copy tree %s: %w", srcRoot, err)
	}
	return diffs, nil
}

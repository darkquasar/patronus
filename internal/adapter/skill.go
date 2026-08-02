package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Body placeholders a skill may use to name its own installed location portably,
// rather than hardcoding one agent's layout (`.claude/skills/...`).
const (
	skillDirPlaceholder  = "{skillDir}"  // the directory this skill installs into
	skillsDirPlaceholder = "{skillsDir}" // its parent, holding all installed skills
)

// transformSkill produces CREATE diffs for a Skill: the SKILL.md body
// (passthrough — Claude/Codex/OpenCode all read it natively) plus every file in
// the artifact's supporting Files directories, copied into the skill directory
// with the skill-dir body placeholders resolved.
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
	ph := e.skillPlaceholdersFor(skillDir, scope)

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
		After:  ph.apply(body),
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
	})

	// 2. Supporting Files directories, copied under the skill dir.
	for _, rel := range art.Files {
		rel = filepath.Clean(rel)
		ops, err := e.copyTree(filepath.Join(srcDir, rel), filepath.Join(skillDir, rel), ad.Tool, scope, string(art.Role), ph)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, ops...)
	}

	return diffs, nil
}

// skillPlaceholdersFor builds the substitution for a skill installed at skillDir.
// The paths are relative to the project at project scope — so a body that names
// them stays valid in a checked-in tree — and absolute at global scope, where the
// skill lands under the home directory, nowhere near the open repo.
//
// Derivation goes through the already-resolved skillDir rather than re-templating
// the adapter marker, so a placeholder can never disagree with where the file
// actually lands.
func (e *Engine) skillPlaceholdersFor(skillDir, scope string) skillPlaceholders {
	if scope != toolpath.ScopeGlobal {
		skillDir = e.resolver.RelativeTo(skillDir)
	}
	return skillPlaceholders{skillDir: skillDir, skillsDir: filepath.Dir(skillDir)}
}

// skillPlaceholders resolves the skill-dir body placeholders for one installed skill.
type skillPlaceholders struct {
	skillDir  string
	skillsDir string
}

// apply substitutes the known placeholders in content. Unknown {…} strings are
// left untouched, matching {toolContext} in internal/recipe: skill bodies carry
// braces in JSON, f-strings, and shell expansions, so only an exact match
// substitutes. Content that is not valid UTF-8 is returned unchanged, keeping
// binary assets byte-identical by construction rather than by luck.
func (s skillPlaceholders) apply(content []byte) []byte {
	if !utf8.Valid(content) {
		return content
	}
	out := strings.ReplaceAll(string(content), skillDirPlaceholder, s.skillDir)
	out = strings.ReplaceAll(out, skillsDirPlaceholder, s.skillsDir)
	return []byte(out)
}

// copyTree enumerates every regular file under srcRoot and emits a CREATE diff
// mapping it to the corresponding path under dstRoot, with the skill-dir body
// placeholders resolved (see skillPlaceholders.apply — a file carrying none is
// copied verbatim).
func (e *Engine) copyTree(srcRoot, dstRoot, tool, scope, role string, ph skillPlaceholders) ([]diff.FileDiff, error) {
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
			After:  ph.apply(content),
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

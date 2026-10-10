package scan

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OpenCodeSkillRoots returns the skill roots OpenCode 1.18.35 reads, as shared
// skill roots carrying the OpenCode walk rules: the two native write roots
// (.opencode/skills, <config>/skills), their singular spellings, and the
// co-visible Claude and .agents roots. The global config base is
// OPENCODE_CONFIG_DIR, else XDG_CONFIG_HOME/opencode, else ~/.config/opencode
// (the toolpath resolver's precedence). Both Claude roots are recursive
// (~/.claude/skills holds synced/<id>/<name>/SKILL.md trees). No root is legacy:
// every one is a place OpenCode still loads from.
func OpenCodeSkillRoots(home, project string, env EnvLookup) []SkillRoot {
	base := openCodeConfigBase(home, env)
	var out []SkillRoot
	add := func(path, origin string, recursive bool) {
		out = append(out, openCodeSkillRoot(path, origin, recursive))
	}
	add(filepath.Join(project, ".opencode/skills"), "project", false)
	add(filepath.Join(project, ".opencode/skill"), "project", false)
	add(filepath.Join(project, ".claude/skills"), "claude-compat", true)
	add(filepath.Join(project, ".agents/skills"), "agents", false)
	add(filepath.Join(base, "skills"), "global", false)
	add(filepath.Join(base, "skill"), "global", false)
	add(filepath.Join(home, ".claude/skills"), "claude-compat", true)
	add(filepath.Join(home, ".agents/skills"), "docs-derived", false)
	return out
}

func openCodeConfigBase(home string, env EnvLookup) string {
	if env != nil {
		if dir, ok := env("OPENCODE_CONFIG_DIR"); ok && dir != "" {
			return dir
		}
		if xdg, ok := env("XDG_CONFIG_HOME"); ok && xdg != "" {
			return filepath.Join(xdg, "opencode")
		}
	}
	return filepath.Join(home, ".config/opencode")
}

func openCodeSkillRoot(path, origin string, recursive bool) SkillRoot {
	return SkillRoot{
		Path:            path,
		Origin:          origin,
		Recursive:       recursive,
		DuplicatePolicy: RefuseDifferentBytesVerifiedException,
		Name:            SkillDirName,
		ConflictMessage: openCodeSkillConflict,
	}
}

func openCodeSkillConflict(name string, a, b Resource) error {
	return fmt.Errorf("opencode skill name %q conflicts at %s and %s; remove one copy or rename", name, a.Path, b.Path)
}

// SkillDirName is the neutral name == directory rule: the frontmatter name
// must equal the SKILL.md parent directory. It carries no target wording, so
// any descriptor whose adapter declares nameSource: dir can use it.
func SkillDirName(raw []byte, path string) (string, error) {
	fields, err := piMarkdownFields(raw)
	if err != nil {
		return "", fmt.Errorf("skill %s: %w", path, err)
	}
	name, _ := fields["name"].(string)
	dir := filepath.Base(filepath.Dir(path))
	if strings.TrimSpace(name) == "" || name != dir {
		return "", fmt.Errorf("skill %s: frontmatter name %q must equal directory %q", path, name, dir)
	}
	return name, nil
}

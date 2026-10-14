package scan

import "path/filepath"

// ClaudeSkillRoots describes the adapter's project and global skill roots.
// Recursive discovery includes synced/<id>/<name>/SKILL.md installations.
// These roots provide static co-visibility evidence, not runtime qualification.
func ClaudeSkillRoots(home, project string) []SkillRoot {
	return []SkillRoot{
		{Path: filepath.Join(project, ".claude/skills"), Origin: "project", Recursive: true, DuplicatePolicy: RefuseAlways, Name: SkillDirName},
		{Path: filepath.Join(home, ".claude/skills"), Origin: "global", Recursive: true, DuplicatePolicy: RefuseAlways, Name: SkillDirName},
	}
}

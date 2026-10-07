package scan

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CodexSkillRoot includes retired roots for conservative admission, not delivery.
type CodexSkillRoot struct {
	Path   string
	Legacy bool
}
type CodexSkill struct {
	Name, Path string
	Legacy     bool
	Content    []byte
}

// CodexSkillRoots records the approved .agents roots and inspects old .codex
// candidates too. Ancestor candidates stop at the nearest repository boundary.
// It performs no loader execution, migration, planning or writes.
func CodexSkillRoots(home, project, config string) []CodexSkillRoot {
	var out []CodexSkillRoot
	seen := map[string]bool{}
	add := func(path string, legacy bool) {
		if !seen[path] {
			seen[path] = true
			out = append(out, CodexSkillRoot{path, legacy})
		}
	}
	add(filepath.Join(home, ".agents/skills"), false)
	add(filepath.Join(home, ".codex/skills"), true)
	add(filepath.Join(config, "skills"), true)
	for dir := project; ; dir = filepath.Dir(dir) {
		add(filepath.Join(dir, ".agents/skills"), false)
		add(filepath.Join(dir, ".codex/skills"), true)
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil || filepath.Dir(dir) == dir {
			break
		}
	}
	return out
}

// CodexSafePath shares the existing filesystem alias/unreadable-input guard.
func CodexSafePath(path string) error {
	if err := PiSafePath(path); err != nil {
		return fmt.Errorf("codex admission: %w", err)
	}
	return nil
}

var _codexSkillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func CodexSkillName(raw []byte, path string) (string, error) {
	fields, err := piMarkdownFields(raw)
	if err != nil {
		return "", fmt.Errorf("codex skill %s: %w", path, err)
	}
	name, _ := fields["name"].(string)
	description, _ := fields["description"].(string)
	if len(name) > 64 || !_codexSkillName.MatchString(name) || name != filepath.Base(filepath.Dir(path)) || strings.TrimSpace(description) == "" {
		return "", fmt.Errorf("codex skill %s: malformed name/description; name must match its directory and use lowercase hyphenated identity", path)
	}
	return name, nil
}

func DiscoverCodexSkills(roots []CodexSkillRoot) ([]CodexSkill, error) {
	return DiscoverCodexSkillsWithOwnership(roots, nil)
}

// DiscoverCodexSkillsWithOwnership permits differing scoped versions only when
// the caller proves both resources are managed, unchanged Codex installations.
// A nil verifier retains the conservative discovery-only conflict policy.
func DiscoverCodexSkillsWithOwnership(roots []CodexSkillRoot, verified func(CodexSkill) bool) ([]CodexSkill, error) {
	var out []CodexSkill
	for _, root := range roots {
		if err := CodexSafePath(root.Path); err != nil {
			return nil, err
		}
		err := filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, err error) error {
			if os.IsNotExist(err) && path == root.Path {
				return nil
			}
			if err != nil {
				return err
			}
			if err := CodexSafePath(path); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() != "SKILL.md" {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			name, err := CodexSkillName(raw, path)
			if err != nil {
				return err
			}
			selected := CodexSkill{Name: name, Path: path, Legacy: root.Legacy, Content: raw}
			for _, other := range out {
				if other.Name == name && other.Path != path && !bytes.Equal(other.Content, raw) {
					if !other.Legacy && !selected.Legacy && verified != nil && verified(other) && verified(selected) {
						continue
					}
					return fmt.Errorf("codex skill name %q conflicts at %s and %s; incompatible same-name resources cannot be shadowed", name, other.Path, path)
				}
			}
			out = append(out, selected)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CodexSkillTree lists every regular file below one skill directory without
// following links. Any symlink, nonregular entry or unreadable path is refused,
// so a caller can prove the complete payload before relocating it. A missing
// directory yields no files.
func CodexSkillTree(dir string) ([]string, error) {
	if err := CodexSafePath(dir); err != nil {
		return nil, err
	}
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == dir {
			return nil
		}
		if err != nil {
			return err
		}
		if err := CodexSafePath(path); err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CodexSkillOccurrences lists every SKILL.md naming the skill across roots,
// with no conflict or shadowing policy. Callers must prove each occurrence's
// ownership themselves; nothing here adopts or plans.
func CodexSkillOccurrences(roots []CodexSkillRoot, name string) ([]CodexSkill, error) {
	var out []CodexSkill
	for _, root := range roots {
		files, err := CodexSkillTree(root.Path)
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			if filepath.Base(path) != "SKILL.md" || filepath.Base(filepath.Dir(path)) != name {
				continue
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			out = append(out, CodexSkill{Name: name, Path: path, Legacy: root.Legacy, Content: raw})
		}
	}
	return out, nil
}

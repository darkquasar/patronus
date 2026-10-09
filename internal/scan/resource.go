package scan

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DuplicatePolicy says how two same-name resources discovered at different
// paths are judged. It is per-root data so each target keeps its historical
// rule without the shared walk naming a target.
type DuplicatePolicy int

const (
	// RefuseAlways refuses any same-name (or alias) resource regardless of its
	// bytes; it is evaluated over the complete inventory after every root was
	// walked (the Pi rule).
	RefuseAlways DuplicatePolicy = iota + 1
	// RefuseDifferentBytesVerifiedException admits identical bytes and refuses
	// differing bytes unless the caller proves both copies are verified, unchanged
	// managed installations and neither root is legacy; it is evaluated as each
	// resource is discovered (the Codex rule).
	RefuseDifferentBytesVerifiedException
)

// Resource is one discovered skill (or, through the Pi wrappers, agent or
// command): exact identity and source, never inferred equivalence.
type Resource struct {
	Kind, Name, Path, Origin string
	Legacy                   bool
	Content                  []byte
	Aliases                  []string
}

// SkillRoot is one discovery root with the walk rules its target applies.
// Every zero flag is the least surprising rule: descend only into directories
// without SKILL.md, consult no ignore files, skip nothing, read no loose
// markdown, use the shared SafePath guard and a generic conflict message.
type SkillRoot struct {
	Path, Origin string
	Legacy       bool
	// Recursive keeps descending below a directory that yielded SKILL.md and
	// recognizes SKILL.md as a file entry anywhere; false treats such a
	// directory as a leaf claimed by its SKILL.md.
	Recursive bool
	// AgentsSkills marks a pinned .agents skill collection: loose markdown is read
	// below the root instead of directly in it.
	AgentsSkills bool
	// IgnoreRules refuses a directory carrying a nonempty .gitignore, .ignore or
	// .fdignore, whose filtering this static walk cannot reproduce.
	IgnoreRules bool
	// SkipDotDirs skips dot-directories and node_modules below the root.
	SkipDotDirs bool
	// LooseMarkdown reads .md files other than SKILL.md as resources (directly in
	// the root, or below it when AgentsSkills is set).
	LooseMarkdown   bool
	DuplicatePolicy DuplicatePolicy
	// SafePath guards the root and every entry; nil uses SafePath.
	SafePath func(path string) error
	// Name derives the resource name from the file bytes; "" ignores the file
	// silently; an error stops discovery and is returned as-is.
	Name func(raw []byte, path string) (string, error)
	// ConflictMessage builds the refusal for two conflicting resources; name is
	// the identity that matched. nil uses a generic message.
	ConflictMessage func(name string, a, b Resource) error
}

func (r SkillRoot) safePath(path string) error {
	if r.SafePath != nil {
		return r.SafePath(path)
	}
	return SafePath(path)
}

// DiscoverSkills walks every root in order and returns the discovered skills.
// Conflicts under RefuseDifferentBytesVerifiedException are detected as each
// resource is found; conflicts under RefuseAlways are detected over the
// complete inventory afterwards, so walk errors keep precedence as before.
func DiscoverSkills(roots []SkillRoot, verified func(Resource) bool) ([]Resource, error) {
	var out []Resource
	var policies []SkillRoot
	for _, root := range roots {
		found, err := walkSkillRoot(root)
		if err != nil {
			return nil, err
		}
		for _, r := range found {
			if root.DuplicatePolicy == RefuseDifferentBytesVerifiedException {
				if err := ResourceConflict(out, r, root.DuplicatePolicy, verified, root.ConflictMessage); err != nil {
					return nil, err
				}
			}
			out = append(out, r)
			policies = append(policies, root)
		}
	}
	for i, r := range out {
		if policies[i].DuplicatePolicy != RefuseAlways {
			continue
		}
		if err := ResourceConflict(out[:i], r, RefuseAlways, verified, policies[i].ConflictMessage); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// walkSkillRoot discovers the resources below one root without any conflict
// check, so a caller composing several kinds can defer identity checks.
func walkSkillRoot(root SkillRoot) ([]Resource, error) {
	if err := root.safePath(root.Path); err != nil {
		return nil, err
	}
	var out []Resource
	add := func(raw []byte, path string) error {
		if root.Name == nil {
			return fmt.Errorf("scan: root %s has no naming rule", root.Path)
		}
		name, err := root.Name(raw, path)
		if err != nil {
			return err
		}
		if name == "" {
			return nil
		}
		out = append(out, Resource{Kind: "skill", Name: name, Path: path, Origin: root.Origin, Legacy: root.Legacy, Content: raw})
		return nil
	}
	err := filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == root.Path {
			return nil
		}
		if err != nil {
			return err
		}
		if err := root.safePath(path); err != nil {
			return err
		}
		if d.IsDir() {
			return walkSkillDir(root, path, add)
		}
		if d.Name() == "SKILL.md" {
			if !root.Recursive {
				return nil // the directory entry already claimed it
			}
			raw, _, err := ReadPiFile(path)
			if err != nil {
				return err
			}
			return add(raw, path)
		}
		if !root.LooseMarkdown || !strings.HasSuffix(path, ".md") {
			return nil
		}
		atRoot := filepath.Dir(path) == root.Path
		if root.AgentsSkills == atRoot {
			return nil
		}
		raw, _, err := ReadPiFile(path)
		if err != nil {
			return err
		}
		return add(raw, path)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func walkSkillDir(root SkillRoot, path string, add func(raw []byte, path string) error) error {
	if path != root.Path && root.SkipDotDirs && (filepath.Base(path) == "node_modules" || strings.HasPrefix(filepath.Base(path), ".")) {
		return filepath.SkipDir
	}
	if root.IgnoreRules {
		for _, ignore := range []string{".gitignore", ".ignore", ".fdignore"} {
			raw, exists, err := ReadPiFile(filepath.Join(path, ignore))
			if err != nil {
				return err
			}
			if exists && len(bytes.TrimSpace(raw)) > 0 {
				return fmt.Errorf("pi skill source %s: ignore rules require qualified discovery", filepath.Join(path, ignore))
			}
		}
	}
	if root.Recursive {
		return nil
	}
	file := filepath.Join(path, "SKILL.md")
	raw, exists, err := ReadPiFile(file)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := add(raw, file); err != nil {
		return err
	}
	return filepath.SkipDir
}

// ResourceConflict judges selected against already discovered resources under
// policy. Identity is kind-scoped and includes aliases. message nil uses a
// generic text; verified may be nil (no exception).
func ResourceConflict(resources []Resource, selected Resource, policy DuplicatePolicy, verified func(Resource) bool, message func(name string, a, b Resource) error) error {
	if message == nil {
		message = genericConflict
	}
	selectedNames := append([]string{selected.Name}, selected.Aliases...)
	for _, other := range resources {
		if other.Kind != selected.Kind || other.Path == selected.Path {
			continue
		}
		otherNames := append([]string{other.Name}, other.Aliases...)
		for _, selectedName := range selectedNames {
			for _, otherName := range otherNames {
				if otherName != selectedName {
					continue
				}
				if policy == RefuseDifferentBytesVerifiedException {
					if bytes.Equal(other.Content, selected.Content) {
						continue
					}
					if !other.Legacy && !selected.Legacy && verified != nil && verified(other) && verified(selected) {
						continue
					}
				}
				return message(selectedName, other, selected)
			}
		}
	}
	return nil
}

func genericConflict(name string, a, b Resource) error {
	return fmt.Errorf("%s %q conflicts at %s and %s; incompatible same-name resources cannot be shadowed", a.Kind, name, a.Path, b.Path)
}

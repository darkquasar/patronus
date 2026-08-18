package remove

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// A skill OWNS the directory holding its files: scan treats that directory as the
// artifact's occupancy marker, so an empty leftover reads as an installed skill.
// Deleting the recorded files was never enough — nothing in the tree pruned the
// directory, and because an empty directory has no state row, no drift pass could
// even see it. Removing a skill left a ghost that only rmdir by hand could clear.
//
// File-shaped artifacts (agent, command, output-style) are the opposite case:
// they SIT IN a shared directory they do not own, and pruning .claude/agents/
// because it happened to empty would delete a directory Patronus never created.
// Type is what tells the two apart, which is why it is the gate.

// skillMarker is the file whose parent identifies a skill's owned root. Every
// built-in skill layout places it at the directory's top level.
const skillMarker = "SKILL.md"

// Prune removes the directories a directory-shaped artifact owned, once its files
// are gone. It is deliberately rmdir-only and deepest-first: os.Remove on a
// directory is non-recursive, so a directory holding anything at all survives.
// RemoveAll on a path under the user's home would destroy files Patronus never
// wrote, which is far worse than the leftover it would clean up.
//
// Call it only when every tracked deletion for the item is logically complete.
// Under that gate a non-empty directory can only mean content Patronus does not
// track, so the retention warning needs no forensics and makes no accusation.
//
// A non-empty directory is a RETENTION, not a failure: it returns a warning and no
// error. Any other failure IS an error, and the caller must keep the state row —
// forgetting a directory Patronus owns and failed to clean is the exact
// invisibility this exists to end.
func Prune(item state.Item) ([]Warning, error) {
	root, ok := ownedRoot(item)
	if !ok {
		return nil, nil
	}

	dirs := ancestorsWithin(item, root)
	for _, dir := range dirs {
		// A lexical containment check is not enough to make a delete safe. Rel()
		// proves the RECORDED path sits under the root, but os.Remove follows
		// symlinked parent components: if a directory inside the skill is a link
		// into the user's own tree, "prune the empty directory under our root"
		// silently becomes "delete an empty directory somewhere else entirely".
		// Resolve against the real filesystem and refuse anything that leaves the
		// tree, and never follow a link when removing.
		real, ok, err := realPathWithin(root, dir)
		if err != nil {
			return nil, fmt.Errorf("remove: prune %s: %w", dir, err)
		}
		if !ok {
			continue // a link out of the tree, or gone: not ours to remove
		}
		err = os.Remove(real)
		switch {
		case err == nil, errors.Is(err, os.ErrNotExist):
			// Gone, or already gone: pruning what is not there is a no-op.
			continue
		case isNotEmpty(err):
			// Authoritative rather than pre-checked: a ReadDir before the remove
			// would be a check/use race. Read the entries only NOW, to name them.
			return []Warning{{
				Item:    item.Artifact,
				Path:    dir,
				Message: "directory retained because it is not empty: " + strings.Join(entriesOf(real), ", "),
			}}, nil
		default:
			return nil, fmt.Errorf("remove: prune %s: %w", dir, err)
		}
	}
	return nil, nil
}

// realPathWithin resolves dir through any symlinks and reports whether the REAL
// directory still lies inside the real root — and whether it is a directory at
// all rather than a link standing in for one. It answers false (not an error)
// when the path is simply gone, which is the ordinary case for a tree that
// already emptied.
//
// The lexical check in within() protects the recorded paths; this protects the
// delete itself. Both are needed: one bounds what we plan, the other bounds what
// the kernel actually acts on.
func realPathWithin(root, dir string) (string, bool, error) {
	if fi, err := os.Lstat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	} else if fi.Mode()&os.ModeSymlink != 0 {
		// A symlink is never a directory WE created, whatever it points at.
		return "", false, nil
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if !within(realRoot, real) {
		return "", false, nil // a linked parent led out of the tree
	}
	return real, true, nil
}

// ownedRoot derives the directory this item owns from its RECORDED paths, and
// reports whether pruning is permitted at all.
//
// Type gates the inference but does not perform it: "skill" says the artifact is
// directory-SHAPED, not WHICH directory. The practical invariant is the unique
// recorded SKILL.md parent, so a modern typed row and a legacy row without Type
// use the same four conditions. All must hold, or nothing is pruned:
//
//  1. at least one recorded CREATE named exactly SKILL.md;
//  2. every file of the item at or below that file's parent;
//  3. no path escaping the root once cleaned;
//  4. exactly ONE plausible root — ambiguity means do not prune.
//
// The root comes from recorded paths ONLY, never from the artifact name or the
// current catalog: those drift apart from what was written after a rename, which
// is a live scenario in this repo rather than a hypothetical.
func ownedRoot(item state.Item) (string, bool) {
	if item.Type != "" && item.Type != "skill" {
		return "", false // a file-shaped artifact never owns its parent
	}

	roots := map[string]bool{}
	for _, f := range item.Files {
		if f.Action == string(diff.Create) && filepath.Base(f.Path) == skillMarker {
			roots[filepath.Dir(filepath.Clean(f.Path))] = true
		}
	}
	if len(roots) != 1 {
		return "", false // no marker, or an ambiguous layout
	}
	var root string
	for r := range roots {
		root = r
	}
	if isSharedContainer(root) {
		// The marker sits DIRECTLY in a container several artifacts share, so its
		// parent is not a directory this item owns — every layout places a skill at
		// <container>/<name>/SKILL.md, and a marker one level up means the state row
		// is legacy, hand-written, or malformed. Deleting the container would take
		// every other artifact's directory with it the moment it emptied.
		return "", false
	}

	for _, f := range item.Files {
		if !within(root, f.Path) {
			return "", false // a file outside the inferred root: not one owned tree
		}
	}
	return root, true
}

// sharedContainers are the directory names the adapter layouts use to HOLD
// artifacts, as opposed to the per-artifact directory beneath them. A skill lives
// at <container>/<name>/SKILL.md in every layout, so a root whose basename is one
// of these is a container Patronus never owns exclusively.
var sharedContainers = map[string]bool{
	"skills":   true,
	"agents":   true,
	"commands": true,
}

// isSharedContainer reports whether root names a directory artifacts share rather
// than one a single artifact owns.
func isSharedContainer(root string) bool {
	return sharedContainers[filepath.Base(root)]
}

// within reports whether path is at or below root, after cleaning, so a "../"
// segment cannot smuggle a file outside the tree past the check.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ancestorsWithin collects every directory between the item's files and its owned
// root (inclusive), deepest first. Skills can nest, and the adapter records their
// files recursively without recording any directory, so the set has to be
// RECONSTRUCTED from file paths rather than read out of state. Deepest-first is
// what lets a nested tree empty from the leaves up in one pass.
func ancestorsWithin(item state.Item, root string) []string {
	seen := map[string]bool{root: true}
	for _, f := range item.Files {
		for dir := filepath.Dir(filepath.Clean(f.Path)); within(root, dir); dir = filepath.Dir(dir) {
			if seen[dir] {
				break
			}
			seen[dir] = true
		}
	}
	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		di, dj := strings.Count(dirs[i], string(filepath.Separator)), strings.Count(dirs[j], string(filepath.Separator))
		if di != dj {
			return di > dj
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

// isNotEmpty reports whether err means "the directory still has contents".
// Platforms disagree: ENOTEMPTY is usual, but some return EEXIST for the same
// condition. Without tolerating both, an ordinary non-empty directory would fail
// the command instead of being retained.
func isNotEmpty(err error) bool {
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST)
}

// entriesOf names a retained directory's IMMEDIATE entries for the warning — not
// a recursive walk, which would bury the useful line in noise.
func entriesOf(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

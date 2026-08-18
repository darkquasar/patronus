package remove

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// skillItem records a directory-shaped artifact rooted at dir, with SKILL.md plus
// any extra relative paths.
func skillItem(dir string, extra ...string) state.Item {
	files := []state.FileState{{Path: filepath.Join(dir, skillMarker), Action: string(diff.Create)}}
	for _, e := range extra {
		files = append(files, state.FileState{Path: filepath.Join(dir, e), Action: string(diff.Create)})
	}
	return state.Item{Artifact: filepath.Base(dir), Type: "skill", Tool: "claude", Scope: "global", Files: files}
}

// write creates path (and its parents) with some content.
func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPruneRemovesEmptiedSkillDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "writing-style")
	item := skillItem(dir)
	// The applier has already deleted the files; only the directory is left.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	warns, err := Prune(item)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("an empty owned directory needs no warning, got %+v", warns)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the emptied skill directory must be gone, stat err = %v", err)
	}
	// And only its own: the shared parent it sat in is untouched.
	if _, err := os.Stat(filepath.Join(root, "skills")); err != nil {
		t.Errorf("the shared parent must survive: %v", err)
	}
}

func TestPruneKeepsDirectoryHoldingUnrecordedFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "writing-style")
	item := skillItem(dir)
	write(t, filepath.Join(dir, "notes.md")) // the user's own file

	warns, err := Prune(item)
	if err != nil {
		t.Fatalf("a non-empty directory is a retention, never an error: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("a directory holding unrecorded files must survive: %v", err)
	}
	if len(warns) != 1 {
		t.Fatalf("want one retention warning, got %+v", warns)
	}
	if !strings.Contains(warns[0].Message, "notes.md") {
		t.Errorf("the warning must name what is there, got %q", warns[0].Message)
	}
	// Neutral about WHO left the entries: it reports the fact, not a culprit.
	for _, accusation := range []string{"you ", "your ", "user"} {
		if strings.Contains(strings.ToLower(warns[0].Message), accusation) {
			t.Errorf("the warning must not accuse the user, got %q", warns[0].Message)
		}
	}
}

func TestPruneNestedDirectoriesDeepestFirst(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "deep-skill")
	item := skillItem(dir, filepath.Join("references", "inner", "note.md"))
	// Files already deleted; the nested tree remains.
	if err := os.MkdirAll(filepath.Join(dir, "references", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Prune(item); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("a nested skill tree must empty from the leaves up in one pass")
	}
}

func TestPruneNeverTouchesFileShapedArtifacts(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "agents")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		t.Fatal(err)
	}
	// An agent SITS IN a shared directory it does not own. Even now that the
	// directory is empty, pruning it would delete something Patronus never made.
	item := state.Item{
		Artifact: "reviewer", Type: "agent", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: filepath.Join(agents, "reviewer.md"), Action: string(diff.Create)}},
	}

	if _, err := Prune(item); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(agents); err != nil {
		t.Errorf("a file-shaped artifact's shared parent must never be pruned: %v", err)
	}
}

func TestPruneLeavesTheOtherSkillAlone(t *testing.T) {
	root := t.TempDir()
	mine := filepath.Join(root, "mine")
	theirs := filepath.Join(root, "theirs")
	if err := os.MkdirAll(mine, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(theirs, skillMarker))

	if _, err := Prune(skillItem(mine)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(theirs); err != nil {
		t.Errorf("removing one skill must not touch another's directory: %v", err)
	}
}

func TestPruneIsIdempotent(t *testing.T) {
	root := t.TempDir()
	item := skillItem(filepath.Join(root, "already-gone"))
	// The directory never existed / was already cleaned by hand.
	warns, err := Prune(item)
	if err != nil {
		t.Fatalf("pruning an absent directory is a no-op, not an error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("nothing to report, got %+v", warns)
	}
}

// Legacy rows carry no Type. They still prune via the SKILL.md-parent invariant —
// but only when the layout is unambiguous.
func TestPruneLegacyRowWithoutType(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "legacy")
	item := skillItem(dir)
	item.Type = ""
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Prune(item); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("an unambiguous legacy skill layout must still prune")
	}
}

func TestPruneRefusesAmbiguousLayouts(t *testing.T) {
	root := t.TempDir()

	noMarker := state.Item{
		Artifact: "no-marker", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: filepath.Join(root, "a", "README.md"), Action: string(diff.Create)}},
	}
	twoMarkers := state.Item{
		Artifact: "two-roots", Tool: "claude", Scope: "global",
		Files: []state.FileState{
			{Path: filepath.Join(root, "b", skillMarker), Action: string(diff.Create)},
			{Path: filepath.Join(root, "c", skillMarker), Action: string(diff.Create)},
		},
	}
	escapes := state.Item{
		Artifact: "escapee", Tool: "claude", Scope: "global",
		Files: []state.FileState{
			{Path: filepath.Join(root, "d", skillMarker), Action: string(diff.Create)},
			{Path: filepath.Join(root, "elsewhere", "stray.md"), Action: string(diff.Create)},
		},
	}

	for _, tt := range []struct {
		name string
		item state.Item
		dirs []string
	}{
		{"no SKILL.md marker", noMarker, []string{"a"}},
		{"more than one plausible root", twoMarkers, []string{"b", "c"}},
		{"a file outside the inferred root", escapes, []string{"d", "elsewhere"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Prune(tt.item); err != nil {
				t.Fatal(err)
			}
			for _, d := range tt.dirs {
				if _, err := os.Stat(filepath.Join(root, d)); err != nil {
					t.Errorf("an ambiguous layout must NOT prune %s: %v", d, err)
				}
			}
		})
	}
}

// The hard constraint: rmdir semantics, never RemoveAll. A directory with content
// survives with everything in it.
func TestPruneIsNeverRecursive(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skill")
	item := skillItem(dir)
	nested := filepath.Join(dir, "user-stuff", "keep.md")
	write(t, nested)

	if _, err := Prune(item); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("a file Patronus never wrote must survive the prune: %v", err)
	}
}

// --- peer-review regressions -------------------------------------------------

// A lexical containment check proves the RECORDED path sits under the root, but
// os.Remove follows symlinked parents: a link inside the skill pointing into the
// user's own tree would turn "prune our empty directory" into deleting an empty
// directory somewhere else entirely.
func TestPruneRefusesToFollowASymlinkOutOfTheTree(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "user-tree", "generated")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "skills", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// assets/ inside the skill is a LINK into the user's own tree.
	link := filepath.Join(dir, "assets")
	if err := os.Symlink(filepath.Join(root, "user-tree"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	item := skillItem(dir, filepath.Join("assets", "generated", "file.txt"))
	if _, err := Prune(item); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outside); err != nil {
		t.Errorf("a directory reached only through a symlink is not ours to delete: %v", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("the symlink itself must not be removed either: %v", err)
	}
}

// A marker recorded DIRECTLY in a shared container means the state row is legacy,
// hand-written, or malformed — not that Patronus owns the container. Deleting it
// would take every other artifact's directory with it the moment it emptied.
func TestPruneRefusesASharedContainerAsRoot(t *testing.T) {
	for _, container := range []string{"skills", "agents", "commands"} {
		t.Run(container, func(t *testing.T) {
			root := t.TempDir()
			shared := filepath.Join(root, ".claude", container)
			if err := os.MkdirAll(shared, 0o755); err != nil {
				t.Fatal(err)
			}
			// The marker sits directly in the container, one level too high.
			item := state.Item{
				Artifact: "malformed", Tool: "claude", Scope: "global",
				Files: []state.FileState{{Path: filepath.Join(shared, skillMarker), Action: string(diff.Create)}},
			}

			if _, err := Prune(item); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(shared); err != nil {
				t.Errorf("the shared %s container must never be pruned: %v", container, err)
			}
		})
	}
}

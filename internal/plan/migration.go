package plan

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

// CodexMigrationScope names the retired and selected skill roots of one
// explicitly selected scope.
type CodexMigrationScope struct {
	Scope  string   // "global" | "local"
	Legacy []string // retired roots, e.g. ~/.codex/skills and $CODEX_HOME/skills
	Dest   string   // selected .agents/skills root
}

// CodexMigrationInput carries both scopes' recorded ownership. Unselected
// scopes are inspected for overlap, never planned or written.
type CodexMigrationInput struct {
	Scope  CodexMigrationScope
	States map[string]*state.State // keyed "global" / "local"
	Roots  []scan.CodexSkillRoot   // every discovery candidate in both scopes
}

// CodexSkillMove is a transient relocation of one recorded Codex skill. It is
// never persisted: Staged ownership (old and new rows) is saved after verified
// destination writes and before retirement; Final ownership is saved only after
// every retirement is verified.
type CodexSkillMove struct {
	Index  int        // position of Source in the selected scope's state
	Source state.Item // recorded row, unchanged
	Staged state.Item
	Final  state.Item
	Writes []diff.FileDiff // CREATE at the destination
	Retire []diff.FileDiff // DELETE of verified legacy sources
	Dirs   []string        // legacy directories to prune when empty, deepest first
}

// CodexSkillMigration proves every recorded legacy Codex skill in the
// selected scope by identity, root, scope and checksum. Any refusal rejects the
// whole selection: there is no partial plan, adoption or force bypass.
func CodexSkillMigration(in CodexMigrationInput) ([]CodexSkillMove, error) {
	sel := in.States[in.Scope.Scope]
	if sel == nil {
		return nil, fmt.Errorf("codex migration: no %s state loaded", in.Scope.Scope)
	}
	var moves []CodexSkillMove
	for i, it := range sel.Items {
		if it.Native != nil {
			continue
		}
		root := ""
		for _, f := range it.Files {
			for _, legacy := range in.Scope.Legacy {
				if _, ok := codexUnder(legacy, f.Path); ok && root == "" {
					root = legacy
				}
			}
		}
		if root == "" {
			continue
		}
		if it.Tool != "codex" {
			return nil, fmt.Errorf("codex migration: %s (%s) records %s under a Codex skill root; foreign ownership is not migrated", it.Artifact, it.Tool, root)
		}
		move, err := planCodexSkillMove(in, i, it, root)
		if err != nil {
			return nil, fmt.Errorf("codex migration %s (%s): %w; old files and ownership preserved", it.Artifact, in.Scope.Scope, err)
		}
		moves = append(moves, move)
	}
	return moves, nil
}

type codexPair struct{ legacy, dest *state.FileState }

func planCodexSkillMove(in CodexMigrationInput, index int, it state.Item, root string) (CodexSkillMove, error) {
	move := CodexSkillMove{Index: index, Source: it}
	if it.Type != "" && it.Type != "skill" {
		return move, fmt.Errorf("type %q is not a skill payload", it.Type)
	}
	if !scan.PiSafeName(it.Artifact) {
		return move, fmt.Errorf("ambiguous skill identity %q", it.Artifact)
	}
	srcDir := filepath.Join(root, it.Artifact)
	dstDir := filepath.Join(in.Scope.Dest, it.Artifact)
	for _, p := range []string{root, srcDir, in.Scope.Dest, dstDir} {
		if err := scan.CodexSafePath(p); err != nil {
			return move, err
		}
	}
	pairs := map[string]*codexPair{}
	var rels []string
	for i := range it.Files {
		f := &it.Files[i]
		if f.Action != string(diff.Create) || f.Section != "" || f.Setting != nil || len(f.Prior) > 0 {
			return move, fmt.Errorf("%s is not a whole-file owned payload (%s); structured or unknown ownership is not converted", f.Path, f.Action)
		}
		if !strings.HasPrefix(f.Checksum, "sha256:") {
			return move, fmt.Errorf("%s lacks recorded checksum evidence", f.Path)
		}
		rel, legacy := codexUnder(srcDir, f.Path)
		if !legacy {
			var ok bool
			if rel, ok = codexUnder(dstDir, f.Path); !ok {
				return move, fmt.Errorf("ambiguous identity: %s is outside %s and %s", f.Path, srcDir, dstDir)
			}
		}
		p := pairs[rel]
		if p == nil {
			p = &codexPair{}
			pairs[rel] = p
			rels = append(rels, rel)
		}
		if (legacy && p.legacy != nil) || (!legacy && p.dest != nil) {
			return move, fmt.Errorf("ambiguous duplicate ownership of %s", f.Path)
		}
		if legacy {
			p.legacy = f
		} else {
			p.dest = f
		}
	}
	sort.Strings(rels)
	if pairs["SKILL.md"] == nil {
		return move, fmt.Errorf("missing recorded SKILL.md evidence")
	}
	if err := codexForeignOwners(in, index, it, srcDir, dstDir); err != nil {
		return move, err
	}
	srcFiles, err := scan.CodexSkillTree(srcDir)
	if err != nil {
		return move, err
	}
	for _, path := range srcFiles {
		rel, _ := codexUnder(srcDir, path)
		if p := pairs[rel]; p == nil || p.legacy == nil {
			return move, fmt.Errorf("unowned file %s preserved; complete owned payload required", path)
		}
	}
	dstFiles, err := scan.CodexSkillTree(dstDir)
	if err != nil {
		return move, err
	}
	for _, path := range dstFiles {
		rel, _ := codexUnder(dstDir, path)
		if p := pairs[rel]; p == nil || p.dest == nil {
			return move, fmt.Errorf("destination collision at %s; unowned file preserved", path)
		}
	}
	staged, final := it, it
	staged.Files, final.Files = nil, nil
	var skill []byte
	for _, rel := range rels {
		p := pairs[rel]
		src, dst := filepath.Join(srcDir, rel), filepath.Join(dstDir, rel)
		var content []byte
		if p.dest != nil {
			raw, exists, err := codexRead(dst)
			if err != nil {
				return move, err
			}
			if !exists || codexSum(raw) != p.dest.Checksum {
				return move, fmt.Errorf("destination %s missing or edited since write", dst)
			}
			if p.legacy != nil && p.legacy.Checksum != p.dest.Checksum {
				return move, fmt.Errorf("ambiguous identity: %s and %s record different checksums", src, dst)
			}
			content = raw
		}
		if p.legacy != nil {
			raw, exists, err := codexRead(src)
			if err != nil {
				return move, err
			}
			if exists && codexSum(raw) != p.legacy.Checksum {
				return move, fmt.Errorf("edited source %s preserved (checksum drift, no --force bypass)", src)
			}
			if !exists && p.dest == nil {
				return move, fmt.Errorf("missing source %s; prior evidence cannot be proven", src)
			}
			if exists {
				info, err := os.Lstat(src)
				if err != nil {
					return move, err
				}
				if p.dest == nil {
					move.Writes = append(move.Writes, codexMoveDiff(it, dst, diff.Create, nil, raw, info.Mode().Perm()))
					content = raw
				}
				move.Retire = append(move.Retire, codexMoveDiff(it, src, diff.Delete, raw, nil, 0))
				staged.Files = append(staged.Files, *p.legacy)
			}
		}
		row := state.FileState{Path: dst, Action: string(diff.Create), Checksum: codexSum(content)}
		staged.Files = append(staged.Files, row)
		final.Files = append(final.Files, row)
		if rel == "SKILL.md" {
			skill = content
		}
	}
	name, err := scan.CodexSkillName(skill, filepath.Join(dstDir, "SKILL.md"))
	if err != nil {
		return move, err
	}
	if name != it.Artifact {
		return move, fmt.Errorf("ambiguous identity: SKILL.md names %q", name)
	}
	if err := codexNameCollisions(in, it, srcDir, dstDir); err != nil {
		return move, err
	}
	move.Staged, move.Final = staged, final
	dirs := map[string]bool{}
	for _, d := range move.Retire {
		for dir := filepath.Dir(d.Path); ; dir = filepath.Dir(dir) {
			dirs[dir] = true
			if dir == srcDir || !strings.HasPrefix(dir, srcDir) {
				break
			}
		}
	}
	for dir := range dirs {
		move.Dirs = append(move.Dirs, dir)
	}
	sort.Slice(move.Dirs, func(i, j int) bool { return len(move.Dirs[i]) > len(move.Dirs[j]) })
	return move, nil
}

// codexForeignOwners refuses any other recorded owner, in either scope, of the
// source or destination skill directory, including a duplicate same-name row.
func codexForeignOwners(in CodexMigrationInput, index int, it state.Item, srcDir, dstDir string) error {
	for scope, s := range in.States {
		if s == nil {
			continue
		}
		for j, other := range s.Items {
			if scope == in.Scope.Scope && j == index {
				continue
			}
			if other.Tool == it.Tool && other.Scope == it.Scope && other.Artifact == it.Artifact && other.Root == it.Root {
				return fmt.Errorf("ambiguous identity: duplicate ownership rows for %s", it.Artifact)
			}
			for _, f := range other.Files {
				_, inSrc := codexUnder(srcDir, f.Path)
				_, inDst := codexUnder(dstDir, f.Path)
				if inSrc || inDst {
					return fmt.Errorf("%s is also recorded by %s (%s, %s); incompatible ownership", f.Path, other.Artifact, other.Tool, scope)
				}
			}
		}
	}
	return nil
}

// codexNameCollisions permits another same-name skill only when a Codex row
// with that artifact owns it with a matching checksum, in either scope.
func codexNameCollisions(in CodexMigrationInput, it state.Item, srcDir, dstDir string) error {
	found, err := scan.CodexSkillOccurrences(in.Roots, it.Artifact)
	if err != nil {
		return err
	}
	for _, r := range found {
		if r.Path == filepath.Join(srcDir, "SKILL.md") || r.Path == filepath.Join(dstDir, "SKILL.md") {
			continue
		}
		owned := false
		for _, s := range in.States {
			if s == nil {
				continue
			}
			for _, other := range s.Items {
				for _, f := range other.Files {
					if f.Path == r.Path && other.Tool == "codex" && other.Artifact == it.Artifact && f.Checksum == codexSum(r.Content) {
						owned = true
					}
				}
			}
		}
		if !owned {
			return fmt.Errorf("same-name skill collision at %s; unowned or edited resource preserved", r.Path)
		}
	}
	return nil
}

func codexMoveDiff(it state.Item, path string, action diff.Action, before, after []byte, mode os.FileMode) diff.FileDiff {
	return diff.FileDiff{Path: path, Action: action, Before: before, After: after, Mode: mode, Artifact: it.Artifact, Type: "skill", Tool: "codex", Scope: it.Scope, Root: it.Root, Version: it.ItemVersion}
}

func codexUnder(dir, path string) (string, bool) {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return rel, true
}

func codexRead(path string) ([]byte, bool, error) {
	if err := scan.CodexSafePath(path); err != nil {
		return nil, false, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return raw, err == nil, err
}

func codexSum(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }

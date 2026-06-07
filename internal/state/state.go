// Package state records what Patronus installed on this machine so later
// lifecycle commands (remove/update/revert, Phase 8) are clean and never clobber
// user edits. Phase 3 is RECORD-ONLY: it writes state but does not read it back
// to revert. The schema deliberately captures the pre-install bytes of
// APPEND/MERGE targets now, because those are the inputs a future revert needs
// and they exist only at apply time — everything else can be recomputed.
//
// Format is plain JSON (stdlib, deterministic, git-diffable). State is kilobytes,
// written once per command, so format performance is irrelevant; readability and
// zero dependencies are what matter.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
)

// Version is the state schema version, bumped when the on-disk shape changes so
// future readers can migrate.
const Version = 1

// State is the full record for one scope's state file.
type State struct {
	Version int    `json:"version"`
	Items   []Item `json:"items"`
}

// Item is one installed artifact at one tool+scope.
type Item struct {
	Artifact    string      `json:"artifact"`
	ItemVersion string      `json:"itemVersion,omitempty"` // the artifact's own version
	Tool        string      `json:"tool"`
	Scope       string      `json:"scope"`
	InstalledAt string      `json:"installedAt,omitempty"` // RFC3339; supplied by caller (pkg stays clockless)
	Files       []FileState `json:"files"`
}

// FileState records one written file. Checksum is the sha256 of the bytes
// Patronus wrote (not the user's surrounding prose), which lets a later command
// tell "untouched since we wrote it" from "user-edited."
type FileState struct {
	Path     string `json:"path"`
	Action   string `json:"action"` // CREATE | APPEND | MERGE
	Checksum string `json:"checksum"`

	// --- forward-compat inputs for Phase 8 revert (captured now, unused now) ---
	// Section is the fenced-section name for an APPEND, so revert can remove
	// exactly that block. Prior is the pre-install file content for APPEND/MERGE,
	// so revert can restore it; it is omitted for CREATE (revert = delete).
	Section string `json:"section,omitempty"`
	Prior   []byte `json:"prior,omitempty"`
}

// Load reads a state file. A missing file is not an error: it yields an empty
// State at the current version (first install on this machine/scope).
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Version: Version}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		s.Version = Version
	}
	return &s, nil
}

// Save writes s atomically as indented, deterministic JSON.
func Save(path string, s *State) error {
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return install.WriteFileAtomic(path, out, 0o644)
}

// Merge upserts newItems into s, keyed by (artifact, tool, scope): re-installing
// the same item replaces its record rather than duplicating it. Returns s for
// chaining. Pure aside from mutating s.
func Merge(s *State, newItems []Item) *State {
	for _, ni := range newItems {
		replaced := false
		for i := range s.Items {
			if s.Items[i].Artifact == ni.Artifact && s.Items[i].Tool == ni.Tool && s.Items[i].Scope == ni.Scope {
				s.Items[i] = ni
				replaced = true
				break
			}
		}
		if !replaced {
			s.Items = append(s.Items, ni)
		}
	}
	return s
}

// FromChangeSet translates the diffs that were actually applied into state items,
// grouped by (artifact, tool, scope). now is an RFC3339 timestamp supplied by
// the caller (this package takes no clock so it stays deterministic in tests).
// Only the listed diffs should be the ones Apply reported as Applied.
func FromChangeSet(applied []diff.FileDiff, now string) []Item {
	type key struct{ artifact, tool, scope string }
	order := []key{}
	byKey := map[key]*Item{}

	for _, d := range applied {
		if d.IsDir {
			continue
		}
		k := key{d.Artifact, d.Tool, d.Scope}
		it, ok := byKey[k]
		if !ok {
			it = &Item{Artifact: d.Artifact, Tool: d.Tool, Scope: d.Scope, InstalledAt: now}
			byKey[k] = it
			order = append(order, k)
		}
		it.Files = append(it.Files, fileState(d))
	}

	out := make([]Item, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	return out
}

// fileState builds the per-file record for one applied diff.
func fileState(d diff.FileDiff) FileState {
	fs := FileState{
		Path:     d.Path,
		Action:   string(d.Action),
		Checksum: checksum(d.After),
	}
	// Capture the revert inputs only where they exist / are needed.
	if d.Action == diff.Append && d.Section != nil {
		fs.Section = d.Section.Name
	}
	if d.Action == diff.Append || d.Action == diff.Merge {
		// Pre-install content the revert must restore. CREATE has none (delete).
		fs.Prior = d.Before
	}
	return fs
}

func checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

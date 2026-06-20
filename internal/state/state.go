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

// Item is one installed artifact or recipe at one tool+scope.
type Item struct {
	Artifact    string      `json:"artifact"`
	ItemVersion string      `json:"itemVersion,omitempty"` // the artifact's own version
	Tool        string      `json:"tool"`
	Scope       string      `json:"scope"`
	InstalledAt string      `json:"installedAt,omitempty"` // RFC3339; supplied by caller (pkg stays clockless)
	Files       []FileState `json:"files"`

	// --- recipe forward-compat for Phase 8 (captured now, unused now) ---
	// SelfWired marks a recipe that installed itself via post-install commands
	// (e.g. ai-memory). PostInstall records the exact commands run so a future
	// revert can attempt the inverse (best-effort) or at least warn that the
	// wiring is not cleanly reversible. Mirrors how Prior/Section were captured
	// in Phase 3 for a Phase-8 reader.
	SelfWired   bool     `json:"selfWired,omitempty"`
	PostInstall []string `json:"postInstall,omitempty"`
}

// FileState records one written file. Checksum is the sha256 of the bytes
// Patronus wrote (not the user's surrounding prose), which lets a later command
// tell whether a file is untouched-since-write or user-edited.
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

// Find returns the items matching the given filters. An empty tool or scope
// matches any value for that field, so Find(name, "", "") returns every recorded
// install of name across tools/scopes — the natural selection for `remove <name>`.
func (s *State) Find(artifact, tool, scope string) []Item {
	var out []Item
	for _, it := range s.Items {
		if it.Artifact != artifact {
			continue
		}
		if tool != "" && it.Tool != tool {
			continue
		}
		if scope != "" && it.Scope != scope {
			continue
		}
		out = append(out, it)
	}
	return out
}

// Remove drops every item matching the filters (the inverse of Merge's upsert),
// keeping the rest in their original order. An empty tool or scope matches any
// value for that field. Returns the number of items removed.
func (s *State) Remove(artifact, tool, scope string) int {
	kept := s.Items[:0]
	removed := 0
	for _, it := range s.Items {
		match := it.Artifact == artifact &&
			(tool == "" || it.Tool == tool) &&
			(scope == "" || it.Scope == scope)
		if match {
			removed++
			continue
		}
		kept = append(kept, it)
	}
	s.Items = kept
	return removed
}

// FromChangeSet translates installed diffs into state items, grouped by
// (artifact, tool, scope). now is an RFC3339 timestamp supplied by the caller
// (this package takes no clock so it stays deterministic in tests). Pass the
// diffs that were actually realized: the applier's Applied (CREATE/APPEND/MERGE/
// FETCH) plus any EXEC diffs the cmd layer ran (self-wiring post-install) — EXEC
// diffs are recorded on their recipe's Item as SelfWired + PostInstall rather
// than as files.
func FromChangeSet(applied []diff.FileDiff, now string) []Item {
	type key struct{ artifact, tool, scope string }
	order := []key{}
	byKey := map[key]*Item{}

	getKey := func(k key) *Item {
		it, ok := byKey[k]
		if !ok {
			it = &Item{Artifact: k.artifact, Tool: k.tool, Scope: k.scope, InstalledAt: now}
			byKey[k] = it
			order = append(order, k)
		}
		return it
	}
	get := func(d diff.FileDiff) *Item {
		it := getKey(key{d.Artifact, d.Tool, d.Scope})
		// A later diff may carry the version when the first one (e.g. a shared
		// composed file) did not; record the first non-empty we see.
		if it.ItemVersion == "" && d.Version != "" {
			it.ItemVersion = d.Version
		}
		return it
	}

	for _, d := range applied {
		if d.IsDir {
			continue
		}
		it := get(d)
		if d.Action == diff.Exec {
			// A wire.run/self command: not a file, recorded as forward-compat revert
			// data on the recipe's item. SelfWired tracks the self-managing
			// (wire.mode:self) provenance — these have no Patronus-recorded inverse,
			// which is what remove reports as "not auto-revertable."
			if d.Exec != nil {
				it.SelfWired = it.SelfWired || d.Exec.SelfManaged
				it.PostInstall = append(it.PostInstall, d.Exec.Display)
			}
			continue
		}
		it.Files = append(it.Files, fileState(d))

		// A composed APPEND file may carry sections from OTHER artifacts folded in
		// (e.g. agents-spine + agent-rules → one CLAUDE.md). Record each under its
		// own artifact so remove strips exactly that section, restoring the file to
		// the state before this contributor was folded in.
		for _, c := range d.Contrib {
			ci := getKey(key{c.Artifact, d.Tool, d.Scope})
			if ci.ItemVersion == "" {
				ci.ItemVersion = c.Version
			}
			ci.Files = append(ci.Files, FileState{
				Path:     d.Path,
				Action:   string(diff.Append),
				Checksum: checksum(d.After),
				Section:  c.Section,
				Prior:    c.Prior,
			})
		}
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
	// FETCH writes a binary, not d.After bytes; record the placed binary's sha so
	// a later scan can tell "unchanged" from "user-replaced." Prefer the digest
	// the applier stamped after placing (the extracted member for an archive);
	// fall back to the pinned download sha for a raw-binary asset / planning.
	if d.Action == diff.Fetch && d.Fetch != nil {
		sum := d.Fetch.PlacedSHA256
		if sum == "" {
			sum = d.Fetch.SHA256
		}
		fs.Checksum = "sha256:" + normalizeHexState(sum)
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

// normalizeHexState strips an optional "sha256:" prefix so a pinned digest is
// stored uniformly with the computed-checksum form.
func normalizeHexState(s string) string {
	if len(s) > 7 && s[:7] == "sha256:" {
		return s[7:]
	}
	return s
}

func checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

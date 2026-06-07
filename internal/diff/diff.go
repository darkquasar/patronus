// Package diff is the shared change-set abstraction for Patronus. Every layer —
// the adapter transform engine, the planner, the dry-run renderer, and the
// (Phase 3) applier — speaks in FileDiffs: a path with its Before and After
// bytes. Conflict classification, SKIP detection, rendering, JSON output, and
// eventual disk writes all derive from this one type, so a single source of
// truth flows from "compute" to "apply" and makes remove/update straightforward
// to add later.
package diff

import (
	"bytes"

	"znkr.io/diff/textdiff"
)

// Action is the kind of change a FileDiff represents. CREATE/APPEND/MERGE are
// intended actions an adapter emits; CONFLICT/SKIP are terminal classifications
// the planner assigns after comparing against the real filesystem.
type Action string

const (
	Create   Action = "CREATE"   // Before absent, After is new content
	Append   Action = "APPEND"   // delimited-section insert/replace (non-destructive)
	Merge    Action = "MERGE"    // structural config edit (never blind overwrite)
	Conflict Action = "CONFLICT" // target exists & differs from a CREATE's After
	Skip     Action = "SKIP"     // After == Before; idempotent, nothing to do
)

// FileDiff is the single source of truth for one path's change. Before is nil
// when the target does not exist. Before/After are excluded from JSON: the
// machine-readable surface is the path + action, while the bytes are retained
// in memory for classification and the Phase-3 applier.
type FileDiff struct {
	Path   string `json:"path"`
	Action Action `json:"action"`
	Before []byte `json:"-"`
	After  []byte `json:"-"`

	// Display / grouping metadata.
	Artifact   string `json:"artifact,omitempty"`   // source artifact name
	Capability string `json:"capability,omitempty"` // what's added: skill|instruction|mcp|...
	Tool       string `json:"tool,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Role       string `json:"role,omitempty"`
	Note       string `json:"note,omitempty"`
	IsDir      bool   `json:"isDir,omitempty"`

	// Section, when set, describes a named fenced-section APPEND edit. The
	// planner uses it to re-fold multiple appends that land on the same file
	// (e.g. codex + opencode both targeting a shared AGENTS.md) into one After.
	Section *SectionEdit `json:"-"`
}

// SectionEdit captures the inputs of an appendSection edit so it can be re-applied
// during composition.
type SectionEdit struct {
	Name string
	Body []byte
}

// Classify decides the terminal Action for a proposed change, preserving
// CREATE/APPEND/MERGE semantics:
//
//	CREATE: target absent -> CREATE; identical -> SKIP; differs -> CONFLICT.
//	APPEND/MERGE: After == Before -> SKIP; otherwise keep the intended action.
//	  These fold the existing content into After by construction, so they are
//	  non-destructive and never produce CONFLICT.
//
// exists reports whether the target file is present on disk.
func Classify(intended Action, before, after []byte, exists bool) Action {
	switch intended {
	case Create:
		if !exists {
			return Create
		}
		if bytes.Equal(before, after) {
			return Skip
		}
		return Conflict
	case Append, Merge:
		if bytes.Equal(before, after) {
			return Skip
		}
		return intended
	default:
		return intended
	}
}

// Unified returns a unified-format (---/+++/@@) diff of Before vs After for the
// verbose dry-run view. It returns "" when there is nothing to show (SKIP, or a
// supporting-dir summary row). For a CREATE the whole content shows as added.
func (d FileDiff) Unified() string {
	if d.IsDir || d.Action == Skip {
		return ""
	}
	return textdiff.Unified(string(d.Before), string(d.After))
}

// ChangeSet is an ordered, deduped/composed set of diffs plus whether this run
// is a dry run (DryRun false only on a real --deploy apply).
type ChangeSet struct {
	Diffs  []FileDiff `json:"diffs"`
	DryRun bool       `json:"dryRun"`
}

// Counts tallies diffs by action, for summaries.
func (cs *ChangeSet) Counts() map[Action]int {
	out := map[Action]int{}
	for _, d := range cs.Diffs {
		if d.IsDir {
			continue
		}
		out[d.Action]++
	}
	return out
}

// The writer that realizes a ChangeSet on disk lives in internal/install
// (Phase 3). Following Go convention we let the consumer define any interface it
// needs rather than declaring a speculative one here; the concrete
// install.Applier consumes the same ChangeSet the planner produced.

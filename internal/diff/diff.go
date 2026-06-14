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
	Fetch    Action = "FETCH"    // download+verify a recipe binary and place it (Phase 4)
	Exec     Action = "EXEC"     // run a self-wiring recipe's post-install command (Phase 4)
	Conflict Action = "CONFLICT" // target exists & differs from a CREATE's After
	Skip     Action = "SKIP"     // After == Before; idempotent, nothing to do

	// Inverse actions (Phase 8 remove/revert): a recorded apply read back from
	// state.json as its undo. They are built already-classified — Classify is not
	// involved — so they reuse the same applier writes as their forward kin.
	Delete   Action = "DELETE"   // remove a CREATEd file (inverse of CREATE)
	Unappend Action = "UNAPPEND" // strip a named fenced section (inverse of APPEND); After is the file without it
	Restore  Action = "RESTORE"  // write recorded Prior bytes back (inverse of MERGE/APPEND full-file restore)
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
	Version    string `json:"version,omitempty"`    // the source item's own version (recorded in state for update)
	Capability string `json:"capability,omitempty"` // what's added: skill|instruction|mcp|...
	Tool       string `json:"tool,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Role       string `json:"role,omitempty"`
	Note       string `json:"note,omitempty"`
	IsDir      bool   `json:"isDir,omitempty"`

	// Intended, when set on a SKIP, is the action this diff WOULD perform if the
	// skip were overridden. The Phase-8 remove path uses it for drift: a file
	// edited since install is emitted as SKIP(Intended=DELETE/UNAPPEND/RESTORE) so
	// the renderer shows it as skipped, and --force promotes Action to Intended.
	// Excluded from JSON (an internal control field).
	Intended Action `json:"-"`

	// Section, when set, describes a named fenced-section APPEND edit. The
	// planner uses it to re-fold multiple appends that land on the same file
	// (e.g. codex + opencode both targeting a shared AGENTS.md) into one After.
	Section *SectionEdit `json:"-"`

	// Fetch, when set, describes a FETCH: a binary to download, verify, and
	// place. It lives only on Action==Fetch diffs (Path is the placement dest;
	// Before/After are empty — the bytes are streamed at apply time, never held
	// in the change set). Excluded from JSON; Note carries the display label.
	Fetch *FetchSpec `json:"-"`

	// Exec, when set, describes a self-wiring recipe's post-install command.
	// It lives only on Action==Exec diffs, which are display-only in the change
	// set: install.Applier skips them, and the cmd layer (runDeploy) runs them
	// via os/exec on --deploy. Excluded from JSON; Note carries the display.
	Exec *ExecSpec `json:"-"`
}

// FetchSpec is the input to a FETCH apply: download URL, expected sha256, the
// final placement path, and (for archive assets) the archive format and the
// member path to extract. Archive is "" for a raw-binary asset.
type FetchSpec struct {
	URL        string
	SHA256     string
	Dest       string
	Archive    string // "" | "tar.gz" | "tgz" | "zip"
	BinaryPath string // member path within the archive (when Archive != "")
	Label      string // human label, e.g. "engram v0.4 (darwin/arm64)"

	// PlacedSHA256 is the sha256 of the binary actually written to Dest, stamped
	// by the applier after extraction. For a raw-binary asset this equals SHA256;
	// for an archive it is the extracted member's digest (which SHA256, the
	// archive's, is not). State records this so a later scan can detect whether
	// the placed binary was replaced.
	PlacedSHA256 string
}

// ExecSpec is one self-wiring post-install command. Command is the argv; Display
// is the human-readable form shown in the dry run.
type ExecSpec struct {
	Command []string
	Display string
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
	// FETCH/EXEC carry no text Before/After (a binary download / a command), so
	// there is no unified diff to show — the renderer surfaces them via Note.
	if d.IsDir || d.Action == Skip || d.Action == Fetch || d.Action == Exec {
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

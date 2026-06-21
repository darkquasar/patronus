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
	"io/fs"

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

	// Display / grouping metadata. Type and Role are the two ontology axes the
	// summary table shows as columns: Type is the item's SHAPE (an artifact's
	// declared type — skill|agent|command|hook|instruction — or a recipe's
	// computed Shape() — wire-only|fetch+wire|fetch+run), Role is the layer it
	// fills (capability|context|memory|tools|...).
	Artifact string `json:"artifact,omitempty"` // source artifact/recipe name
	Version  string `json:"version,omitempty"`  // the source item's own version (recorded in state for update)
	Type     string `json:"type,omitempty"`     // shape: artifact type or recipe Shape()
	Tool     string `json:"tool,omitempty"`
	Scope    string `json:"scope,omitempty"`
	Role     string `json:"role,omitempty"` // the layer it fills
	Note     string `json:"note,omitempty"`
	IsDir    bool   `json:"isDir,omitempty"`

	// Mode is the file permission for a CREATE write when it must differ from the
	// default 0o644 — set to 0o755 for an executable hook script. Zero means "use
	// the applier's default." Excluded from JSON (an apply-time control field).
	Mode fs.FileMode `json:"-"`

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

	// Contrib lists the ADDITIONAL artifacts whose APPEND sections were folded
	// into this one composed FileDiff (the first contributor stays in Artifact /
	// Section). It exists because several instruction/output-style artifacts can
	// append distinct fenced sections to ONE file (e.g. agents-spine + agent-rules
	// → CLAUDE.md): the applier writes the file once, but state must record each
	// section under its own artifact so remove can strip exactly that section, and
	// the dry-run table must show a row per contributor. Empty for the common
	// single-contributor case.
	Contrib []SectionContrib `json:"-"`

	// Setting, when set, marks this MERGE as a settings list-append (a hook
	// registration) rather than a scalar config merge. It carries the element's
	// identity + target so the planner re-folds multiple appends onto one
	// settings file and state/remove pull exactly this element. nil for a scalar
	// MERGE (MCP wiring, native-switch toggle), which round-trips via Prior.
	Setting *SettingEdit `json:"-"`

	// SettingContrib lists ADDITIONAL artifacts whose settings elements were
	// folded into this one composed MERGE (the MERGE-side analogue of Contrib).
	// Empty for the common single-contributor case.
	SettingContrib []SettingContrib `json:"-"`

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

// ExecSpec is one wire.run command. Command is the argv; Display is the
// human-readable form shown in the dry run. SelfManaged is true when the command
// comes from a wire.mode:self recipe (the recipe's own installer wires it), which
// is what remove reports as "not auto-revertable"; false for a wire.mode:run
// command that Patronus itself runs. Advisory is true for a display-only command
// Patronus must NOT run (an install-only recipe's package-install line): it is
// shown so the user can run it, but the EXEC runner skips it — distinct from
// SelfManaged, which IS executed.
type ExecSpec struct {
	Command     []string
	Display     string
	SelfManaged bool
	Advisory    bool
}

// SectionEdit captures the inputs of an appendSection edit so it can be re-applied
// during composition.
type SectionEdit struct {
	Name string
	Body []byte
}

// SectionContrib records one additional artifact that contributed a fenced
// section to a shared composed file: enough identity (artifact, version) and the
// section name for state to record a removable per-artifact item, plus Prior —
// the file's bytes as they were BEFORE this contributor's section was folded in —
// so remove can reverse each section independently and in order.
type SectionContrib struct {
	Artifact string
	Version  string
	Section  string
	Prior    []byte
}

// SettingEdit captures the inputs of a settings MERGE so the planner can re-fold
// it onto an accumulated config (several settings edits can land on one file). It
// has two forms, distinguished by IdentityKey:
//   - LIST-APPEND (IdentityKey != ""): append/replace Elem in the array at Dotted,
//     keyed by Identity — a hook registration. Remove strips exactly that element.
//   - SCALAR SET (IdentityKey == ""): set ScalarValue at Dotted — a statusline /
//     sandbox toggle. Remove restores the prior value (wholesale Prior).
//
// Either way the edit is re-foldable, so a scalar set folded after some hooks no
// longer clobbers them. The full FileTarget travels along for re-parse/serialize.
type SettingEdit struct {
	Target      FileTargetRef  // file + format the merge applies to
	Dotted      string         // resolved path, e.g. "hooks.PreToolUse" (list) or "statusLine" (scalar)
	IdentityKey string         // array form: element field carrying the identity ("" => scalar set)
	Identity    string         // array form: this element's identity value
	Elem        map[string]any // array form: the element to (re-)append
	ScalarValue any            // scalar form: the value to set at Dotted
}

// FileTargetRef is the minimal file/format descriptor a SettingEdit needs to
// re-merge or remove without importing the manifest package into diff. The
// adapter/planner fills it from the layout's FileTarget.
type FileTargetRef struct {
	File   string
	Format string
}

// SettingContrib records one additional artifact whose settings edit was folded
// into a shared config file (e.g. several hooks into one settings.json). Like
// SectionContrib it carries identity for a per-artifact removable record. Edit
// carries the merge intent, and remove reverses it SURGICALLY (strip the array
// element for a list edit, delete the key for a scalar) — so no whole-file Prior
// is needed: a revert never disturbs edits that folded into the same file.
type SettingContrib struct {
	Artifact string
	Version  string
	Edit     *SettingEdit
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

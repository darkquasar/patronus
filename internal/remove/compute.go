// Package remove computes the inverse of a recorded install: it reads the state
// items Patronus wrote (internal/state) and produces a diff.ChangeSet of UNDO
// actions on the same spine the installer and renderer already speak. There is no
// new write machinery — DELETE/UNAPPEND/RESTORE flow through the existing
// install.Applier exactly as CREATE/APPEND/MERGE do.
//
// Undos that land on the SAME file compose. Several artifacts can be wired into
// one settings file, and computing each undo independently from the same original
// bytes made the last write resurrect what the earlier ones removed — so an
// artifact could survive the very command that removed it. Compute folds every
// selected settings edit on a path onto one evolving buffer and emits a single
// write, mirroring what the install planner already does. Because one write can
// then settle several artifacts, Compute also returns a per-contributor outcome
// ledger: the change set says what to WRITE, the ledger says who was UNDONE.
//
// Safety mirrors install's never-clobber-unconfirmed stance: every undo is gated
// on the recorded sha256 still matching what's on disk. If a CREATEd file or an
// APPENDed section was edited since install (drift), the row is emitted as a SKIP
// carrying a note; the cmd layer turns --force into "treat drift as its intended
// undo." Self-wiring recipes (EXEC) have no clean inverse and are reported as
// warnings, never auto-reverted.
package remove

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

// ReadExisting reads a target file's current bytes; ok is false when it does not
// exist. Mirrors adapter.ReadExisting so the cmd layer can reuse one reader.
type ReadExisting func(path string) ([]byte, bool, error)

// Warning is a non-fatal advisory surfaced to the user (drift skipped, a
// self-wired recipe that can't be auto-reverted, an orphaned binary).
type Warning struct {
	Item    string // artifact/recipe name
	Path    string // file path, when the warning is about one file ("" otherwise)
	Message string
}

// Outcome is the structured verdict for ONE contributor's ONE recorded file.
// Composition means several contributors can share a single physical write, so
// the diff alone can no longer say who was undone: the ledger carries that, and
// downstream code (state retirement, directory pruning) matches on these codes
// rather than on human-facing Note prose.
type Outcome string

const (
	// Applied — this contribution is in a diff that the applier will write.
	Applied Outcome = "applied"
	// AlreadyAbsent — the target file (or appended section) is already gone, so
	// the removal is logically satisfied with nothing to write.
	AlreadyAbsent Outcome = "already-absent"
	// SettingAbsent — the settings file parsed, but our key/element is not in it.
	SettingAbsent Outcome = "setting-absent"
	// DriftSkipped — the file changed since install; held open for a later --force.
	DriftSkipped Outcome = "drift-skipped"
	// UnreadableSkipped — the settings CONTENT could not be parsed (not an I/O
	// error, which aborts Compute outright).
	UnreadableSkipped Outcome = "unreadable-skipped"
	// UnsafeLegacySkipped — a pre-compose whole-file MERGE row whose path has
	// another recorded contributor, so restoring its snapshot would destroy that
	// contributor's wiring. Never promotable under --force.
	UnsafeLegacySkipped Outcome = "unsafe-legacy-skipped"
	// AmbiguousSkipped — two folded settings edits target overlapping keys, so no
	// order-independent composition exists. Never promotable under --force.
	AmbiguousSkipped Outcome = "ambiguous-skipped"
	// UnknownAction — the state row records an action this version cannot invert.
	UnknownAction Outcome = "unknown-action"
)

// Complete reports whether this outcome means the contribution's removal is
// logically satisfied — either it was written, or there was nothing left to do.
// The incomplete codes deliberately hold a state row open so a later run (or a
// --force) can finish the job.
func (o Outcome) Complete() bool {
	switch o {
	case Applied, AlreadyAbsent, SettingAbsent:
		return true
	default:
		return false
	}
}

// LedgerEntry is one contributor's verdict on one recorded path. Identity is the
// full (artifact, tool, scope, path) tuple because composition collapses several
// of these onto one diff, and path alone can no longer distinguish them.
type LedgerEntry struct {
	Effect   state.FileState
	Artifact string
	Tool     string
	Scope    string
	Path     string
	Outcome  Outcome
}

// Ledger is the per-contributor outcome record Compute returns alongside the
// change set. The change set says what to WRITE; the ledger says who was UNDONE.
type Ledger []LedgerEntry

// Result bundles Compute's three outputs: the physical change set, the
// per-contributor ledger, and the user-facing warnings.
type Result struct {
	ChangeSet *diff.ChangeSet
	Ledger    Ledger
	Warnings  []Warning
}

// Compute turns recorded state items into an inverse change set, a
// per-contributor outcome ledger, and warnings. read supplies current on-disk
// bytes so each undo can be gated against drift. The returned ChangeSet is
// already classified (DELETE/UNAPPEND/RESTORE/SKIP); Classify is not involved. A
// drift row is a SKIP whose Note explains why; the cmd layer rewrites those to
// their intended action under --force.
//
// occupancy tells Compute about contributors it was NOT asked to remove: it maps
// a recorded path to every artifact that recorded a MERGE on it, selected or not.
// The legacy whole-file arm needs that view — a snapshot restore is unsafe
// whenever anyone else is wired into the same file, and looking only at the
// selection would miss an installed sibling. Pass nil when there is no wider
// state to consult; the legacy arm then sees only the selection.
func Compute(items []state.Item, read ReadExisting, occupancy Occupancy) (Result, error) {
	return ComputeWithForce(items, read, occupancy, false)
}

// ComputeWithForce changes preservation policy only for selected concrete effects.
// Structural parse/identity failures remain non-promotable.
func ComputeWithForce(items []state.Item, read ReadExisting, occupancy Occupancy, force bool) (Result, error) {
	var (
		res   Result
		files []fileIntent
	)
	for _, it := range items {
		// A self-wiring recipe ran post-install commands with no recorded inverse;
		// warn rather than invent an uninstall. Its (non-EXEC) files, if any, are
		// still reverted below.
		if it.SelfWired {
			cmds := ""
			if len(it.PostInstall) > 0 {
				cmds = ": ran " + joinCmds(it.PostInstall)
			}
			res.Warnings = append(res.Warnings, Warning{
				Item:    it.Artifact,
				Message: fmt.Sprintf("self-wired recipe %q cannot be auto-reverted%s — undo it manually", it.Artifact, cmds),
			})
		}
		for _, f := range it.Files {
			files = append(files, fileIntent{item: it, file: f})
		}
	}

	// Modern settings and supported sections compose per path below; other rows
	// retain their legacy independent inverse behavior.
	byPath := map[string][]fileIntent{}
	var pathOrder []string
	// Pi and Codex sections sharing a physical file invert together. Independent
	// UNAPPEND writes would resurrect sections removed by an earlier write.
	piPaths := map[string][]fileIntent{}
	var piPathOrder []string
	for _, fi := range files {
		if (fi.item.Tool == "pi" || fi.item.Tool == "codex") && fi.file.Action == string(diff.Append) {
			if _, seen := piPaths[fi.file.Path]; !seen {
				piPaths[fi.file.Path] = nil
				piPathOrder = append(piPathOrder, fi.file.Path)
			}
		}
	}
	for _, fi := range files {
		if _, shared := piPaths[fi.file.Path]; shared {
			piPaths[fi.file.Path] = append(piPaths[fi.file.Path], fi)
			continue
		}
		if isModernSetting(fi.file) {
			if _, seen := byPath[fi.file.Path]; !seen {
				pathOrder = append(pathOrder, fi.file.Path)
			}
			byPath[fi.file.Path] = append(byPath[fi.file.Path], fi)
			continue
		}
		d, w, out, err := fileUndo(fi.item, fi.file, read, occupancy)
		if err != nil {
			return Result{}, err
		}
		if w != nil {
			res.Warnings = append(res.Warnings, *w)
		}
		res.ChangeSet = appendDiff(res.ChangeSet, d)
		res.Ledger = append(res.Ledger, ledgerEntry(fi, out))
	}

	for _, path := range pathOrder {
		group, err := composeSettingGroup(path, byPath[path], read, occupancy, force)
		if err != nil {
			return Result{}, err
		}
		res.Warnings = append(res.Warnings, group.warnings...)
		res.Ledger = append(res.Ledger, group.ledger...)
		for _, d := range group.diffs {
			res.ChangeSet = appendDiff(res.ChangeSet, d)
		}
	}

	for _, path := range piPathOrder {
		group, err := composeSections(path, piPaths[path], read, occupancy, force)
		if err != nil {
			return Result{}, err
		}
		res.Warnings = append(res.Warnings, group.Warnings...)
		res.Ledger = append(res.Ledger, group.Ledger...)
		for _, d := range group.ChangeSet.Diffs {
			res.ChangeSet = appendDiff(res.ChangeSet, d)
		}
	}
	if res.ChangeSet == nil {
		res.ChangeSet = &diff.ChangeSet{}
	}
	if force {
		res = Promote(res)
	}
	return res, nil
}

// Contributor identifies one recorded writer of a config file. The name alone is
// not the identity: state is keyed by (artifact, tool, scope), so the SAME
// artifact installed for two tools or two scopes is two independent records that
// can both be wired into one absolute path.
type Contributor struct {
	Artifact string
	Tool     string
	Scope    string
	Setting  *diff.SettingEdit
	Section  string // APPEND owner; not a structural config claim
}

func (c Contributor) owner() adapter.SettingOwner {
	return adapter.SettingOwner{Artifact: c.Artifact, Tool: c.Tool, Scope: c.Scope}
}

// Label renders a contributor for a user-facing warning, naming the tool/scope
// only when it is what distinguishes this record from the one being removed.
func (c Contributor) Label() string {
	if c.Tool == "" && c.Scope == "" {
		return c.Artifact
	}
	return c.Artifact + " (" + strings.TrimSpace(c.Tool+" "+c.Scope) + ")"
}

// Occupancy reports recorded config and section contributors on each path,
// including ones this command was not asked to remove. The legacy whole-file restore consults it to prove sole ownership
// before overwriting a file someone else may be wired into.
//
// Paths are absolute, so an entry from ANY scope's state file belongs here: what
// is being protected is the file, not the bookkeeping that happens to mention it.
type Occupancy map[string][]Contributor

// othersOn returns the contributors recorded on path that are not the given
// record. An unknown path yields nothing, which is the honest answer when no
// wider state view was supplied.
func (o Occupancy) othersOn(path string, self Contributor) []string {
	var out []string
	for _, c := range o[path] {
		if c.owner() == self.owner() {
			continue
		}
		out = append(out, c.Label())
	}
	return out
}

// fileIntent pairs a recorded file with the item that owns it, so the composition
// pass keeps each row's identity after grouping by path.
type fileIntent struct {
	item state.Item
	file state.FileState
}

// isModernSetting reports whether a recorded row is a structural settings edit —
// the shape that composes. A MERGE without a Setting is a pre-compose whole-file
// snapshot and takes the legacy arm instead.
func isModernSetting(f state.FileState) bool {
	return f.Action == string(diff.Merge) && f.Setting != nil
}

func appendDiff(cs *diff.ChangeSet, d diff.FileDiff) *diff.ChangeSet {
	if cs == nil {
		cs = &diff.ChangeSet{}
	}
	cs.Diffs = append(cs.Diffs, d)
	return cs
}

func ledgerEntry(fi fileIntent, out Outcome) LedgerEntry {
	return LedgerEntry{
		Artifact: fi.item.Artifact,
		Tool:     fi.item.Tool,
		Scope:    fi.item.Scope,
		Path:     fi.file.Path,
		Outcome:  out,
		Effect:   fi.file,
	}
}

// settingGroup is the result of composing every selected modern settings row on
// one path: the diffs to write (one composite RESTORE at most, plus a SKIP per
// refused contributor), their ledger entries, and any warnings.
type settingGroup struct {
	diffs    []diff.FileDiff
	ledger   []LedgerEntry
	warnings []Warning
}

// composeSettingGroup folds every selected settings edit on one path onto a
// SINGLE evolving buffer and emits ONE RESTORE for it.
//
// This is the mirror of what install already does: plan.composeByPath re-folds
// each edit onto the accumulated result rather than keeping the last. Remove used
// to compute every row independently from the same original bytes and write them
// in sequence, so the second write resurrected what the first had removed — the
// artifact the user asked to remove survived the command.
//
// Only edits proven INDEPENDENT are folded. Ordering within state is not a
// trustworthy install chronology, so a fold whose result would depend on order is
// refused rather than guessed at: overlapping targets become non-promotable SKIPs.
func composeSettingGroup(path string, group []fileIntent, read ReadExisting, occupancy Occupancy, force bool) (settingGroup, error) {
	var out settingGroup

	current, exists, err := read(path)
	if err != nil {
		return settingGroup{}, fmt.Errorf("remove: read %s: %w", path, err)
	}
	if !exists {
		// Nothing to strip from a file that is gone. Logically satisfied, per
		// contributor.
		for _, fi := range group {
			d := baseDiff(fi)
			d.Action = diff.Skip
			d.Note = "settings file absent — nothing to remove"
			out.diffs = append(out.diffs, d)
			out.ledger = append(out.ledger, ledgerEntry(fi, AlreadyAbsent))
		}
		return out, nil
	}

	// All-or-nothing on parse failure: a buffer we cannot read is a buffer we must
	// not partially rewrite. Probe with EVERY row's edit, not just the first —
	// rows on one path can disagree about the file's format, and a probe that
	// only tried one of them would let the disagreement surface mid-fold.
	for _, fi := range group {
		unreadable := probeParse(current, fi.file.Setting)
		if unreadable == "" {
			continue
		}
		return refuseGroup(group, path, unreadable, current), nil
	}

	if force {
		return forceSettingGroup(path, group, current), nil
	}

	// Include unselected contributors: reversing a child can destroy an owned
	// parent just as surely as a selected overlap can. No --force promotion.
	for _, fi := range group {
		self := adapter.SettingOwner{Artifact: fi.item.Artifact, Tool: fi.item.Tool, Scope: fi.item.Scope}
		for _, other := range occupancy[path] {
			if other.owner() == self {
				continue
			}
			if err := adapter.CheckSettingPair(fi.file.Setting, self, other.Setting, other.owner()); err != nil {
				return refuseOverlap(group, path, err.Error(), current), nil
			}
			if adapter.SettingEditsOverlap(fi.file.Setting, other.Setting) {
				return refuseOverlap(group, path, "overlaps recorded owner "+other.Label()+"; explicit migration required", current), nil
			}
		}
	}

	foldable, ambiguous := partitionByOverlap(group)
	if len(ambiguous) > 0 {
		return refuseOverlap(group, path, "overlapping selected settings keys; explicit migration required", current), nil
	}

	// Fold the independent edits onto one buffer, in stable identity order so the
	// output bytes are deterministic. Order cannot affect correctness here: every
	// row left in foldable targets a disjoint key.
	sortByIdentity(foldable)
	buf := current
	var landed []fileIntent
	for _, fi := range foldable {
		stripped, found, unreadable := stripSetting(buf, fi.file.Setting)
		if unreadable != "" {
			// Every row's edit parsed the original bytes, so reaching here means a
			// fold produced something a later row cannot read. Discard the partial
			// buffer and refuse the whole group: an unreadable config is a
			// recoverable warn-and-skip the user can fix and re-run, never a fatal
			// and never a partial write.
			return refuseGroup(group, path, unreadable, current), nil
		}
		if !found {
			d := baseDiff(fi)
			d.Action = diff.Skip
			d.Before = current
			d.Note = "setting absent — nothing to remove"
			out.diffs = append(out.diffs, d)
			out.ledger = append(out.ledger, ledgerEntry(fi, SettingAbsent))
			continue
		}
		buf = stripped
		landed = append(landed, fi)
	}

	if len(landed) == 0 {
		return out, nil
	}

	// One physical write, N logical contributions. The first contributor owns the
	// diff; the rest ride RestoreContrib so every output surface still shows a row
	// per artifact the user asked to remove.
	composite := baseDiff(landed[0])
	composite.Action = diff.Restore
	composite.Before = current
	composite.After = buf
	for _, fi := range landed[1:] {
		composite.RestoreContrib = append(composite.RestoreContrib, diff.RestoreContrib{
			Artifact: fi.item.Artifact,
			Version:  fi.item.ItemVersion,
		})
	}
	out.diffs = append(out.diffs, composite)
	for _, fi := range landed {
		out.ledger = append(out.ledger, ledgerEntry(fi, Applied))
	}
	return out, nil
}

// refuseGroup skips every contributor on a path with the same unreadable-config
// warning, writing nothing. It is the all-or-nothing answer to a file we cannot
// parse: the removal is not done, so each row is held open for a re-run once the
// user has repaired the file.
func refuseGroup(group []fileIntent, path, unreadable string, current []byte) settingGroup {
	var out settingGroup
	for _, fi := range group {
		d := baseDiff(fi)
		d.Action = diff.Skip
		d.Before = current
		d.Note = "settings conflict — skipped"
		out.diffs = append(out.diffs, d)
		out.ledger = append(out.ledger, ledgerEntry(fi, UnreadableSkipped))
		out.warnings = append(out.warnings, Warning{Item: fi.item.Artifact, Path: path, Message: unreadable})
	}
	return out
}

// partitionByOverlap splits a path's settings rows into those that can be folded
// onto one buffer and those whose targets overlap.
//
// Two edits COMMUTE when they touch disjoint state: distinct list identities
// under one list path, or scalar keys at unrelated dotted paths. They DO NOT
// commute when they name the same scalar key, when one dotted path is an ancestor
// of another (a scalar set beneath which another edit nests), or when two list
// rows share a (Dotted, IdentityKey, Identity). Anything that does not commute is
// ambiguous, because state carries no trustworthy install chronology to break the
// tie with.
func partitionByOverlap(group []fileIntent) (foldable, ambiguous []fileIntent) {
	conflicted := make([]bool, len(group))
	for i := range group {
		for j := i + 1; j < len(group); j++ {
			if adapter.SettingEditsOverlap(group[i].file.Setting, group[j].file.Setting) {
				conflicted[i], conflicted[j] = true, true
			}
		}
	}
	for i, fi := range group {
		if conflicted[i] {
			ambiguous = append(ambiguous, fi)
			continue
		}
		foldable = append(foldable, fi)
	}
	return foldable, ambiguous
}

// sortByIdentity orders rows deterministically so composed output bytes are
// stable across runs. It is a display/reproducibility concern only: correctness
// never depends on this order, because overlapping rows were already refused.
func sortByIdentity(rows []fileIntent) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.item.Artifact != b.item.Artifact {
			return a.item.Artifact < b.item.Artifact
		}
		return a.file.Setting.Dotted < b.file.Setting.Dotted
	})
}

// probeParse reports a user-facing message when current cannot be parsed at all,
// using the recorded edit's structural reader. It distinguishes "the file is
// malformed" from "our key is not there", which is the difference between holding
// a state row open and retiring it.
func probeParse(current []byte, edit *diff.SettingEdit) string {
	if edit == nil {
		return ""
	}
	_, _, err := adapter.SettingStatus(current, edit)
	if err != nil {
		return err.Error()
	}
	return ""
}

// baseDiff builds the identity-carrying shell every undo row shares, so the
// renderer groups a removal exactly like the install row it reverses.
func baseDiff(fi fileIntent) diff.FileDiff {
	return diff.FileDiff{
		Path:     fi.file.Path,
		Artifact: fi.item.Artifact,
		Version:  fi.item.ItemVersion,
		Tool:     fi.item.Tool,
		Scope:    fi.item.Scope,
		Root:     fi.item.Root,
	}
}

// fileUndo builds the inverse diff for one recorded file, along with its
// structured outcome. It may also return a warning (drift skipped). The returned
// diff always carries the item's identity metadata so the renderer groups it like
// an install row.
func fileUndo(it state.Item, f state.FileState, read ReadExisting, occupancy Occupancy) (diff.FileDiff, *Warning, Outcome, error) {
	base := baseDiff(fileIntent{item: it, file: f})

	current, exists, err := read(f.Path)
	if err != nil {
		return base, nil, "", fmt.Errorf("remove: read %s: %w", f.Path, err)
	}

	base.Before = current

	// Pi never had a supported whole-file configuration inverse or checksumless
	// file adoption. Preserve insufficient evidence, even under --force.
	if it.Tool == "pi" && ((f.Action == string(diff.Merge) && f.Setting == nil) || f.Checksum == "") {
		base.Action = diff.Skip
		base.Note = "insufficient Pi ownership evidence — retained"
		return base, &Warning{Item: it.Artifact, Path: f.Path, Message: base.Note}, UnsafeLegacySkipped, nil
	}

	switch f.Action {
	case string(diff.Create), string(diff.Fetch):
		// Undo of a created/placed file is a delete, gated on the on-disk bytes
		// still matching what we wrote.
		if !exists {
			base.Action = diff.Skip
			base.Note = "already absent"
			return base, nil, AlreadyAbsent, nil
		}
		if drift := driftsFromChecksum(current, f.Checksum); drift {
			base.Action = diff.Skip
			base.Intended = diff.Delete
			base.Before = current
			base.Note = "user-edited since install — skipped (use --force)"
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "modified since install; not removed (use --force)"}, DriftSkipped, nil
		}
		base.Action = diff.Delete
		base.Before = current // so the verbose unified diff shows the removal
		return base, nil, Applied, nil

	case string(diff.Append):
		// Undo of an append is a surgical un-append of exactly our fenced section.
		if !exists {
			base.Action = diff.Skip
			base.Note = "file absent — nothing to un-append"
			return base, nil, AlreadyAbsent, nil
		}
		stripped, found := adapter.RemoveSection(current, f.Section)
		if !found {
			// Our fenced section is ALREADY GONE from the file (e.g. a later rebuild
			// dropped it — the beads→ticket migration is the canonical case). The file
			// work is already done, so this is a SUCCESSFUL no-op, not a skip: emit an
			// UNAPPEND whose After is the unchanged file. That lands it in the applier's
			// Applied set, which is what retires the orphaned state row — a SKIP would
			// strand the row forever, and `--force` cannot rescue it (a SKIP with no
			// Intended action has nothing to promote), so `scan` would report MISSING
			// with no way to ever clean it up.
			base.Action = diff.Unappend
			base.Before = current
			base.After = current
			base.Section = &diff.SectionEdit{Name: f.Section}
			base.Note = "section already absent — retiring the record (no file change)"
			return base, nil, Applied, nil
		}
		// Drift: the user changed the file outside our fenced section since install.
		// We can't compare just the section body (the recorded checksum is of the
		// whole post-install file, not the block alone), so we use a well-defined
		// invariant: un-appending our block should yield exactly what was there
		// BEFORE we appended — the recorded Prior. When the file did not exist before
		// install, Prior is nil and the expected stripped result is empty (the file
		// should contain only our block). If the stripped form differs from that
		// expectation, the user edited around our section → skip unless --force.
		expected := f.Prior
		// AppendSection treats an absent OR whitespace-only file as "empty" and makes
		// the block the whole content, so un-appending yields empty bytes. Match that:
		// when the pre-install file was absent/blank, the expectation is blank too, and
		// we compare trimmed so a lone trailing newline is not mistaken for drift.
		var drift bool
		if len(bytes.TrimSpace(expected)) == 0 {
			drift = len(bytes.TrimSpace(stripped)) != 0
		} else {
			drift = !bytes.Equal(stripped, expected)
		}
		if drift {
			base.Action = diff.Skip
			base.Intended = diff.Unappend
			base.Before = current
			base.After = stripped
			base.Section = &diff.SectionEdit{Name: f.Section}
			base.Note = "file changed since install — skipped (use --force)"
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "instructions file changed since install; section not removed (use --force)"}, DriftSkipped, nil
		}
		base.Action = diff.Unappend
		base.Before = current
		base.After = stripped
		base.Section = &diff.SectionEdit{Name: f.Section}
		return base, nil, Applied, nil

	case string(diff.Merge):
		// A settings MERGE is undone SURGICALLY — strip exactly our array element
		// (a hook) or delete exactly our key (a scalar setting), leaving every
		// sibling (other artifacts' edits, the user's) intact. This is the MERGE-side
		// twin of APPEND's un-section, and unlike a whole-file Prior restore it is
		// correct even when other edits folded into the same file after ours.
		if f.Setting != nil {
			if !exists {
				base.Action = diff.Skip
				base.Note = "settings file absent — nothing to remove"
				return base, nil, AlreadyAbsent, nil
			}
			stripped, found, unreadable := stripSetting(current, f.Setting)
			if unreadable != "" {
				// An unparseable settings file becomes a user-facing warning + SKIP,
				// not a fatal: the user can fix it and re-run.
				base.Action = diff.Skip
				base.Note = "settings unreadable — skipped"
				return base, &Warning{Item: it.Artifact, Path: f.Path, Message: unreadable}, UnreadableSkipped, nil
			}
			if !found {
				base.Action = diff.Skip
				base.Note = "setting absent — nothing to remove"
				return base, nil, SettingAbsent, nil
			}
			base.Action = diff.Restore // write the surgically-edited bytes
			base.Before = current
			base.After = stripped
			return base, nil, Applied, nil
		}
		// A pre-compose MERGE row carries only a whole-file snapshot of what the
		// config looked like before this artifact was installed. Writing it back
		// reverts the file WHOLESALE: every other artifact wired into it since, and
		// every user edit made since, disappear along with our own contribution.
		//
		// That is safe only when this artifact is provably the sole contributor to
		// the path. Otherwise refuse. The snapshot cannot be turned into a surgical
		// undo after the fact: reconstructing this artifact's structural
		// contribution would mean diffing successive snapshots, and state carries
		// neither a trustworthy contributor ordering nor any way to tell a user's
		// intervening edit from one of ours. Heuristic recovery is not safe removal.
		//
		// The refusal is deliberately NOT promotable (no Intended): --force means
		// "I accept losing my own edit to this file", never "I accept losing another
		// artifact's wiring".
		self := Contributor{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope}
		if others := occupancy.othersOn(f.Path, self); len(others) > 0 {
			base.Action = diff.Skip
			base.Note = "shared config with a pre-compose record — skipped"
			return base, &Warning{
				Item:    it.Artifact,
				Path:    f.Path,
				Message: fmt.Sprintf("recorded before per-key removal existed, and %s also writes this file; not restored — remove the setting by hand", strings.Join(others, ", ")),
			}, UnsafeLegacySkipped, nil
		}
		// Undo of a scalar merge restores the recorded pre-install bytes wholesale.
		if !exists {
			// The merged file is gone; restoring Prior would resurrect a file the
			// user deleted. Skip and note.
			base.Action = diff.Skip
			base.Note = "file absent — not restoring"
			return base, nil, AlreadyAbsent, nil
		}
		if drift := driftsFromChecksum(current, f.Checksum); drift {
			base.Action = diff.Skip
			base.Intended = diff.Restore
			base.Before = current
			base.After = f.Prior
			base.Note = "config changed since install — skipped (use --force)"
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "config changed since install; not restored (use --force)"}, DriftSkipped, nil
		}
		base.Action = diff.Restore
		base.Before = current
		base.After = f.Prior
		return base, nil, Applied, nil

	default:
		base.Action = diff.Skip
		base.Note = "unknown recorded action " + f.Action
		return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "unknown recorded action " + f.Action}, UnknownAction, nil
	}
}

// Promote rewrites every drift SKIP (a SKIP carrying an Intended action) in r's
// change set to that intended action, so --force turns "skipped because edited"
// into the real undo, and re-marks those contributors Applied in the ledger so
// state retirement sees the same reality the applier will.
//
// Rows that are SKIP for a benign reason (already absent, section missing) carry
// no Intended and are left alone. So do the refusals that composition introduced:
// an unsafe pre-compose restore and an ambiguous overlapping edit deliberately
// carry no Intended, because --force is consent to lose YOUR OWN edit to a file,
// not consent to lose another artifact's wiring.
//
// Mutates r in place and returns it.
func Promote(r Result) Result {
	promoted := map[string]bool{}
	for i := range r.ChangeSet.Diffs {
		d := &r.ChangeSet.Diffs[i]
		if d.Action == diff.Skip && d.Intended != "" {
			d.Action = d.Intended
			d.Note = ""
			promoted[d.Artifact+"\x00"+d.Tool+"\x00"+d.Scope+"\x00"+d.Path] = true
		}
	}
	for i := range r.Ledger {
		e := &r.Ledger[i]
		if e.Outcome == DriftSkipped && promoted[e.Artifact+"\x00"+e.Tool+"\x00"+e.Scope+"\x00"+e.Path] {
			e.Outcome = Applied
		}
	}
	var warnings []Warning
	for _, w := range r.Warnings {
		removed := false
		for _, d := range r.ChangeSet.Diffs {
			if d.Artifact == w.Item && d.Path == w.Path && promoted[d.Artifact+"\x00"+d.Tool+"\x00"+d.Scope+"\x00"+d.Path] {
				removed = true
			}
		}
		if !removed {
			warnings = append(warnings, w)
		}
	}
	r.Warnings = warnings
	return r
}

// driftsFromChecksum reports whether current's sha256 differs from the recorded
// checksum (a "sha256:<hex>" string). An empty recorded checksum is treated as
// "unknown" → no drift (we can't prove an edit, so we don't block the undo).
func driftsFromChecksum(current []byte, recorded string) bool {
	if recorded == "" {
		return false
	}
	sum := sha256.Sum256(current)
	got := "sha256:" + hex.EncodeToString(sum[:])
	return got != recorded
}

// stripSetting surgically reverses the recorded settings edit (a hook element or
// a scalar key) from current, returning the edited bytes, whether anything was
// found, and a non-empty warning message if the settings file could not be
// parsed. It folds the error into a message string so the caller branches on a
// value (warn + skip), keeping the "unparseable config is recoverable, not fatal"
// contract.
func stripSetting(current []byte, edit *diff.SettingEdit) (stripped []byte, found bool, warning string) {
	out, found, err := adapter.RemoveSettingEdit(current, edit)
	if err != nil {
		return nil, false, "settings conflict; setting not removed: " + err.Error()
	}
	return out, found, ""
}

// joinCmds renders recorded post-install commands compactly for a warning.
func joinCmds(cmds []string) string {
	out := ""
	for i, c := range cmds {
		if i > 0 {
			out += "; "
		}
		out += c
	}
	return out
}

func refuseOverlap(group []fileIntent, path, message string, current []byte) settingGroup {
	out := refuseGroup(group, path, message, current)
	for i := range out.diffs {
		out.diffs[i].Note = "overlapping settings ownership — skipped"
	}
	for i := range out.ledger {
		out.ledger[i].Outcome = AmbiguousSkipped
	}
	return out
}

// composeSections validates all selected Pi or Codex sections against the same original
// bytes, then strips them in one physical write. The existing ledger records
// each logical inverse; no new persisted ownership or section-body hash exists.
func composeSections(path string, group []fileIntent, read ReadExisting, occupancy Occupancy, force bool) (Result, error) {
	current, exists, err := read(path)
	if err != nil {
		return Result{}, fmt.Errorf("remove: read %s: %w", path, err)
	}
	sort.SliceStable(group, func(i, j int) bool { return group[i].file.Section < group[j].file.Section })
	sections := map[string]bool{}
	allowed := map[string]bool{}
	for _, fi := range group {
		f := fi.file
		expected := fi.item.Artifact
		if fi.item.Tool == "pi" {
			expected = "pi:" + expected
		}
		if (fi.item.Tool != "pi" && fi.item.Tool != "codex") || fi.item.Tool != group[0].item.Tool || fi.item.Scope != group[0].item.Scope || fi.item.Root != group[0].item.Root || f.Action != string(diff.Append) || f.Section != expected || sections[f.Section] {
			return refusePiSections(group, "ambiguous section ownership or incompatible actions/scopes", current), nil
		}
		sections[f.Section] = true
		allowed[f.Section] = true
		self := adapter.SettingOwner{Artifact: fi.item.Artifact, Tool: fi.item.Tool, Scope: fi.item.Scope}
		claims := 0
		occupied := map[string]bool{}
		for _, other := range occupancy[path] {
			if fi.item.Tool == "codex" && occupied[other.Section] {
				return refusePiSections(group, "ambiguous recorded sibling ownership", current), nil
			}
			occupied[other.Section] = true
			if other.Section == "" || other.Tool != fi.item.Tool || other.Scope != fi.item.Scope {
				return refusePiSections(group, "incompatible recorded context ownership", current), nil
			}
			allowed[other.Section] = true
			if other.Tool == "codex" && other.Section != other.Artifact {
				return refusePiSections(group, "incompatible recorded section identity", current), nil
			}
			if other.Section != f.Section {
				continue
			}
			claims++
			if other.owner() != self || claims > 1 {
				return refusePiSections(group, "overlapping recorded section ownership", current), nil
			}
		}
		if !exists {
			continue
		}
		if f.Checksum == "" || ((!force || fi.item.Tool == "codex") && driftsFromChecksum(current, f.Checksum)) {
			return refusePiSections(group, "context changed or lacks checksum evidence", current), nil
		}
		start := []byte("<!-- patronus:start " + f.Section + " -->")
		end := []byte("<!-- patronus:end " + f.Section + " -->")
		from, to := bytes.Index(current, start), bytes.Index(current, end)
		if bytes.Count(current, start) != 1 || bytes.Count(current, end) != 1 || from < 0 || to < from+len(start) {
			return refusePiSections(group, "missing, duplicated or unordered section markers", current), nil
		}
		afterEnd := to + len(end)
		if (from > 0 && current[from-1] != '\n') || from+len(start) >= len(current) || current[from+len(start)] != '\n' || current[to-1] != '\n' || (afterEnd < len(current) && current[afterEnd] != '\n') || bytes.Contains(current[from+len(start):to], []byte("<!-- patronus:")) {
			return refusePiSections(group, "ambiguous or nested section markers", current), nil
		}
	}
	if exists && group[0].item.Tool == "codex" {
		if err := adapter.ValidateSectionMarkers(current, allowed); err != nil {
			return refusePiSections(group, err.Error(), current), nil
		}
	}
	result := Result{ChangeSet: &diff.ChangeSet{}}
	if !exists {
		for _, fi := range group {
			d := baseDiff(fi)
			d.Action = diff.Skip
			d.Note = "context file absent — nothing to remove"
			result.ChangeSet.Diffs = append(result.ChangeSet.Diffs, d)
			result.Ledger = append(result.Ledger, ledgerEntry(fi, AlreadyAbsent))
		}
		return result, nil
	}
	buf := current
	var names []string
	for _, fi := range group {
		var found bool
		buf, found = adapter.RemoveSection(buf, fi.file.Section)
		if !found {
			return refusePiSections(group, "section could not be stripped after folding", current), nil
		}
		names = append(names, fi.file.Section)
		result.Ledger = append(result.Ledger, ledgerEntry(fi, Applied))
	}
	composite := baseDiff(group[0])
	composite.Action = diff.Unappend
	composite.Before = current
	composite.After = buf
	composite.Section = &diff.SectionEdit{Name: group[0].file.Section}
	for _, fi := range group[1:] {
		composite.Contrib = append(composite.Contrib, diff.SectionContrib{Artifact: fi.item.Artifact, Version: fi.item.ItemVersion, Section: fi.file.Section})
	}
	composite.Note = "remove " + group[0].item.Tool + " sections: " + strings.Join(names, ", ")
	result.ChangeSet.Diffs = append(result.ChangeSet.Diffs, composite)
	return result, nil
}

func refusePiSections(group []fileIntent, message string, current []byte) Result {
	result := Result{ChangeSet: &diff.ChangeSet{}}
	for _, fi := range group {
		d := baseDiff(fi)
		d.Action = diff.Skip
		d.Before = current
		d.Note = message + " — retained"
		result.ChangeSet.Diffs = append(result.ChangeSet.Diffs, d)
		result.Ledger = append(result.Ledger, ledgerEntry(fi, AmbiguousSkipped))
		result.Warnings = append(result.Warnings, Warning{Item: fi.item.Artifact, Path: fi.file.Path, Message: d.Note})
	}
	return result
}

// forceSettingGroup deletes the selected union, never a competing prior value.
func forceSettingGroup(path string, group []fileIntent, current []byte) settingGroup {
	sort.SliceStable(group, func(i, j int) bool { return len(group[i].file.Setting.Dotted) < len(group[j].file.Setting.Dotted) })
	buf := current
	var out settingGroup
	var removed []*diff.SettingEdit
	for _, fi := range group {
		covered := false
		for _, parent := range removed {
			if parent.IdentityKey == "" && (parent.Dotted == fi.file.Setting.Dotted || strings.HasPrefix(fi.file.Setting.Dotted, parent.Dotted+".")) {
				covered = true
				break
			}
		}
		if !covered {
			next, _, err := adapter.DeleteSettingEdit(buf, fi.file.Setting)
			if err != nil {
				return refuseGroup(group, path, err.Error(), current)
			}
			buf = next
			removed = append(removed, fi.file.Setting)
		}
		out.ledger = append(out.ledger, ledgerEntry(fi, Applied))
	}
	d := baseDiff(group[0])
	d.Before = current
	d.After = buf
	d.Action = diff.Restore
	d.Note = "force: delete selected setting union (no prior restoration)"
	out.diffs = []diff.FileDiff{d}
	return out
}

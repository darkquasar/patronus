// Package remove computes the inverse of a recorded install: it reads the state
// items Patronus wrote (internal/state) and produces a diff.ChangeSet of UNDO
// actions on the same spine the installer and renderer already speak. There is no
// new write machinery — DELETE/UNAPPEND/RESTORE flow through the existing
// install.Applier exactly as CREATE/APPEND/MERGE do.
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

// Compute turns recorded state items into an inverse change set plus warnings.
// read supplies current on-disk bytes so each undo can be gated against drift.
// The returned ChangeSet is already classified (DELETE/UNAPPEND/RESTORE/SKIP);
// Classify is not involved. A drift row is a SKIP whose Note explains why; the
// cmd layer rewrites those to their intended action under --force.
func Compute(items []state.Item, read ReadExisting) (*diff.ChangeSet, []Warning, error) {
	var (
		diffs    []diff.FileDiff
		warnings []Warning
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
			warnings = append(warnings, Warning{
				Item:    it.Artifact,
				Message: fmt.Sprintf("self-wired recipe %q cannot be auto-reverted%s — undo it manually", it.Artifact, cmds),
			})
		}

		for _, f := range it.Files {
			d, w, err := fileUndo(it, f, read)
			if err != nil {
				return nil, nil, err
			}
			if w != nil {
				warnings = append(warnings, *w)
			}
			diffs = append(diffs, d)
		}
	}
	return &diff.ChangeSet{Diffs: diffs}, warnings, nil
}

// fileUndo builds the inverse diff for one recorded file. It may also return a
// warning (drift skipped). The returned diff always carries the item's identity
// metadata so the renderer groups it like an install row.
func fileUndo(it state.Item, f state.FileState, read ReadExisting) (diff.FileDiff, *Warning, error) {
	base := diff.FileDiff{
		Path:     f.Path,
		Artifact: it.Artifact,
		Version:  it.ItemVersion,
		Tool:     it.Tool,
		Scope:    it.Scope,
	}

	current, exists, err := read(f.Path)
	if err != nil {
		return base, nil, fmt.Errorf("remove: read %s: %w", f.Path, err)
	}

	switch f.Action {
	case string(diff.Create), string(diff.Fetch):
		// Undo of a created/placed file is a delete, gated on the on-disk bytes
		// still matching what we wrote.
		if !exists {
			base.Action = diff.Skip
			base.Note = "already absent"
			return base, nil, nil
		}
		if drift := driftsFromChecksum(current, f.Checksum); drift {
			base.Action = diff.Skip
			base.Intended = diff.Delete
			base.Before = current
			base.Note = "user-edited since install — skipped (use --force)"
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "modified since install; not removed (use --force)"}, nil
		}
		base.Action = diff.Delete
		base.Before = current // so the verbose unified diff shows the removal
		return base, nil, nil

	case string(diff.Append):
		// Undo of an append is a surgical un-append of exactly our fenced section.
		if !exists {
			base.Action = diff.Skip
			base.Note = "file absent — nothing to un-append"
			return base, nil, nil
		}
		stripped, found := adapter.RemoveSection(current, f.Section)
		if !found {
			base.Action = diff.Skip
			base.Note = "section absent — nothing to un-append"
			return base, nil, nil
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
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "instructions file changed since install; section not removed (use --force)"}, nil
		}
		base.Action = diff.Unappend
		base.Before = current
		base.After = stripped
		base.Section = &diff.SectionEdit{Name: f.Section}
		return base, nil, nil

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
				return base, nil, nil
			}
			stripped, found, unreadable := stripSetting(current, f.Setting)
			if unreadable != "" {
				// An unparseable settings file becomes a user-facing warning + SKIP,
				// not a fatal: the user can fix it and re-run.
				base.Action = diff.Skip
				base.Note = "settings unreadable — skipped"
				return base, &Warning{Item: it.Artifact, Path: f.Path, Message: unreadable}, nil
			}
			if !found {
				base.Action = diff.Skip
				base.Note = "setting absent — nothing to remove"
				return base, nil, nil
			}
			base.Action = diff.Restore // write the surgically-edited bytes
			base.Before = current
			base.After = stripped
			return base, nil, nil
		}
		// Undo of a scalar merge restores the recorded pre-install bytes wholesale.
		if !exists {
			// The merged file is gone; restoring Prior would resurrect a file the
			// user deleted. Skip and note.
			base.Action = diff.Skip
			base.Note = "file absent — not restoring"
			return base, nil, nil
		}
		if drift := driftsFromChecksum(current, f.Checksum); drift {
			base.Action = diff.Skip
			base.Intended = diff.Restore
			base.Before = current
			base.After = f.Prior
			base.Note = "config changed since install — skipped (use --force)"
			return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "config changed since install; not restored (use --force)"}, nil
		}
		base.Action = diff.Restore
		base.Before = current
		base.After = f.Prior
		return base, nil, nil

	default:
		base.Action = diff.Skip
		base.Note = "unknown recorded action " + f.Action
		return base, &Warning{Item: it.Artifact, Path: f.Path, Message: "unknown recorded action " + f.Action}, nil
	}
}

// Promote rewrites every drift SKIP (a SKIP carrying an Intended action) in cs to
// that intended action, so --force turns "skipped because edited" into the real
// undo. Rows that are SKIP for a benign reason (already absent, section missing)
// carry no Intended and are left alone. Mutates cs in place and returns it.
func Promote(cs *diff.ChangeSet) *diff.ChangeSet {
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Action == diff.Skip && d.Intended != "" {
			d.Action = d.Intended
			d.Note = ""
		}
	}
	return cs
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
		return nil, false, "settings file unparseable; setting not removed: " + err.Error()
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

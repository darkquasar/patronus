// Package drift reconciles what Patronus RECORDED against what is actually on disk
// — and against what the catalog says the source is NOW.
//
// This exists because Patronus already recorded the truth and then never read it
// back. internal/state has stored a sha256 for every deployed file since Phase 3
// (state.go:52-58, whose doc comment names this exact use verbatim: "which lets a
// later command tell whether a file is untouched-since-write or user-edited"), and
// internal/remove has had a function literally called driftsFromChecksum
// (compute.go:237) — wired into `remove`, and nothing else. No install-path or
// scan-path caller ever compared a deployed artifact to its source.
//
// The consequence, in production: the installed team-research skill drifted behind
// its source and instructed the agent to call a tool that does not exist. Nothing
// detected it. This package is what detects it.
//
// This mirrors the binary-side fix in recipe.classifyFetch, which likewise reads
// back a digest Patronus had been recording and ignoring.
package drift

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

// Verdict is one deployed file's reconciliation outcome.
type Verdict string

const (
	// OK means recorded, on-disk, and source all agree — or the path is empty and
	// unrecorded, which is not our business.
	OK Verdict = "OK"
	// Stale means on-disk matches what we wrote, but the source has moved on.
	// `install` should re-deploy. This is the team-research bug.
	Stale Verdict = "STALE"
	// UserEdited means on-disk differs from what we recorded writing. Report it;
	// never silently overwrite it.
	UserEdited Verdict = "USER-EDITED"
	// UnmanagedShadow means a file occupies a path Patronus would deploy to, and
	// Patronus never wrote it. state.json alone is blind to this.
	UnmanagedShadow Verdict = "UNMANAGED-SHADOW"
	// OrphanedState means a state row whose item is no longer in the catalog.
	OrphanedState Verdict = "ORPHANED-STATE"
	// Missing means we recorded writing the file, but it is gone.
	Missing Verdict = "MISSING"
)

// Finding is one file's verdict, ready to render.
type Finding struct {
	Path    string
	Item    string
	Verdict Verdict
	Detail  string
}

// Classify decides one deployed file's verdict.
//
//   - current/exists: the file's bytes on disk (exists=false means it is absent).
//   - recorded: the FileState.Checksum Patronus stored when it wrote the file
//     ("sha256:<hex>"), or "" when there is NO state row for this path.
//   - source/hasSource: the bytes the catalog would deploy now (hasSource=false
//     means the item is no longer in the catalog).
//
// The order of the checks is the point:
//
//  1. A file with no state row -> UNMANAGED SHADOW. This is the only verdict that
//     catches the defect that motivated this package: the stale skill was never in
//     state.json at all, so a check that walks only state.json reports nothing wrong
//     while the agent runs a dead protocol.
//  2. On-disk != recorded -> USER-EDITED, and this outranks ORPHANED-STATE: bytes
//     the user may have authored are at risk, which is the fact worth reporting even
//     when the item has left the catalog.
//  3. On-disk == recorded but the source moved -> STALE.
func Classify(current []byte, exists bool, recorded string, source []byte, hasSource bool) Verdict {
	if !exists {
		if recorded == "" {
			return OK // nothing here, nothing recorded: not our business
		}
		return Missing
	}
	if recorded == "" {
		// A file occupies a path we WOULD deploy to, and we never wrote it.
		return UnmanagedShadow
	}
	if checksum(current) != recorded {
		return UserEdited
	}
	if !hasSource {
		// We wrote it and it is untouched, but the catalog no longer has the item.
		return OrphanedState
	}
	if !bytes.Equal(current, source) {
		return Stale
	}
	return OK
}

// ClassifySection reconciles ONE fenced section of a composed APPEND file
// (CLAUDE.md / AGENTS.md), where several artifacts each fold a
// `<!-- patronus:start <name> -->` block into one shared file. Whole-file
// Classify cannot judge these: the deployed file is a fold of MANY sources, so it
// never equals any single source's would-be bytes, and every contributor records
// the SAME whole-file checksum — so a second contributor's legitimate append reads
// as a user edit. The reconciliation that IS meaningful is per section body.
//
//   - onDisk/present: the body currently inside this file's fenced block for this
//     section (present=false means the block is absent from the deployed file).
//   - source/hasSource: the body the catalog would fold in now for this section
//     (hasSource=false means the artifact is no longer in the catalog).
//
// Deliberate limitation: state records a whole-file checksum per APPEND, never a
// per-section one, so there is no recorded section body to diff against. This means
// a section whose on-disk body differs from the source is reported STALE — the
// actionable verdict, since `install`'s AppendSection is idempotent and re-folds it
// — rather than distinguishing "the user hand-edited our block" from "the source
// moved on." Both resolve the same way (re-run install), so the merged verdict
// loses no actionable information; recording a per-section checksum to split them is
// future work.
func ClassifySection(onDisk []byte, present bool, source []byte, hasSource bool) Verdict {
	if !present {
		// We recorded folding this section in, and the block is gone from the file.
		return Missing
	}
	if !hasSource {
		// Our block is still in the file, but the catalog no longer has the artifact
		// that owns it — the record diverged from reality (e.g. a removed instruction).
		return OrphanedState
	}
	// Compare on the fence's OWN normalization. The composer trims trailing newlines
	// off the body before writing it between the markers, and the extractor trims
	// surrounding newlines off what it reads back — so the raw source body (which may
	// carry a trailing newline the fence would drop) must be trimmed the same way, or
	// every section reads as STALE over a newline the file never stored.
	if !bytes.Equal(trimFence(onDisk), trimFence(source)) {
		return Stale
	}
	return OK
}

// trimFence normalizes a section body the way the fenced-block composer/extractor
// do — stripping surrounding newlines — so a source body and an on-disk body are
// compared on the bytes the fence actually stores, not on incidental trailing
// whitespace.
func trimFence(b []byte) []byte { return bytes.Trim(b, "\n") }

// checksum matches the form internal/state records ("sha256:<hex>"), so the two can
// be compared directly. It mirrors state.checksum and remove.driftsFromChecksum —
// the same digest, finally read on a path other than `remove`.
func checksum(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

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

// checksum matches the form internal/state records ("sha256:<hex>"), so the two can
// be compared directly. It mirrors state.checksum and remove.driftsFromChecksum —
// the same digest, finally read on a path other than `remove`.
func checksum(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

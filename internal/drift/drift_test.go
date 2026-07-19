package drift

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// TestClassify covers every verdict — and the one that matters is UnmanagedShadow:
// it is the ONLY verdict that would have caught the defect that motivated this
// guard. The stale team-research skill was never in state.json at all (it was placed
// by hand or by another tool), so a check that walks only state.json reports NOTHING
// WRONG while the agent keeps executing a protocol from a deleted era.
func TestClassify(t *testing.T) {
	ours := []byte("what patronus wrote\n")
	moved := []byte("what the source says NOW\n")
	edited := []byte("what the user typed\n")

	tests := []struct {
		name      string
		current   []byte
		exists    bool
		recorded  string
		source    []byte
		hasSource bool
		want      Verdict
	}{
		{
			name:    "untouched and current",
			current: ours, exists: true, recorded: sum(ours),
			source: ours, hasSource: true,
			want: OK,
		},
		{
			// THE TEAM-RESEARCH BUG. The deployed copy is exactly what we wrote, but
			// the source moved on and nothing re-deployed it. The agent runs stale
			// prose forever, and every status says "installed".
			name:    "source moved on -> STALE",
			current: ours, exists: true, recorded: sum(ours),
			source: moved, hasSource: true,
			want: Stale,
		},
		{
			// The user edited an installed skill. REPORT it. Never silently overwrite.
			name:    "user edited -> USER-EDITED",
			current: edited, exists: true, recorded: sum(ours),
			source: ours, hasSource: true,
			want: UserEdited,
		},
		{
			// The one that matters. A file sits at a path we WOULD deploy to, and we
			// have NO state row for it — we never wrote it. state.json alone is blind
			// to this. It is exactly what bit this project.
			name:    "deployed but unrecorded -> UNMANAGED SHADOW",
			current: edited, exists: true, recorded: "",
			source: ours, hasSource: true,
			want: UnmanagedShadow,
		},
		{
			// A state row for an item the catalog no longer has (e.g. bd, still
			// recorded as installed alongside tk). The record diverged from reality.
			name:    "state row, no source -> ORPHANED STATE",
			current: ours, exists: true, recorded: sum(ours),
			source: nil, hasSource: false,
			want: OrphanedState,
		},
		{
			name:    "recorded but gone -> MISSING",
			current: nil, exists: false, recorded: sum(ours),
			source: ours, hasSource: true,
			want: Missing,
		},
		{
			// Neither on disk nor recorded: a path the catalog could deploy to but
			// nothing has. Silence is correct — an uninstalled item is not drift.
			name:    "absent and unrecorded -> OK",
			current: nil, exists: false, recorded: "",
			source: ours, hasSource: true,
			want: OK,
		},
		{
			// USER-EDITED outranks ORPHANED-STATE: the user's bytes are at risk, and
			// that is the fact worth reporting even when the item left the catalog.
			name:    "edited and no source -> USER-EDITED",
			current: edited, exists: true, recorded: sum(ours),
			source: nil, hasSource: false,
			want: UserEdited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.current, tt.exists, tt.recorded, tt.source, tt.hasSource)
			if got != tt.want {
				t.Errorf("Classify = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClassifySection covers the per-section reconciliation of composed APPEND
// files (CLAUDE.md/AGENTS.md), which whole-file Classify cannot judge: the file is
// a fold of many sources, so it never equals any single one, and every contributor
// records the same whole-file checksum. The verdict that matters here is STALE — it
// is what finally catches a section (e.g. agent-rules) whose source moved on while
// the composed file was never re-folded.
func TestClassifySection(t *testing.T) {
	ours := []byte("the section body we fold in\n")
	moved := []byte("the section body the source has NOW\n")

	tests := []struct {
		name      string
		onDisk    []byte
		present   bool
		source    []byte
		hasSource bool
		want      Verdict
	}{
		{
			name:   "section present and current -> OK",
			onDisk: ours, present: true, source: ours, hasSource: true,
			want: OK,
		},
		{
			// The composed-file bug this task closes: the source section moved on
			// (agent-rules gained a heuristic) but the file was never re-folded.
			name:   "source section moved on -> STALE",
			onDisk: ours, present: true, source: moved, hasSource: true,
			want: Stale,
		},
		{
			// A trailing newline the fence drops must NOT read as drift — the composer
			// trims it on write, the extractor trims it on read, so the comparison does
			// too. Without this, every section is a false STALE.
			name:   "differs only by a trailing newline the fence drops -> OK",
			onDisk: []byte("body"), present: true, source: []byte("body\n"), hasSource: true,
			want: OK,
		},
		{
			// Our fenced block is gone from the file (e.g. beads, replaced by ticket).
			name:   "section block absent -> MISSING",
			onDisk: nil, present: false, source: ours, hasSource: true,
			want: Missing,
		},
		{
			// The block is still there, but the artifact that owns it left the catalog.
			name:   "section present, source gone -> ORPHANED-STATE",
			onDisk: ours, present: true, source: nil, hasSource: false,
			want: OrphanedState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySection(tt.onDisk, tt.present, tt.source, tt.hasSource)
			if got != tt.want {
				t.Errorf("ClassifySection = %v, want %v", got, tt.want)
			}
		})
	}
}

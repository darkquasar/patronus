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

package scan

import (
	"github.com/darkquasar/patronus/internal/nativepi"
)

// NativeStatus separates scoped declaration/metadata readiness from loading or
// arbitrary npm-file integrity. Errors remain visible instead of becoming absent.
type NativeStatus struct {
	Artifact    string               `json:"artifact"`
	Identity    nativepi.Identity    `json:"identity"`
	Provenance  string               `json:"provenance"`
	Observation nativepi.Observation `json:"observation"`
	Error       string               `json:"error,omitempty"`
}

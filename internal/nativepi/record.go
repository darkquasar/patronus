package nativepi

// Record is bounded direct identity and operation evidence, not an npm file
// inventory. TrackedAt is not an installation date.
type Record struct {
	Operation   Operation   `json:"intent"`
	Provenance  string      `json:"provenance"`
	TrackedAt   string      `json:"trackedAt,omitempty"`
	InstalledAt string      `json:"installedAt,omitempty"`
	Observed    Observation `json:"observed"`
	Outcome     string      `json:"outcome"`
	Pending     *Pending    `json:"pending,omitempty"`
}

type Pending struct {
	Operation Operation   `json:"operation"`
	Before    Observation `json:"before"`
}

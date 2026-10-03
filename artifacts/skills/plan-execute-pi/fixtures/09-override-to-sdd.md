# Fixture 09: explicit sdd override on an otherwise-solo plan

Expected mode: sdd
Isolates: Rule 1 in the other direction. Assessed on its own the plan is solo (no hard
trigger, no two soft signals), and the explicit request overrides that.
Invocation: "Run this in sdd mode."

**Goal:** Add a `--quiet` flag that suppresses progress output.

## Global Constraints

- `--quiet` suppresses progress lines only. Errors still print to stderr.

### Task 1: Add the flag

**Files:**
- Modify: `cmd/sync/sync.go:30-64`
- Test: `cmd/sync/sync_test.go`

- [ ] Add `--quiet` as a bool, defaulting false.
- [ ] Assert stdout is empty and stderr carries the error on a failed sync with the flag.

### Task 2: Route progress writes through the flag

**Files:**
- Modify: `internal/progress/writer.go:15-40`

- [ ] Give `Writer` a `Quiet bool`; return early from `Line` when set.
- [ ] Assert `Line` writes nothing when Quiet.

# Fixture 02: exactly one soft signal

Expected mode: solo
Isolates: one soft signal does not reach Rule 3's threshold of two.
Invocation: "Execute this plan."

**Goal:** Add a `--format=json` flag to the `report` command.

## Global Constraints

- The existing human-readable output is unchanged when the flag is absent.

### Task 1: Add the flag and the JSON encoder

**Files:**
- Modify: `cmd/report/report.go:40-95`
- Test: `cmd/report/report_test.go`

- [ ] Add `--format` with values `text` (default) and `json`.
- [ ] Marshal the existing `Report` struct with `encoding/json`.
- [ ] Assert both formats in `report_test.go` against golden files.

### Task 2: Document the flag

**Files:**
- Modify: `docs/cli.md:210-224`

- [ ] Add a `--format` row to the `report` command's flag table, with a JSON example.

### Task 3: Choose the JSON shape

**Files:**
- Modify: `cmd/report/report.go:40-95`

The `Report` struct's field names are snake_case in the database layer and camelCase in
the HTTP layer, and either is a plausible wire shape for the CLI. Pick one and apply it
consistently.

- [ ] Add struct tags fixing the chosen casing.
- [ ] Assert the chosen casing in `report_test.go`.

# Fixture 01: hard trigger, one implementation task

> Retained historical routing case. Current `solo` means inline/sequential execution; `sdd` means phase-separated leaf assignments. Expected mode admits no reviewer, fixer or runtime campaign. See README.md for current use.

Expected mode: sdd
Isolates: Rule 2 (hard trigger) beating Rule 3's two-task floor.
Invocation: "Execute this plan."

**Goal:** Migrate the `sessions` table to store `expires_at` as a UTC timestamp instead
of a Unix integer.

## Global Constraints

- The migration runs once, forward only. There is no down-migration.
- Existing rows must retain their expiry instant exactly.

### Task 1: Migrate the sessions.expires_at column

**Files:**
- Create: `migrations/0042_sessions_expires_at_utc.sql`
- Modify: `internal/session/store.go:88-140`

- [ ] Write `migrations/0042_sessions_expires_at_utc.sql`: add `expires_at_utc TIMESTAMPTZ`,
      backfill it from `to_timestamp(expires_at)`, drop `expires_at`, rename.
- [ ] Update `store.go` to read and write the new column type.
- [ ] Run `go test ./internal/session/`.

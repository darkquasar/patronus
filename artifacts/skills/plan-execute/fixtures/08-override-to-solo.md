# Fixture 08: explicit solo override on a hard-trigger plan

Expected mode: solo
Isolates: Rule 1. The plan has a clear hard trigger, and the user's explicit request wins
outright. The record must say the mode was requested, not assessed, and it should note
the trigger it is overriding.
Invocation: "Run this solo."

**Goal:** Rotate the signing key used for session cookies.

## Global Constraints

- Sessions signed with the old key must keep validating until they expire.

### Task 1: Add the new key and dual-verify

**Files:**
- Modify: `internal/auth/sign.go:20-75`
- Test: `internal/auth/sign_test.go`

- [ ] Sign with the new key; verify against new then old.
- [ ] Assert a cookie signed with the old key still validates.

### Task 2: Retire the old key after the expiry window

**Files:**
- Modify: `internal/auth/sign.go:20-75`, `config/auth.yaml:12`

- [ ] Remove the old key from the verify set.
- [ ] Assert an old-key cookie is now rejected.

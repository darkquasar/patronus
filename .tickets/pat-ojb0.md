---
id: pat-ojb0
status: open
deps: [pat-mpn7]
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface, security]
---
# SECURITY: hash an archive-delivered binary instead of trusting its presence

Plan Task 11 — THE PRODUCTION SECURITY HOLE. ***CAN ONLY LAND AFTER TASK 10 (pat-?) — it is what BREAKS stubBinary***: 17 dummy bytes -> archive -> today SKIP (never hashed) -> after the fix HASH -> mismatch -> FETCH -> ~16 tests break. recipe.go:207-209 SKIPs an archive on MERE PRESENCE, unhashed: every re-run is unverified, so an attacker who writes to the dest once gets Patronus to launder the trojan as 'verified' forever — and gitleaks-guard EXECUTES one of these on EVERY COMMIT. The datum ALREADY EXISTS: apply.go:188-191 stamps PlacedSHA256, state.go:263-272 persists it, and the only readers are on the remove path. This READS THE HASH WE ALREADY RECORD. Files: internal/recipe/recipe.go, cmd/patronus/install.go, internal/recipe/recipe_test.go. NEVER weaken the sha check to keep a test green.

## Acceptance Criteria

A tampered binary at ~/.patronus/bin/<x> is DETECTED (classifyFetch -> FETCH, not SKIP); TestTamperedArchiveBinaryIsRefetched passes; a binary with NO recorded digest re-FETCHes


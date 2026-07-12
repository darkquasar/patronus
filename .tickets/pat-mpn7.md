---
id: pat-mpn7
status: open
deps: [pat-0y3f, pat-1gyd, pat-k1h1, pat-9v6i, pat-2rf3]
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Delete testdata/tk, serveBinaries, and stubBinary

Plan Task 10. PRECONDITION: grep -rn stubBinary cmd/ must be ZERO before starting — if it is not, a Class-A migration is unfinished; do NOT delete it out from under a caller. Deletes 47KB of vendored third-party bash. builtRegistry SURVIVES for the one Class-B test, but serves the catalog only, never a binary.

## Acceptance Criteria

grep -rn stubBinary cmd/ -> 0 hits; cmd/patronus/testdata/tk gone; no //go:embed of third-party bytes anywhere; suite green


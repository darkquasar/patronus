---
id: pat-mpn7
status: closed
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


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 10: Delete the vendored bytes and the stubs' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 10: Delete the vendored bytes and the stubs". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

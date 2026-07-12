---
id: pat-0y3f
status: closed
deps: [pat-il8m]
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Migrate integration_test.go's Class-A tests onto the fixture

Plan Task 4 — the TEMPLATE for tasks 5-7. CLASSIFY EVERY SITE FIRST: Class A (mechanism) -> rename to fixture names; Class B (catalog contents) -> DO NOT TOUCH. Drop stubBinary at each site (the fixture serves a real invented binary, so FETCH now runs for real). Files: cmd/patronus/integration_test.go.

## Acceptance Criteria

integration_test.go names zero real catalog items; go test -race ./... green; the classification is recorded in the commit message


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 4: Migrate integration_test.go's own Class-A tests' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 4: Migrate `integration_test.go`'s own Class-A tests". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

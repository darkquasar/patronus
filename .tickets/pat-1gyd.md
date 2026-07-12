---
id: pat-1gyd
status: open
deps: [pat-il8m, pat-0y3f]
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Migrate the golang/hardened/l1/core-consolidated profile tests (Class A)

Plan Task 5. Files: cmd/patronus/{golang_profile,hardened_profile,l1_profile,core_consolidated}_integration_test.go. Point them at fix-all / fix-extends / fix-flavoured. WARNING: if a test asserts a PRODUCT GUARANTEE about a real profile's contents, that is Class B — leave it on the real catalog and keep it off the fetch path.

## Acceptance Criteria

Each of the 4 files is a separate green commit carrying its classification; core_profile_integration_test.go is NOT among them


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 5: Migrate the profile integration tests (Class A)' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 5: Migrate the profile integration tests (Class A)". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

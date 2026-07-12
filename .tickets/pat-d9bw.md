---
id: pat-d9bw
status: open
deps: [pat-ojb0, pat-udo1]
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Verify the test-surface acceptance criteria against the artifacts

Plan Task 13. A green suite is a CLAIM. This task opens the artifacts: bump a pin and watch nothing break; break the Class-B guarantee and watch it fail; re-run the tamper proof. If the Class-B check passes with grilling REMOVED, the sweep silently converted a product guarantee into a tautology — the single worst outcome of this plan.

## Acceptance Criteria

Corrupting the pin in recipes/tk.yaml breaks ZERO tests (today ~30) — SHOWN, not asserted; and commenting grilling out of core.yaml still fails the Class-B test


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 13: Verify the acceptance criteria — against the artifacts, not the claims' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 13: Verify the acceptance criteria — against the artifacts, not the claims". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

---
id: pat-9v6i
status: open
deps: [pat-il8m, pat-0y3f]
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 2
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Retarget the hook + output-style baselines at the fixture catalog

Plan Task 7. Files: cmd/patronus/{hook,outputstyle}_integration_test.go. Class D — they ALREADY invent their items (smoke-hook, smoke-style); only their BASELINE build was the real catalog, dragging every real pin in for nothing. Keep the injected smoke-* items exactly as they are.

## Acceptance Criteria

TestHook* and TestOutputStyle* pass unchanged; neither builds the real catalog any more


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 7: Retarget hook_integration_test.go and outputstyle_integration_test.go' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 7: Retarget `hook_integration_test.go` and `outputstyle_integration_test.go`". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

---
id: pat-2uah
status: open
deps: []
links: []
created: 2026-07-12T04:46:52Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface, security]
---
# Deny-all fetcher in TestMain — make 'no network' structural, not conventional

Plan Task 3 (R-SEC). Files: cmd/patronus/main_test.go (new). No TestMain exists today — the enforcement point is free. Defaults at registry.go:21 / install.go:528 are LIVE HTTPFetcher{}: the suite is offline only because every test REMEMBERS withRemoteEnv. That is a convention, not a control. Land BEFORE the sweep touches 40 call sites.

## Acceptance Criteria

A test that forgets withRemoteEnv PANICS with the URL it tried to reach — proven by running the guard, not by a green suite


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 3: The deny-all TestMain (R-SEC)' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 3: The deny-all `TestMain` (R-SEC)". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

---
id: pat-2rf3
status: closed
deps: []
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Keep core_profile on the REAL catalog, but off the fetch path (Class B)

Plan Task 9. ***THE MOST DANGEROUS TASK IN THIS PLAN.*** core_profile_integration_test.go:20-25 (the coreSkills slice) asserts 'the core profile really ships grilling, tdd, ...' — a PRODUCT GUARANTEE. A fixture cannot express it: the real names ARE the assertion. Renaming them to fixture names would turn a real assertion into a tautology that passes forever while testing nothing — THE EXACT FAILURE THIS WHOLE RESEARCH EXISTS TO PREVENT. DO NOT RENAME. DO NOT point this file at fixtureRegistry. The ONLY change: stop it installing a binary (assert against the PLAN, not the placement), so it never fetches or hashes — otherwise the archive-SKIP fix turns it red and the 'fix' would be to weaken a sha check or vendor 15MB of gitleaks.

## Acceptance Criteria

coreSkills STILL holds the real names, AND commenting 'grilling' out of profiles/core.yaml makes TestCoreProfileClaude FAIL. If that check passes with grilling removed, the test has become a tautology — STOP.


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 9: core_profile_integration_test.go — the Class-B test that STAYS' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 9: `core_profile_integration_test.go` — the Class-B test that STAYS". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

---
id: pat-il5d
status: open
deps: [pat-ojb0]
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 2
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Release-note the archive re-verification; record the fork-PR decision in ci.yml

Plan Task 12. Task 11 CHANGES PRODUCTION BEHAVIOR — a previously-SKIPped binary may now re-FETCH. Users who hand-placed a binary need to know. And ci.yml:9 triggers on any pull_request incl. forks: safe today ONLY because permissions: contents: read and no secrets.*. Record that as a DECISION, not an accident.

## Acceptance Criteria

CHANGELOG names the behavior change (a hand-placed binary WILL be re-fetched); ci.yml carries the fork-PR decision as a comment


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 12: Release notes for the behavior change' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 12: Release notes for the behavior change". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

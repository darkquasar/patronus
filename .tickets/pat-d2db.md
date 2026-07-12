---
id: pat-d2db
status: open
deps: [pat-q360, pat-il8m]
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle, security]
---
# Wire the drift guard into patronus scan (two passes)

Plan Task 9 (R7). Files: cmd/patronus/{scan.go,scan_test.go}, internal/render/render.go. NOTE: runScan does not exist — add it (mirror runInstall, install_test.go:20-29). TWO PASSES, and the second IS the point: pass 1 (state -> disk) catches STALE/USER-EDITED/MISSING/ORPHANED; pass 2 (catalog -> disk) catches the UNMANAGED SHADOW — you CANNOT find it by walking state.json, because BY DEFINITION it has no state row. That is exactly how the stale skill hid. Do NOT hand-roll the deploy-path resolution: install already computes it — call it. Two implementations of 'where does this item land' is precisely the divergence this plan is about. Its test uses fixtureRegistry -> depends on the test-surface stream.

## Acceptance Criteria

patronus scan NAMES an unmanaged shadow file on disk — verified by planting one and seeing the verdict fire, not by a unit test


## Notes

**2026-07-12T04:49:02Z**

CROSS-STREAM DEPENDENCY: TestScanReportsDrift uses fixtureRegistry from the test-surface stream (pat-il8m). It is a Class-A test — it asserts Patronus's BEHAVIOR, not the catalog's contents — so it must land on the fixture, not the real catalog. If you must start before the fixture exists, use builtRegistry as a STOPGAP and migrate it when pat-il8m closes; do not leave it bound to the real catalog.

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 9: Wire the drift guard into patronus scan' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 9: Wire the drift guard into `patronus scan`". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

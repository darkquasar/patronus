---
id: pat-mdda
status: closed
deps: []
links: []
created: 2026-07-12T04:46:52Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Add fixtureCatalog — a temp-dir catalog whose pins are sha256(invented bytes)

Plan Task 1. Files: cmd/patronus/fixture_catalog_test.go (new). DiscoverRoot needs artifacts/ AND adapters/ as dirs; recipe key is 'deliver:' not 'delivery:'; description: is required. Additive — suite stays green.

## Acceptance Criteria

go test ./cmd/patronus/ -run TestFixtureCatalogBuilds passes; the fix-bin pin equals sha256 of bytes the test itself wrote


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 1: The fixture catalog' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 1: The fixture catalog". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

---
id: pat-il8m
status: closed
deps: [pat-mdda]
links: []
created: 2026-07-12T04:46:52Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Add fixtureRegistry; move serveBinaries out of serveTree

Plan Task 2. Files: cmd/patronus/fixture_catalog_test.go, cmd/patronus/integration_test.go. serveTree serves the CATALOG only; binaries become the caller's business. Ordering: build BEFORE withRemoteEnv (it t.Chdirs where DiscoverRoot fails by design).

## Acceptance Criteria

TestFixtureRegistryServesBothDeliveryShapes passes; go test -race ./... still green (builtRegistry keeps serving real binaries during the sweep)


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → 'Task 2: fixtureRegistry — build and serve the fixture' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md → "## Task 2: `fixtureRegistry` — build and serve the fixture". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

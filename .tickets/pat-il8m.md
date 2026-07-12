---
id: pat-il8m
status: open
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


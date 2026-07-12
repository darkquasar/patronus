---
id: pat-0y3f
status: open
deps: [pat-il8m]
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Migrate integration_test.go's Class-A tests onto the fixture

Plan Task 4 — the TEMPLATE for tasks 5-7. CLASSIFY EVERY SITE FIRST: Class A (mechanism) -> rename to fixture names; Class B (catalog contents) -> DO NOT TOUCH. Drop stubBinary at each site (the fixture serves a real invented binary, so FETCH now runs for real). Files: cmd/patronus/integration_test.go.

## Acceptance Criteria

integration_test.go names zero real catalog items; go test -race ./... green; the classification is recorded in the commit message


---
id: pat-1gyd
status: open
deps: [pat-il8m, pat-0y3f]
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Migrate the golang/hardened/l1/core-consolidated profile tests (Class A)

Plan Task 5. Files: cmd/patronus/{golang_profile,hardened_profile,l1_profile,core_consolidated}_integration_test.go. Point them at fix-all / fix-extends / fix-flavoured. WARNING: if a test asserts a PRODUCT GUARANTEE about a real profile's contents, that is Class B — leave it on the real catalog and keep it off the fetch path.

## Acceptance Criteria

Each of the 4 files is a separate green commit carrying its classification; core_profile_integration_test.go is NOT among them


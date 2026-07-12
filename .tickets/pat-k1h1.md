---
id: pat-k1h1
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
# Migrate update_test, orchestration, reground_hooks (Class A)

Plan Task 6. Files: cmd/patronus/{update_test,orchestration_integration_test,reground_hooks_integration_test}.go. update_test builds+serves by hand (:18-21) with cwd = package dir, so DiscoverRoot finds the REAL repo — retarget with t.Chdir(fixtureCatalog(t)) before the build.

## Acceptance Criteria

All three green on the fixture; update_test's explicit runBuild is retargeted at the fixture root, preserving build-before-withRemoteEnv


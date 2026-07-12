---
id: pat-d9bw
status: open
deps: [pat-ojb0, pat-udo1]
links: []
created: 2026-07-12T04:47:38Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Verify the test-surface acceptance criteria against the artifacts

Plan Task 13. A green suite is a CLAIM. This task opens the artifacts: bump a pin and watch nothing break; break the Class-B guarantee and watch it fail; re-run the tamper proof. If the Class-B check passes with grilling REMOVED, the sweep silently converted a product guarantee into a tautology — the single worst outcome of this plan.

## Acceptance Criteria

Corrupting the pin in recipes/tk.yaml breaks ZERO tests (today ~30) — SHOWN, not asserted; and commenting grilling out of core.yaml still fails the Class-B test


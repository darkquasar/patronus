---
id: pat-zr8z
status: open
deps: []
links: []
created: 2026-07-12T04:46:52Z
type: epic
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
tags: [test-surface]
---
# Test surface — unbind the suite from the catalog's contents

Stream 'test-surface' of docs/specs/01-lifecycle-and-test-surface. Plan: test-surface-plan.md (13 tasks). STRICT ORDER: fixture catalog -> migrate Class-A file-by-file -> delete stubs -> ONLY THEN close the archive-SKIP hole (step 3 is what BREAKS stubBinary). Groups only; tk ready reads deps, never parent.

## Acceptance Criteria

Bumping a pin in recipes/tk.yaml breaks 0 tests (today: ~30); a tampered archive binary at the dest is DETECTED


## Notes

**2026-07-12T04:49:12Z**

NOT ACTIONABLE — this is a GROUPING epic. It appears in 'tk ready' because ready/blocked read ONLY deps, never parent (an epic has no deps, so it always looks ready). That inertness is documented, not a bug: epics group and display; tk dep orders. Work the children.

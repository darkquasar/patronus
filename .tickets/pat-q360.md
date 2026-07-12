---
id: pat-q360
status: open
deps: []
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle, security]
---
# Drift guard: classify a deployed file against its recorded sha AND its source

Plan Task 8 (R7). Files: internal/drift/{drift.go,drift_test.go} (new). CORRECTED PREMISE: Patronus ALREADY records a sha for every deployed file (state.go:52-58, whose comment names this exact use VERBATIM) and ALREADY has a function literally called driftsFromChecksum (remove/compute.go:237) — wired into remove and NOTHING ELSE. This is NOT 'start recording hashes'. It is 'READ THE HASH YOU ALREADY RECORD'. Do not add a hashing pass. The verdict that MATTERS is UNMANAGED SHADOW: the stale team-research skill was NEVER IN state.json (placed by hand or another tool), so a check that walks state.json alone reports nothing wrong while the agent runs a dead protocol.

## Acceptance Criteria

go test ./internal/drift/ green, with a case for EACH of OK / STALE / USER-EDITED / UNMANAGED-SHADOW / ORPHANED-STATE / MISSING


---
id: pat-q360
status: closed
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


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 8: The drift guard (R7)' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 8: The drift guard (R7)". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-12T10:17:20Z**

DONE. internal/drift/{drift.go,drift_test.go}. 8 subtests green (the 6 required verdicts + 2 ordering cases I added: absent+unrecorded->OK, edited+no-source->USER-EDITED). Premise verified against the code before writing: FileState.Checksum (state.go:52-58) and driftsFromChecksum (remove/compute.go:237) both already existed; this reads them, it does not add a hashing pass. DEVIATION from the plan's Classify: USER-EDITED is checked BEFORE ORPHANED-STATE (the plan had !hasSource first, which would report a user-edited file whose item left the catalog as ORPHANED-STATE and hide the at-risk bytes). Mutation-tested: forcing 'recorded=="" -> OK' (the production state.json-blindness bug) fails the UNMANAGED-SHADOW case; the plan's original ordering fails the edited+no-source case. Commit efc8872.

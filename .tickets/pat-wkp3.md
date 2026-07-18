---
id: pat-wkp3
status: open
deps: []
links: []
created: 2026-07-18T11:28:20Z
type: task
priority: 2
assignee: darkquasar
tags: [lifecycle, security]
---
# Drift guard: check composed/APPEND instruction files (multi-source fold)


## Notes

**2026-07-18T11:28:20Z**

Pass 1 byte-compares a deployed file against ONE source's would-be After bytes. Instructions (CLAUDE.md/AGENTS.md) are APPEND-composed: a fold of MANY sources into one file, delimited by <!-- patronus:start <name> --> markers. So a single-source compare never matches, and composed files are currently NOT drift-checked. Fix direction: reconcile PER MARKED SECTION (parse the fenced blocks, compare each against its source's contribution), reusing internal/adapter's appendSection composition. Found while wiring pat-d2db. state.json's FileState.Section field (state.go) already records the section name per APPEND — read it, like the checksum.

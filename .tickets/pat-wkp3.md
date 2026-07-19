---
id: pat-wkp3
status: closed
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

**2026-07-19T00:01:55Z**

DONE. Composed/APPEND instruction files (CLAUDE.md/AGENTS.md) now drift-checked PER SECTION. Added drift.ClassifySection (OK/STALE/MISSING/ORPHANED-STATE per fenced block) + wired reconcileDrift pass 1b: APPEND state rows -> sectionRows (keyed path+section, not path — many share a path); their path removed from whole-file rows; on-disk body via adapter.SectionBody, source body via wouldSection (each per-name diff's Section.Body). TWO real bugs found & fixed while wiring: (1) capture Section.Body even on SKIP — plan.Compute downgrades an unchanged APPEND to SKIP but keeps Section; gating on Action==Append gave false ORPHANED for every already-installed section. (2) ClassifySection trims trailing newlines both sides — buildBlock TrimRights + SectionBody Trims, so a raw source body's trailing \n the fence drops read as STALE on EVERY section. Tests: TestClassifySection (5 verdicts incl the newline case) + TestScanReportsComposedSectionDrift (install 2 instructions into 1 CLAUDE.md, move ONLY one's source, assert per-section STALE for it + silence for the other + no false USER-EDITED). VERIFIED BY BREAKING: disabled the section capture -> composed test FAILED; restored -> PASS. Four gates green. BONUS: the guard now catches the real pat-t5a0 gap live — agent-rules 'antipattern' heuristic is in source but not the deployed CLAUDE.md -> reported STALE.

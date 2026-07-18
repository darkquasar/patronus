---
id: pat-2dgf
status: closed
deps: [pat-06p3, pat-xdxv]
links: []
created: 2026-07-12T04:48:22Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# Kill tasks.md — tk is the single work-graph

Plan Task 2 (R2). Files: artifacts/skills/team-implement/{SKILL.md,TASKS-TEMPLATE.md(delete),TEAMMATE-TEMPLATE.md,PROVENANCE-GUIDE.md}, brainstorming/SKILL.md:140, team-research/SKILL.md:14,185. TWO LOSSES that must be STATED, not lost silently: (1) ids go opaque (pat-a1b2, no --id flag) — the concern survives only via --tags; (2) tk query reads ONLY frontmatter, so the file list is NOT machine-queryable and the core invariant becomes a HUMAN READING STEP. An invariant nobody checks is not an invariant.

## Acceptance Criteria

grep -rn 'tasks.md' artifacts/ -> 0 hits; TASKS-TEMPLATE.md deleted; AND the skill explicitly states that 'no two teammates edit the same file' has DEGRADED into a Team-Lead reading step


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 2: Kill tasks.md' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 2: Kill `tasks.md`". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-12T04:59:27Z**

SCOPE ADDED (2026-07-12): team-implement's Phase-2 seeding recipe must mint a RESOLVABLE pointer — --external-ref docs/specs/NN-slug/<stream>-plan.md (the FILE, not the folder: ADR-0003 guarantees a folder holds many plans, so a folder ref is ambiguous BY CONSTRUCTION), plus a 'PLAN: <file> → <verbatim section heading>' line in -d, plus the note that docs/specs/ is gitignored.

**2026-07-18T23:36:48Z**

DONE (Task 2). tasks.md is dead: Phase 2 'Check for tasks.md' -> 'Seed the tk work-graph' (one epic to group, one task per plan task, tk dep for order, --external-ref names the PLAN FILE + -d names the SECTION HEADING verbatim + GITIGNORED warning). Phase 3 reads from tk graph. PROVENANCE-GUIDE: task ids -> ticket ids (pat-a1b2), research/<domain> layout -> docs/specs/NN-slug/, prefixed names. TASKS-TEMPLATE.md deleted (git rm). TWO LOSSES stated in the skill: (1) ids opaque, no --id flag, concern lives in --tags; (2) tk query reads only frontmatter so 'no two teammates edit the same file' DEGRADES from a lookup to a Team-Lead READING step. Acceptance: grep tasks.md|TASKS-TEMPLATE across artifacts/ = 0; template gone; invariant + 'reading step' both stated. Clean-body rule applied.

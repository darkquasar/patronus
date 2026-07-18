---
id: pat-0y6h
status: closed
deps: [pat-d2db]
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# Fix the .agents/ copy and verify the lifecycle acceptance criteria

Plan Task 10. Files: .agents/skills/plan-review/SKILL.md:61 ('plan -> beads'; the source says 'plan -> ticket' since 793c06b). Then diff every .agents/ skill against its source and re-sync. The acceptance criterion that MATTERS: the guard's whole justification is that it catches the originating defect. If it does not, the guard is a claim, not a control.

## Acceptance Criteria

grep -rn beads .agents/ -> 0; AND patronus scan demonstrably WOULD HAVE CAUGHT the stale team-research skill that started this branch


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 10: Fix the .agents/ copy and verify every acceptance criterion' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 10: Fix the `.agents/` copy and verify every acceptance criterion". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-18T23:43:37Z**

DONE (Task 10, its OWN criterion). (1) .agents/skills/plan-review/SKILL.md re-synced from source: 'plan -> beads' -> 'plan -> ticket', plus the two other drifts my Task 3/4 edits introduced (prefixed names line 33, the tk-mirror Next block). grep beads .agents/ = 0; diff vs source = IN SYNC. .agents/ is the codex PROJECT deploy path (internal/adapter/builtin/codex.yaml: project: .agents/skills/{name}/SKILL.md), hand-committed in 793c06b. (2) THE ACCEPTANCE THAT MATTERS: patronus scan demonstrably catches the originating defect. Installed team-research/team-implement still carry TeamCreate/team_name (2 and 3 hits); source=0; scan reports STALE (in-repo .claude deploy) + ORPHANED-STATE (global ~/.claude). The guard is a control, not a claim. CAVEAT: the full Task-10 capstone battery is not all-green yet — 'If your platform supports' hedges = 3 until pat-uxdx (Task 5) lands, and the plan->tk mirror shape is pat-nlf8 (Task 7). Those are separate ready tickets; pat-0y6h's own two-part criterion is met. Re-deploy (patronus install ... --tool claude --global) is how the STALE/ORPHANED installed copies reconcile; NOT hand-edited.

---
id: pat-0y6h
status: open
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


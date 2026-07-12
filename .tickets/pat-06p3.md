---
id: pat-06p3
status: open
deps: []
links: []
created: 2026-07-12T04:48:22Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# The ticket instruction must tell the truth about tk's surface

Plan Task 1 (R3). Files: artifacts/instructions/ticket/INSTRUCTIONS.md (:33-35, :166), artifacts/skills/writing-plans/SKILL.md (:166). The instruction STATES A FALSEHOOD: 'the graph is flat, no epic to hang them from' — tk HAS -t epic and --parent. The TRUE statement: they are grouping/display only; ready/blocked read only deps. And the port dropped the nouns: it only ever demonstrates 'tk create title', so -p is never taught while tk ready SORTS BY PRIORITY — every ticket lands at the default 2 and the ordering signal is DEAD. A live regression against beads, which did teach -p.

## Acceptance Criteria

grep -rn 'no epic|graph is flat' artifacts/ -> 0 hits; AND running the taught 'tk create' with -p/--acceptance/--tags/--external-ref actually works, with -p 1 sorting above -p 2 in tk ready


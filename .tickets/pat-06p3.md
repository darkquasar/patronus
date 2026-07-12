---
id: pat-06p3
status: closed
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


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 1: Tell the truth about tk's command surface' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 1: Tell the truth about tk's command surface". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-12T04:59:27Z**

SCOPE ADDED (2026-07-12): the create surface must also teach a pointer that RESOLVES — --external-ref names the FILE the work is specified in (never a folder holding several candidates), -d names the SECTION verbatim, and the ticket says out loud when the target is in a gitignored dir. Rationale: a ticket whose reference an agent cannot follow is a ticket nobody can do.

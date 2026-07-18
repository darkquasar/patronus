---
id: pat-jl1r
status: closed
deps: []
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# Team protocol: the lead PULLS; native isolation is USED, not banned; research needs no worktree

Plan Task 6 (R6). Files: artifacts/skills/team-{implement,research}/{SKILL.md,*-TEMPLATE.md}. DO NOT MODIFY patronus.yaml. THREE empirical corrections: (1) team-implement:79-81 forbids isolation:worktree 'so you retain ownership of the branches to merge' — EMPIRICALLY FALSE: the agent's commit is reachable and 'git merge worktree-agent-<id> --no-ff' WORKS. It bans the right tool for a wrong reason. (2) THE LEAD MUST PULL — n=7, reproducible: EVERY subagent went idle without delivering, including two under a contract saying 'your findings MUST be the text of your final message'. Prompt contracts enforce PROHIBITIONS; they CANNOT enforce DELIVERY. Assign a path, then GO READ IT. Completion = the artifact exists. (3) team-research needs NO worktree — DELETE the phase, don't fix it. DO NOT make the skill 'capability-detecting': markdown cannot execute if(hasNativeWorktrees). Patronus already flavours per target at BUILD time. FLAVOUR IT; DON'T BRANCH IN IT.

## Acceptance Criteria

grep -rn 'TeamCreate|TeamDelete|team_name|shutdown_request' artifacts/skills/ -> 0; grep -rn 'Monitor .TaskList' -> 0; team-research has NO worktree phase; patronus.yaml targets: [claude] UNCHANGED


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 6: The team protocol — flavour it, and make the lead PULL' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 6: The team protocol — flavour it, and make the lead PULL". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-18T22:05:35Z**

DONE (pat-jl1r/Task 6). team-{implement,research} SKILL.md + both templates rewritten: native isolation:worktree (ban reversed), lead PULLS (completion=artifact, patience clause), team-research has no worktree phase, Monitor-TaskList removed, shutdown handshake removed. User steer applied: bodies are POSITIVE present-tense for a fresh LLM — all backward-referencing archaeology ('do not pass deprecated team_name', 'this reverses an earlier ban', 'used to mandate') was STRIPPED, not just reworded, because a fresh session never knew the prior state. Acceptance: grep for TeamCreate|TeamDelete|team_name|shutdown_request AND Monitor TaskList both = 0; patronus.yaml targets:[claude] UNCHANGED; suite green. Codex flavour filed as pat-7i2i.

**2026-07-18T22:05:35Z**

RESIDUE FOR pat-2dgf (Task 2, blocked on pat-xdxv): Phase 0 + Phase 2 of team-implement/SKILL.md still reference tasks.md / completeness: / TASKS-TEMPLATE.md. I did NOT touch those — that full kill is pat-2dgf's verifiable outcome. My Task-6 edits use the tk seam vocabulary the plan's Task 6 gives (tk ready -T <concern>, seed the tk graph), so TEAMMATE-TEMPLATE + Coordination Protocol are already tk-based; pat-2dgf must reconcile Phase 0/2 to match and delete TASKS-TEMPLATE.md.

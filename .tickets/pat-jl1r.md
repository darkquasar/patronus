---
id: pat-jl1r
status: open
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


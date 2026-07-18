---
id: pat-7i2i
status: open
deps: [pat-jl1r]
links: []
created: 2026-07-18T22:05:05Z
type: task
priority: 3
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md
parent: pat-54y7
tags: [lifecycle]
---
# Author the codex flavour of team-implement/team-research

One protocol, two flavours. The Claude flavour uses the Agent tool's native isolation:worktree unconditionally (done in pat-jl1r). The Codex flavour must hand-roll it with git worktree + codex exec + a supervisor. Do NOT branch on capability inside one skill — a markdown file cannot execute if(hasNativeWorktrees), and Patronus already flavours per target at BUILD time (patronus.yaml targets:). PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md -> 'Task 6' Step 6. NOTE: docs/specs/ is GITIGNORED.

## Acceptance Criteria

artifacts/skills/team-implement-codex/patronus.yaml has targets: [codex] and uses explicit git worktree; profiles/core.yaml wires team-implement@codex


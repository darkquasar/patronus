---
id: pat-4v2t
status: open
deps: [pat-kwwi]
links: []
created: 2026-07-19T01:56:53Z
type: task
priority: 3
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md
parent: pat-7i2i
tags: [codex-parity]
---
# Author team-implement-codex artifact

The codex flavour of team-implement. Hand-roll isolation with git worktree; spawn implementers via codex exec; a supervisor PULLS (completion = the committed branch, per pat-jl1r's protocol). One protocol, two flavours — do NOT branch on capability in prose. NOTE: docs/specs/ is GITIGNORED.

## Acceptance Criteria

artifacts/skills/team-implement-codex/patronus.yaml has targets: [codex]; SKILL.md uses explicit git worktree + codex exec + a supervisor (not the Agent tool's native isolation)


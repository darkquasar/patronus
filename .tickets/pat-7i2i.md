---
id: pat-7i2i
status: open
deps: []
links: []
created: 2026-07-18T22:05:05Z
type: epic
priority: 3
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md
tags: [codex-flavour]
---
# Codex flavour of the team skills (one protocol, two flavours)

EPIC. The Claude flavour uses the Agent tool's native isolation:worktree unconditionally (shipped in pat-jl1r). The Codex flavour hand-rolls the same protocol with `git worktree` + `codex exec` + a supervisor. The rule that holds across both: do NOT branch on capability inside one skill — a markdown file cannot execute if(hasNativeWorktrees), and Patronus already flavours per target at BUILD time (patronus.yaml targets:). So this is a SEPARATE codex-flavoured artifact per skill, wired per target.

Grouped children (see `tk dep tree pat-7i2i`), each a single verifiable outcome:
- team-implement-codex artifact
- team-research-codex artifact
- profile wiring (team-{implement,research}@codex)

Deferred deliberately by the lifecycle plan (Task 6 Step 6: "file the Codex flavour as a ticket — do not build it here"). A fresh session should grill the user on the codex spawn model (codex exec + worktree supervisor) before authoring. PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md -> 'Task 6' Step 6. NOTE: docs/specs/ is GITIGNORED.


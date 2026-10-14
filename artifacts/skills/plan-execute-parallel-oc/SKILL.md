---
name: plan-execute-parallel-oc
description: "OpenCode only: Use when executing approved independent plan tasks through native Task subagents in explicitly allocated worktrees. CLI workers require user choice."
---

# Native parallel execution in OpenCode

Prefer OpenCode's native Task tool for work it can perform. No CLI supervisor is
a prerequisite. A Task subagent may share the lead cwd; it does not automatically
create or enforce an isolated worktree. Never run concurrent writers in the lead
checkout, even on disjoint files.

Read the approved plan/spec, current source/base/head, effective instructions and
latest handoff. Reuse valid scope approval; this skill grants no additional Git,
integration, permission, installation, publication or cleanup authority.

## Decompose and prepare

Admit independent deliverables with disjoint paths and accepted input contracts.
Distinguish authoring dependencies from integration/verification dependencies.
If the plan has no writer-ready parallel split, ask whether the user wants the
first wave decomposed before writers start, unless already authorized. Keep
briefs and ownership in the existing plan/handoff, not a new scheduler or ledger.

While current writers work, use native read-only `@patronus-researcher-oc` Task
subagents under `{skillsDir}/dispatching-parallel-agents-oc/SKILL.md` to prepare
independent next-wave briefs where discovery is useful. Scouts use stable read
inputs, return complete text and never write source/reports, run builds or
delegate. Check briefs against accepted outputs, reconcile changed seams and
launch dependent writers only after actual acceptance barriers clear. Respect
available slots and project resource limits; preparation creates no extra
review gate or root-budget form.

## Allocate and launch native writers

Inspect actual Task availability, the installed `@patronus-writer-oc` role and
its permissions, completion and stop controls. Read
`{skillsDir}/using-git-worktrees-oc/SKILL.md` under the Git-allocation grant.
Use one allocated root per writer with disjoint path claims. Explicitly carry
accepted dirty/untracked predecessor inputs into a base worktree; do not hide,
stash or commit owner work merely to create a baseline.

Where native allocation is unavailable, authorized Git worktree creation or the
bundled allocation-only script may prepare roots:
`python3 {skillDir}/scripts/prepare-worktrees.py ALLOCATION.json`.
Read `{skillDir}/references/worktree-allocation.md` only when using the script.
This setup needs no supervisor installation, launches no agents and does not
activate CLI mode. A leaf may perform
only an exact coordinator-approved allocation, not pick new roots or spawn work.

Give each Task writer the absolute allocated root/base, owned/excluded paths,
predecessors, strategy, checks and return format. The writer verifies that root
before writing and directs tools explicitly to it. If Task tools or permissions
cannot target the assigned worktree, report that exact limit before parallel
writes; do not claim a prompt changes cwd or silently widen permissions.
Read-only native assistance remains available within its grant. No headless
`opencode run`, inline permission override or CLI worker is substituted.

Workers are leaves with no delegation, shared-progress writes, commits,
integration or cleanup. Run TDD or focused postchecks from the task brief and
serialize heavy operations under the project resource lock. Return complete
findings through Task delivery, not mandatory supervisor result JSON.

## Settle and integrate

Read every final response and actual check result. Confirm native completion or
stop, including any background jobs, before retry, capture or integration.
Unknown liveness retains allocations and blocks those actions. Check complete
tracked/untracked patches, ownership and input hashes, then apply accepted
patches serially under the integration grant. Drift or conflicts stop the affected
integration; never discard dirty source. Fresh composed checks and the planned
independent whole-change review remain, with at most one bounded batched correction/disposition. Fixers are leaves and
cannot launch reviewers; changed interfaces, safety invariants or acceptance
scope return to the owner. Earlier reviews require a named changed safety or
compatibility invariant and its acceptance point before dependent consumption;
they never replace aggregate review.
No automatic per-task review wave, commit, push or worktree removal.

Worktree ownership and OpenCode permissions are cooperative controls, not an OS
sandbox. Check the actual tools and permissions for the selected run.

## Explicitly chosen CLI automation

Only if the user selects CLI worker mode, read
`{skillsDir}/isolated-writer-supervisor/SKILL.md` and its mode gate, then
`{skillDir}/references/cli-workers.md` for OpenCode-specific prechecks and argv.
The shared helper is optional. No failed-native-route fallback, automatic model/
provider change or cross-harness delegation is authorized.

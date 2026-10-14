---
name: plan-execute-parallel-cx
description: "Codex only: Use when executing approved, independent plan tasks with native subagents assigned isolated worktrees. No CLI supervisor is required."
---

# Parallel execution in Codex

Use Codex's available native subagent tools for work the harness can perform
natively. Do not invoke `codex exec`, another harness or
`isolated-writer-supervisor` to replace native delegation. CLI orchestration is
a separate, explicitly user-selected mode, not this skill's prerequisite or an
automatic fallback. Native subagents do not depend on a subprocess supervisor.

Read the approved plan/spec, effective instructions, current source/base/head and
latest handoff. Existing scope approval remains valid. This skill grants no
additional Git mutations, integration, deployment, publication or cleanup.

## Split by actual dependencies

Parallelize independent deliverables, not merely numbered tasks. Distinguish an
input needed to start authoring from an output needed to integrate or verify.
For example, a registry change and skill-package renames can be authored in
parallel in separate worktrees, then joined before consumer and catalog checks.
Keep common files and final integration with one owner. Use the existing plan or
handoff for ownership and progress; do not add a scheduler, request JSON, result
schema, duplicated tracker or per-task review wave.

If the plan lacks writer-ready, non-overlapping tasks, do not invent an execution
split or simply refuse parallelism. Explain the missing decomposition and ask
whether the user wants the first wave decomposed before writers start. Reuse an
existing explicit decomposition grant. Within that grant, use read-only native
scouts to propose task briefs from approved requirements, then admit only checked
claims and accepted interfaces. Keep the decomposition in the existing plan.

## Prepare the next wave while this one writes

Once the first wave is admitted, overlap its writers with read-only native scouts
for independent next-wave questions, through
`{skillsDir}/dispatching-parallel-agents-cx/SKILL.md`. Give each scout a stable
revision/read snapshot, relevant requirements, expected interface and one brief
to return. No source/report writes, builds, environment mutation or delegation.
Do not scout work whose brief is already sufficient.

The coordinator checks returned briefs and reconciles ownership, anchors and
assumptions against accepted current-wave outputs. Preparation may overlap
implementation; dependent writers start only after their actual input/acceptance
barriers clear. Re-scout changed seams only. Continue this pipeline while useful
independent work and native capacity remain, leaving capacity for preparation
and integration rather than filling every slot with writers. Existing host and
resource limits still apply. This is one plan and handoff, not another scheduler,
progress ledger, root budget or automatic review wave.

## Allocate isolated writers

Inspect the native launch, completion/wait and stop controls actually exposed in
the session. Use the current model/provider and existing authentication; do not
change engines. Report missing capabilities precisely.

Under the approved Git grant, allocate one worktree per writer at the agreed
base outside the lead checkout, following
`{skillsDir}/using-git-worktrees-cx/SKILL.md`. Preserve dirty and untracked
predecessor inputs: a clean base worktree omits them. Explicitly carry required
accepted inputs into the allocated roots and bind their hashes, or report a
missing input. Never stash, commit or discard owner changes to manufacture a
baseline.

Give each native worker its absolute allocated root, base, owned paths,
predecessors, task and acceptance checks, applicable skills, existing resource
limits and return format. The worker verifies the allocated root before its first
write and uses that root explicitly for file and shell operations. The native
tool may begin in the lead cwd; prompting is cooperative, not tool-enforced
isolation. If tools cannot target the assigned root, do not admit parallel
writers. Read-only native assistance may still be useful within its grant.

Workers are leaves: no nested delegation, new CLI agent sessions, shared progress
writes, commits, integration or cleanup. They write only owned files. Never run
concurrent writers in the lead checkout, even with disjoint paths.

When native worktree creation is unavailable, use authorized Git allocation or
the bundled allocation-only script:
`python3 {skillDir}/scripts/prepare-worktrees.py ALLOCATION.json`.
Read `{skillDir}/references/worktree-allocation.md` only when using this script.
The coordinator approves exact roots/base/claims first. A leaf may carry out that
specific allocation, not choose new allocations or launch agents. This utility
creates worktrees only; native launch/wait/stop remain with the coordinator.
No supervisor installation, CLI skill or worker argv/result contract is needed.
Missing optional allocation tooling does not itself block a usable native/Git
allocation path, and cannot authorize CLI worker launch.

## Collect, check and integrate

Launch only ready independent tasks within the host and approved resource limits.
Keep critical-TDD and focused-postcheck strategies from their briefs; serialize
resource-heavy checks under the existing project lock. Capture command results
and complete final worker responses. A finished message alone is not acceptance.

Use native wait/stop controls. Before retry, capture or integration, confirm the
worker and any started background jobs have stopped. Unknown liveness retains
allocations and blocks those actions. No automatic retry or relaunch.

Check each settled root's base/head, complete tracked/untracked diff, file
ownership, required-input hashes and actual tests. Preserve complete patches and
return evidence in the existing handoff. Do not manufacture JSON receipts or
attestation files for native workers. A worker claim of green tests does not
replace reading the command results.

With the separate integration grant, apply accepted patches serially to the
checked integration state. Preserve dirty predecessor work; on drift, conflicting
patches or unexpected edits, stop the affected integration and report the exact
problem. Run fresh composed checks and the planned independent whole-change review,
with at most one bounded batched correction/disposition. Fixers are leaves and
cannot launch reviewers; changed interfaces, safety invariants or acceptance
scope return to the owner. Earlier reviews require a named changed safety or
compatibility invariant and its acceptance point before dependent consumption;
they never replace aggregate review. Do not add reviews simply because tasks
were delegated. Commits, push and worktree removal remain separate grants.

## Limits and reporting

Worktrees and ownership checks provide cooperative isolation, not an OS sandbox.
Native tools do not automatically allocate or enforce separate roots. Report
observed launch, settlement, capture and verification behavior for this run.

If the native route is unavailable, retain evidence and ask for direction. Do not
silently substitute CLI workers or serial execution. If the user explicitly
chooses subprocess automation, read the installed
`{skillsDir}/isolated-writer-supervisor/SKILL.md` and its mode gate under that
separate grant. It is optional and must not be loaded for ordinary native work.

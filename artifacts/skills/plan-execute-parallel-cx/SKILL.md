---
name: plan-execute-parallel-cx
description: "Codex only: run disjoint isolated writers through the bundled supervisor helper. Native Codex worker qualification is runtime-pending (Partial)."
---

# Parallel execution in Codex

This skill ships a bundled supervisor helper, `{skillDir}/scripts/supervisor.py`
(Python 3 standard library, POSIX only). Its isolation, timeout, ownership and
integration rules are exercised by the bundled fake-worker suite
`{skillDir}/fixtures/test_supervisor.py`, which drives real local processes in
invented scratch repositories. Native Codex workers under the supervisor are
**Partial**: runtime-pending until a separately authorized native qualification.
Native Codex workers do not establish write isolation by themselves; only the
supervisor's per-attempt worktrees do.
Never run concurrent writers in the lead checkout.

Read the approved spec/plan, `meta.yaml`, research, effective project instructions,
root/base/head, assigned files, prior outputs and stage/Git grants. Confirm one
research effort in `docs/specs/NN-slug/`, one shared synthesis and one spec/plan
pair per stream. File presence is not execution authorization.

## Request binding

Write a request matching `{skillDir}/references/request.schema.json` and check it
with `python3 {skillDir}/scripts/supervisor.py validate REQUEST.json` before any
launch. The request binds the recorded base commit, the task brief and its hash,
the evidence directory, a worktree root outside the lead checkout, maximum
concurrency and, per task: worker argv, exclusive owned paths, timeout, grace
period, bounded retry budget (`maxAttempts`, default one) and predecessors.
Overlapping claims, escaping or `.git` paths, unknown predecessors, cycles and a
reused run identity are rejected before anything is created.

## Writers

`python3 {skillDir}/scripts/supervisor.py run REQUEST.json` starts every attempt in
a new worktree and branch from the recorded base, never the lead checkout. The
worker receives its bindings and a supervisor-owned result path through
`PATRONUS_SUPERVISOR_*` variables and must write a result whose run, task,
attempt, root and base match. Independent writers may overlap up to the
concurrency limit. A dependent writer waits until its predecessor is verified and
captured, then receives that captured patch applied in its fresh worktree, with
the input manifest at `PATRONUS_SUPERVISOR_INPUTS`.

On timeout the supervisor signals the worker's process group with SIGTERM, waits
the bounded grace period, escalates to SIGKILL and must confirm stop before
validation, retry or capture. A descendant that outlives its worker is stopped the
same way and the attempt is rejected. Unknown stop state (a surviving member, or a
Linux process still working inside the worktree) blocks the run and preserves all
evidence; nothing further launches. Supervisor errors or SIGINT/SIGTERM interruption
cancel admission and attempt bounded stop/settlement of every started worker before
exit. Concurrent workers cannot capture after the run is blocked. Unknown state
blocks capture, integration, retry and cleanup; retain the worktrees, logs and
result evidence for an explicit owner decision, never infer permission to replay.

Acceptance rejects a missing or malformed result, binding mismatch, nonzero exit,
worker-reported failure, HEAD or branch drift (including commits), non-owned
changes, staged residue, ignored-file residue, unreported changes and claimed but
absent changes. Intended uncommitted edits inside the owned paths are the capture,
not a dirty result. Accepted attempts are captured as exact binary patches with
patch and per-file SHA-256 after confirmed settlement; rejected settled attempts
keep an evidence patch. A retry is explicit and bounded, uses a fresh worktree from
the recorded base and retains every failed attempt. The supervisor never deletes a
worktree or branch; cleanup is a separate grant after capture is verified.

## Integration

Integration is disabled by default. It needs `integration.enabled`, the checked
integration head and worktree in the request, plus a separate explicit grant
passed as `integrate REQUEST.json --grant-integration`. Captures apply serially in
dependency order against the latest checked integration head; head drift, a dirty
or drifted tree, a changed capture hash or a patch that does not apply cleanly
(`merge-conflict`) stops before mutation. It never commits. Commit, push, discard
and worktree removal remain separate grants. No automatic cleanup follows from a
completion notification.

## Limits

Ownership is cooperative, enforced from Git's view of the worktree. It is not an
OS filesystem sandbox: a process can write outside its worktree or through a
symlink without detection, and a descendant that leaves the process group and the
worktree directory is not observable. Process discovery uses `/proc` on Linux and
a process-group probe elsewhere; Windows is blocked. Conflict detection is
conservative (`git apply --check`, no three-way merge). SIGKILL, host loss or another
uncatchable hard kill cannot run settlement; retained evidence alone then does not
prove workers stopped. Process-group/cwd observation does not detect arbitrary
escaped processes outside both boundaries. Only the invented Linux fixtures are
observed here, not native Codex or qualification of other platforms. Fake-worker tests do not
qualify native Codex workers, their authentication or their sandbox.

## Authorized alternative

If the supervisor, Git, Python or native workers are unavailable or auth-blocked,
stop the lane as blocked with partial evidence retained, report the Partial limit
and ask the lead/owner. Serial execution via `{skillsDir}/plan-execute-cx/SKILL.md`
needs explicit lead authorization; it is never a failed-engine fallback. Never
change model, provider, protocol or isolation to make progress.

Read-only parallel investigation may be separately authorized through
`{skillsDir}/dispatching-parallel-agents-cx/SKILL.md`. It grants no writer access.

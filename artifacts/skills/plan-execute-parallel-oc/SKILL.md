---
name: plan-execute-parallel-oc
description: "OpenCode only: run disjoint isolated writers through the required shared isolated-writer-supervisor with headless `opencode run` workers. Native OpenCode worker qualification is runtime-pending (Partial)."
---

# Parallel execution in OpenCode

This skill uses the required shared supervisor helper
`{skillsDir}/isolated-writer-supervisor/scripts/supervisor.py` (Python 3 standard
library, POSIX only). Its isolation, timeout, ownership and integration rules are
exercised by that skill's fake-worker suite
`{skillsDir}/isolated-writer-supervisor/fixtures/test_supervisor.py`, which drives
real local processes in invented scratch repositories. Native OpenCode workers
under the supervisor are **Partial**: runtime-pending until a separately
authorized native qualification. Native OpenCode subagents (Task tool) share the
lead checkout and do not establish write isolation; only the supervisor's
per-attempt worktrees do. At most one native subagent writer may act in a shared
cwd. Never run concurrent writers in the lead checkout.

Read the approved spec/plan, `meta.yaml`, research, effective project instructions,
root/base/head, assigned files, prior outputs and stage/Git grants. Confirm one
research effort in `docs/specs/NN-slug/`, one shared synthesis and one spec/plan
pair per stream. File presence is not execution authorization.

## Working set precheck

A fresh attempt worktree contains tracked files only, and OpenCode's project
discovery stops at the worktree's own Git directory. Project-scoped `.opencode/`
content and `opencode.json` are therefore invisible to a worker unless they are
committed on the recorded base. Before writing any request, run

```sh
sh {skillDir}/scripts/worker-precheck.sh <base-ref>            # tracked at the base
sh {skillDir}/scripts/worker-precheck.sh <base-ref> --global   # installed at global scope
```

It exits 0 only when the complete working set is present in the checked place:
the `patronus-writer-oc` agent (`mode: subagent`), the hook runner plugin
`patronus-hook-runner.ts`, `patronus/hooks.json` registering `block-secrets-oc`
and `gitleaks-guard-oc`, both executable guard scripts, and an `opencode.json`
whose `permission.edit` map carries the `*`, `*.env` and `.env.*` keys
(`permission-defaults-oc`). Every missing identity is named on stderr. A
non-empty tracked listing or a populated directory alone never passes. If both
forms fail, stop: the worker would run without the writer role and without the
guards. Installing or committing them is a separate grant, not a step of this
skill. The global root is `OPENCODE_CONFIG_DIR`, else `XDG_CONFIG_HOME/opencode`,
else `~/.config/opencode`.

## Request binding

Write a request matching
`{skillsDir}/isolated-writer-supervisor/references/request.schema.json` and check
it with `python3 {skillsDir}/isolated-writer-supervisor/scripts/supervisor.py
validate REQUEST.json` before any launch. The request binds the recorded base
commit, the task brief and its hash, the evidence directory `runDir`, a worktree
root outside the lead checkout, maximum concurrency and, per task: worker argv,
exclusive owned paths, timeout, grace period, bounded retry budget
(`maxAttempts`, default one) and predecessors. Overlapping claims, escaping or
`.git` paths, unknown predecessors, cycles and a reused run identity are
rejected before anything is created.

## Worker argv

Each task's `command` is a headless OpenCode run of the writer role:

```json
["env", "OPENCODE_CONFIG_CONTENT=<inline JSON>",
 "opencode", "run", "--format", "json", "--auto",
 "--agent", "patronus-writer-oc", "-m", "<provider/model>",
 "--dir", "<attempt worktree>", "<brief reference>"]
```

The inline JSON is exactly

```json
{"agent":{"patronus-writer-oc":{"permission":{"external_directory":{"<runDir>/*":"allow"}}}}}
```

with the request's recorded `runDir` substituted. Render it from `REQUEST.json`
rather than typing it:

```sh
python3 -c 'import json,sys; r=json.load(open(sys.argv[1])); print(json.dumps({"agent":{"patronus-writer-oc":{"permission":{"external_directory":{r["runDir"]+"/*":"allow"}}}}},separators=(",",":")))' REQUEST.json
jq -c '{agent:{"patronus-writer-oc":{permission:{external_directory:{(.runDir+"/*"):"allow"}}}}}' REQUEST.json
```

Why this shape:

- The writer agent file keeps `external_directory: {"*": deny}`. The inline rule
  admits writes outside the worktree under `runDir` only, which is where the
  supervisor result file (`runDir/tasks/<key>/attempt-N/result.json`, given to
  the worker as `PATRONUS_SUPERVISOR_RESULT`) lives. Documented trade: the worker
  may write anywhere under the evidence root, not only its own attempt
  directory. Whether the agent-file deny and the inline allow combine as
  last-match-wins is runtime-pending until the worker qualification row observes
  it; a precheck pass does not settle it.
- `--auto` flips `ask` outcomes to `allow` only; explicit `deny` rules stay
  enforced, and `edit` is already allowed inside the worktree for the writer
  role. It is not an authorization step and widens nothing beyond the inline
  rule.
- `OPENCODE_CONFIG_CONTENT` is the highest-precedence configuration source short
  of managed files; its value is never recorded in logs, reports or state.
- `<brief reference>` is the run message: name the brief at `<runDir>/brief.md`
  (the supervisor copies it there, hash-checked) and instruct the writer to read
  its `PATRONUS_SUPERVISOR_*` bindings and write the result file at
  `PATRONUS_SUPERVISOR_RESULT`. The argv is executed directly, so spell the
  literal `runDir`; no shell expansion happens.
- Exit codes and the `--format json` event stream are not documented contracts
  and are not relied on. The result file is the contract: a result whose run,
  task, attempt, root and base match the bindings, with `status` and
  `changedFiles`. Missing or malformed results are rejected by the supervisor.

## Writers

`python3 {skillsDir}/isolated-writer-supervisor/scripts/supervisor.py run REQUEST.json`
starts every attempt in a new worktree and branch from the recorded base, never
the lead checkout. The worker receives its bindings and a supervisor-owned result
path through `PATRONUS_SUPERVISOR_*` variables and must write a result whose run,
task, attempt, root and base match. Independent writers may overlap up to the
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

## Resource protocol

When the brief is silent, apply these defaults and record them in the request
and the task briefs: build parallelism `-j2`; builds run under
`flock -s /tmp/patronus-gates.lock`; benchmarks and multi-process test suites run
under `flock -x /tmp/patronus-gates.lock`; keep a 500 MB memory reserve; admit an
expensive job only with at least 750 MiB of free disk. Workers inherit the same
lock path through their worktrees, so a worker never runs an expensive job
outside the lock. A brief that records its own limits governs.

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
worktree directory is not observable. The `external_directory` permission is
OpenCode's own rule for its tools and many bash commands, not a kernel boundary.
Process discovery uses `/proc` on Linux and a process-group probe elsewhere;
Windows is blocked. Conflict detection is conservative (`git apply --check`, no
three-way merge). SIGKILL, host loss or another uncatchable hard kill cannot run
settlement; retained evidence alone then does not prove workers stopped.
Process-group/cwd observation does not detect arbitrary escaped processes outside
both boundaries. Only the invented Linux fixtures are observed here, not native
OpenCode or qualification of other platforms. Fake-worker tests and the precheck
fixture do not qualify native OpenCode workers, the permission merge, their
authentication or their sandbox.

## Authorized alternative

If the supervisor, Git, Python, the `opencode` executable, the writer role or the
guards are unavailable or auth-blocked, stop the lane as blocked with partial
evidence retained, report the Partial limit and ask the lead/owner. Serial
execution via `{skillsDir}/plan-execute-oc/SKILL.md` needs explicit lead
authorization; it is never a failed-engine fallback. Never change model,
provider, protocol or isolation to make progress.

Read-only parallel investigation may be separately authorized through
`{skillsDir}/dispatching-parallel-agents-oc/SKILL.md`. It grants no writer access.

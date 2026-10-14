---
name: dispatching-parallel-subagents
description: "Coordinator-used primitive for admitted leaf subagent launch, wait/stop, collection and settlement, for read-only leaves or separately authorized isolated writers, on Claude, Codex and OpenCode."
---

# Dispatching parallel subagents

The current coordinator applies this guidance to one admitted leaf or to a set.
Leaves complete their assignment, return complete evidence and stop. They never
nest delegation, admit successors, add gates or write shared progress. Dispatch
is subordinate to the caller's stage authority.

## Admission preflight

Before launch, inspect the native launch, wait and stop primitives, the leaf
role and its permissions, authentication and output delivery that this session
actually exposes. Use the observed native schema. A missing or auth-blocked
primitive blocks that lane: no CLI model call, provider/model change, protocol
switch, inline substitute or serial fallback.

Admit leaves within available native capacity and existing host/resource
limits. Ordinary work needs no root-wide numerical budget or start ledger;
explicitly authorized campaign limits still apply. Independent questions may run
in parallel. Shared state, causal dependencies or an unresolved common design
decision need a coordinator ordering decision before launch.

## Modes

- **Read-only:** no worktree, stable read inputs, complete text returned to the
  coordinator, which persists it. Read-only intent is cooperative scope, not an
  operating-system sandbox.
- **Isolated writer:** only under a separate execution grant, in a root prepared
  by the `using-git-worktrees` skill. The writer verifies root, base and required
  input hashes before its first write and writes only owned paths. Never run
  concurrent writers in the lead checkout, even on disjoint files. Fill
  `writer-brief.md` (in this skill's directory) for each writer.

## Brief inputs

Use the caller's existing brief fields; this skill adds no schema:

- outcome or question, and the acceptance point;
- exact root/base/head;
- accepted predecessor inputs with hashes;
- read paths, or owned paths plus the assigned absolute root for writers;
- relevant instructions;
- testing strategy taken from the caller's brief;
- authorized checks and applicable host/resource limits;
- return and stop conditions.

The testing tier comes from the caller's brief and the host's TDD skill; this
skill does not classify it.

## Collection and outputs

Wait with the native wait mechanism, then read every complete final response. A
dispatch receipt, terminal status or session title is not delivery. Verify
consequential claims against current source before synthesis. Record per leaf:

- association with the actual native child identity;
- the complete final response;
- actual check evidence;
- observed settlement;
- limitations.

Writers also return the assigned root, base/head and the complete tracked and
untracked delta for caller inspection.

**Settlement** is `settled` only when the native child has completed or has been
stopped through an exposed control, **and** every background job or descendant
it started has stopped. Anything less is `unknown`. Stop only with exposed
controls, and never relaunch a live leaf.

## Failure boundaries

- A missing capability blocks that lane.
- Unknown settlement keeps allocations and blocks dependent retry, capture and
  integration.
- Missing or empty output blocks acceptance; preserve partial evidence.
- Independent admitted work may continue under its own grant.
- None of these triggers a fixer, reviewer, retry, engine change or serial
  substitution. A retry needs coordinator admission, confirmed settlement and
  retained failed evidence.

## Not owned

This skill does not own research questions or synthesis, decomposition,
readiness, product acceptance, integration, review requirements, rubrics,
disposition or progress. It can launch a reviewer the caller requires but never
creates that requirement. The CLI supervisor (`isolated-writer-supervisor`) is
explicit-only: used only when the user selects it, never a native prerequisite
or fallback.

## Claude

- Launch leaves with the Agent tool and use the wait/stop controls the session
  exposes.
- Read-only leaves need no write, task or messaging tool.
- A writer may use native `isolation: "worktree"` only when every required
  accepted input is already in the commit that isolation starts from. The
  coordinator records the returned path and branch.
- Otherwise the coordinator prepares the root through `using-git-worktrees` and
  passes it as an absolute root, which gives cooperative isolation like Codex.

## Codex

- Use the exposed native worker and wait tools under their observed schema.
- Codex does not allocate or enforce a separate writer root. The assigned
  absolute root is cooperative scope.

## OpenCode

- Use the Task tool with `@patronus-researcher-oc`, `@patronus-web-researcher-oc`
  (only when web access is authorized) and `@patronus-writer-oc`. Check each
  role with `opencode agent list`; an absent role blocks the lane.
- Native depth is 1: a subagent cannot spawn subagents.
- Each Task call returns the child's final response when its session ends.
- Task subagents share the lead cwd, and the `patronus-writer-oc` role denies
  `external_directory`. Check actual assigned-root access before a parallel
  write; never bypass it with Bash.

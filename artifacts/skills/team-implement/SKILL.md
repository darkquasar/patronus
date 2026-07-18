---
name: team-implement
description: "/team-implement — Spec-Driven Team Implementation. Use when the user wants to implement a feature from existing research (spec.md + plan.md). Spawns parallel teammate agents, each owning a concern boundary. Requires explicit invocation."
---

# /team-implement — Spec-Driven Team Implementation

You are executing a **spec-driven team implementation**. Research has already been done by someone else. Your job is to read the specs, understand the architecture, define concern boundaries, and spin up a team of agents to build it.

**You are the Team Lead.** Your job is to orchestrate, not to write the code yourself: you plan, spawn parallel teammates in native worktree isolation, coordinate, then merge their branches and clean up. The full protocol is in the [Coordination Protocol](#coordination-protocol) section at the end of this skill — read it before Phase 4.

---

## Phase 0: Identify the Research Domain

The user will provide a feature-folder path under `docs/specs/` (e.g. `docs/specs/07-logging-improvement/`). If they didn't, ask for it — or list the folders under `docs/specs/` whose `meta.yaml` shows `spec` and `plan` complete but `tasks` still `false`.

1. **Read the folder's `meta.yaml`** — confirm `completeness.spec` and `completeness.plan` are `true` before proceeding. Then scan the folder for these files:
   - `research.md` — background research and findings
   - `spec.md` — the specification (required)
   - `plan.md` — the implementation plan (required)
   - `tasks.md` — pre-existing task breakdown (optional)
   - `*-findings.md` — any additional research findings
2. **Read ALL found files** in the feature folder. Understand the full picture before proceeding.
3. **Read the project's instructions file** (`CLAUDE.md` / `AGENTS.md` if present) — internalize the project's conventions and constraints.
4. **Read `tasks/lessons.md`** if it exists — internalize past mistakes.

If there is no `spec.md` or `plan.md`, STOP and tell the user: "This domain has no spec or plan. Run `/team-research` first before using `/team-implement`."

---

## Phase 1: Understand the Codebase

Before defining any boundaries, you MUST understand the existing project structure:

1. **Read the project's directory structure** — understand where source code lives, how it's organized, what frameworks/languages are in use.
2. **Identify the file types in use** — `.ts`, `.tsx`, `.py`, `.go`, `.rs`, `.sql`, `.toml`, `.yaml`, etc. This determines the provenance header format.
3. **Read key files referenced by the spec/plan** — if the spec says "modify `src/foo/bar.ts`", read that file. Understand the patterns, conventions, and style already in place.
4. **Check for existing tests** — understand how tests are structured so teammates can follow the same patterns.

---

## Phase 2: Check for tasks.md

Look for `tasks.md` in the feature folder. This is the implementation task breakdown.

**If `tasks.md` does NOT exist**, you must create it before proceeding:

1. Enter plan mode.
2. Analyze `spec.md` and `plan.md` to extract every discrete implementation task.
3. Group tasks by concern boundary (the domains that will become teammate assignments).
4. Write `tasks.md` in the feature folder, then set `tasks: true` in the folder's `meta.yaml` and bump `updated:`. See [TASKS-TEMPLATE.md](TASKS-TEMPLATE.md) for the format.

**Present `tasks.md` to the user for review before proceeding.** Do not spawn teammates until the user approves the task breakdown.

---

## Phase 3: Define Concern Boundaries

From `tasks.md`, identify 2-5 concern boundaries that become teammates. Each boundary must:

1. **Own a separable set of files** — no two teammates should edit the same file. If overlap exists, either merge those concerns into one teammate or define a clear contract (e.g., "teammate A writes the type, teammate B imports it").
2. **Have clear inputs and outputs** — what does this boundary produce that others consume? (types, schemas, API contracts, config entries)
3. **Map to a coherent domain** — not random task buckets, but logical architectural boundaries (e.g., "data layer + schema", "API routes + middleware", "UI components + hooks", "infrastructure + config", "test suite")

**Maximum 5 teammates.** If the work is small enough for 2, use 2. Don't create teammates for the sake of parallelism.

---

## Phase 4: Spawn the Team

Follow the [Coordination Protocol](#coordination-protocol) at the end of this skill.

**Implementers WRITE code in parallel — that is the one and only reason they need isolation.**
Get it from the `Agent` tool's `isolation: "worktree"`; the tool creates the worktree and branch,
so you don't set up any git worktree yourself.

```
Agent({ subagent_type: "general-purpose", name: "<teammate>", isolation: "worktree",
        run_in_background: true, prompt: <TEAMMATE-TEMPLATE, filled in> })
```

Issue **ALL spawns in a single message** with parallel `Agent` calls.

**Record what the spawn RETURNS**, not what you named it:
- **`agentId`** and **`worktreeBranch`** are the durable handles.
- **The branch name is `worktree-agent-<id>`; you don't choose it.** Read it from the result.
- **A task's `owner` field is agent-set data and can drift from the name you assigned** — address
  teammates by the name you spawned them with, not by a task's `owner`.

The agent commits to `worktree-agent-<id>`, that commit is reachable from the main repo, and the
lead merges it back with `git merge worktree-agent-<id> --no-ff` (Phase 6).

**⚠️ A worktree with a commit is not auto-reclaimed, and `.claude/` is gitignored, so the leftover
is invisible to `git status`. Cleanup is MANDATORY and it is yours (Phase 6, Cleanup).**

### Critical Spawn Rules

- **Do NOT paste code into the prompt.** Do NOT tell teammates HOW to implement things. Define the boundary, point to the specs, and let them figure it out. They have full access to the codebase and can read any file.
- **Do NOT assign individual tasks in the prompt.** Point them at their concern's ready set — `tk ready -T <concern>`. They will pull tasks themselves.
- **Spawn all teammates in a single message** with parallel `Agent` calls.

---

## Phase 5: Coordinate, then PULL

While teammates work:

1. **Coordinate dependencies** — when teammate A completes work that teammate B needs, use `SendMessage` to tell B to pull from A's branch.
2. **Unblock** — if a teammate is stuck, provide targeted guidance. Point to patterns in the codebase, clarify spec intent. Still do not write their code.
3. **Do NOT do the teammates' work.** Your job is orchestration, not implementation.
4. **The lead PULLS — it does not wait to be pushed.** See the Coordination Protocol's PULL rule. A returned message is not the deliverable; the committed branch is. When a background agent notifies you it has finished, go and confirm the commit exists on `worktree-agent-<id>` — do not synthesize from a status.

---

## Phase 6: Merge and Verify

When all teammates' tasks are complete (a background agent auto-notifies on completion — that notification is your signal to go PULL, not the deliverable itself):

1. **Confirm each branch has the commit** — the deliverable is the commit on `worktree-agent-<id>`, not a returned message. Read the branch name from each spawn's result; never trust a task's `owner`.
2. **Merge branches** sequentially into the parent branch with `--no-ff` (a completed agent has already returned — there is nothing to shut down):
   ```bash
   git merge worktree-agent-<id> --no-ff -m "Merge <teammate> work: <summary>"
   ```
3. **Resolve conflicts** if any — do not force-push or skip.
4. **Run verification**:
   - Type checking (e.g., `tsc --noEmit`, `mypy`, `cargo check`)
   - Tests (e.g., `npm test`, `pytest`, `cargo test`)
   - Any spec-defined verification steps
5. **Close the finished tickets** — `tk close <id>` for each. Closing is what unblocks the next ready item; work left open stalls whoever picks up next.
6. **Write `provenance.md`** — in the feature folder, list every created/modified file with ticket ids and change summaries. See [PROVENANCE-GUIDE.md](PROVENANCE-GUIDE.md).
7. **Record this stream's `epic:` in `meta.yaml`** — the tk epic id printed at seeding — and bump `updated:`. Do not rename the folder; the `NN-slug` name is stable.
8. **Cleanup (MANDATORY)** — remove the native worktrees and delete the teammate branches (see the Coordination Protocol's Cleanup step). Nothing is reclaimed once a worktree has a commit, and `.claude/` is gitignored, so a leak is invisible to `git status`.

---

## Phase 7: Report

Provide a summary to the user:

```
## Implementation Complete

**Domain**: <research domain>
**Spec**: <path to spec.md>
**Branch**: <current branch>

### What was built
- <high-level summary per concern boundary>

### Files changed
- <list of created/modified files, grouped by concern>

### Verification
- <type-check results>
- <test results>

### Provenance
See `<research-domain>/provenance.md` for full file-to-spec traceability.

### Next steps
- <anything the spec deferred or flagged as future work>
```

---

## Hard Rules (Non-Negotiable)

1. **Never start coding without an approved tk graph.**
2. **Never prescribe code to teammates.** Define boundaries, point to specs, let them write.
3. **Maximum 5 teammates.** Prefer fewer when the work allows it.
4. **No two teammates should edit the same file.** Resolve overlaps before spawning.
5. **Write `provenance.md`** in the feature folder listing every created/modified file. Do NOT add provenance headers to source files. See [PROVENANCE-GUIDE.md](PROVENANCE-GUIDE.md).
6. **Run verification before declaring done.** A description of a test is not a test.
7. **Follow the [Coordination Protocol](#coordination-protocol) to the letter** (plan, spawn parallel teammates in native isolation, coordinate, PULL, merge, cleanup).
8. **Present the seeded graph (`tk ls` + `tk ready`) to the user for approval** before spawning any teammates.
9. **The lead PULLS.** Completion is the artifact — a committed branch — never a status or a returned message. Do not synthesize from a status.

---

## Coordination Protocol

This is the protocol the Team Lead follows end-to-end. Implementation **writes code**, so each
teammate works in an isolated worktree on its own branch. Get that isolation from the `Agent`
tool's `isolation: "worktree"`; the tool creates the worktree and its `worktree-agent-<id>` branch,
that branch is reachable from the main repo, and the lead merges it back. Spawn teammates as
parallel `Agent` calls and coordinate by `name` via `SendMessage`.

**Team sizing:** Maximum 5 teammates for implementation work — coordination overhead dominates
beyond that. Use 2 for focused parallel work, 3–5 for broader feature builds with distinct
domains. Every teammate must own a clearly separable domain (files, modules, layers); if two
would edit the same files, merge them into one.

### The lead PULLS. Completion is the artifact, not a message.

A subagent can finish its work and go idle **without ever reporting it back** — even one told its
findings must be its final message. Treat delivery as your job to collect, not the member's job to
push:

1. **Assign each member an explicit output at spawn** — for implementers, the deliverable is a
   **commit on its `worktree-agent-<id>` branch**; for read-only members, a `<workdir>/<member>-findings.md`
   path you choose and put in the prompt.
2. **Go and read that artifact.** A returned message is a convenience, not the deliverable.
3. **Completion = the artifact exists** — the commit is on the branch, or the file is non-empty. A
   `completed` status can accompany a stream that produced nothing, so confirm the artifact itself.
4. **If the artifact is missing after the agent has terminated, the stream did not happen.** Re-run
   it, or do it inline — do not synthesize from a status.
5. Members also return their findings as text, as a redundant channel; read the artifact regardless.

**⚠️ The patience clause.** A missing artifact while the agent is **still running** means nothing —
wait. Background agents auto-notify on completion, and *that* notification is your cue to go read the
artifact. Do not poll early and do not declare an agent dead before it terminates.

**Do not use `TaskList` as a progress signal.** A self-reported status is neither liveness nor a
deliverable — read the artifact after the agent terminates.

**`git status` after every member terminates**, to confirm no teammate left the working tree
mutated outside its worktree branch.

### Step 1: Plan & seed the tk graph
1. Enter plan mode. Identify the parallel work streams. Each stream becomes a teammate.
2. Seed the tk graph from the plan: one epic to group, one ticket per plan task, `tk dep` for order. Tag each ticket with its concern (`--tags <concern>`) so teammates can pull with `tk ready -T <concern>`.
3. Present the seeded graph (`tk ls` + `tk ready`) to the user and get approval before spawning.

### Step 2: Spawn teammates in native isolation
Spawn ALL teammates in a **single message with parallel `Agent` calls** for maximum concurrency.
For each: `subagent_type: "general-purpose"`, `isolation: "worktree"`, a stable `name`, optional
`run_in_background: true`, and a filled-in [TEAMMATE-TEMPLATE.md](TEAMMATE-TEMPLATE.md) prompt. The
tool creates the worktree and its branch for you. **Record the returned `agentId` and
`worktreeBranch`**; the branch name is assigned by the tool, and a task's `owner` field is not a
reliable handle — address teammates by the name you gave them.

### Step 3: Coordinate, then PULL
- `SendMessage` (address teammates by `name`) to unblock, redirect, or share context. When a
  teammate completes work another depends on, notify the downstream teammate to pull.
- **PULL, per the rule above.** When a background agent notifies completion, go and confirm the
  commit exists on `worktree-agent-<id>`. Orchestrate; do not do the teammates' work.

### Step 4: Merge (Orchestrator only)
When all teammates have committed to their branches (confirmed by PULL, not by status):
1. **Merge each branch into the parent** sequentially, from the main project directory (NOT a worktree), resolving conflicts as you go:
   ```bash
   git merge worktree-agent-<id> --no-ff -m "Merge <teammate> work: <summary>"
   # ... repeat for each teammate
   ```
   Use `--no-ff` to preserve branch history. If a merge conflicts, resolve it manually — do not force or skip.
2. **Verify the merged result** — run tests, check for regressions, confirm everything integrates.

### Step 5: Clean up (MANDATORY)
A worktree with a commit is not auto-reclaimed, and `.claude/` is gitignored — **so the leftover is
invisible to `git status`.** For each member:
```bash
git worktree remove --force .claude/worktrees/agent-<id>
git branch -D worktree-agent-<id>
```
Then `git worktree list` and `git status` to confirm the tree is clean. **If something goes wrong
during merge:** do not force it — stop, re-plan, and consider whether work needs redoing or a
teammate needs to rebase.

### Step 6: Let agents terminate on their own
Background agents auto-notify on completion; there is no shutdown handshake to perform and no team to
tear down. Track status with `TaskUpdate`, not with structured JSON status messages.

$ARGUMENTS

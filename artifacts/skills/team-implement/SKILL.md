---
name: team-implement
description: "/team-implement — Spec-Driven Team Implementation. Use when the user wants to implement a feature from existing research (spec.md + plan.md). Spawns parallel teammate agents, each owning a concern boundary. Requires explicit invocation."
---

# /team-implement — Spec-Driven Team Implementation

You are executing a **spec-driven team implementation**. Research has already been done by someone else. Your job is to read the specs, understand the architecture, define concern boundaries, and spin up a team of agents to build it.

**You are the Team Lead.** Your job is to orchestrate, not to write the code yourself: you plan, create worktrees, spawn parallel teammates, coordinate, then merge their branches and clean up. The full protocol is in the [Coordination Protocol](#coordination-protocol) section at the end of this skill — read it before Phase 4.

---

## Phase 0: Identify the Research Domain

The user will provide a research domain path (a directory containing spec documents). If they didn't, ask for it.

1. **Scan the research directory** for these files (names may vary slightly):
   - `research.md` — background research and findings
   - `spec.md` — the specification (required)
   - `plan.md` — the implementation plan (required)
   - `tasks.md` — pre-existing task breakdown (optional)
   - `*-findings.md` — any additional research findings
2. **Read ALL found files** in the research directory. Understand the full picture before proceeding.
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

Look for `tasks.md` in the research domain directory. This is the implementation task breakdown.

**If `tasks.md` does NOT exist**, you must create it before proceeding:

1. Enter plan mode.
2. Analyze `spec.md` and `plan.md` to extract every discrete implementation task.
3. Group tasks by concern boundary (the domains that will become teammate assignments).
4. Write `tasks.md` in the research domain directory. See [TASKS-TEMPLATE.md](TASKS-TEMPLATE.md) for the format.

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

Follow the [Coordination Protocol](#coordination-protocol) at the end of this skill. Implementers WRITE code, so each needs an isolated git worktree on its own branch that you (the Team Lead) own and merge back:

1. **Create the task board** — `TaskCreate` for every item in `tasks.md`, with `addBlockedBy`/`addBlocks` for dependency ordering.
2. **Create one worktree + branch per teammate** that you control (so you can merge them afterward), branching from the current HEAD:
   ```bash
   git worktree add .claude/worktrees/<teammate-name> -b team/<team-name>/<teammate-name> HEAD
   ```
3. **Spawn teammates** — issue ALL spawns in a **single message with parallel `Agent` calls**. For each: `subagent_type: "general-purpose"`, a stable `name` (how you address it via `SendMessage`), optional `run_in_background: true`, and a `prompt` (see [TEAMMATE-TEMPLATE.md](TEAMMATE-TEMPLATE.md)) that directs the teammate to `cd` into its assigned worktree as its first action and confirm the `cd` before doing anything else. Do NOT pass the deprecated `team_name`, and do NOT use the `Agent` tool's `isolation: "worktree"` here — you created the worktrees yourself so you retain ownership of the branches to merge.

**Why parent-created worktrees (not the `Agent` tool's `isolation: "worktree"`)**: per Anthropic's
docs, worktree-isolated subagents (a) branch from your **default branch (origin/HEAD), NOT the
parent's current HEAD** unless `worktree.baseRef: "head"` is set — so on a feature branch they'd
get a stale baseline and diffs that may not merge cleanly — and (b) are **never auto-merged back**:
a changed worktree is just *preserved on its branch*, and the subagent returns only a text summary,
not a diff. By creating the worktrees yourself from the current `HEAD`, you branch from the right
baseline AND own branches you can merge. (Anthropic's only first-party parallel-code pattern,
`/batch`, sidesteps merging entirely by having each agent open a **pull request** — a valid
alternative if you'd rather integrate via PRs than direct merge.) Coordinate via `SendMessage` (by
name) and track via the `Task*` board.

### Critical Spawn Rules

- **Do NOT paste code into the prompt.** Do NOT tell teammates HOW to implement things. Define the boundary, point to the specs, and let them figure it out. They have full access to the codebase and can read any file.
- **Do NOT assign individual tasks in the prompt.** Point them to `tasks.md` and their concern section. They will pick up tasks themselves.
- **Spawn all teammates in a single message** with parallel `Task` calls.

---

## Phase 5: Monitor and Coordinate

While teammates work:

1. **Monitor** `TaskList` for progress.
2. **Coordinate dependencies** — when teammate A completes work that teammate B needs, use `SendMessage` to tell B to pull from A's branch.
3. **Unblock** — if a teammate is stuck, provide targeted guidance. Point to patterns in the codebase, clarify spec intent. Still do not write their code.
4. **Do NOT do the teammates' work.** Your job is orchestration, not implementation.

---

## Phase 6: Merge and Verify

When all teammates' tasks are complete (you're notified as each background agent finishes):

1. **Merge branches** sequentially into the parent branch with `--no-ff` (a completed agent has already returned — there is nothing to shut down).
2. **Resolve conflicts** if any — do not force-push or skip.
4. **Run verification**:
   - Type checking (e.g., `tsc --noEmit`, `mypy`, `cargo check`)
   - Tests (e.g., `npm test`, `pytest`, `cargo test`)
   - Any spec-defined verification steps
5. **Update `tasks.md`** — mark all completed tasks with `[x]`.
6. **Write `provenance.md`** — in the research domain directory, list every created/modified file with task IDs and change summaries. See [PROVENANCE-GUIDE.md](PROVENANCE-GUIDE.md).
7. **Rename the research domain folder** — prefix the folder name with `done-` to signal that the research has been implemented (e.g., `research/phase-08/skills-tools-model/` → `research/phase-08/done-skills-tools-model/`). Use `mv` to rename in place.
8. **Cleanup** — remove the worktrees you created and delete the teammate branches (see the Coordination Protocol's Cleanup step).

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

1. **Never start coding without an approved `tasks.md`.**
2. **Never prescribe code to teammates.** Define boundaries, point to specs, let them write.
3. **Maximum 5 teammates.** Prefer fewer when the work allows it.
4. **No two teammates should edit the same file.** Resolve overlaps before spawning.
5. **Write `provenance.md`** in the research domain directory listing every created/modified file. Do NOT add provenance headers to source files. See [PROVENANCE-GUIDE.md](PROVENANCE-GUIDE.md).
6. **Run verification before declaring done.** A description of a test is not a test.
7. **Follow the [Coordination Protocol](#coordination-protocol) to the letter** (plan, create worktrees, spawn parallel teammates, coordinate, merge, cleanup).
8. **Present `tasks.md` to user for approval** before spawning any teammates.

---

## Coordination Protocol

This is the protocol the Team Lead follows end-to-end. Implementation **writes code**, so each
teammate works in a dedicated git worktree on its own branch — created and owned by YOU (the
Team Lead) so you can merge each branch back after completion. Spawn teammates as parallel
`Agent` calls (there is a single implicit team — no team to create, no `team_name` to pass).
Coordinate by `name` via `SendMessage`; track via the `Task*` board.

**Team sizing:** Maximum 5 teammates for implementation work — coordination overhead dominates
beyond that. Use 2 for focused parallel work, 3–5 for broader feature builds with distinct
domains. Every teammate must own a clearly separable domain (files, modules, layers); if two
would edit the same files, merge them into one.

### Step 1: Plan & create the task board
1. Enter plan mode. Identify the parallel work streams. Each stream becomes a teammate.
2. `TaskCreate` to define all work items upfront with clear descriptions and acceptance criteria.
3. `TaskUpdate` with `addBlockedBy`/`addBlocks` to express ordering constraints.

### Step 2: Set up worktrees (MANDATORY before spawning)
Determine the parent branch (the branch you are on now — typically `main` or a feature branch). Create one worktree + branch per teammate that you own:
```bash
# Repeat for each teammate
git worktree add .claude/worktrees/<teammate-name> -b team/<team-name>/<teammate-name> HEAD
```
Naming convention: `team/<team-name>/<teammate-name>` (e.g., `team/auth-feature/backend`).

### Step 3: Spawn teammates
Spawn ALL teammates in a **single message with parallel `Agent` calls** for maximum concurrency.
For each: `subagent_type: "general-purpose"`, a stable `name` (how you address it via
`SendMessage`), optional `run_in_background: true`, and a prompt that directs the teammate to
`cd` into its assigned worktree as its FIRST action and confirm the `cd` before doing anything
else. Do NOT pass the deprecated `team_name`, and do NOT use the `Agent` tool's built-in
`isolation: "worktree"` — you created the worktrees yourself so you own the branches to merge.

### Step 4: Assign & coordinate
- `TaskUpdate` with `owner` to assign tasks; teammates report status via `TaskUpdate`.
- `SendMessage` (address teammates by `name`) to unblock, redirect, or share context. When a
  teammate completes work another depends on, notify the downstream teammate to pull.
- Monitor `TaskList`/`TaskGet` for progress. Do NOT do the teammates' work — you orchestrate.

### Step 5: Merge (Orchestrator only)
When all teammates' tasks are complete (you're notified as each background agent finishes) and
each has committed to its branch:
1. **Merge each branch into the parent** sequentially, from the main project directory (NOT a worktree), resolving conflicts as you go:
   ```bash
   git merge team/<team-name>/<teammate-name> --no-ff -m "Merge <teammate-name> work: <summary>"
   # ... repeat for each teammate
   ```
   Use `--no-ff` to preserve branch history. If a merge conflicts, resolve it manually — do not force or skip.
2. **Verify the merged result** — run tests, check for regressions, confirm everything integrates.

### Step 6: Cleanup
After successful merge and verification:
```bash
git worktree remove .claude/worktrees/<teammate-name>   # repeat per teammate
git branch -d team/<team-name>/<teammate-name>          # repeat per teammate
rmdir .claude/worktrees 2>/dev/null
```
**If something goes wrong during merge:** do not force it — stop, re-plan, and consider whether work needs redoing or a teammate needs to rebase.

$ARGUMENTS

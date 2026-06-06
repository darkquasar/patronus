---
name: team-implement
description: "/team-implement — Spec-Driven Team Implementation. Use when the user wants to implement a feature from existing research (spec.md + plan.md). Spawns parallel teammate agents, each owning a concern boundary. Requires explicit invocation."
---

# /team-implement — Spec-Driven Team Implementation

You are executing a **spec-driven team implementation**. Research has already been done by someone else. Your job is to read the specs, understand the architecture, define concern boundaries, and spin up a team of agents to build it.

**You are the Team Lead.** Your job is to orchestrate, not to do the manual labor: you plan, spawn, assign, coordinate, merge, and clean up. The full team-lifecycle protocol you follow is in the [Team Lifecycle Protocol](#team-lifecycle-protocol) section at the end of this skill — read it before Phase 4.

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

Follow Steps 1-4 of the [Team Lifecycle Protocol](#team-lifecycle-protocol) exactly. You MUST use **Team Mode** (not naive subagent spawning):

1. **Create team** via `TeamCreate`.
2. **Create tasks** via `TaskCreate` for every item in `tasks.md`, with `addBlockedBy`/`addBlocks` for dependency ordering.
3. **Create worktrees manually** — one per teammate, branching from current HEAD:
   ```bash
   git worktree add .claude/worktrees/<teammate-name> -b team/<team-name>/<teammate-name> HEAD
   ```
4. **Spawn teammates** using the `Task` tool **with `team_name` and `name` parameters** (this is what makes them Team members, not subagents). Do NOT use `isolation: "worktree"` — you already created worktrees manually. Spawn ALL teammates in a **single message with parallel `Task` calls**. Use the template in [TEAMMATE-TEMPLATE.md](TEAMMATE-TEMPLATE.md).

**Why Team Mode, not subagents**: Teammates work in isolated git worktrees on dedicated branches. The Team Lead merges their branches after completion. This allows parallel file writes without conflicts and preserves a merge-able git history. Naive subagents (`Task` without `team_name`) don't get worktrees, can't be coordinated via `SendMessage`, and can't be tracked via `TaskList`.

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

When all teammates report completion:

1. **Shut down teammates** — `SendMessage` with `type: "shutdown_request"` to each. Wait for confirmation.
2. **Merge branches** sequentially into the parent branch with `--no-ff`.
3. **Resolve conflicts** if any — do not force-push or skip.
4. **Run verification**:
   - Type checking (e.g., `tsc --noEmit`, `mypy`, `cargo check`)
   - Tests (e.g., `npm test`, `pytest`, `cargo test`)
   - Any spec-defined verification steps
5. **Update `tasks.md`** — mark all completed tasks with `[x]`.
6. **Write `provenance.md`** — in the research domain directory, list every created/modified file with task IDs and change summaries. See [PROVENANCE-GUIDE.md](PROVENANCE-GUIDE.md).
7. **Rename the research domain folder** — prefix the folder name with `done-` to signal that the research has been implemented (e.g., `research/phase-08/skills-tools-model/` → `research/phase-08/done-skills-tools-model/`). Use `mv` to rename in place.
8. **Cleanup** — remove worktrees, delete teammate branches, `TeamDelete`.

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
7. **Follow the [Team Lifecycle Protocol](#team-lifecycle-protocol) to the letter** for team lifecycle (create, spawn, coordinate, merge, cleanup).
8. **Present `tasks.md` to user for approval** before spawning any teammates.

---

## Team Lifecycle Protocol

This is the team-lifecycle protocol the Team Lead follows end-to-end. **Team Mode** means `TeamCreate` + spawning via the `Task` tool **with** `team_name` and `name` parameters — NOT naive subagent spawning. Members work in isolated git worktrees on dedicated branches; the Team Lead merges those branches after completion. This allows parallel file writes without conflicts and preserves a merge-able git history. Naive subagents (`Task` without `team_name`) don't get worktrees, can't be coordinated via `SendMessage`, and can't be tracked via `TaskList`.

**Team sizing:** Maximum 5 members per team for implementation work — coordination overhead dominates beyond that. Use 2 for focused parallel work, 3–5 for broader feature builds with distinct domains. Every member must own a clearly separable domain (files, modules, layers); if two members would edit the same files, merge them into one.

### Step 1: Plan & Create Team
1. Enter plan mode. Identify the parallel work streams. Each stream becomes a member.
2. `TeamCreate` to initialize the team.
3. `TaskCreate` to define all work items upfront with clear descriptions and acceptance criteria.
4. `TaskUpdate` with `addBlockedBy`/`addBlocks` to express ordering constraints.

### Step 2: Set Up Worktrees (MANDATORY before spawning)
Determine the parent branch (the branch you are on now — typically `main` or a feature branch). Create one worktree + branch per member:
```bash
# Repeat for each member
git worktree add .claude/worktrees/<member-name> -b team/<team-name>/<member-name> HEAD
```
Naming convention: `team/<team-name>/<member-name>` (e.g., `team/auth-feature/backend`).

### Step 3: Spawn Members
Use the `Task` tool with `team_name` and `name` parameters. **Do NOT use `isolation: "worktree"`** — you already created the worktrees manually. Spawn ALL members in a **single message with parallel `Task` calls** for maximum concurrency. Each spawn prompt must direct the member to `cd` into its worktree as its first action and confirm the `cd` before doing anything else. (See the spawn template referenced by this skill.)

### Step 4: Assign & Coordinate
- `TaskUpdate` with `owner` to assign tasks to members.
- `SendMessage` to unblock, redirect, or share context between members. When a member completes work another depends on, notify the downstream member to pull.
- Monitor `TaskList` for progress. Do NOT do the members' work — your job is orchestration.

### Step 5: Merge (Orchestrator Only)
When all members have committed and all tasks are complete:
1. **Shut down all members first** — `SendMessage` with `type: "shutdown_request"` for each. Wait for confirmation.
2. **Merge each branch into the parent** sequentially, from the main project directory (NOT a worktree), resolving conflicts as you go:
   ```bash
   git merge team/<team-name>/<member-name> --no-ff -m "Merge <member-name> work: <summary>"
   # ... repeat for each member
   ```
   Use `--no-ff` to preserve branch history. If a merge conflicts, resolve it manually — do not force or skip.
3. **Verify the merged result** — run tests, check for regressions, confirm everything integrates.

### Step 6: Cleanup
After successful merge and verification:
```bash
git worktree remove .claude/worktrees/<member-name>   # repeat per member
git branch -d team/<team-name>/<member-name>          # repeat per member
rmdir .claude/worktrees 2>/dev/null
```
Then `TeamDelete` to clean up team metadata. **If something goes wrong during merge:** do not force it — stop, re-plan, and consider whether work needs redoing or a member needs to rebase.

$ARGUMENTS

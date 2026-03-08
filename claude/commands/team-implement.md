# /team-implement — Spec-Driven Team Implementation

You are executing a **spec-driven team implementation**. Research has already been done by someone else. Your job is to read the specs, understand the architecture, define concern boundaries, and spin up a team of agents to build it.

**You are the Team Lead. Follow CLAUDE.md Section 2A exactly.**

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
3. **Read `CLAUDE.md`** — you will enforce its team protocol.
4. **Read `tasks/lessons.md`** if it exists — internalize past mistakes.

If there is no `spec.md` or `plan.md`, STOP and tell the user: "This domain has no spec or plan. Run research first before using /team-implement."

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
4. Write `tasks.md` in the research domain directory with this structure:

```markdown
# <Domain Name> — Implementation Tasks

**Source**: spec.md, plan.md (this directory)
**Created**: <date>
**Status**: Not started

---

## Concern: <boundary-name> (e.g., "Storage Layer", "API Routes", "UI Components")

### Tasks

- [ ] `<task-id>` — <clear description of what to build>
  - Files: <expected files to create or modify>
  - Acceptance: <how to verify this is done>
  - Refs: <which section of spec.md or plan.md defines this>

- [ ] `<task-id>` — <next task>
  ...

## Concern: <next-boundary-name>

### Tasks

- [ ] ...
```

Each task must have:
- A short ID (e.g., `A1`, `B3`, `C2`)
- Clear description of WHAT to build (not HOW)
- Expected files to create or modify
- Acceptance criteria (testable/verifiable)
- Reference back to the spec/plan section that defines it

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

Follow CLAUDE.md Section 2A Steps 1-4 exactly:

1. **Create team** via `TeamCreate`.
2. **Create tasks** via `TaskCreate` for every item in `tasks.md`, with `addBlockedBy`/`addBlocks` for dependency ordering.
3. **Create worktrees** — one per teammate, branching from current HEAD.
4. **Spawn teammates** in parallel using the prompt template below.

### Teammate Spawn Prompt Template

```
You are "<teammate-name>" on team "<team-name>".

FIRST ACTION — run this immediately:
  cd <absolute-path-to-project>/.claude/worktrees/<teammate-name>
Confirm the cd succeeded before doing anything else. All your work happens in this directory.

You are on branch: team/<team-name>/<teammate-name>

## Your Concern Boundary

<description of what this teammate owns — the domain, not the tasks>

## Reference Files (READ THESE FIRST)

- `<path-to-research-dir>/spec.md` — the specification you are implementing
- `<path-to-research-dir>/plan.md` — the implementation plan
- `<path-to-research-dir>/tasks.md` — your assigned tasks (look for your concern section)
- `CLAUDE.md` — project conventions (read Section 2B for your operating instructions)
- `tasks/lessons.md` — past mistakes to avoid (if it exists)

## Your Responsibilities

<list of concern-level responsibilities — NOT task-by-task prescriptions>

## Peer Branches

<list of peer branches they may need to pull from>

## Code Provenance — MANDATORY

Every file you CREATE or SIGNIFICANTLY MODIFY must include a provenance header.
This is non-negotiable. Add it as the FIRST thing in the file, before any imports or code.

The header links each file back to the research documents that drove the change, creating a traceable trail from spec to implementation.

Pick the correct comment syntax for the file type:

For TypeScript / JavaScript / Java / C# / Go / Rust (.ts, .tsx, .js, .jsx, .java, .cs, .go, .rs):

/**
 * @spec <relative-path-to-spec.md>
 * @plan <relative-path-to-plan.md>
 * @tasks <task-ids that drove changes, e.g., "A1, A2, A3">
 * @changed <date> — <one-line summary of what changed>
 */

For Python (.py):

# @spec <relative-path-to-spec.md>
# @plan <relative-path-to-plan.md>
# @tasks <task-ids>
# @changed <date> — <summary>

For SQL migration files (.sql):

-- @spec <relative-path-to-spec.md>
-- @plan <relative-path-to-plan.md>
-- @tasks <task-ids>
-- @changed <date> — <summary>

For CSS / SCSS (.css, .scss):

/* @spec <relative-path-to-spec.md>
 * @plan <relative-path-to-plan.md>
 * @tasks <task-ids>
 * @changed <date> — <summary>
 */

For TOML / YAML / Shell / Dockerfile config files (.toml, .yaml, .yml, .sh, Dockerfile):

# @spec <relative-path-to-spec.md>
# @plan <relative-path-to-plan.md>
# @tasks <task-ids>
# @changed <date> — <summary>

For HTML / JSX templates (.html):

<!-- @spec <relative-path-to-spec.md>
     @plan <relative-path-to-plan.md>
     @tasks <task-ids>
     @changed <date> — <summary> -->

Rules:
- If the file ALREADY has a provenance header, APPEND a new @changed line. Do not replace existing @changed entries — they form a changelog.
- If you only make a trivial change (fixing a typo, adjusting whitespace), skip the header update.
- The @spec and @plan paths are relative to the project root.
- Use the actual task IDs from tasks.md.

## Workflow

1. Read all reference files first. Understand the spec before writing any code.
2. Read existing source files you'll be modifying — understand patterns and conventions.
3. Use TaskUpdate to mark your tasks `in_progress` as you start them.
4. Commit after each logical unit of work (small, atomic commits with clear messages).
5. Use TaskUpdate to mark tasks `completed` when done, with proof it works.
6. Use SendMessage to report progress or flag blockers to the Team Lead.
7. Check TaskList after completing work for any new assignments.

Read CLAUDE.md Section 2B for your full operating instructions.
```

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
6. **Cleanup** — remove worktrees, delete teammate branches, `TeamDelete`.

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
All modified files include @spec/@plan/@tasks/@changed headers linking back to the research documents.

### Next steps
- <anything the spec deferred or flagged as future work>
```

---

## Hard Rules (Non-Negotiable)

1. **Never start coding without an approved `tasks.md`.**
2. **Never prescribe code to teammates.** Define boundaries, point to specs, let them write.
3. **Maximum 5 teammates.** Prefer fewer when the work allows it.
4. **No two teammates should edit the same file.** Resolve overlaps before spawning.
5. **Every created/modified file gets a provenance header** in the appropriate comment syntax.
6. **Run verification before declaring done.** A description of a test is not a test.
7. **Follow CLAUDE.md Section 2A to the letter** for team lifecycle (create, spawn, coordinate, merge, cleanup).
8. **Present `tasks.md` to user for approval** before spawning any teammates.

$ARGUMENTS

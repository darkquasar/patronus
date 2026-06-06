---
name: team-research
description: "/team-research — Spec-Driven Team Research. Use when the user wants to investigate an unknown domain, produce validated findings, and synthesize them into research.md + spec.md + plan.md. Spawns parallel researcher agents. Requires explicit invocation."
---

# /team-research — Spec-Driven Team Research

You are executing a **spec-driven team research** phase. Your job is to deeply investigate an unknown domain, produce validated findings, and synthesize them into three deliverables that feed directly into `/team-implement`:

1. **`research.md`** — raw findings, evidence, constraints, trade-offs
2. **`spec.md`** — the technical specification (what to build)
3. **`plan.md`** — the implementation plan (how to build it, phased)

You do NOT produce `tasks.md` — that's `/team-implement`'s job.

**You are the Team Lead.** Your job is to orchestrate, not to do the manual labor: you plan, spawn, assign, coordinate, merge, and clean up. The full team-lifecycle protocol you follow is in the [Team Lifecycle Protocol](#team-lifecycle-protocol) section at the end of this skill — read it before Phase 3.

---

## Phase 0: Define the Research Domain

The user will provide a research question, problem space, or feature area. If the description is too vague, ask clarifying questions until you have:

1. **The problem statement** — what are we trying to solve or understand?
2. **The output destination** — a directory under `research/` where deliverables will land. If none exists, create one with a descriptive name (e.g., `research/07-architecture-deep-dive/logging-improvement/`).
3. **Scope boundaries** — what's in scope and what's explicitly out of scope.
4. **Success criteria** — what does "research complete" look like? What questions must be answered?

---

## Phase 1: Preliminary Survey (Solo — Before Spawning)

Before creating any team, YOU do the initial reconnaissance. This prevents spawning agents into a void.

1. **Read the project's instructions file** (`CLAUDE.md` / `AGENTS.md` if present) — internalize the project's architecture, conventions, and constraints.
2. **Read `tasks/lessons.md`** if it exists — avoid past mistakes.
3. **Read the project structure** — understand what exists, what frameworks are in use, where the code lives.
4. **Read existing research** — check for prior research in the `research/` directory that overlaps with or informs this domain. Don't duplicate work that's already been done.
5. **Identify the unknowns** — list the specific questions that need answers. Each unknown becomes a potential research stream.

---

## Phase 2: Decompose into Research Streams

Enter plan mode. Break the research domain into 2-4 parallel research streams. Each stream investigates one unknown.

A research stream is NOT a task — it's a **question that requires investigation**. Good streams:
- "How does Cloudflare's X feature actually work under constraints Y and Z?"
- "What's the storage cost model for approach A vs approach B?"
- "What are the API contracts we'd need to integrate with system X?"
- "What patterns exist in our codebase that this change would affect?"

Bad streams (too vague or too implementation-focused):
- "Research the backend" — not a question
- "Write the schema" — that's implementation, not research
- "Figure out everything" — not decomposed

For each stream, define:
- **The question** it answers
- **The approach** — what to read, test, spike, or explore
- **The evidence required** — what constitutes a valid finding (code samples, benchmarks, API docs, existing patterns)
- **The output** — a `*-findings.md` file in the research directory

**Maximum 4 research streams.** If the domain is narrow enough for 2, use 2.

Present the research plan to the user for approval before proceeding.

---

## Phase 3: Spawn the Research Team

Follow Steps 1-4 of the [Team Lifecycle Protocol](#team-lifecycle-protocol) exactly. You MUST use **Team Mode** (not naive subagent spawning):

1. **Create team** via `TeamCreate`.
2. **Create tasks** via `TaskCreate` — one per research stream, plus synthesis tasks for the deliverables.
3. **Create worktrees manually** — one per researcher, branching from current HEAD:
   ```bash
   git worktree add .claude/worktrees/<researcher-name> -b team/<team-name>/<researcher-name> HEAD
   ```
4. **Spawn researchers** using the `Task` tool **with `team_name` and `name` parameters** (this is what makes them Team members, not subagents). Do NOT use `isolation: "worktree"` — you already created worktrees manually. Spawn ALL researchers in a **single message with parallel `Task` calls**. Use the template in [RESEARCHER-TEMPLATE.md](RESEARCHER-TEMPLATE.md).

**Why Team Mode, not subagents**: Researchers work in isolated git worktrees on dedicated branches. The Team Lead merges their branches after completion. This allows parallel file writes without conflicts and preserves a merge-able git history. Naive subagents (`Task` without `team_name`) don't get worktrees, can't be coordinated via `SendMessage`, and can't be tracked via `TaskList`.

### Critical Spawn Rules

- **Do NOT prescribe answers.** Define the question and approach, let the researcher investigate.
- **Spawn all researchers in a single message** with parallel `Task` calls.
- **Each researcher writes to their own `*-findings.md` file** — no file conflicts.

---

## Phase 4: Monitor and Coordinate

While researchers work:

1. **Monitor** `TaskList` for progress.
2. **Cross-pollinate** — if researcher A discovers something that affects researcher B's stream, use `SendMessage` to share the context.
3. **Redirect** — if a researcher hits a dead end, provide alternative angles or reframe the question.
4. **Do NOT do the researchers' work.** Your job is orchestration, not investigation.

---

## Phase 5: Synthesize Deliverables

When all researchers report completion:

1. **Shut down researchers** — `SendMessage` with `type: "shutdown_request"` to each. Wait for confirmation.
2. **Merge branches** sequentially into the parent branch with `--no-ff`.
3. **Read ALL `*-findings.md` files** produced by the team.
4. **Synthesize the three deliverables.** You write these yourself — this is the Team Lead's core job. Use the templates in [DELIVERABLE-TEMPLATES.md](DELIVERABLE-TEMPLATES.md).

### Deliverable Gate (MANDATORY before proceeding)

Before moving to Phase 6, verify all three deliverables exist in the research directory:

```bash
ls <research-dir>/research.md <research-dir>/spec.md <research-dir>/plan.md
```

If any file is missing, **STOP and write it now**. Do not proceed to cleanup without all three files committed. This gate exists because it's easy to write findings files and forget the synthesis step — the synthesis IS the deliverable, not the raw findings.

---

## Phase 6: Capture Lessons

Before cleanup, review the entire research process for lessons learned. Update `tasks/lessons.md` with any new entries. See [LESSONS-FORMAT.md](LESSONS-FORMAT.md) for the format and criteria.

---

## Phase 7: Cleanup and Report

1. **Verify all files are committed** to the parent branch.
2. **Cleanup** — remove worktrees, delete researcher branches, `TeamDelete`.
3. **Present the deliverables** to the user:

```
## Research Complete

**Domain**: <research domain>
**Directory**: <path to research directory>

### Deliverables

- `research.md` — consolidated findings from <N> research streams
- `spec.md` — technical specification
- `plan.md` — phased implementation plan
- `*-findings.md` — <N> raw findings files (appendix)

### Key Findings Summary
- <top 3-5 findings that shape the spec>

### Lessons Captured
- <N> new entries added to `tasks/lessons.md` (or "No new lessons")

### Decisions Needed
- <any open questions or trade-offs that require user input>

### Next Step
Run `/team-implement <research-dir>` to begin implementation.
```

---

## Hard Rules (Non-Negotiable)

1. **Never start spawning without an approved research plan.** Present the streams to the user first.
2. **Never skip the preliminary survey.** You must understand the codebase before decomposing the research.
3. **Maximum 4 researchers.** Prefer fewer when the domain allows it.
4. **Every finding needs evidence.** Opinions without evidence don't go in the spec.
5. **The Team Lead writes the deliverables.** Researchers produce raw findings; synthesis is your job.
6. **The output is three files: `research.md`, `spec.md`, `plan.md`.** No `tasks.md` — that's `/team-implement`'s responsibility. Do not proceed past Phase 5 without all three files written and committed.
7. **Follow the [Team Lifecycle Protocol](#team-lifecycle-protocol) to the letter** for team lifecycle (create, spawn, coordinate, merge, cleanup).
8. **Touch the actual code/system.** "I believe X works this way" is not a finding. "I read X at line Y and confirmed Z" is.
9. **Existing research is prior art.** Check `research/` before investigating something that may already be answered.
10. **Capture lessons before cleanup.** Surprising discoveries, constraint corrections, and process mistakes go in `tasks/lessons.md`. Routine findings stay in `research.md`.

---

## Team Lifecycle Protocol

This is the team-lifecycle protocol the Team Lead follows end-to-end. **Team Mode** means `TeamCreate` + spawning via the `Task` tool **with** `team_name` and `name` parameters — NOT naive subagent spawning. Members work in isolated git worktrees on dedicated branches; the Team Lead merges those branches after completion. This allows parallel file writes without conflicts and preserves a merge-able git history. Naive subagents (`Task` without `team_name`) don't get worktrees, can't be coordinated via `SendMessage`, and can't be tracked via `TaskList`.

**Team sizing:** Maximum 4 members per team — coordination overhead dominates beyond that. Use 2 for focused parallel work, 3–4 for broader builds with distinct domains. Every member must own a clearly separable domain (files, modules, layers); if two members would edit the same files, merge them into one.

### Step 1: Plan & Create Team
1. Enter plan mode. Identify the parallel work streams (max 4). Each stream becomes a member.
2. `TeamCreate` to initialize the team.
3. `TaskCreate` to define all work items upfront with clear descriptions and acceptance criteria.
4. `TaskUpdate` with `addBlockedBy`/`addBlocks` to express ordering constraints.

### Step 2: Set Up Worktrees (MANDATORY before spawning)
Determine the parent branch (the branch you are on now — typically `main` or a feature branch). Create one worktree + branch per member:
```bash
# Repeat for each member (max 4)
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

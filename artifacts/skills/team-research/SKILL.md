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

**You are the Team Lead.** Your job is to orchestrate, not to do the investigation yourself: you plan the streams, spawn parallel researchers, coordinate them, and synthesize their findings. The full protocol is in the [Coordination Protocol](#coordination-protocol) section at the end of this skill — read it before Phase 3.

---

## Phase 0: Define the Research Domain

The user will provide a research question, problem space, or feature area. If the description is too vague, ask clarifying questions until you have:

1. **The problem statement** — what are we trying to solve or understand?
2. **The output destination** — a per-feature folder under `docs/specs/`: `docs/specs/NN-slug/` (sequential number + short slug, e.g. `docs/specs/07-logging-improvement/`). Scan `docs/specs/` for the highest existing `NN` and increment. All deliverables — `research.md`, `spec.md`, `plan.md`, the `*-findings.md` appendix, and a `meta.yaml` manifest — land in this one folder. `docs/specs/` is gitignored; ensure `.gitignore` contains `/docs/specs/` (add it and tell the user if missing).
3. **Scope boundaries** — what's in scope and what's explicitly out of scope.
4. **Success criteria** — what does "research complete" look like? What questions must be answered?

---

## Phase 1: Preliminary Survey (Solo — Before Spawning)

Before creating any team, YOU do the initial reconnaissance. This prevents spawning agents into a void.

1. **Read the project's instructions file** (`CLAUDE.md` / `AGENTS.md` if present) — internalize the project's architecture, conventions, and constraints.
2. **Read `tasks/lessons.md`** if it exists — avoid past mistakes.
3. **Read the project structure** — understand what exists, what frameworks are in use, where the code lives.
4. **Read existing research** — check for prior feature folders under `docs/specs/` (their `research.md` / `spec.md`) that overlap with or inform this domain. Don't duplicate work that's already been done.
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

Follow the [Coordination Protocol](#coordination-protocol) at the end of this skill. Research is **read-only** — researchers investigate and write findings; they do not edit the codebase, so there are no worktrees, branches, or merges:

1. **Create the task board** — `TaskCreate` one task per research stream, plus synthesis tasks for the deliverables (`research.md`, `spec.md`, `plan.md`). Use `TaskUpdate` with `addBlockedBy` where a synthesis task must follow its streams.
2. **Spawn researchers** — issue ALL researcher spawns in a **single message with parallel `Agent` calls** for maximum concurrency. Each `Agent` call sets:
   - `subagent_type: "Explore"` — read-only investigation (researchers don't modify files).
   - `name`: a stable researcher name (e.g. `researcher-auth`) — how you address it via `SendMessage`.
   - `prompt`: the stream's question + brief (see [RESEARCHER-TEMPLATE.md](RESEARCHER-TEMPLATE.md)). Direct the researcher to write its findings to its own `*-findings.md` in the research directory and to call `TaskUpdate` to mark its task complete.
   - Optionally `run_in_background: true` to run async and be notified on completion.
   Researchers need no git worktrees — each writes only its own findings file, so nothing conflicts.

**Why parallel agents**: each researcher investigates one unknown concurrently and writes its own `*-findings.md` — no shared mutable state, no file conflicts, no merge step. The Team Lead coordinates via `SendMessage` (by name) and tracks via the `Task*` board, then synthesizes the findings.

### Critical Spawn Rules

- **Do NOT prescribe answers.** Define the question and approach, let the researcher investigate.
- **Spawn all researchers in a single message** with parallel `Agent` calls.
- **Each researcher writes to their own `*-findings.md` file** — no file conflicts.

---

## Phase 4: Coordinate, then PULL

While researchers work:

1. **Cross-pollinate** — if researcher A discovers something that affects researcher B's stream, use `SendMessage` to share the context.
2. **Redirect** — if a researcher hits a dead end, provide alternative angles or reframe the question.
3. **PULL each stream's findings yourself** (see the Coordination Protocol's PULL rule). When a background agent notifies completion, go and read its `*-findings.md` — a returned message is not the deliverable. Track status with `TaskUpdate`, not by watching a progress board.
4. **Do NOT do the researchers' work.** Your job is orchestration, not investigation.

---

## Phase 5: Synthesize Deliverables

When all researchers' tasks are complete (you're notified as each background agent finishes):

1. **Read ALL `*-findings.md` files** produced by the researchers.
2. **Synthesize the three deliverables.** You write these yourself — this is the Team Lead's core job. Use the templates in [DELIVERABLE-TEMPLATES.md](DELIVERABLE-TEMPLATES.md).
3. **Write the folder's `meta.yaml`.** Since team-research produces research, spec, and plan, create `docs/specs/NN-slug/meta.yaml` with those three flags `true` (leave `tasks: false` — that's `/team-implement`'s job):

   ```yaml
   slug: NN-slug
   intent: "One line: what this feature is."
   created: <today, YYYY-MM-DD>     # from context; do not invent
   updated: <today, YYYY-MM-DD>
   completeness:
     research: true
     spec:     true
     plan:     true
     tasks:    false
   ```

(Researchers are read-only `Explore` agents writing findings files — no branches to merge, no cleanup. A completed background agent has already returned; let it terminate on its own.)

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

1. **Verify all three deliverables are written** (the Deliverable Gate above).
2. **No cleanup needed** — read-only researchers leave no worktrees or branches behind; the
   `*-findings.md` files stay in the research directory as the audit trail.
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
6. **The output is three files plus `meta.yaml`: `research.md`, `spec.md`, `plan.md` in `docs/specs/NN-slug/`.** No `tasks.md` — that's `/team-implement`'s responsibility. Do not proceed past Phase 5 without all three files written and `meta.yaml` updated (research/spec/plan `true`). `docs/specs/` is gitignored — do not commit the deliverables.
7. **Follow the [Coordination Protocol](#coordination-protocol) to the letter** for the research lifecycle (plan streams, spawn parallel researchers, coordinate, synthesize).
8. **Touch the actual code/system.** "I believe X works this way" is not a finding. "I read X at line Y and confirmed Z" is.
9. **Existing research is prior art.** Check `docs/specs/` (prior feature folders) before investigating something that may already be answered.
10. **Capture lessons before finishing.** Surprising discoveries, constraint corrections, and process mistakes go in `tasks/lessons.md`. Routine findings stay in `research.md`.

---

## Coordination Protocol

This is the protocol the Team Lead follows end-to-end. Research is **read-only**: researchers
investigate the codebase/web and each writes its own `*-findings.md`. Because no researcher
edits shared files, they need **no git worktrees, no branches, and no merge step** — that isolation
machinery is only for teammates that write code (that is team-implement's job, not this one's).

**Team sizing:** Maximum 4 researchers — coordination overhead dominates beyond that. Use 2 for
a focused question, 3–4 for a broad domain with distinct unknowns. Each researcher owns one
clearly separable stream; if two streams would investigate the same thing, merge them into one.

### Step 1: Plan & create the task board
1. Enter plan mode. Identify the parallel research streams (max 4). Each stream becomes a researcher.
2. `TaskCreate` to define every stream + the synthesis tasks (`research.md`, `spec.md`, `plan.md`) upfront, with clear questions and acceptance criteria.
3. `TaskUpdate` with `addBlockedBy`/`addBlocks` to express ordering (synthesis after its streams).

### Step 2: Spawn researchers
Spawn ALL researchers in a **single message with parallel `Agent` calls** for maximum
concurrency. For each: `subagent_type: "Explore"` (read-only), a stable `name` (how you address
it via `SendMessage`), and a `prompt` carrying the stream question + brief. Optionally
`run_in_background: true` for async completion notifications. Tell each researcher to write its
findings to its own `*-findings.md` and to `TaskUpdate` its task to complete when done. Researchers
need no worktrees — they write no shared code.

### The lead PULLS. Completion is the artifact, not a message.

A researcher can finish and go idle **without ever reporting back** — even one told its findings must
be its final message. So collect delivery yourself rather than waiting to be pushed:

1. **Assign each researcher an explicit output path at spawn** — `<research-dir>/<stream>-findings.md`.
   You choose it and put it in the prompt.
2. **Go and read that file.** A returned message is a convenience, not the deliverable.
3. **Completion = the file exists and is non-empty.** A `completed` status can accompany a stream
   that produced nothing, so confirm the file itself.
4. **If the file is missing after the agent has terminated, the stream did not happen** — re-run it
   or do it inline; do not synthesize from a status.

**⚠️ The patience clause.** A missing file while the agent is **still running** means nothing — wait.
Background agents auto-notify on completion, and that notification is your cue to go read the file.
Do not poll early.

### Step 3: Coordinate
- `SendMessage` (address researchers by `name`) to share cross-stream context or redirect a
  researcher that hits a dead end. Researchers report status via `TaskUpdate`, not JSON messages.
- Orchestrate, and ultimately synthesize; do not do the researchers' work. Read the findings files
  after each agent terminates rather than watching a progress board.

### Step 4: Collect & synthesize
As each researcher terminates (you're notified for background agents), read its `*-findings.md`.
When all streams are in, synthesize `research.md` + `spec.md` + `plan.md` yourself (the Team
Lead's core job). The findings files are the audit trail; there is nothing to merge or clean up.

$ARGUMENTS

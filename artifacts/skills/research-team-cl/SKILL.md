---
name: research-team-cl
description: "/research-team-cl — team research. Use when the user wants to investigate an unknown domain, produce validated findings, and synthesize them into research.md. Spawns parallel researcher agents. Requires explicit invocation."
---

# /research-team-cl — Team Research

You are executing a **spec-driven team research** phase. Your job is to deeply investigate an unknown domain and produce validated findings — one consolidated **`<slug>-research.md`** — plus the `meta.yaml` skeleton that the downstream stages fill in. A folder is **one research effort with many streams**; a stream is **one spec and one plan** (ADR-0003). This skill owns the **research** stage only: it investigates the unknown, it does not author the specs or the plans.

The single deliverable is:

1. **one `<slug>-research.md`** — raw findings, evidence, constraints, trade-offs, shared by every stream
2. **the folder `meta.yaml`** — one `research:` entry plus a `streams:` skeleton with `spec: null`/`plan: null` per stream you identified

You do NOT author `<stream>-spec.md` — that is `spec-brainstorming-cl`'s job (it reads this research and specifies the known). You do NOT write `<stream>-plan.md` — that is `plan-writing-cl`' job. You do NOT seed the tk work-graph — that is the job of whichever skill executes the plan: `plan-writing-cl` seeds it when it mirrors a plan, and `/plan-execute-parallel-cl` seeds it when the plan reaches it unmirrored. Each downstream skill fills the fields it owns: `spec-brainstorming-cl` fills `spec:`, `plan-writing-cl` fills `plan:`, and the seeder fills `epic:` and `tasks:`.

**You are the Team Lead.** Your job is to orchestrate, not to do the investigation yourself: you plan the streams, spawn parallel researchers, coordinate them, and synthesize their findings. The full protocol is in the [Coordination Protocol](#coordination-protocol) section at the end of this skill — read it before Phase 3.

---

## Phase 0: Define the Research Domain

The user will provide a research question, problem space, or feature area. If the description is too vague, ask clarifying questions until you have:

1. **The problem statement** — what are we trying to solve or understand?
2. **The output destination** — a research-effort folder under `docs/specs/`: `docs/specs/NN-slug/` (sequential number + short slug, e.g. `docs/specs/07-logging-improvement/`). Scan `docs/specs/` for the highest existing `NN` and increment. The deliverable — one `<slug>-research.md`, the `<stream>-<who>-findings.md` appendix, and a `meta.yaml` manifest with a `streams:` skeleton — lands in this one folder. (The `<stream>-spec.md` and `<stream>-plan.md` are authored later by `spec-brainstorming-cl` and `plan-writing-cl` into the same folder.) Confirm `docs/specs/` is gitignored: `git check-ignore -q docs/specs/ && echo ignored`; if not, add `/docs/specs/` to `.gitignore` and tell the user.
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
- **The output** — complete findings text returned to the lead, who retains it as a `*-findings.md` file in the research directory

**Maximum 4 research streams.** If the domain is narrow enough for 2, use 2.

Present the research plan to the user for approval before proceeding.

---

## Phase 3: Spawn the Research Team

Follow the [Coordination Protocol](#coordination-protocol) at the end of this skill. Researchers are **read-only**: they investigate and return complete findings text. The lead writes all findings files and shared deliverables, so there are no researcher worktrees, branches, or merges:

1. **Create the task board** — `TaskCreate` one task per research stream, plus one synthesis task for the `<slug>-research.md` deliverable. Use `TaskUpdate` with `addBlockedBy` where the synthesis task must follow its streams. (No spec/plan synthesis tasks — those stages are downstream skills, not this one.)
2. **Spawn researchers** — issue ALL researcher spawns in a **single message with parallel `Agent` calls** for maximum concurrency. Each `Agent` call sets:
   - `subagent_type: "Explore"` — read-only investigation (researchers don't modify files).
   - `name`: a stable researcher name (e.g. `researcher-auth`) for result association and `SendMessage` when exposed.
   - `prompt`: the stream's question + brief (see [RESEARCHER-TEMPLATE.md](RESEARCHER-TEMPLATE.md)). Direct the researcher to return complete findings in its final response, not create files or update the task board. The lead owns `TaskUpdate` after collecting and retaining the findings.
   - Optionally `run_in_background: true` to run async and be notified on completion.
   Researchers need no git worktrees: they do not write files, including reports.

**Why parallel agents**: each researcher investigates one unknown concurrently and returns its findings without shared writes. The Team Lead coordinates through available native result/message delivery, owns the `Task*` board and findings files, then synthesizes the findings.

### Critical Spawn Rules

- **Do NOT prescribe answers.** Define the question and approach, let the researcher investigate.
- **Spawn all researchers in a single message** with parallel `Agent` calls.
- **Each researcher returns complete findings text.** Only the lead persists findings files.

---

## Phase 4: Coordinate, then PULL

While researchers work:

1. **Cross-pollinate**: when native messaging is exposed, share discoveries that affect another stream. Otherwise reconcile them from the returned findings.
2. **Redirect**: when native messaging is exposed, provide alternative angles for a dead end. Otherwise collect the limitation for lead disposition.
3. **Collect each stream's complete result yourself** (see the Coordination Protocol's collection rule). A completion notification is a cue to retrieve the full response, not evidence by itself. Read and retain the findings as `*-findings.md`, then update the task with `TaskUpdate`.
4. **Do NOT do the researchers' work.** Your job is orchestration, not investigation.

---

## Phase 5: Synthesize the Research Deliverable

When all researchers' tasks are complete (you're notified as each background agent finishes):

1. **Read ALL complete researcher results and the `*-findings.md` files retained by the lead.**
2. **Synthesize the one `<slug>-research.md`** — raw findings, evidence, constraints, and trade-offs, consolidated across every researcher. You write this yourself — this is the Team Lead's core job. Use the template in [DELIVERABLE-TEMPLATES.md](DELIVERABLE-TEMPLATES.md). Do NOT author any `<stream>-spec.md` or `<stream>-plan.md` — the research investigates the unknown; specifying and planning are downstream stages.
3. **Write the folder's `meta.yaml` skeleton.** A folder is one research effort with many streams; a stream is one spec + one plan (ADR-0003). Write the one `research:` file and one entry per independent work stream the research identified, leaving `spec:` and `plan:` **null** — the downstream skills fill them:

   ```yaml
   slug: NN-slug
   intent: "One line: what this investigation was."
   created: <today, YYYY-MM-DD>     # from context; do not invent
   updated: <today, YYYY-MM-DD>

   research: <slug>-research.md     # ONE per folder; naming it is what marks research done

   streams:
     - slug: <stream>              # THE name: <stream>-spec.md, <stream>-plan.md, --tags <stream>
       intent: "One line: what this stream is."
       spec: null                  # spec-brainstorming-cl fills this in
       plan: null                  # plan-writing-cl fills this in
       epic: null                  # the tk seeder (plan-writing-cl or plan-execute-parallel-cl)
                                   # fills this in with the tk epic id
       tasks: []                   # the same seeder fills this in: one entry per plan
                                   # task, {number, id, name}
   ```

   **One stream = one spec + one plan.** If the research forks into pieces that are independently
   specifiable, reviewable, and shippable, that is **more than one stream** — add a stream. They
   share the folder's one research doc; they share nothing else.

   **Name the file; do not assert a flag.** When a downstream skill authors a spec it writes
   `spec: <stream>-spec.md`, never `spec: true`. Naming the file and claiming it exists are the same
   act. Until then the field is `null`, not a promise.

   **The manifest must be checkable, not believable. Two invariants:**

   1. Every filename `meta.yaml` names **resolves to a file on disk.**
   2. Every `*-spec.md` / `*-plan.md` in the folder **is named in `meta.yaml`.**

   Both are greppable — that is why the schema names files rather than asserting flags.

(Researchers are read-only `Explore` agents returning text, not writing files. Confirm their results are collected and their runs have settled; there are no branches to merge.)

### Deliverable Gate (MANDATORY before proceeding)

Before moving to Phase 6, verify the manifest is honest — the `<slug>-research.md` exists and is named in `meta.yaml`, and the `streams:` skeleton is present:

```bash
# The research doc meta.yaml names resolves on disk.
ls <research-dir>/<slug>-research.md
```

If the research doc is missing, **STOP and write it now** — the synthesis IS the deliverable, not the raw findings, and it's easy to write findings files and forget it. Do not proceed to cleanup until `<slug>-research.md` exists and is named in `meta.yaml` alongside the `streams:` skeleton.

---

## Phase 6: Capture Lessons

Before cleanup, review the entire research process for lessons learned. Update `tasks/lessons.md` with any new entries. See [LESSONS-FORMAT.md](LESSONS-FORMAT.md) for the format and criteria.

---

## Phase 7: Cleanup and Report

1. **Verify the research deliverable is written** (the Deliverable Gate above).
2. **No cleanup needed** — read-only researchers leave no worktrees or branches behind; the
   `*-findings.md` files stay in the research directory as the audit trail.
3. **Present the deliverable** to the user:

```
## Research Complete

**Domain**: <research domain>
**Directory**: <path to research directory>

### Deliverable

- `<slug>-research.md` — consolidated findings shared across the streams
- `meta.yaml` — the streams skeleton (spec/plan still null; downstream skills fill them)
- `<stream>-<who>-findings.md` — raw findings files (appendix)

### Key Findings Summary
- <top 3-5 findings that shape the spec>

### Lessons Captured
- <N> new entries added to `tasks/lessons.md` (or "No new lessons")

### Decisions Needed
- <any open questions or trade-offs that require user input>

### Next Step
The research is done; the design is not. Author the spec from these findings with
**`spec-brainstorming-cl`** — it reads this `<slug>-research.md` and specifies the known, one
stream at a time. (Consider **`grilling-cl`** first if you want to stress-test the findings
before they harden into a spec.)
```

---

## Hard Rules (Non-Negotiable)

1. **Never start spawning without an approved research plan.** Present the streams to the user first.
2. **Never skip the preliminary survey.** You must understand the codebase before decomposing the research.
3. **Maximum 4 researchers.** Prefer fewer when the domain allows it.
4. **Every finding needs evidence.** Opinions without evidence don't go in the spec.
5. **The Team Lead writes the deliverables.** Researchers produce raw findings; synthesis is your job.
6. **The output is exactly one `<slug>-research.md` plus a `meta.yaml` streams skeleton (`spec: null`/`plan: null`), in `docs/specs/NN-slug/`.** Authoring the specs is `spec-brainstorming-cl`'s job, writing the plans is `plan-writing-cl`', and seeding the tk graph is `/plan-execute-parallel-cl`'s. Do not proceed past Phase 5 until `<slug>-research.md` exists and is named in `meta.yaml`. `docs/specs/` is gitignored — do not commit the deliverable.
7. **Follow the [Coordination Protocol](#coordination-protocol) to the letter** for the research lifecycle (plan streams, spawn parallel researchers, coordinate, synthesize).
8. **Touch the actual code/system.** "I believe X works this way" is not a finding. "I read X at line Y and confirmed Z" is.
9. **Existing research is prior art.** Check `docs/specs/` (prior feature folders) before investigating something that may already be answered.
10. **Capture lessons before finishing.** Surprising discoveries, constraint corrections, and process mistakes go in `tasks/lessons.md`. Routine findings stay in `research.md`.

---

## Coordination Protocol

This is the protocol the Team Lead follows end-to-end. Researchers are **read-only**:
they investigate the codebase/web within the approved scope and return complete findings text.
Only the lead persists `*-findings.md` files. Researchers need **no git worktrees, no branches, and no merge step** because they write no files. That isolation
machinery is only for teammates that write code (that is plan-execute-parallel-cl's job, not this one's).

**Team sizing:** Maximum 4 researchers — coordination overhead dominates beyond that. Use 2 for
a focused question, 3–4 for a broad domain with distinct unknowns. Each researcher owns one
clearly separable stream; if two streams would investigate the same thing, merge them into one.

### Step 1: Plan & create the task board
1. Enter plan mode. Identify the parallel research streams (max 4). Each stream becomes a researcher.
2. `TaskCreate` to define every research stream + the one synthesis task (the `<slug>-research.md`) upfront, with clear questions and acceptance criteria. No spec/plan synthesis tasks — those are downstream stages.
3. `TaskUpdate` with `addBlockedBy`/`addBlocks` to express ordering (synthesis after its streams).

### Step 2: Spawn researchers
Spawn ALL researchers in a **single message with parallel `Agent` calls** for maximum
concurrency. For each: `subagent_type: "Explore"` (read-only), a stable `name` (for result
association and messaging when exposed), and a `prompt` carrying the stream question + brief. Optionally
`run_in_background: true` for async completion notifications. Tell each researcher to return
complete findings in its final response. The lead persists those findings and updates the task
board. Researchers need no worktrees: they write neither source nor reports.

### The lead collects complete results and persists the evidence

1. **Choose a retained output path for each stream** at spawn, such as
   `<research-dir>/<stream>-findings.md`. This is the lead's write target, not a worker assignment.
2. **Wait for the researcher to settle and retrieve its complete final response** through the
   native result mechanism exposed by the session. A receipt, status or short summary is not
   the findings. Do not assume `Explore` has file-write, `SendMessage` or `TaskUpdate` tools.
3. **Read and validate the returned evidence, then persist the full findings** at the chosen
   path. Preserve limitations and unresolved questions, not just a success summary. The lead
   marks the task complete only after the substantive result is retained and readable.
4. **An absent worker-created file is expected, not failed research.** Do not rerun completed
   investigation merely because no worker wrote a file. If the actual result is missing or
   incomplete, retrieve available output or request the missing evidence within the existing
   budget. If delivery or persistence remains blocked, retain partial evidence and report it;
   do not synthesize from a status or silently switch to a write-capable role.

A completion notification is a cue to collect the result. While the researcher is running,
wait through the available native mechanism rather than polling for a file it cannot create.

### Step 3: Coordinate
- Use `SendMessage` when exposed for the selected agent to share cross-stream context or
  redirect a researcher that hits a dead end. Do not require workers to call unavailable tools.
- The lead owns task status, result collection and persistence. Read the complete results
  after each agent settles rather than treating a progress board as evidence.

### Step 4: Collect & synthesize
As each researcher settles, collect its complete findings response and persist its `*-findings.md`.
When all streams are in, synthesize the one `<slug>-research.md` yourself (the Team Lead's core
job) and write the `meta.yaml` streams skeleton. You do NOT author the specs or plans — those are
downstream stages. The findings files are the audit trail; there is nothing to merge or clean up.

---

**Next:** the research is written. The design is not yet settled — author the spec from these
findings with **`spec-brainstorming-cl`** (it reads this research and scales its dialogue to how much
the findings already answer). Consider **`grilling-cl`** first to stress-test the findings before they
harden into a spec. (Suggestions, not gates.)

$ARGUMENTS

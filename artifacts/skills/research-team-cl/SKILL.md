---
name: research-team-cl
description: "/research-team-cl — team research. Use when the user wants to investigate an unknown domain, produce validated findings, and synthesize them into research.md. Spawns parallel researcher agents. Requires explicit invocation."
---

# /research-team-cl — Team Research

You are executing a **spec-driven team research** phase. Your job is to deeply investigate an unknown domain and produce validated findings — one consolidated **`<slug>-research.md`** — plus the `meta.yaml` skeleton that the downstream stages fill in. A folder is **one research effort with many streams**; a stream is **one spec and one plan** (ADR-0003). This skill owns the **research** stage only: it investigates the unknown, it does not author the specs or the plans.

The single deliverable is:

1. **one `<slug>-research.md`** — raw findings, evidence, constraints, trade-offs, shared by every stream
2. **the folder `meta.yaml`** — one `research:` entry plus a `streams:` skeleton with `spec: null`/`plan: null` per stream you identified

You do NOT author specs or plans: spec-writing-cl owns `spec:` and
plan-writing-cl owns `plan:`. Research does not seed tk or create a task-board
mirror. At plan acceptance the coordinator records one progress authority in
the existing brief/plan. Qualified native state that actually owns delegated runs
wins (Pi today); otherwise explicit owner/project selection, then existing tracker
or one durable plan-associated record. UI todos alone are not run-owning state.
During research consume the existing brief's authority, not a new selection ledger.
Only an authorized selected-tracker owner may later seed accepted plan work once
and fill optional `epic:`/`tasks:` references. Metadata indexes intent/artifacts,
not mirrored mutable status. Respect actual ignore/tracking policy.

Only the coordinator admits research leaves and successor stages. A delegated
researcher returns complete text and stops: no agents, fixes, reviews, scope
expansion or shared progress writes. Read effective instructions, approved brief,
latest handoff and current source initially and after resume. Preserve valid
approval within scope; ask only for changed scope or a real unresolved choice.
Do not replay settled investigation. Existing host/resource and explicitly granted
campaign limits apply, without mandatory numerical root budgets or start ledgers.

**You are the Team Lead.** Your job is to orchestrate, not to do the investigation yourself: you plan the streams, spawn parallel researchers, coordinate them, and synthesize their findings. The full protocol is in the [Coordination Protocol](#coordination-protocol) section at the end of this skill — read it before Phase 3.

---

## Phase 0: Define the Research Domain

The user will provide a research question, problem space, or feature area. If the description is too vague, ask clarifying questions until you have:

1. **The problem statement** — what are we trying to solve or understand?
2. **The output destination:** use the authorized research-effort folder, normally
   `docs/specs/NN-slug/` (for example `docs/specs/07-logging-improvement/`). User
   preferences override this default; preserve existing allocated outputs. For a
   new folder inspect existing numbering. Research, full findings appendices and
   `meta.yaml` land here; specs/plans follow only under their own stage grants.
   Inspect actual ignore/tracking policy, for example `git check-ignore`, and
   disclose ignored local outputs in the handoff. Do not modify `.gitignore`,
   force-add files or infer commit/publication authority from this default.
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

Break the domain into independent research questions, proportionate to the unknowns
and available host capacity. A focused question may stay one stream; no minimum
agent count or compulsory generic budget form. Each admitted leaf answers one unknown.

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

Use only independent streams that add evidence; merge overlapping questions.

Consume existing valid research approval; present genuinely unresolved scope or
material choices for approval before proceeding. A phase label is not a new grant.

---

## Phase 3: Spawn the Research Team

Follow the [Coordination Protocol](#coordination-protocol) at the end of this skill. Researchers are **read-only**: they investigate and return complete findings text. The lead writes all findings files and shared deliverables, so there are no researcher worktrees, branches, or merges:

1. **Use the existing progress authority.** Retain questions, dependencies and
   acceptance in the approved brief. Use `TaskCreate`/`TaskUpdate` only if that
   surface is selected and available, never as a mandatory mirror of tk or native
   run state. Synthesis depends on retained complete findings, not task status.
2. **Spawn admitted researchers** within actual host capacity. Independent calls
   may share one message when supported; do not exceed applicable limits to launch
   every stream at once. Each `Agent` call sets:
   - `subagent_type: "Explore"` — read-only investigation (researchers don't modify files).
   - `name`: a stable researcher name (e.g. `researcher-auth`) for result association and `SendMessage` when exposed.
   - `prompt`: the stream question + relevant brief/source anchors (see
     [RESEARCHER-TEMPLATE.md](RESEARCHER-TEMPLATE.md)). Require complete final text,
     no file or shared-status writes. The coordinator retains results and updates
     only the selected authority after collection.
   - Optionally `run_in_background: true` to run async and be notified on completion.
   Researchers need no git worktrees: they do not write files, including reports.

Verify actual Agent read-only role, permissions/authentication and native result,
wait/stop controls before launch. Missing primitives stop the lane without model/
provider/CLI fallback. The coordinator controls concurrent admission, persists
complete findings and synthesizes only after all required results settle.

### Critical Spawn Rules

- **Do NOT prescribe answers.** Define the question and approach, let the researcher investigate.
- **Launch only coordinator-admitted leaves** within actual capacity; preparation
  does not admit dependent work or a later research wave.
- **Each researcher returns complete findings text.** Only the lead persists findings files.

---

## Phase 4: Coordinate, then PULL

While researchers work:

1. **Cross-pollinate**: when native messaging is exposed, share discoveries that affect another stream. Otherwise reconcile them from the returned findings.
2. **Redirect**: when native messaging is exposed, provide alternative angles for a dead end. Otherwise collect the limitation for lead disposition.
3. **Collect each complete result yourself** through native delivery. A completion
   notification is only a retrieval cue. Read and retain full `*-findings.md`
   evidence, then update only the selected authority when authorized.
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
       spec: null                  # spec-writing-cl fills this in
       plan: null                  # plan-writing-cl fills this in
       epic: null                  # optional reference only when tk is selected
       tasks: []                   # optional references, not a mutable status mirror
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

Before handoff, review durable project-relevant lessons. Update the existing
authorized `tasks/lessons.md` only within the write grant, otherwise return proposed
lessons for coordinator disposition. See [LESSONS-FORMAT.md](LESSONS-FORMAT.md).

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
**`spec-writing-cl`** — it reads this `<slug>-research.md` and specifies the known, one
stream at a time. (Consider **`grilling-cl`** first if you want to stress-test the findings
before they harden into a spec.)
```

---

## Hard Rules (Non-Negotiable)

1. **Spawn only within approved research scope.** Consume valid approval already
   present; resolve genuine new choices, not repetitive consent.
2. **Never skip the preliminary survey.** You must understand the codebase before decomposing the research.
3. **Admit only independent leaves within actual host/resource limits.** No
   mandatory root numerical budget or automatic successor wave.
4. **Every finding needs evidence.** Opinions without evidence don't go in the spec.
5. **The Team Lead writes the deliverables.** Researchers produce raw findings; synthesis is your job.
6. **Deliver one research synthesis and honest metadata** in the authorized folder,
   with retained full findings. Specs/plans remain downstream subjects. No automatic
   tk seeding, ignore-policy rewrite or commit/publication follows. Required output
   persistence failure blocks the affected transition; retain partial evidence.
7. **Follow the [Coordination Protocol](#coordination-protocol) to the letter** for the research lifecycle (plan streams, spawn parallel researchers, coordinate, synthesize).
8. **Touch the actual code/system.** "I believe X works this way" is not a finding. "I read X at line Y and confirmed Z" is.
9. **Existing research is prior art.** Check `docs/specs/` (prior feature folders) before investigating something that may already be answered.
10. **Capture authorized lessons or return proposals.** Routine findings stay in
    research.md. Settle all researchers and retain complete outputs before handoff;
    unknown liveness/capture blocks dependents and preserves allocations. No cleanup,
    publication or successor stage is automatically admitted.

---

## Coordination Protocol

This is the protocol the Team Lead follows end-to-end. Researchers are **read-only**:
they investigate the codebase/web within the approved scope and return complete findings text.
Only the lead persists `*-findings.md` files. Researchers need **no git worktrees, no branches, and no merge step** because they write no files. That isolation
machinery is only for teammates that write code (that is plan-execute-parallel-cl's job, not this one's).

**Team sizing:** the coordinator admits independent questions within available
capacity and applicable limits. Each leaf owns one clearly separable question;
merge overlap rather than manufacture extra starts or an obligatory count.

### Step 1: Consume the approved brief and authority
1. Identify independent questions, acceptance and retained output paths in the
   existing brief. Do not rewrite settled work or add a status ledger.
2. Native UI tasks are optional only when selected/available; no compulsory board
   or tk mirror. Synthesis follows substantive retained results, not status flags.

### Step 2: Spawn researchers
Launch admitted independent researchers with native `Agent` calls within actual
capacity. For each: `subagent_type: "Explore"` (read-only), a stable `name` (for result
association and messaging when exposed), and a `prompt` carrying the stream question + brief. Optionally
`run_in_background: true` for async completion notifications. Tell each researcher to return
complete findings in its final response. The coordinator persists findings and
updates only the selected authority. Researchers write neither source nor reports.

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
   grant and applicable limits. If delivery or persistence remains blocked, retain partial evidence and report it;
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
findings with **`spec-writing-cl`** (it reads this research and scales its dialogue to how much
the findings already answer). Consider **`grilling-cl`** first to stress-test the findings before they
harden into a spec. (Suggestions, not gates.)

$ARGUMENTS

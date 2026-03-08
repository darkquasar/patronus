# /team-research — Spec-Driven Team Research

You are executing a **spec-driven team research** phase. Your job is to deeply investigate an unknown domain, produce validated findings, and synthesize them into three deliverables that feed directly into `/team-implement`:

1. **`research.md`** — raw findings, evidence, constraints, trade-offs
2. **`spec.md`** — the technical specification (what to build)
3. **`plan.md`** — the implementation plan (how to build it, phased)

You do NOT produce `tasks.md` — that's `/team-implement`'s job.

**You are the Team Lead. Follow CLAUDE.md Section 2A exactly.**

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

1. **Read `CLAUDE.md`** — internalize the project's architecture, conventions, and constraints.
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

Follow CLAUDE.md Section 2A Steps 1-4 exactly:

1. **Create team** via `TeamCreate`.
2. **Create tasks** via `TaskCreate` — one per research stream, plus synthesis tasks for the deliverables.
3. **Create worktrees** — one per researcher, branching from current HEAD.
4. **Spawn researchers** in parallel using the prompt template below.

### Researcher Spawn Prompt Template

```
You are "<researcher-name>" on team "<team-name>".

FIRST ACTION — run this immediately:
  cd <absolute-path-to-project>/.claude/worktrees/<researcher-name>
Confirm the cd succeeded before doing anything else. All your work happens in this directory.

You are on branch: team/<team-name>/<researcher-name>

## Your Research Stream

**Question**: <the specific question this researcher answers>

**Approach**: <what to investigate — read source code, search docs, run spikes, analyze patterns>

**Evidence required**: <what constitutes a valid finding>

## Output

Write your findings to: `<research-dir>/<stream-name>-findings.md`

Your findings file MUST include:
- **Summary** — 2-3 sentence answer to the research question
- **Evidence** — code snippets, API responses, benchmarks, documentation excerpts that support the findings
- **Constraints discovered** — hard limits, gotchas, undocumented behaviors
- **Trade-offs** — if multiple approaches exist, compare them with pros/cons
- **Recommendations** — your informed opinion on what approach to take and why
- **Open questions** — anything you couldn't resolve that the team should know about

## Reference Files

- `CLAUDE.md` — project conventions (read Section 2B for your operating instructions)
- `tasks/lessons.md` — past mistakes to avoid (if it exists)
- Any existing research in `research/` that's relevant to your stream

## Rules

1. **Show your work.** Findings without evidence are opinions, not research.
2. **Touch the actual code.** Don't theorize about how something works — read it, trace it, test it.
3. **Note surprises.** If something behaves differently than expected, that's a critical finding.
4. **Stay in your lane.** Answer YOUR question. Don't speculatively investigate other streams.
5. **Commit your findings file** when done. Small atomic commits.
6. Use SendMessage to report completion or flag blockers to the Team Lead.
7. Use TaskUpdate to mark your task `in_progress` when starting and `completed` when done.

Read CLAUDE.md Section 2B for your full operating instructions.
```

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
4. **Synthesize the three deliverables.** You write these yourself — this is the Team Lead's core job.

### Deliverable 1: `research.md`

The consolidated research document. Structure:

```markdown
# <Domain Name> — Research

**Date**: <date>
**Status**: Complete
**Authors**: Team <team-name> (AI-assisted research)

## Problem Statement

<what we set out to understand>

## Scope

**In scope**: <what was investigated>
**Out of scope**: <what was explicitly excluded>

## Key Findings

### <Finding Area 1>

<synthesized findings from relevant streams, with evidence>

### <Finding Area 2>

<synthesized findings>

...

## Constraints & Hard Limits

<all discovered constraints, consolidated and deduplicated>

## Trade-off Analysis

<comparison of approaches where multiple options exist, with recommendations>

## Open Questions

<unresolved items that need future investigation or user decisions>

## Appendix: Stream Findings

- [<stream-1>-findings.md](<relative-path>) — <one-line summary>
- [<stream-2>-findings.md](<relative-path>) — <one-line summary>
...
```

### Deliverable 2: `spec.md`

The technical specification derived from the research. Structure:

```markdown
# <Domain Name> — Specification

**Date**: <date>
**Status**: Draft
**Source**: research.md (this directory)

## Overview

<1-2 paragraph description of what will be built and why>

## Goals

1. <goal 1>
2. <goal 2>
...

## Non-Goals

- <what this does NOT do>

## Architecture

<how the solution fits into the existing system — diagrams, data flow, component relationships>

## Detailed Design

### <Component/Module 1>

<interface contracts, data schemas, behavior specifications>

### <Component/Module 2>

...

## Dependencies

- <what this depends on — existing code, external services, CF features>

## Constraints

- <hard limits from research that shape the design>

## Testing Strategy

- <how correctness will be verified>

## Migration / Rollout

- <if applicable, how to transition from current state to new state>

## Future Considerations

- <things deferred but worth noting for later phases>
```

### Deliverable 3: `plan.md`

The implementation plan derived from the spec. Structure:

```markdown
# <Domain Name> — Implementation Plan

**Date**: <date>
**Status**: Draft
**Source**: spec.md (this directory)

## Prerequisites

- <what must be true before implementation starts>

## Phase Breakdown

### Phase 1: <name>

**Goal**: <what this phase achieves>
**Estimated scope**: <small / medium / large>

- <high-level work item 1>
- <high-level work item 2>
...

**Verification**: <how to confirm this phase is complete>

### Phase 2: <name>

...

## Concern Boundaries

<how the work naturally splits into parallel streams for /team-implement>

| Boundary | Owns | Produces | Consumes |
|----------|------|----------|----------|
| <name>   | <files/modules> | <outputs> | <inputs from others> |
| ...      | ...  | ...      | ...      |

## Risk & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| ...  | ...    | ...        |

## Deploy Order

<if relevant, the order in which changes should be deployed>

## Definition of Done

- [ ] <criterion 1>
- [ ] <criterion 2>
...
```

---

## Phase 6: Capture Lessons

Before cleanup, review the entire research process for lessons learned. Update `tasks/lessons.md` with any new entries. Lessons come from two sources:

### From the research findings
- **Surprising discoveries** — things that behaved differently than expected (e.g., platform limits, API quirks, silent failures). These are the most valuable lessons because they prevent future teams from hitting the same gotchas.
- **Constraint corrections** — if research invalidated a prior assumption in `MEMORY.md` or elsewhere, note the correction and what made the wrong assumption feel correct.

### From the research process
- **Orchestration mistakes** — if a researcher was blocked, mis-scoped, or produced unusable output, capture why and how to prevent it next time.
- **Stream decomposition issues** — if streams overlapped too much, were too broad, or missed a critical angle, note the better decomposition.

Each lesson follows the format already in `tasks/lessons.md`:

```markdown
## <date>: <short title>

**Mistake/Discovery**: <what happened>

**Why it felt correct**: <why the wrong assumption was plausible>

**The actual lesson**: <the real takeaway>

**Rule**: <a concrete, actionable rule to follow going forward>
```

**Only add lessons that are durable and project-relevant.** Don't log routine findings — those belong in `research.md`. Lessons are for patterns that should change how future work is done.

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
6. **The output is three files: `research.md`, `spec.md`, `plan.md`.** No `tasks.md` — that's `/team-implement`'s responsibility.
7. **Follow CLAUDE.md Section 2A to the letter** for team lifecycle (create, spawn, coordinate, merge, cleanup).
8. **Touch the actual code/system.** "I believe X works this way" is not a finding. "I read X at line Y and confirmed Z" is.
9. **Existing research is prior art.** Check `research/` before investigating something that may already be answered.
10. **Capture lessons before cleanup.** Surprising discoveries, constraint corrections, and process mistakes go in `tasks/lessons.md`. Routine findings stay in `research.md`.

$ARGUMENTS

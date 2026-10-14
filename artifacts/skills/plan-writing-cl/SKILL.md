---
name: plan-writing-cl
description: Use when you have a spec or requirements for a multi-step task, before touching code
---

# Plan Writing

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. Explicit test strategy. Commits only with separate authority.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the plan-writing-cl skill to create the implementation plan."

## Spec Gate (run this FIRST)

plan-writing-cl is **plan-only**. It does not author specs. Before planning anything, confirm a spec
exists for this stream, a plan is a promise to build a spec, and there is nothing to promise
without one, except an explicitly owner-declined spec retained in the brief.

Check for a `<stream>-spec.md` that is **named in the folder's `meta.yaml`** (`streams[].spec:`)
**and resolves on disk.** This is a check for the SPEC specifically, a folder can have a
`meta.yaml` and a `research.md` but no spec, and that still redirects. Do not read the "create
`meta.yaml` if absent" step below as license to proceed without a spec.

- **Spec explicitly declined by the owner** → retain the decision and plan against
  the accepted requirements. Do not invent a spec or require renewed consent.
- **No spec named, or the named file is absent (without explicit decline)** → **STOP.** Suggest `spec-writing-cl` for this
  stream: "This stream has no spec. Author it with `spec-writing-cl` first, it reads the
  research (if any) and specifies what to build; I'll plan from that." Do NOT synthesize a spec, do
  NOT infer one from `research.md`, and do NOT proceed to planning.
- **A `<stream>-plan.md` already exists for this stream** (an older `research-team-cl` or a prior run
  may have written one) → do NOT silently overwrite reviewed or edited work. Report it and ask:
  revise in place, or supersede? And note the two spec cases:
  - **plan exists + its spec exists** → ask revise-vs-supersede as above.
  - **plan exists but NO spec** → with explicit spec-decline, use accepted requirements;
    otherwise redirect to `spec-writing-cl`. Do not reverse-engineer a spec.
- **Spec present (no plan yet)** → plan normally (the rest of this skill).

Spec ownership stays with `spec-writing-cl`. plan-writing-cl sets `streams[].plan:` only, never
`streams[].spec:`.

**Save plans to:** `docs/specs/NN-slug/<stream>-plan.md`, alongside the spec they implement, one plan per spec (find the `<stream>-spec.md` you're planning from; your plan takes the same `<stream>` prefix). After writing it, fill in **your stream's** `plan:` in the folder's `meta.yaml`, name the file (`plan: <stream>-plan.md`), never assert `plan: true`, and bump `updated:`. If the folder has no `meta.yaml` yet, create one (see the `spec-writing-cl` skill's "Spec folder & meta.yaml" for the shape). Respect actual repository tracking/ignore policy; authoring grants no commit, ignore-rule change or publication. (User preferences for plan location override this default.)
- (User preferences for plan location override this default)

## Scope Check

If the spec covers multiple independent subsystems, suggest breaking this into separate plans, one per subsystem. Each plan should produce working, testable software on its own.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Task Right-Sizing

A task is the smallest independently verifiable behavior-sized deliverable. When drawing task boundaries: fold setup,
configuration, scaffolding, and documentation steps into the task whose
deliverable needs them; split where interfaces, ownership or acceptance can be checked independently,
not to manufacture reviewer gates. Each task ends with an
independently testable deliverable.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Capture results" - step; commit only if separately authorized

## Plan Document Header

**Every plan MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** The coordinator selects plan-execute-cl or
> plan-execute-parallel-cl from ownership/dependencies. Steps are intent, not a
> second mutable progress system. Read the accepted brief and current source.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

## Global Constraints

[The spec's project-wide requirements, version floors, dependency limits,
naming and copy rules, platform requirements, one line each, with exact
values copied verbatim from the spec. Every task's requirements implicitly
include this section.]

---
```

For critical behavioral bugs/boundaries use public RED/GREEN and relevant negative/
legacy regressions through tdd-cl. Other features/behavior default to TDD; when a
strategy is absent apply and record that default, not a manufactured stop.
Exceptions need a concrete reason and owner agreement unless already approved.
Pure refactors stay green; characterize uncertain legacy first. Nonbehavioral
prose/manifests use focused postchecks. Real classification/exception ambiguity
needs an owner decision. Adapt the example; do not force a red cycle on prose.

## Task Structure

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks, exact signatures]
- Produces: [what later tasks rely on, exact function names, parameter
  and return types. A task's implementer sees only their own task; this
  block is how they learn the names and types neighboring tasks use.]

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: intended public-behavior failure, not a missing function/setup error

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Capture actual focused checks and changed-byte evidence**

Record commands/options, actual exits, readable logs and checked inputs.
Commit only if a separate grant includes it.
````

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures**, never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code, the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task

## Remember
- Exact file paths always
- Complete code in every step, if a step changes code, show the code
- Exact commands with expected output
- DRY, YAGNI, behavior-sized verification and separately authorized commits

## Inline author check

Check spec/requirements coverage, placeholders and cross-task type/interface
consistency yourself. Fix gaps within the authoring grant. This is not a dispatched
review and does not replace independent plan review.

## One progress authority

At plan acceptance the coordinator records the sole mutable progress authority
once in the existing brief/plan. Native mission/workflow state that actually owns
delegated runs stays authoritative; a project tracker receives references only.
Without run-owning state use explicit project/owner selection, otherwise the
existing project tracker or, if absent, one durable plan-associated record.
Native UI todos alone do not outrank a project tracker.

No checklist-to-todo expansion, tk mirror or competing ledger. Metadata indexes
documents; evidence logs record observations, not mirrored progress. If tk is
selected, seed once under its authorized owner after acceptance, respecting actual
ignore/tracking policy. Never commit ignored .tickets/ or require a tracker install.
Existing ticket pointers name the actual plan file and exact task heading; follow
them, do not reconstruct missing requirements from a stale title.

## Review and execution handoff

One independent plan review is required for every written plan, including one
task, unless explicitly owner-declined and retained in the brief. The coordinator
uses plan-review-cl once for this subject. No dispatched Self-Review, automatic
second-model discovery or consent cycle. A spec, when present, keeps its separate
spec-review-cl subject. Review exceptions do not waive required tests or authority.

Consume full findings and the one bounded correction/disposition through
receiving-code-review-cl. Corrected bytes are parent-verified, not independently
re-reviewed; material changes or insufficient proof require an owner decision.
Use the existing review/evidence routes, not another ledger or automatic wave.

The coordinator selects execution mode from concrete ownership/dependencies:
disjoint isolated owners may use plan-execute-parallel-cl; shared/ordered work
uses plan-execute-cl, whose solo/SDD choice never creates automatic task reviews.
Pass relevant requirements, source/interface anchors, paths, settled predecessors,
strategy, acceptance and evidence references, not a growing transcript.
Verify real tool/auth/read-only/isolation/settlement requirements before launch.
Missing tools stop that lane without provider fallback.

On resume read effective instructions, accepted brief, latest handoff and exact
source/results; use the one authority and never replay settled work. Existing valid
grants remain valid within scope. Handoff does not self-admit implementation,
publication or cleanup. Ask only for a real changed scope or unresolved choice.

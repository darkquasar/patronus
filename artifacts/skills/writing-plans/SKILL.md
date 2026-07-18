---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the writing-plans skill to create the implementation plan."

**Save plans to:** `docs/specs/NN-slug/<stream>-plan.md`, alongside the spec they implement — one plan per spec (find the `<stream>-spec.md` you're planning from; your plan takes the same `<stream>` prefix). After writing it, fill in **your stream's** `plan:` in the folder's `meta.yaml` — name the file (`plan: <stream>-plan.md`), never assert `plan: true` — and bump `updated:`. If the folder has no `meta.yaml` yet, create one (see the brainstorming skill's "Spec folder & meta.yaml" for the shape). `docs/specs/` is gitignored — do not commit the plan. (User preferences for plan location override this default.)
- (User preferences for plan location override this default)

## Scope Check

If the spec covers multiple independent subsystems, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Task Right-Sizing

A task is the smallest unit that carries its own test cycle and is worth a
fresh reviewer's gate. When drawing task boundaries: fold setup,
configuration, scaffolding, and documentation steps into the task whose
deliverable needs them; split only where a reviewer could meaningfully
reject one task while approving its neighbor. Each task ends with an
independently testable deliverable.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

## Plan Document Header

**Every plan MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** Use the executing-plans skill to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

## Global Constraints

[The spec's project-wide requirements — version floors, dependency limits,
naming and copy rules, platform requirements — one line each, with exact
values copied verbatim from the spec. Every task's requirements implicitly
include this section.]

---
```

## Task Structure

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks — exact signatures]
- Produces: [what later tasks rely on — exact function names, parameter
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
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** — never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code — the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task

## Remember
- Exact file paths always
- Complete code in every step — if a step changes code, show the code
- Exact commands with expected output
- DRY, YAGNI, TDD, frequent commits

## Self-Review

**Dispatch a reviewer. If subagents are available, dispatch one.** A fresh subagent starts with a
clean context window, unsullied by the accumulated context of the session that produced the work. If
the host has no subagents, do it inline and say so.

*The author cannot have fresh eyes on their own work.* You know what each step *meant*; a reviewer
with a clean context reads only what each step *says*. That gap is the entire value of the review —
running the checklist against yourself closes none of it.

Give the reviewer the plan and the spec, and this checklist:

1. **Spec coverage:** point to the task implementing each spec requirement. List the gaps.
2. **Placeholder scan:** any "TBD", "add appropriate error handling", "similar to Task N", or a step
   that says *what* without showing *how*?
3. **Type consistency:** do the types, signatures, and names in later tasks match what earlier tasks
   defined? A function called `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

Fix what it finds. If a spec requirement has no task, add the task.

## Optional: Mirror Tasks Into Ticket

The plan's `- [ ]` checkboxes remain the canonical task capture. This mirror is additive, and
never required.

If ticket is available — `tk` on PATH or at `~/.patronus/bin/tk` — offer it after the plan is
saved:

> "Mirror these plan tasks into the ticket work-graph, so completion is tracked durably?"

If the user accepts: create one ticket per task with `tk create` (it prints the new id — capture
it), then add dependency edges matching the plan's build order with `tk dep <task> <depends-on>`.

Use the **full** create surface — `tk create "<title>" -t task -p <0..4> --tags <concern>
--acceptance "<the one check that closes it>" -d "PLAN: <file> → '<section heading>'. Files: …"
--external-ref <the file the work is specified in>`. The defaults cost you: **`tk ready` sorts by
priority**, so a ticket created without `-p` lands at the default `2` and the ordering signal is
dead. See the `ticket` instruction for why each flag earns its place.

**Epics group; only `tk dep` orders.** tk *does* have `-t epic` and `--parent` — but they are
**grouping and display only**: `tk ready` and `tk blocked` read **only `deps`**, never `parent`. An
epic never blocks and never unblocks anything. So use `--parent`/`--tags` to group a plan's tasks
under one heading, and use `tk dep` to encode the build order. A dependency edge is checkable; a
parent link is not.

The point is not bookkeeping — it is that a checkbox lives in a file the context window may lose,
while the work-graph survives a compaction, a new session, and a hand-off to another agent.
`tk ready` then answers "what's next" without re-reading the plan.

If ticket is absent or the user declines, the checkboxes are the source of truth. Say which one you
are using, so the next session knows where to look.

## Execution Handoff

After saving the plan, hand off to the **executing-plans** skill to implement it task-by-task with
review checkpoints. **If subagents are available, dispatch one per task** (with review between
tasks) — a fresh subagent starts with a clean context window, unsullied by the accumulated context
of the session that produced the work. If the host has no subagents, execute inline and say so.

The Self-Review above is your own pass over the plan. For an independent one — a fresh reviewer who
reads what the plan *says* rather than what you meant — run the **plan-review** skill before
handing off. It applies the same coverage / placeholder / type-consistency checks plus engineering,
design, DevEx, and strategy lenses, and it is advisory: it closes planning, it does not block it.

**Next:** the plan is written. Consider **`plan-review`** before building — a fresh subagent reads
what each step *says*, where the author knows what each step *meant*. Then `executing-plans`.
(Suggestion, not a gate.)

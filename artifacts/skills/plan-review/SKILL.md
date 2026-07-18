---
name: plan-review
description: Use at the END of the planning phase, before implementation — dispatches a reviewer applying plan basics (coverage, bite-sized, no placeholders, type consistency) plus engineering / design / devex / strategy lenses, surfacing Critical/Important/Minor findings. Advisory; NOT an implementation pre-flight.
---

# Plan Review

Close the planning phase by reviewing the plan before anyone builds from it. The defect this gate
catches most often is **coverage** — a spec requirement that no task actually implements, which
nobody notices until the feature ships without it.

This gate is **advisory**. It surfaces findings — it does not block, and it does not edit the plan.
You decide what to act on.

It runs **once**, at the end of planning. It is not an implementation pre-flight and does not
re-run per task.

**Core principle:** a plan is a promise to build the spec. Check the promise before keeping it.

## When to Use

**Use it when:**
- `writing-plans` has produced a plan and the next step is implementation.
- You are about to execute someone else's plan and want an independent read first.

**Skip it when:**
- The plan is a single task. There is no build order to get wrong and no coverage to miss.

## How to Review

**1. Identify the plan — and the spec it implements.**

Find the plan (`docs/specs/NN-slug/<stream>-plan.md` in the research-effort folder) *and* the spec it was written from (`<stream>-spec.md`, the same `<stream>` prefix, in the same folder). You
cannot review a plan without both: most of the rubric is a comparison between them.

If there is no spec, say so — a plan with no spec to check against can only be reviewed for
internal consistency, not for coverage.

**2. Dispatch the reviewer.**

Dispatch a `general-purpose` subagent, filling the template at [plan-reviewer.md](plan-reviewer.md).

**Placeholders:**
- `{DESCRIPTION}` — one line on what the plan builds
- `{PLAN_PATH}` — path to the plan document
- `{SPEC_PATH}` — path to the spec it implements

Dispatch a subagent rather than reviewing inline: the author of a plan knows what each step *meant*
and will read the gaps closed. A fresh reviewer reads only what is on the page — which is all an
implementer will get.

**3. Present the findings.**

Group them Critical / Important / Minor, with task references. Name the lenses the reviewer skipped.

**4. Decide, and say what you decided.**

Offer the choice plainly: fix the findings, accept them and proceed, or revise the plan. Do not
block, and do not proceed past a Critical finding without the user explicitly accepting it.

This is also the natural point for the **plan → ticket** offer: once the plan is accepted, mirroring
its tasks into the work-graph is what makes them survive a context compaction. See `writing-plans`.

## The Rubric

The reviewer applies **plan basics** (coverage, bite-sized steps, no placeholders, type
consistency, idiom-aligned, right-sized tasks) plus four lenses — **engineering**, **design**
(user-facing UI only), **DevEx** (developer-facing only), and **strategy** — skipping any lens that
does not apply.

Full rubric: [plan-reviewer.md](plan-reviewer.md)

## Red Flags

**Never:**
- Review the plan without the spec. Coverage is the point, and coverage is a comparison.
- Proceed past a Critical finding without the user explicitly accepting it.
- Treat "the plan looks thorough" as a pass. Thoroughness and coverage are different properties —
  a plan can be exhaustive about the wrong things.

## Where This Sits

`brainstorming` → `spec-review` → `writing-plans` → **plan-review** → `executing-plans`.

Its sibling gate, `spec-review`, closes the spec phase the same way.

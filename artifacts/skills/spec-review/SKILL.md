---
name: spec-review
description: Use at the END of the spec phase, before writing-plans — dispatches a reviewer applying spec basics plus engineering / design / devex / strategy lenses, surfacing Critical/Important/Minor findings. Advisory; you decide whether to proceed.
---

# Spec Review

Close the spec phase by reviewing what was written, before anyone plans from it. A defect caught in
the spec costs a paragraph; the same defect caught after implementation costs a rewrite.

This gate is **advisory**. It surfaces findings — it does not block, and it does not edit the spec.
You decide what to act on.

**Core principle:** review the spec before it becomes a plan, not after it becomes code.

## When to Use

**Use it when:**
- A spec or design doc has been written — by `brainstorming`, by `team-research`, or by hand — and
  the next step is `writing-plans`.
- You are about to plan from someone else's spec and want an independent read first.

**Skip it when:**
- The change is small enough that the "spec" is a sentence in the conversation. There is nothing to
  review.

## How to Review

**1. Identify the spec.**

Find the spec or design doc under review — usually the most recent one written this session. If
more than one candidate exists, ask which.

**2. Dispatch the reviewer.**

Dispatch a `general-purpose` subagent, filling the template at [spec-reviewer.md](spec-reviewer.md).

**Placeholders:**
- `{DESCRIPTION}` — one line on what the spec proposes
- `{SPEC_PATH}` — path to the spec document
- `{CONTEXT}` — anything the reviewer needs that the spec omits (constraints, prior decisions)

Dispatch a subagent rather than reviewing inline: a fresh reviewer reads what the spec *says*,
while you would read what you *meant*. That gap is the whole point of the gate. For a quick pass
on a short spec you may apply the rubric inline, but you will find less.

**3. Present the findings.**

Group them Critical / Important / Minor, with section references. Name the lenses the reviewer
skipped — a skipped lens is not a clean one.

**4. Decide, and say what you decided.**

Offer the choice plainly: fix the findings, accept them and proceed to planning, or revise the
spec. Do not block, and do not silently proceed past a Critical finding — if the user accepts one,
record that they did.

## The Rubric

The reviewer applies **spec basics** (testable, complete, unambiguous, scoped, idiom-aware) plus
four lenses — **engineering**, **design** (user-facing only), **DevEx** (developer-facing only),
and **strategy** — skipping any lens that does not apply to this spec.

Full rubric: [spec-reviewer.md](spec-reviewer.md)

## Red Flags

**Never:**
- Skip the gate because the spec "seems fine" — that judgment is exactly what the gate is testing.
- Proceed past a Critical finding without the user explicitly accepting it.
- Let the reviewer edit the spec. It advises; the author decides.

**A spec that fails this gate quietly** is one where every requirement is agreeable and none is
falsifiable. If the reviewer cannot find anything to disagree with, check that the spec actually
committed to something.

## Where This Sits

`brainstorming` (or `team-research`) → **spec-review** → `writing-plans` → `plan-review` → build.

Its sibling gate, `plan-review`, closes the planning phase the same way.

**Next:** once the findings are addressed (or consciously declined), the next hop is
**`writing-plans`**.

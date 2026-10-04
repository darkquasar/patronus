---
name: grilling-pi
description: Optional, bounded interview about a plan or design. Use when the user wants to stress-test a plan before building, or uses a 'grill' trigger phrase; no stage advance.
---

## Interview scope

Interview only when requested or accepted, within the task's question/time budget.
If no budget is given, agree a bound before starting. Stop when the user declines,
the bound is reached, or the authorized questions are resolved; report remaining
unknowns instead of extending the interview or blocking already authorized work.
A delegated worker routes unresolved decisions to its coordinator and waits for
an answer; it does not launch agents or change stages.

Walk down each branch of the design tree within that scope, resolving dependencies
between decisions one-by-one. For each question, provide your recommended answer.

Ask the questions one at a time, waiting for feedback on each question before continuing. Asking multiple questions at once is bewildering.

If a question can be answered by exploring the codebase within your read scope,
explore the codebase instead.

When what you're interrogating is structural — a control flow, a data flow, a component boundary, a state machine — draw it as a compact ASCII diagram before or alongside the question (diagram-explain-pi charset: `+---+` boxes, `=>` sync, `~>` async, `>` `<` `^` `v` arrows, ≤100 wide), unless the user's format forbids it. Ambiguity that survives prose rarely survives a picture: the gap you're probing becomes a box nobody can label, or an arrow nobody can point.

---

## When the grilling is done

The point of the interview is to make the idea sharp enough to *act on*.
grilling-pi is a generic stress-tester — anything with a design tree (a raw idea,
research, or a spec) flows in — and its spirit hands the sharpened idea back
**upstream** (to research or design-settling) or lets the user proceed.
**grilling-pi has NO forward hop into planning or execution.** It produces clarity,
not an artifact or a stage grant. Do not suggest `plan-writing-pi`,
`plan-review-pi`, `plan-execute-pi`, or `plan-execute-parallel-pi` as an automatic
next stage.

The skill names below are optional suggestions, not dependencies or automatic
invocations. Offer a named skill only if that selected Pi skill is available;
read its installed SKILL.md only when the task authorizes that work. If absent,
report the limitation and offer to return the findings to the coordinator, not
a legacy-name fallback, installation, or an unapproved stage transition.

**Tailor the outbound suggestion to where you entered from.** Detect it cheaply from files in the
current `docs/specs/NN-slug/` effort — whether a `<slug>-research.md` and/or a `<stream>-spec.md`
exist. Entry-awareness only tunes *which upstream offer to surface*; it never routes forward.

- **A `<stream>-spec.md` is present** (the common steady state — a spec exists, with or without
  research): file presence cannot tell "grilling to harden a fresh spec" from "grilling a spec
  that's already sharp," so **ASK, do not auto-route**:

  > "The spec looks sharper now. Want me to harden it further with `spec-brainstorming-pi` — fold what
  > we surfaced back into the spec — or return the findings without changing stage?"

- **Research only** (a `<slug>-research.md` exists, no spec yet):

  > "The findings look sharp now. Author the spec from them with `spec-brainstorming-pi`,
  > if you authorize that stage? Or return the findings?"

  Do NOT re-suggest `research-team-pi` here — that's a loop.

- **Entered from research-team-pi** (research exists, you were called to stress-test before the spec):
  same as research-only — offer `spec-brainstorming-pi`; do not loop back to `research-team-pi`.

- **Called cold** (no `research.md` and no `<stream>-spec.md` present) — the full two-option menu:

  > "The idea looks sharp now. Two possible directions for your approval:
  > - **The domain has real unknowns** (several things you'd have to go investigate before the
  >   design is even tractable) → the `research-team-pi` skill.
  > - **You know the domain; it's the design that needs settling** → the `spec-brainstorming-pi` skill.
  >
  > Or neither; I can return the findings without changing stage."

**Offer; do not gate.** The user may decline and proceed however they like within
their task authority. Spec-present is the ask-the-user case, not a confident pick.

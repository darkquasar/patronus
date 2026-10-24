---
name: grilling
description: Bounded interview about consequential plan or design choices on request. Use when the user wants to stress-test a plan before building, or uses any 'grill' trigger phrases.
---

## Interview scope

Read the current brief, project instructions and actual subject before asking.

Resolve questions answered by source or valid prior decisions yourself. Own routine
technical choices within scope. Distinguish facts, assumptions, owner-fixed
constraints and agent choices; label low-impact, reversible observable defaults
as agent-selected. Escalate unresolved consequential goals, scope, external
contracts, security, preservation, irreversible commitments or material costs
with evidence and a recommendation. Explain consequences and show a concrete
distinguishing example when interpretations compete; wait for the owner's answer.
Necessary clarification needs no separate interview grant.

An additional stress-test interview runs only on owner request or explicit
acceptance, within a declared finite question/time bound, one consequential
question at a time, with recommendation, consequences and an actual wait.
Stop when decisions are settled or the bound is reached. Manufacture no minimum
question count, extend no bound automatically and never rerun a valid approval.
Completion grants no next stage.

Before adopting new or materially expanded blocking verification criteria or
substantial test/evaluation machinery, surface a consequential owner choice.
Explain the protected goal and value, evidence, scope and useful granularity,
time/cost and maintenance burden, exact blocked stage/condition and a simpler
alternative (including informational comparison). Recommend an option, show a
concrete distinguishing example, ask one question and wait. Keep an undecided
proposal unresolved; continue unrelated authorized development. Reuse settled
policy and existing required checks without repeated permission. Ordinary relevant
test selection remains agent-owned.

When what you're interrogating is structural — a control flow, a data flow, a component boundary, a state machine — draw it as a compact ASCII diagram before or alongside the question (diagram-explain charset: `+---+` boxes, `=>` sync, `~>` async, `>` `<` `^` `v` arrows, ≤100 wide). Ambiguity that survives prose rarely survives a picture: the gap you're probing becomes a box nobody can label, or an arrow nobody can point.

---

## When the grilling is done

The point of the interview is to make the idea sharp enough to *act on*. grilling is a generic
stress-tester — anything with a design tree (a raw idea, research, or a spec) flows in — and its
spirit hands the sharpened idea back **upstream** (to research or requirements clarification) or lets the user
proceed. **grilling has NO forward hop into planning or execution.** It produces clarity, not an
artifact, so it has no edge into the plan stage; that routing is `plan-review`'s, not grilling's.
Do not suggest `plan-writing`, `plan-review`, `executing-plans`, or `plan-execute-parallel`.

**Tailor the outbound suggestion to where you entered from.** Detect it cheaply from files in the
current `docs/specs/NN-slug/` effort — whether a `<slug>-research.md` and/or a `<stream>-spec.md`
exist. Entry-awareness only tunes *which upstream offer to surface*; it never routes forward.

- **A `<stream>-spec.md` is present** (the common steady state — a spec exists, with or without
  research): file presence cannot tell "grilling to harden a fresh spec" from "grilling a spec
  that's already sharp," so **ASK, do not auto-route**:

  > "The spec looks sharper now. Want me to harden it further with `spec-writing` — fold what
  >  we surfaced back into the spec — or just build it however you like?"

- **Research only** (a `<slug>-research.md` exists, no spec yet) — unambiguous next hop:

  > "The findings look sharp now. Author the spec from them with `spec-writing`? (Or just
  >  build it however you like.)"

  Do NOT re-suggest `research-team` here — that's a loop.

- **Entered from research-team** (research exists, you were called to stress-test before the spec):
  same as research-only — offer `spec-writing`; do not loop back to `research-team`.

- **Called cold** (no `research.md` and no `<stream>-spec.md` present) — the full two-option menu:

  > "The idea looks sharp now. Two ways forward — your call:
  >  - **The domain has real unknowns** (several things you'd have to go investigate before the
  >    design is even tractable) → `/research-team`.
  >  - **You know the domain; the outcomes or requirements need clarification** → the `spec-writing` skill.
  >
  >  Or neither, if you'd rather just build it."

**Offer; do not gate.** The user may decline and proceed however they like. Spec-present is the
ask-the-user case, not a confident pick.

Offer named upstream skills only when installed and readable. Missing optional
companions are disclosed, not installed or substituted. Only an authorized lead
writes shared artifacts. This interview grants no edits, dispatch, Git mutation,
planning, implementation, publication or cleanup.

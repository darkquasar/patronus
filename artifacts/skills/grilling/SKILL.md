---
name: grilling
description: Interview the user relentlessly about a plan or design. Use when the user wants to stress-test a plan before building, or uses any 'grill' trigger phrases.
---

Interview me relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For each question, provide your recommended answer.

Ask the questions one at a time, waiting for feedback on each question before continuing. Asking multiple questions at once is bewildering.

If a question can be answered by exploring the codebase, explore the codebase instead.

When what you're interrogating is structural — a control flow, a data flow, a component boundary, a state machine — draw it as a compact ASCII diagram before or alongside the question (diagram-explain charset: `+---+` boxes, `=>` sync, `~>` async, `>` `<` `^` `v` arrows, ≤100 wide). Ambiguity that survives prose rarely survives a picture: the gap you're probing becomes a box nobody can label, or an arrow nobody can point.

---

## When the grilling is done

The point of the interview is to make the idea sharp enough to *act on*. grilling is a generic
stress-tester — anything with a design tree (a raw idea, research, or a spec) flows in — and its
spirit hands the sharpened idea back **upstream** (to research or design-settling) or lets the user
proceed. **grilling has NO forward hop into planning or execution.** It produces clarity, not an
artifact, so it has no edge into the plan stage; that routing is `plan-review`'s, not grilling's.
Do not suggest `plan-writing`, `plan-review`, `executing-plans`, or `team-implement`.

**Tailor the outbound suggestion to where you entered from.** Detect it cheaply from files in the
current `docs/specs/NN-slug/` effort — whether a `<slug>-research.md` and/or a `<stream>-spec.md`
exist. Entry-awareness only tunes *which upstream offer to surface*; it never routes forward.

- **A `<stream>-spec.md` is present** (the common steady state — a spec exists, with or without
  research): file presence cannot tell "grilling to harden a fresh spec" from "grilling a spec
  that's already sharp," so **ASK, do not auto-route**:

  > "The spec looks sharper now. Want me to harden it further with `spec-brainstorming` — fold what
  >  we surfaced back into the spec — or just build it however you like?"

- **Research only** (a `<slug>-research.md` exists, no spec yet) — unambiguous next hop:

  > "The findings look sharp now. Author the spec from them with `spec-brainstorming`? (Or just
  >  build it however you like.)"

  Do NOT re-suggest `team-research` here — that's a loop.

- **Entered from team-research** (research exists, you were called to stress-test before the spec):
  same as research-only — offer `spec-brainstorming`; do not loop back to `team-research`.

- **Called cold** (no `research.md` and no `<stream>-spec.md` present) — the full two-option menu:

  > "The idea looks sharp now. Two ways forward — your call:
  >  - **The domain has real unknowns** (several things you'd have to go investigate before the
  >    design is even tractable) → `/team-research`.
  >  - **You know the domain; it's the design that needs settling** → the `spec-brainstorming` skill.
  >
  >  Or neither, if you'd rather just build it."

**Offer; do not gate.** The user may decline and proceed however they like. Spec-present is the
ask-the-user case, not a confident pick.

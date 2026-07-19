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

The point of the interview is to make the idea sharp enough to *act on*. When it is:

> "The idea looks sharp now. Two ways forward — your call:
>  - **The domain has real unknowns** (several things you'd have to go investigate before the
>    design is even tractable) → `/team-research`.
>  - **You know the domain; it's the design that needs settling** → the `brainstorming` skill.
>
>  Or neither, if you'd rather just build it."

**Offer; do not gate.** The user may decline and proceed however they like.

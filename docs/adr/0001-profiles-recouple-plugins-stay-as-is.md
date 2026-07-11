---
status: accepted
date: 2026-06-24
---

# Opinionated profiles may re-couple vendored skills; plugins keep upstreams as-is

## Context

Patronus vendors skills from self-contained upstream "universes" (obra/superpowers,
mattpocock/skills, and others). Each upstream's skills were designed to cross-reference
and reinforce *each other* — their value came partly from that internal orchestration.
Vendoring individual skills into Patronus decoupled them from that native wiring, leaving
a bag of capable-but-loose parts that don't hand off to one another. At the same time,
Patronus is building a "plugin" concept for delivering upstreams intact.

## Decision

Two distinct delivery lanes, with different mutation rights:

1. **Plugins deliver upstreams as-is.** Anyone who wants a pristine upstream universe
   (e.g. all of superpowers, internally coupled) installs the plugin. No Patronus edits.

2. **Opinionated profiles may re-couple vendored artifacts into Patronus's own flow.**
   A profile — especially `core` — earns the right to edit a vendored skill's body so it
   hands off to sibling skills the *Patronus* way (e.g. brainstorming explicitly invoking
   the diagram-explain charset at design-presentation points). `core` is Patronus's own
   engineered lifecycle, built *from* vendored parts but wired to *our* sequence — not a
   neutral re-listing of other people's skills.

**Mutation sub-rule (how much a profile may change a vendored artifact):**

- **Edit-in-place** when the upstream's structure survives and we are *adding hand-offs*
  or small couplings. Flip the `attribution.note` from "verbatim" to "adapted" and record
  what was re-coupled; keep the upstream license, copyright, and pinned commit.
- **Author a fresh sibling** (a Patronus-authored artifact that supersedes the vendored
  one) when we would be rewriting **more than half** of the body. Cleaner provenance when
  little of the original remains (precedent: `skills-dispatch-activate`, an authored
  variant adapted from a superpowers hook).

## Consequences

- Re-coupled skills **diverge from upstream**; future upstream updates will not merge
  cleanly. We own those bodies. Mitigated by the pinned commit + `attribution.note`
  recording the fork.
- Therefore re-couple **sparingly** — only at high-value seams where the coupling earns
  its maintenance cost — not across every vendored skill.
- This is the **template for the broader "production-ready virtual engineering team"
  reframe**: sequencing grilling into the design loop, language-idiom auto-detection,
  spec/plan review gates, and plan→beads sync are all instances of "re-couple loose
  vendored parts into Patronus's orchestrated flow." This ADR governs all of them.

## Considered alternatives

- **Keep everything verbatim (option A).** Rejected: it preserves the loose-parts model
  and prevents core from becoming an orchestrated lifecycle — the opposite of the goal.
- **Fork wholesale into Patronus-authored skills.** Rejected as the default: discards
  upstream attribution value and maintenance leverage for skills whose structure we
  largely keep. Reserved (author-fresh) only for >50% rewrites.

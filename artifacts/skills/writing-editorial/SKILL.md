---
name: writing-editorial
description: >
  The canonical writing and editorial style guide, and the on-demand action for critiquing prose
  against it. This is the single source of truth for how written output should read: tone,
  punctuation, and craft. Use it in two ways. (1) REFERENCE while composing: consult it whenever you
  are writing prose a reader will see, so the house style is there from the first draft rather than
  fixed afterwards. That covers notebook prose and .ipynb/marimo markdown cells, docstrings and
  module headers, READMEs, design docs, ADRs, specs, PR descriptions, commit bodies, lessons and
  topic explanations, Confluence pages, emails, and longer messages. (2) EDITORIAL REVIEW: invoke
  WHENEVER the user asks to "review this draft", "critique my writing", "check this against my
  style", "does this follow my rules", "edit this prose", "clean up this text", "make this sound
  like me", or pastes a paragraph/doc/email/message and asks how to improve the writing. The rules
  live in four tier files applied in sequence: tier-0 strips machine phrasings (ungated), tier-1
  fixes mechanics and the mirrored swap (nearly everything), tier-2 removes machine tells and
  protects the author's voice (gated on stakes), tier-3 works on meaning and movement (gated on
  stakes, and the only pass that may restructure). Do NOT use it to change code logic, to lint code
  style, or as a general writing-from-scratch generator; it governs how prose reads, not what it
  says.
---

# Writing editorial

This skill holds no rules. Every rule lives in a tier file, applied in sequence, so a caller can load
one tier without paying for the other three.

```
 raw draft
     |
     v
 tier-0   anti-slop phrasings          ungated, every piece     LOCAL
     |
     v
 tier-1   mechanics + mirrored swap    nearly everything        LOCAL
     |
     v
 tier-2   machine tells + craft        gated on stakes          LOCAL + PROTECT
     |                                 emits PRESERVE ----------+
     v                                                          |
 tier-3   meaning + connective tissue  gated on stakes          COMPOSE
     |    reads PRESERVE, must not flatten those spans <--------+
     v
 edited draft + change report
```

**An editor works span by span; a composer may restructure.** That is why the local passes run first
and the compositional pass runs last. By the time tier-3 reads the draft, the slop, the mechanical
faults, and the machine tells are gone, so its whole attention goes to whether the reasoning holds
and the paragraphs move.

The first three tiers repair the span that violates a rule, and a repair may need words the original
did not have. What none of them may do is restructure the piece or introduce material the author
never had. tier-3 is the one pass allowed to reorder, add a concession, or repair an ending.

## The four tiers

| Tier | File | Owns | Gate | Operation |
|---|---|---|---|---|
| 0 | `{skillDir}/tier-0.md` | machine phrasings, 11 catalogue entries | ungated | local |
| 1 | `{skillDir}/tier-1.md` | em-dashes, quote punctuation, the mirrored swap | nearly everything | local |
| 2 | `{skillDir}/tier-2.md` | tropes, word tiers, variance, what to protect | stakes | local + protect |
| 3 | `{skillDir}/tier-3.md` | reasoning, movement, bridges | stakes | compose |

Worked cases for the mirrored-swap rule ship beside it, in `{skillDir}/tier-1-fixtures.md`.

## When each tier applies

**tier-0** is ungated. Every piece of prose with a reader, at any length, in any register.

**tier-1** applies to nearly everything with a reader: lessons, docs, emails, Slack messages, PR
descriptions, commit bodies, this file itself. The only writing it skips is where the mechanics
genuinely do not matter, such as throwaway scratch notes or machine-read output. When in doubt, apply
it.

tier-0 and tier-1 have their own, broader gates; the closed exception list below governs tier-2 and
tier-3 only. The two gates are deliberately different, which is the whole reason the mirrored-swap
rule sits in tier-1.

**tier-2 and tier-3** are live whenever the prose has a reader who will judge it, or carries a claim
that reader will act on. **Length is not the test.** A 60-word job application answer, a PR
description, a Slack message arguing for a decision, a two-sentence answer to "why did you pick this
approach": all live. So are the obvious cases, the design docs, architecture writeups, proposals,
emails, and every lesson or topic explanation.

They are off only where there is no argument to articulate: a status ping, a one-line factual answer,
a code comment, a commit subject line. That is the whole exception list. **When in doubt, they are
live**, because short and high-stakes is exactly where these rules earn the most, and it is the case
a length test gets wrong.

## Dispatch: one subagent or four

On invocation for a review, ask once:

```
Four tiers to apply. How do you want them run?

  [1] One subagent, all four tiers in sequence     cheaper, one context
  [2] A fresh subagent per tier                    higher fidelity, ~4x tokens

  Recommended: [2] for anything published or high-stakes.
```

A fresh context per tier stops each pass from inheriting the previous pass's frame, which is the same
reason the pipeline exists at all. The cost is roughly four times the tokens, because each agent
re-reads the draft.

Recommend per-tier when the prose has real stakes (a published post, a PR description, a design doc),
and single-agent for quick passes. The skill recommends; the user decides.

When run per-tier, each subagent receives the draft as it stands, its own tier file, and, for tier-3,
the PRESERVE list. It does **not** receive the other tier files, which is what keeps the perspectives
from poisoning each other.

**Where subagent dispatch is unavailable**, skip the question, apply the four tiers in sequence
within the single context, and say that fresh-context isolation was unavailable. The tiers still
apply in order; only the isolation is lost.

## Editorial review workflow

When invoked to review a draft rather than to write from scratch, return targeted edits, not a
rewrite of the whole thing:

1. Decide the tiers in scope. tier-0 and tier-1 always apply. tier-2 and tier-3 apply unless the
   draft is on the closed exception list above. Length does not decide it. **When in doubt, they are
   live.**
2. Apply the tiers in order, each on the previous one's output.
3. For every hit, quote the offending span, name the tier-local rule id, and give the fix inline.
   Concrete beats abstract: show the rewritten sentence. Where a rule offers two honest repairs
   (tier-3.15), show which one fits and why.
4. Preserve the author's voice and meaning. **Fix how it reads, never what it claims.** When a fix
   would change the meaning, flag it and ask rather than guess.
5. Do not invent problems. If a passage is clean, say so and move on; silence on a paragraph means it
   passed. Remember tier-2.19: **leave the author's voice intact, and do not sand prose down to bare
   facts in the name of the rules.** This is the guide's main defense against over-editing, and it
   survives the split into tiers because the router is the one file every review passes through.

Report format:

```
EDITS:    rule id -> quoted span -> suggested fix
PRESERVE: (from tier-2; carried forward, or "(none)")
```

Lead with the edits that matter most, and skip preamble.

## Adding rules

This guide is meant to grow. When the user gives a new rule, add it to the right tier file with the
same shape as the others, and do not compress it into a bare command. Keep three things: **the rule,
the reasoning behind it, and worked examples (a Don't/Do pair, or a short before-and-after) with the
commentary that says what each example demonstrates.** The reasoning and the commentary matter most:
a rule with its why gets applied intelligently in cases the examples never covered, while a bare
imperative gets misfired or ignored.

Assign the tier by scope and by operation:

- tier-0 if it names a phrasing that is wrong at any length in any context. These are the shortest to
  add and the cheapest to apply, so prefer them when a rule really is unconditional.
- tier-1 if it governs sentence mechanics that hold everywhere, and can be applied without an open
  editorial weighing.
- tier-2 if it names a machine tell or governs surface craft, and the fix stays inside the offending
  span.
- tier-3 if it governs how prose carries reasoning or teaches, or if applying it may require
  restructuring.

A correction of the form "this specific phrasing reads as machine-made" belongs in tier-0, not buried
in a stakes-gated tier where a scope judgment can skip it. A rule that fits none cleanly earns a new
tier of its own.

## Known limits

- tier-1 carries one judgment call. The mirrored-swap rule is not a pure mechanic like the other two
  entries in that file. It is written as two mechanical checks wrapped around one narrow judgment to
  keep the file's character. Scope-gating is what let the pattern survive a stakes-gated rule, which
  is why it sits at this gate.
- Lexical substitution alone is becoming a tell. Widely circulated word lists mean scrubbing exactly
  one list is itself detectable. Lean on the structural rules (variance, cluster density,
  preservation) rather than on word swaps.

## References

For the human maintaining this guide, not for the model applying it. These are the sources the rules
were distilled from, kept here so the provenance is not lost.

- Wikipedia, "Signs of AI writing": the catalogue of machine-generated tells behind the
  puffery, participle-summary, weasel-attribution, and vogue-word entries in tier-0.
  <https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing>
- George Orwell, "Politics and the English Language": the source of the meaning-and-precision,
  connective-tissue, and image-rhythm-voice principles in tier-3, including the rule to break any
  rule when it serves the meaning.
  <https://www.orwellfoundation.com/the-orwell-foundation/orwell/essays-and-other-works/politics-and-the-english-language/>
- The Gods of good narrative: the working name for the third source, on positive-first
  exposition and using contrast only to close a live interpretive branch.
- ossa-ma, "AI Writing Tropes to Avoid" (tropes.fyi): 33 tropes across word choice, sentence
  structure, paragraph structure, tone, formatting, and composition. Source of the mirrored-swap
  characterization and the cluster-density principle in tier-2.
- conorbronsdon, "Avoid AI Writing", MIT licensed: source of the 1A/1B word-tier split, the
  split-sentence negation shape, the never-inject list and its provenance rule, and the burstiness
  framing.

Only rules and patterns are extracted from these sources. No source text is reproduced wholesale.
See the NOTICE file beside this one for the full attribution.

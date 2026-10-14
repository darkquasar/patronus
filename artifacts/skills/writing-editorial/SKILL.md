---
name: writing-editorial
description: >
  Optional editorial review. Use only on an explicit editorial-review invocation:
  "review this draft", "critique my writing", "check this against my style",
  "edit this prose", or "clean up this text". General prose composition alone
  does not activate this workflow. Applies the four bundled tiers to the selected
  draft, preserving the author's voice and reporting edits. Optional section
  fan-out requires the caller's choice. Supply trail-root for span-anchored records.
---

# Writing editorial

Activation requires an explicit editorial-review request. Do not load this
workflow automatically while composing ordinary prose. A workflow caller that
needs an editorial pass must explicitly invoke that pass on its selected draft.
The tier gates below apply only after invocation; "every piece" describes the
reviewed draft, not an implicit trigger for unrelated writing.

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
     |    emits CONTRAST-LEDGER  ------------------------------+
     v                                                         |
 tier-2   machine tells + craft        gated on stakes         | LOCAL + PROTECT
     |    emits PRESERVE  --------------------------------+    |
     v                                                    |    |
 tier-3   meaning + connective tissue  gated on stakes    |    | COMPOSE
     |    reads PRESERVE, and must not flatten it  <------+    |
     |    reads the LEDGER, and composes within it  <----------+
     v
 edited draft + change report

Both carry past tier-3 into any voice pass downstream. A stage that writes new
sentences is bound by the ledger it did not open.
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
| 1 | `{skillDir}/tier-1.md` | em-dashes, quote punctuation, the mirrored swap | nearly everything | local, emits ledger |
| 2 | `{skillDir}/tier-2.md` | tropes, word tiers, variance, what to protect | stakes | local + protect |
| 3 | `{skillDir}/tier-3.md` | reasoning, movement, bridges | stakes | compose |

Worked cases for the mirrored-swap rule ship beside it, in `{skillDir}/tier-1-fixtures.md`.

Two further references govern how the tiers are applied to a long draft, rather than what they
judge: `{skillDir}/sectioning.md` (cutting, fan-out, merge, tier-3 projection) and
`{skillDir}/edit-record.md` (the two schemas a caller joins on). Neither changes a tier's verdict.

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

## Execution choice and local reads

Apply the four tiers inline in order by default. Resolve `{skillDir}` to this
installed skill's directory and read its own sidecars, never a same-name foreign
copy. Carry tier-1's contrast ledger and tier-2's PRESERVE list into later passes.
This workflow does not require a host-specific tool, mission API or companion skill.

Only the coordinator may admit optional fresh-context per-tier or independent
section work through the host's available native tools under the existing task
grant. Leaves return their assigned edits and stop; they never dispatch another
pass. A trail-root binds outputs, not fanout. Sectioning may run inline when the
caller needs records without delegation. Required independent isolation that is
unavailable is a disclosed blocker, not permission to switch engines or modes.

Each admitted per-tier leaf receives its own rubric, the current draft and prior
emitted ledger/PRESERVE state, not earlier reasoning transcripts. The coordinator
assigns disjoint outputs, settles all runs, reads complete results and merges in
order. Read `{skillDir}/sectioning.md` and `{skillDir}/edit-record.md` when the
caller requests section records or authorizes sectioning. These sidecars specify
text mechanics, not delegation authority. No automatic review/fixer chain follows.

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
EDITS:           rule id -> quoted span -> suggested fix
PRESERVE:        (from tier-2; carried forward, or "(none)")
CONTRAST-LEDGER: (from tier-1; what is retained, what remains)
```

Under sectioning, the merge pass edits the settled per-section text, so it adds a
fourth block. **Report it whether or not `trail-root` is supplied**: without it this is the only
account the caller gets of what the merge changed.

```
MERGE:           which section keeps the ledger allowance, and how many surplus
                 instances were rewritten positive; which whole-document tier-2
                 rules fired; how many consistency conflicts were resolved
```

Lead with the edits that matter most, and skip preamble.

## trail-root: an optional caller-supplied path

**This skill writes no files unless the caller supplies `trail-root`.** With none, both modes behave
exactly as above: review mode returns targeted edits, compose mode returns an edited draft, and
nothing appears on disk.

With an authorized `trail-root` supplied, the section artifacts described in `{skillDir}/sectioning.md`
may be written under it by their assigned owner. A read-only leaf returns full content for caller
persistence. The caller owns that path. **Never** invent one, resolve one from the repository, or ask
the user where a trail should live: that decision belongs to the calling skill.

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

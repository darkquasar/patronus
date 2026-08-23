# The liveness audit

Runs **per section**, after voicing. Per-section is the point: a piece can average acceptable
figures while one section is six identical four-sentence blocks, and that is exactly where
decay hides.

Voice decay is the failure this exists to catch: transformation front-loads where it is
cheapest and most visible, then falls back to editing what is in front of it. A live opening
and a dead fifth paragraph is the signature.

## The criteria

All read from the profile and the spine rather than invented. Each scores 0, 1 or 2, and each
**names the evidence the score rests on**.

| Criterion | 0 | 1 | 2 |
|---|---|---|---|
| Named moves present | none of the profile's moves appear | one appears | two or more appear, quoted |
| Governing metaphor | absent | mentioned once, decoratively | touched and carried further than the previous section did |
| Concreteness | abstractions dominate; no image | some concrete nouns and verbs | the section's abstract point lands on a specific image |
| Paragraph spread | every paragraph within one sentence of the same length | some variation | a long paragraph and a short one or a fragment |
| Signposts | above the corpus rate | at it | below it, connections carried by the prose |
| Assigned frame-break or aside | assigned and absent | present but perfunctory | present and doing work |
| Assigned claims | one or more not made | all made | all made, and made in this voice |

A section carrying a metaphor the spine did not assign **fails the metaphor row regardless of
how well it is written**. Compliance is a criterion, not just quality.

## Flat is a score, not an impression

A section is **flat** when it scores 0 on "named moves present", **or** when its total is 4 or
below out of 14. Either condition fires one rework.

Sections not assigned a frame-break or aside score that row at **the mean of the other six rows,
rounded to the nearest whole number, rounding 0.5 up.** The row being imputed is excluded from
that mean, or the calculation would depend on itself. An unassigned section is **not** penalised
for following the spine.

The thresholds are a starting point rather than a measurement. They are set where a section
with no moves and no metaphor fails while a restrained but live section passes, and they are
expected to move once run against real drafts. What matters is that the trigger is written
down and reproducible.

## Failure handling

**Flat sections go back once**, with the audit's findings. If the second attempt is still flat,
**accept it and flag it in the output**, naming the section and what it failed. An unbounded
rework loop burns tokens on a section the model cannot make live; a flag tells the user where
to intervene by hand.

## Rhythm numbers are reported, never scored

Median length, share past 26 words and shape uniformity, per `{skillDir}/weights.md`, are
reported **beside** the score and **never** enter it. They are guard rails, not criteria.
Where the profile carries `rhythm_source: unavailable`, report them as unavailable rather than
computing a cross-language number.

Low sentence-length variance is a fact about a text, not a defect: a deliberately clipped
passage is low-variance and may be the best paragraph in the piece. Any threshold over it would
be invented, and a gate creates a perverse gradient, teaching the pipeline to pad variance and
delete transitions to escape a proxy.

## A one-section piece

A short piece cuts to a single section, `00-preamble`, and **the pipeline runs normally over it**.
Every stage still applies: the profile, the spine, one voice subagent, this audit, and the codex
read. Stitching has one section to place and adds no cross-section continuity.

"Which section is flat" is not a question a one-section piece can answer, so report the audit for
the piece as a whole. **Do not cut a headless draft into paragraph-sized sections to manufacture a
fan-out.** The upstream cut rules forbid it, and subagents reasoning about fragments is worse than
one subagent reasoning about the piece.

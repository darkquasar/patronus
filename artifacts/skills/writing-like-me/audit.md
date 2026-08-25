# The liveness audit

## Write the findings to the file you were given

**The output is a file, and the file is the deliverable.** Write it with the file-writing tool to
the path the dispatching instruction names, beside the drafts in the run's own directory, normally
`AUDIT.md`. Write it **even where the analysis is incomplete**: where context runs short, stop
analysing and write what you have. A partial findings file is worth more than a fuller reply.

A reply reaches the caller only as your final text, so an audit that ends on a tool call arrives as
nothing at all. The file survives however the turn ends.

Runs over the **whole piece** by default, since one writer produces it. Score section by section
only where the refinement pass ran, or where a whole-piece read has already found the decay and
needs to localise it: a piece can average acceptable figures while one stretch is six identical
four-sentence blocks.

**A per-section score diagnoses where to look. It is never a set of obligations the writer owed.**

Voice decay is the failure this exists to catch: transformation front-loads where it is
cheapest and most visible, then falls back to editing what is in front of it. A live opening
and a dead fifth paragraph is the signature.

## The criteria

All read from the profile and the spine rather than invented. Each scores 0, 1 or 2, and each
**names the evidence the score rests on**.

| Criterion | 0 | 1 | 2 |
|---|---|---|---|
| Named moves, earning their use | none appear, **or** those present are ornamental | one appears and is load-bearing | two or more appear, quoted, each doing work the plain sentence could not |
| Governing metaphor | absent | mentioned once, decoratively | touched and carried further than the previous `metaphor_locations` section did |
| Concreteness | abstractions dominate; no image | some concrete nouns and verbs | the section's abstract point lands on a specific image |
| Paragraph spread | every paragraph within one sentence of the same length | some variation | a long paragraph and a short one or a fragment |
| Signposts | above the corpus rate | at it | below it, connections carried by the prose |
| Assigned frame-break or aside | assigned and absent | present but perfunctory | present and doing work |
| Assigned seam | absent, discharged only by an adverbial hedge, **or resolved anywhere in the piece** | present and standing, but immaterial: resolving it would not change the conclusion | present, standing, and material |
| Assigned claims | one or more not made, per its type | all made | all made, and made in this voice |
| Transformation (edit mode) | the section is materially the input: no substantive operation | one substantive operation | two or more, and the section reads as this author rather than as the source tidied |

A section carrying a **governing** metaphor the spine did not assign **fails the metaphor row
regardless of how well it is written**: a second image competing to govern the piece is the failure.
**Local imagery that serves one passage and is done is never scored as an unassigned metaphor**, and
scoring it that way is what strips the figurative register out of every unallocated section.

**The metaphor row applies only to sections in `metaphor_locations`.** A section outside that list
is not assigned the image, so the row is excluded from its total under the unassigned-row rule
below. Leaving the image alone there is compliance, and scoring it as absence is what drives the
decorative recurrence the allocation exists to prevent.

**Route is a plan, not a debt.** A section is not required to prove its path was indispensable.
Requiring that taught every section to manufacture a turn that later prose visibly used, which is the
same distributed proof obligation as the old moves gate. Dead structure is caught by the pressure
progression, claims fidelity and performed sameness instead, none of which asks a section to leave
forensic evidence of how it travelled.

**What counts as a substantive operation**, for the transformation row. Percentage-unchanged is the
wrong measure: synonym substitution moves it a long way without doing any editorial work, and a
single reconceived paragraph may move it very little. Count operations, not characters:

| Operation | Present where |
|---|---|
| mood changed | a declarative became interrogative or concessive, or the reverse |
| architecture changed | paragraphs merged, split, reordered, or a minor point subordinated |
| enumeration dismantled | items were **chosen because they advance the claim** and the rest cut, a hierarchy was introduced, an item was interpreted rather than named, or the inventory was replaced by the principle that generates it |
| reader relation introduced | the section turns to address, question or implicate the reader where the input did not |
| route changed | the claim lands by a different path than the input took to it |

**Converting a bulleted list to a comma-separated list is not dismantling it.** A comma-string is
still an inventory: the reader receives the same accumulation of nouns, and the prose effect is
identical. Scoring the typographic conversion is what teaches a pipeline to produce paragraphs that
could be pasted back into bullets without loss. The test is whether anything was **chosen**.

**A rewritten sentence is not by itself an operation.** Rephrasing every sentence while preserving
the input's mood, architecture and route is the failure this row exists to catch: it is what "the
voice stage left the source paragraphs untouched" looks like from the inside, and it can score well
on every other row, since concreteness, spread and signposts are properties the input already had.

In compose mode the row does not apply: the input carried no voice, so there is nothing to
transform against.

**The claims row reads each claim's `type`.** A claim is made when it lands as its type requires:
an `assertion` affirmed declaratively, a `question` raised, a `concession` granted without becoming
the conclusion, a `hypothesis` carried (hedged is conforming, not weakened), an `evidence` specific
surviving without being promoted into a thesis. An untyped claim is an `assertion`.

**An interrogative rendering of an `assertion` does not by itself make it.** Where a voicer records
a `question_surface`, the claim is made only where a `landing_surface` also lands it declaratively,
here or in another section. A question with no landing anywhere in the piece is a claim not made,
and the whole-piece coverage pass is where that is finally decided.

**A move reproduced past its meaning scores 0 on the moves row regardless of how well it is
written.** Presence is not the test. Test each instance by rewriting it flat, in the plainest
sentence carrying the same information, and asking what the flat version no longer does.

**What counts as loss.** The move is load-bearing where the flat rewrite drops one of these:

| Loss | The move was carrying |
|---|---|
| a distinction | the two halves name genuinely different things, and flattening merges them |
| a claim | the shape asserts something the plain sentence does not say |
| an image the argument later uses | a later passage refers back to it |
| the reader's position | it turns to address, question or implicate the reader, and flat prose does not |
| a stance | hedging or bluntness the plain version neutralises |

Where the flat rewrite loses **only** rhythm, emphasis or symmetry, the move was ornament and
scores 0. Sounding better is not carrying something.

**Frequency is not the test either, and a counter would get this backwards.** A move may recur
several times in one paragraph, each instance passing the test above on its own, and that passage
can be the strongest in the section. One instance that passes none of it still fails: two clauses
set in opposition where the second only adds a fact, or restates the first at another scale, is
parallelism doing rhetorical work the content did not earn.

**Three instances can pass where one fails**, so score the instance rather than the tally.

Where `~/.claude/patronus/voice/examples.md` exists, read its
`## The moves row: earned against ornamental` heading for a passing and a failing instance in this
author's own register. **It illustrates and never governs**: the criteria and thresholds here hold,
and an example that appears to contradict one is being misread.

## Cheap proof surfaces

Three shapes recur across runs, survive every prohibition written against them, and share one
cause: **each is the cheapest available way to show that a claim was discharged.** A list performs
completeness. A mirrored opposition performs precision. A flat declarative performs confidence. All
three are cheaper than the thinking they stand in for, and a writer holding an obligation reaches
for whichever is nearest.

**Do not gate these numerically.** A rate produced nine identical "I think" tics; a counter produced
ornamental moves. An assertiveness threshold teaches lexical fog, where the verdict survives wearing
"arguably"; a per-paragraph list cap teaches paragraph-splitting and semicolons. **Report passages,
never counts**, and return them for reconception rather than repair. Hedging a declarative, merging
a list or softening an opposition preserves the non-thinking underneath.

### The shape is not the defect. Test the function.

The construct this pipeline has banned for three runs appears, unaltered, in the passage the author
named as the only one that works:

> "An unmapped forest is not empty. It is just not answerable, and a place becomes answerable at the
> moment somebody writes down enough about it that a stranger can ask where something is and be told."

Structurally that is "The runner is incidental. The approval is not.", which he rejected. The
syntax is identical and the function is opposite:

| | the negation | what the next sentence does |
|---|---|---|
| earned | "An unmapped forest is not empty." | travels into new territory: answerability, the stranger who can ask |
| cheap | "Skip it and you do not get a weaker record." | delivers the verdict immediately and closes |

**A negation is a hinge or a verdict, and only its consequence tells you which.** Banning the shape
catches both instances equally, which is why three runs of prohibition changed nothing. Test by
counterfactual, and require the auditor to name evidence rather than to feel a difference:

- **Negation**: replace it with its positive proposition, or delete it. **Does the passage still
  introduce the same distinction, mechanism or consequence, with the same force?** If yes, it was
  emphasis. If no, name the distinction the negation creates and **point to where the piece later
  uses it**. The payoff may arrive two paragraphs on, so do not judge on the next sentence alone.
- **Opposition**: do the two poles create a distinction later reasoning **uses**, or restate one
  conclusion with reversed polarity? Do not require a scale: some real distinctions are categorical
  ("authenticated but not authorised" has no useful continuum), and demanding a gradient there
  produces vagueness.
- **Enumeration**: does the selection, ordering or interpretation of the items change the reader's
  model? Examples can carry scope, credibility or consequence without changing the headline
  proposition, so semantic redundancy alone does not condemn a list. What condemns it is items
  arriving unchosen and uninterpreted. **A paragraph that admits its own inventory while still
  delivering it has not been redeemed by the admission**: the reader still reads the list.
- **Closure**: does the sentence sound more certain than the reasoning around it has earned? Not
  whether a hedge is present. What warrants this degree of closure?

### Vagueness forces confidence

The author's own repair shows the generator:

> "Passing lint proves a rule is structurally acceptable. It leaves open whether the telemetry exists"

became

> "A rule that passes lint proves that it is structurally acceptable, it's what lint is there for. But
> unless you bake in data source validation as part of linting, you won't know whether telemetry
> exists at all"

He did not soften it. He **added a mechanism**, and the conditional fell out of the mechanism. The
original was assertive *because it was abstract*: it held nothing specific enough to be conditional
about.

So where a passage reads as overclaimed, **test for an absent mechanism before prescribing a hedge**.
Overclaiming has other sources too: missing scope, unacknowledged counterevidence, causal confusion,
or plain unsupported certainty. Diagnose which before returning it. Returning any of them for a
qualifier produces vagueness, which is the worse defect.

### Merit-announcements and intensifiers

Two lexical habits substitute a claim *about* the material for the material:

- **The merit-announcement**: "worth naming", "worth noting", "earns its place", "the part worth
  copying". Test by removal: **delete the phrase expressing merit. If the proposition and its warrant
  survive unchanged, it was an attention announcement.** If deleting it removes an evaluation the
  piece is actually arguing, keep it. ("Whether the second surface is worth what it costs" is the
  argument; "a shape worth naming" is throat-clearing before the shape.)
- **`exactly`**: test truth-conditionally. **Delete it. If the sentence's factual commitment changes
  and the surrounding evidence warrants that precision, keep it.** If only emphasis weakens, report
  it. In a repair construction ("not X, exactly"), keep it where it marks X as an approximate but
  inadequate category and the sentence then supplies a better one.

Both are reported as passages, and **neither is a find-and-replace**. Where the announcement is
compensating for weak material, deleting the phrase leaves the weakness and hides the symptom: return
the passage for reconception instead.

## Flat is a score, not an impression

A section is **flat** when its total falls at or below **a fifth of its available maximum**.

**A section with no named move is not thereby flat.** That rule was itself a distributed proof
obligation: every section had to exhibit a recognisable move, so a writer with nothing to show
manufactured one, which is the ornamental compliance the moves row then scored 0 for. A deliberately
plain section carrying the argument cleanly is a legitimate finished state, and in a piece that
modulates, some sections must be plain. **Where a section scores 0 on moves, report it and say
whether the plainness reads as chosen or as absence**; let the total decide whether it reworks.

**A row for something the section was not assigned is excluded, and the maximum scales down.**
Of the nine rows, five are conditional and four always apply:

| Always | Conditional on |
|---|---|
| named moves | metaphor: the section is in `metaphor_locations` |
| concreteness | frame-break or aside: one is present to judge |
| paragraph spread | seam: one is present to judge |
| signposts | claims: claims were assigned |
| | transformation: the run is in edit mode |



Worked, for an edit-mode section assigned no frame-break, no seam, outside `metaphor_locations`,
carrying two claims:

| | rows | maximum | flat at or below |
|---|---|---|---|
| all nine apply | 9 | 18 | 3 |
| this section | 6 | 12 | 2 |

Six rows: the four that always apply, plus claims and transformation. **Round the threshold down**,
so a maximum of 14 gives 2 rather than 2.8. Nothing is imputed.

Imputation is what this replaces, and it was a defect: scoring an unassigned row at the mean of the
others **rounded up** handed a section that scored 1, 1, 1 on three rows a 2 for a row it was never
asked to fill. The weakest sections gained the most. An unassigned section is still **not** penalised
for following the spine, which is what exclusion achieves without inventing a score.

The thresholds are a starting point rather than a measurement. They are set where a section
with no moves and no metaphor fails while a restrained but live section passes, and they are
expected to move once run against real drafts. What matters is that the trigger is written
down and reproducible.

## The piece-level pass

Per-section scoring cannot see a failure that lives in the composition. Eight sections that each
individually pass compose into a draft with no frame-break, no reader address and no unresolved
question in it anywhere, and every one of them scored well.

**Run this once, after stitching, over the finished piece.** It does not replace the per-section
rows, which localise a failed *execution*; this catches a failed *allocation*.

| Check | Fails when | On failure |
|---|---|---|
| Signature effects | an effect in `signature_set` is absent from the piece entirely, or every instance of one is ornamental | reported |
| Frame-breaks, reader address, asides | none of a named act appears anywhere in the piece | reported |
| Unresolved seams | the piece resolves everything it raised, or a seam is discharged only by an adverbial hedge | **blocks** |
| Claims coverage | a claim no section made as its type requires, including an `assertion` opened as a question and never landed | **blocks** |
| Inherited sections | a section scored 0 on transformation | **blocks** |
| **Pressure progression** | a section leaves the governing pressure unchanged: it restates, supports or adds another reason | **blocks** |
| **Overstatement** | a rendering asserts more than its claim and that claim's `evidence_bounds` license | **blocks** |
| **Performed sameness** | sections share one route **and** perform it identically: the same entry/turn/exit mechanics, the same section shape, or flat-replacement equivalence | **blocks** |
| **Unallocated inventories** | an inventory of five or more items that no section's `required_specifics` names | **blocks** |
| **Post-voice compliance** | an em-dash or punctuation inside a closing quote (absolute, no comparison needed); a machine signpost or a second mirrored construction **that the upstream edit records show the tiers removed** | **blocks** |
| Route concentration | more than half the sections share one route | reported |
| Register modulation | adjacent sections do detectably the same reader-work throughout, or a `device` never returns to its `grounding` | reported |
| Paragraph openers | one opener word in more than a quarter of paragraphs, or more than a third of paragraphs opening on a back-reference to the one before | reported |
| Hedge convergence | one hedge lexeme in more than a third of sections | reported |
| Section architecture | the same shape repeated across most sections | reported |
| **Wandering** | the piece never leaves its own frame: no passage departs to a concept outside the argument and returns changed | **reported, never blocking** |

**Effects are not counted.** `signature_set` names what the piece should carry, not how many times.
A counter tells a writer to manufacture instances, and manufactured instances are what the removal
test below then scores as ornament. Ask whether the effect is present and earned.

**Every signature instance takes the removal test**, the same one the moves row uses: delete it and
ask what the piece no longer does. A reader-directed "Think about that" claiming `reader_turn`, or a
stray image claiming `figurative_grounding`, fails it. Ornament satisfies no allocation.

**Seams must be distinct.** Two allocations discharged by two phrasings of the same open question
count once. Uniqueness is judged on the `undecided` proposition, not on the wording.

**A blocking failure returns the piece to the stage that owns it** and does not fire a per-section
rework: a missing allocation is the spine's or the stitching's. Where the second attempt still
fails, **accept it and flag it prominently**, naming the check and the sections that could have
carried the missing thing. The bound matters more than the gate here, and an unbounded loop over a
piece-level miss is worse than a flagged draft.

**Frame-break, reader address and aside are counted separately.** One insertion does not discharge
three allocations: an aside is not necessarily reader-facing, and a frame-break is not necessarily
an aside.

**Route concentration is metadata; performed sameness is the defect.** A piece may legitimately derive
throughout: a causal postmortem, a cumulative technical argument. Eight sections labelled `derivation`
is a fact about the labels. Eight sections each opening on a proposition, listing support and closing
categorically is the regularity a reader feels. Report the concentration and block only on the second.
A one-section piece concentrates at 100% by arithmetic and is never blocked for it.

**The pressure check is what separates a piece from a tour.** Every section in `running_order` appears
in `governing_pressure.progression`, and each entry says what became harder, stranger or more
consequential. A section that intensifies nothing, complicates nothing and releases nothing is a
polished object: true, well made, and skippable. Ask directly whether a reader could stop after each
section and feel finished. Where they could, the piece stopped pressing there.

**Overstatement is not measured by counting qualifiers.** Do not hunt for "usually" or "more like":
compare each rendered assertion against its claim and that claim's `evidence_bounds`. A claim may be
legitimately universal and need no softening; a bounded claim rendered as a flat universal is the
failure. Requiring a qualifier where the evidence supports the universal produces vagueness, which is
the opposite defect. **Scope words are how a rendering stays inside its bound, never a quota.**

**Inventories are judged semantically, not by length.** An enumeration earns its place where its
completeness is the argument, which is what `required_specifics` records at spine time. The check
here is the total: several individually-defensible inventories compose into a catalogue, and only the
finished piece shows how many there are.

**The compliance gate reports; it never rewrites.** Two different things sit in this row and only one
is a regression. House mechanics are **absolute**: an em-dash is a violation whether or not any tier
ever removed one, and needs no comparison. A machine signpost or a second mirrored construction is a
**regression only against the record**: read the upstream `edits.yaml` to tell "the voice stage put
this back" from "this survived every stage", because the two call for different fixes and the gate
cannot tell them apart without looking.

Where the profile exempts a move from a tier rule, **classify the instance and report it**; do not
count it as a regression, and do not remove it. **Findings go back to the voicer**, which holds the
profile, the claims and the route. A stage that cannot read the profile must never edit the finished
prose: it would lawfully delete a corpus-backed move the voice stage lawfully introduced, and the
piece would have two authorities over its own voice. Where the profile exempts a move from a tier
rule, the gate classifies the instance and reports it rather than removing it.

**Wandering is reported and never enforced, deliberately.** The thing being looked for is a passage
that leaves the argument's own frame, goes to a concept related but outside it, and brings the reader
back holding a new lens on the same mechanism. One such passage in a piece has been worth more to
this author than any number of executed allocations.

It is reported rather than gated because **we do not yet have a measure that separates a real
excursion from fancy dress**, and a gate without that measure would buy five imitations of the one
passage that worked. Allocating excursions in advance produces exactly that: told to depart to
alchemy and return to validation, a writer will produce crucibles and lead into gold and announce
that linting is similar. So the spine names where literal exposition is unlikely to suffice and
leaves the destination open, and this row observes what actually happened.

Report, per instance: **where it left, what it went to, where it came back, and what the reader can
now see about the mechanism that they could not before.** Where the fourth is missing, the passage
decorated rather than wandered, and say so. Where the piece has none at all, say that too: a piece
that never leaves its own frame is not thereby broken, and this is the row that notices it has not.

Accumulate these observations across runs. **When there is enough evidence to say what separates the
earned excursion from the decorative one, this row can become a criterion. Until then it is an
observation, and treating it as a target would teach the pipeline to manufacture departures.**

**Hedge convergence needs this pass to be visible at all.** A run allocated its hedges per section
and met its rate; nine of nine came back as the identical phrase "I think", each voicer reaching for
it independently with no sight of the others. Every section passed. Only the composed piece shows it.

**Report the piece-level result beside the per-section table**, and where a check fails, name the
sections that could carry the missing thing. A failure here is the spine's or the stitching's, not
one section's, so it does not fire a per-section rework.

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

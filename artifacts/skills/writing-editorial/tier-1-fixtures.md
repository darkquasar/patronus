# tier-1.3 fixtures

Worked cases for the mirrored-swap rule. Each is a passage, never a bare sentence: step 2 asks what
a reader has been led to expect at this point, which only the surrounding text can answer.

Read the candidate in **bold**. `STEP` is the step the algorithm reaches before deciding.

**A bold candidate is a specimen under test, not a model to copy.** Most of them are prose this rule
kills: every case in the failing set, case 11, cases 12 through 14, case 16, and all three in case
19 are rewritten positive, and each carries its rewrite. Only four bold spans in this file survive as
written: case 10, case 15, the first candidate in case 17, and the second in case 18. Cases 17 and 18
are the ones to read carefully, because a candidate that passes every test is still rewritten there
once the budget is spent. The verdict is always on the `STEP` line. Never infer it from the prose.

## Staged reveals: the failing set

All nine resolve at step 2. None consumes the budget.

```
1.  The redesign shipped last week to a chorus of praise.
    **It's not bold. It's backwards.**
    STEP 2 -> staged reveal. Fails (a): no reader was holding
    "bold" as a belief to correct; it was introduced to be knocked
    down. Rewrite: "The redesign reverts three years of layout work".

2.  The tube runs directly into the stomach.
    **Feeding isn't nutrition. It's dialysis.**
    STEP 2 -> staged reveal. Fails (a). Rewrite positive: name what
    the tube actually does.

3.  The new build lands in half the time.
    **The headline isn't the speed. The real story is Y.**
    STEP 2 -> staged reveal. Fails (a) and (c): "the real story"
    announces a reveal rather than correcting a live misreading.

4.  Most engineers blame the compiler first.
    **Half the bugs you chase aren't in your code. They're in your head.**
    STEP 2 -> staged reveal. Fails (b): the mirror is an aphorism, and
    holding the negated belief changes nothing the reader does next.

5.  The invoice arrived on Tuesday.
    **This isn't a price increase. It's a betrayal of trust.**
    STEP 2 -> staged reveal. Fails (a). "of trust" completes the noun
    phrase; it does not explain why the increase amounts to betrayal.

6.  The handset ships in three colours.
    **This isn't just a phone. It's a statement.**
    STEP 2 -> staged reveal. Fails (a) and (b).

7.  The proposal has been circulating since March.
    **This isn't about X, it's about Y.**
    STEP 2 -> staged reveal. Joined topology; same verdict.

8.  Customers cancelled in three waves.
    **It's not the price. It's not the features. It's the trust.**
    STEP 2 -> staged reveal. Countdown topology. Two negations
    stacked before one reveal.

9.  The crash reproduces on every third boot.
    **Not a bug. Not a feature. A fundamental design flaw.**
    STEP 2 -> staged reveal. Countdown topology.
```

## The two-lead-in pair: same candidate, opposite verdicts

This pair is the sharpest check in the set. The candidate sentences are **identical**. Only the
lead-in differs, and the verdicts differ because of it.

```
10. The client blocks until the handler returns.
    **The service is not synchronous. It is asynchronous.**
    STEP 3 -> LIVE CORRECTION. (a) the lead-in invited exactly the
    synchronous reading; (b) it changes how they write the calling
    code; (c) nothing before the candidate had excluded it.
    Kept, if it is the first live correction in the piece.

11. Every call returns a promise.
    **The service is not synchronous. It is asynchronous.**
    STEP 2 -> staged reveal. Fails (c) ALONE: "returns a promise"
    already excludes the synchronous reading, so the negation
    corrects nobody. Rewrite: "The service is asynchronous".
```

## Each condition failing alone

So a bug in any single condition is caught rather than masked by another.

```
12. FAILS (a) ONLY.
    The build now runs on the vendored toolchain.
    **The parser is not hand-rolled. It is generated.**
    (b) holds: knowing which it is changes whether you edit the
    grammar or the source.
    (c) holds: the toolchain sentence says nothing either way
    about how the parser was produced.
    (a) fails: nothing has raised the question, and a reader
    arriving here holds no belief about it to correct. The
    negated half was introduced so it could be knocked down.
    -> staged reveal. Rewrite: "The parser is generated".

13. FAILS (b) ONLY.
    The logo sat in the top-left corner for years.
    **The mark is not navy. It is midnight blue.**
    (a) holds: a reader could easily have believed navy.
    (c) holds: nothing already excluded navy.
    (b) fails: nothing the reader does or understands next changes.
    -> staged reveal. Rewrite: "The mark is midnight blue".

14. FAILS (c) ONLY.
    Every write appends a new version to the log, and the previous
    version stays readable at its own offset.
    **Updates are not destructive. They append a new version.**
    (a) holds: destructive update is what a reader brings to any
    store, from every database they have used before.
    (b) holds: it changes how you reason about recovery.
    (c) fails: the lead-in already said, in so many words, that the
    previous version survives. The belief was live on arrival and is
    closed by the time the candidate speaks, so the negation
    corrects nobody.
    -> staged reveal. Rewrite: "Each update appends a new version".
```

## `not X but Y because Z`, resolving in opposite directions

```
15. Z EXPLANATORY -> passes.
    The vendor raised the rate midway through the term.
    **It isn't a price increase. It's a betrayal, because the
    contract fixed this rate for three years.**
    STEP 3 -> live correction. The because-clause does work no
    reader could infer from the mirror alone.

16. Z RESTATING -> fails.
    The vendor raised the rate midway through the term.
    **It isn't X. It's Y, which is the real story.**
    STEP 2 -> staged reveal. The trailing clause restates the reveal
    instead of explaining it. Appended words do not convert a foil
    into a correction.
```

## The budget

```
17. TWO LIVE CORRECTIONS IN ONE PIECE.
    The health check posts its result to the dashboard, where the
    release manager reviews it before signing off.
    **The gate is not advisory. It blocks the rollout.**
    Downstream, the retry wrapper reads as belt-and-braces.
    **The retries are not redundant. They are the only thing
    covering the broker's at-most-once delivery.**
    FIRST -> step 3, LIVE CORRECTION, kept. It is the first in
    the piece.
    SECOND -> step 3, passes (a), (b), and (c) on its own: a
    reader who has just been told the wrapper looks redundant
    holds exactly that belief, correcting it changes whether they
    delete the wrapper, and nothing prior excluded it. It is
    rewritten positive anyway, because the budget is one RETAINED
    live correction per piece: "The retries are the only thing
    covering the broker's at-most-once delivery".
    This is the case that shows the budget binds a passing
    candidate, not merely a failing one.

18. ONE STAGED REVEAL PLUS ONE LIVE CORRECTION.
    The migration ran clean on staging last Thursday.
    **This isn't a schema change. It's a data change.**
    Rolling back is a one-line command that repoints the alias at
    the previous snapshot.
    **The rollback is not instant. It replays four hours of
    write-ahead log.**
    FIRST -> step 2, staged reveal. Fails (a): no reader was
    holding "schema change" as a belief; it was introduced to be
    knocked down. Rewritten positive: "The migration rewrites
    every row in place". It never reaches step 3, so it does NOT
    consume the budget.
    SECOND -> step 3, LIVE CORRECTION, kept. (a) the one-line
    command invited exactly the instant-rollback reading; (b) it
    changes whether you plan a maintenance window; (c) nothing
    prior closed it. The budget is still unspent when it arrives.
    This is the case that distinguishes a correct budget from one
    that counts both classes. A budget that counted the reveal
    would wrongly rewrite the correction.

19. THREE STAGED REVEALS, NO LIVE CORRECTION.
    The dashboard went out to the whole company on Monday.
    **This isn't a reporting tool. It's a culture change.**
    Adoption sat at nine percent by Friday.
    **The problem isn't discoverability. It's trust.**
    Two teams had already built their own.
    **Not a rollout. Not a pilot. A rediscovery of what people
    were already doing.**
    ALL THREE -> step 2. The first two fail (a) and (b): the
    mirrors are aphorisms and no reader held the negated belief.
    The third is countdown topology, same verdict. All three are
    rewritten positive. The budget never engages, because nothing
    reached step 3 to spend it on.
```

## Out of scope: the rule never fires

```
20. NO MIRRORED REPLACEMENT.
    **They work, hence why I am wary of them.**
    STEP 1 -> out of scope. A negation with no mirrored replacement
    is not this pattern. Not flagged.

21. NO NEGATION.
    **The batch job runs hourly, whereas the stream is continuous.**
    STEP 1 -> out of scope. Contrast without negation. Not flagged.

22. INSIDE A QUOTATION.
    She wrote: "It's not bold. It's backwards."
    STEP 1 -> out of scope. Never edited. Quoted text stays exactly
    what was said, whatever the rule would say about it unquoted.
```

# weights: rhythm calibration for the voice pass

Read this before rewriting, and check the finished draft against it. Applying a voice flattens rhythm on its own, and this file is the counterweight. It is shipped content, not your corpus.

Two things go wrong, and only the first is about length.

## 1. The tails get clipped

Voice work rewards punch, so a pass reaches for compression: it splits qualifications into separate sentences and simplifies clauses. There are many natural prompts to break a sentence and almost none to join two, so the spread narrows from both ends and every sentence drifts toward the middle.

**The corpus's centre is a target. Its extremes are permission, not a quota.** A corpus of 284 sentences has a longest that a 16-sentence draft will not contain, and should not manufacture: that sentence is a one-in-284 event, and demanding it produces a conspicuous long sentence the thought never earned. What the maximum tells you is that sentences of that length are *allowed* in this voice.

What to aim at:

- Keep the corpus's typical sentence length, and do not drift below it.
- Carry its share of long sentences. If roughly a fifth run past 26 words, a 16-sentence draft wants around three of them.
- Keep the genuinely short sentences too, because clipping the bottom tail flattens as surely as clipping the top.
- Let one sentence reach the corpus's upper range, near its 90th percentile rather than its maximum.

**Drift runs both ways, though only one direction is common.** A draft materially narrower or wider than the corpus has drifted. Narrowing is the observed failure and the one to watch; a manufactured 60-word sentence is the other.

**When the long sentences have gone, find where you split a claim from its qualification and rejoin them.** Do not add decorative fragments or alternate short and long to manufacture a spread.

**Never shorten a structurally sound long sentence to regularize the draft.** Long sentences are part of the voice rather than defects awaiting simplification, and in reasoning prose they are usually where the reasoning lives.

Long sentences come from a small set of moves, so reach for the move rather than the word count: a claim followed by two or three subordinated qualifications; an enumeration held in one sentence because its items are parallel; a mechanism stated and its consequence named in a trailing clause.

## 2. The shapes repeat, which no length rule catches

**Varied lengths with identical architecture still reads monotonous.** This is the failure a length-only check passes clean. Consider:

> It is idempotent. Running reconciliation twice changes nothing the second time, because the second pass finds no diff. This is what makes "redeploy everything" stop being frightening. A deploy-on-merge pipeline has no such guarantee, which is why teams that run one end up afraid of their own tooling and start hand-scoping deploys to the rules they touched.

Lengths of 3, 15, 9, and 30 words: an excellent spread that satisfies every rule above. It still plods, because all four sentences are built the same way. Each opens on its subject and runs subject, verb, object, and both long ones append their reasoning as a trailing `because` or `which is why` clause. The variance is in the lengths and nowhere else.

Scan consecutive sentences for these, and vary whichever has gone uniform:

| Dimension | Uniform when |
|---|---|
| Opening | every sentence starts with its subject |
| Where the weight sits | the main clause always comes first and qualification always trails |
| Clause count | every sentence is one main clause plus one dependent |
| Connector | the same joint (`because`, `which is why`, `so`) does the work each time |
| Mood | every sentence declares; none asks, conditions, or commands |

The repairs are structural. Front a subordinate clause so a sentence opens on its condition rather than its subject. Fuse two related sentences into one with a semicolon or a colon. Put the consequence first and the mechanism after. Let one sentence carry three parallel items instead of appending a reason. Ask a real question where the argument turns.

**Three consecutive sentences sharing an opening pattern or a clause structure is the threshold.** At that point the shape has become the rhythm, whatever the lengths are doing.

## What not to do

**A short/long alternation is itself a pattern.** Vary the variance.

**Three or more same-shape fragments in a row is staccato drama, not variance.** Keep the one fragment that earns its emphasis, and fold the rest into ordinary sentences.

**Uniformity is the fault, not length.** A run of same-length sentences reads robotic at any length, and the repair is to widen the spread, never to shorten the run.

## Report what you found

Say, in a line: the corpus's typical length and long-sentence share, the same two for your output, your longest sentence, and any shape dimension you had to vary. That makes a narrowed draft visible to the audit instead of leaving it to be noticed later by a reader.

Structural regularity is a stronger signal than vocabulary. A draft in the author's words with the rhythm flattened still reads as machine-made.

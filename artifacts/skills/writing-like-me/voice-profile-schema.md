# The voice profile

Location: `~/.claude/patronus/voice/voice-profile.md`, beside the corpus. Derived from the
corpus, cached, and the user's to edit.

**Every entry is a named move with corpus evidence attached.** Adjectives are not entries:
"playful" is unactionable and uncheckable, while a move with quotations can be both followed
and audited.

## Shape

```markdown
## Move: coins a term, then uses it as established
Evidence (en): "structural lag", "DAIKII", "delusion of progress",
"The Inevitable Kraken of Doom"
Frequency: ~1 per 400 words

## Move: thinks in contrast
Evidence (en): "Hunting is more about X than it is about Y" (x4 in one paragraph);
shortcuts/longcuts; signal/noise
Note: a cognitive habit, not a tic. Tier-1.3's ban does not apply to it.

## Move: mythic register on professional matter
Evidence (en): "dark magic", "archmages", "automaton-golems"

## Move: breaks frame to address the reader
Evidence (en): "Wait a second bro, are you saying...", "But is it?"

## Move: concrete image carrying an abstract point
Evidence (en): footbridges and stairs and a table where the ocean comes into view

## Rates measured from the corpus
rhythm_source: english-pool
rhythm_source_format: short-form        # what the corpus IS, not what the draft should be
logical_signposts: 0.4 per paragraph
asides_self_correction: 1 per 300 words
hedge_rate_corpus: 2.2 per 1000 words   # measured
hedge_rate_target: 3.0 per 1000 words   # author's choice for long-form; deliberately above corpus
median_sentence: 16
pct_past_26: 17
longest: 73
```

The `## Move:` heading form is what the audit joins on. A profile entry without evidence is
not a move; drop it rather than shipping an adjective.

## Extraction reads every file, whatever the language

The pool may hold Spanish material alongside English, in parallel files:

```
~/.claude/patronus/voice/
  short-form.md      long-form.md        English
  short-form-es.md   long-form-es.md     Spanish
```

The named moves are ways of thinking rather than artefacts of a language: reaching for mythic
imagery when describing institutional decay, coining a term and then using it as established,
breaking frame to address the reader, landing an abstract point on a concrete image. These
transfer, and an author's less professionally guarded writing is often where the voice is
clearest.

Two constraints on how they are used:

- **Evidence quotations stay in their source language**, labelled with it: `Evidence (es):`.
  Translating evidence would fabricate quotations the author never wrote.
- **Rhythm numbers come from the English pool only.** Spanish runs longer per clause, so
  pooling sentence-length distributions across languages produces a target matching neither.
  **With no English pool, set `rhythm_source: unavailable` and report the rhythm guard rails
  as unavailable** rather than computing a misleading number.

Cross-lingual voice transfer is real but weaker than same-language transfer. Where both pools
are substantial, the matching-language pool leads and the other supplements.

## Cached or fresh

At run start, ask:

```
Use the cached voice profile, or re-extract fresh from the corpus?
(fresh surfaces different moves; cached is reproducible)
```

Fresh extraction is not merely a cache miss. A second pass over the corpus surfaces different
moves, and that variance is wanted for creative work. Reproducibility is wanted when comparing
runs, or when the profile has been hand-tuned.

**A fresh extraction writes `voice-profile.<timestamp>.md` and asks before replacing the
cache. A hand-edited profile is never overwritten silently.**

With no corpus, degrade: print the resolved path, say what to put there, and run
editorial-only.

## The numbers are a guard rail, not a criterion

Median length, share past 26 words and longest sentence stay in the profile and stay in the
prompt, for the reason `{skillDir}/weights.md` gives: applying a voice narrows the spread from
both ends unless something counters it.

**They are not a success criterion.** A run may match every number and still fail the liveness
gate. **Never present distributional agreement as evidence the voice landed.**

## Cadence does not cross a format boundary

`rhythm_source_format` records what the corpus is. **When the draft's format differs from it, the
rhythm numbers are context rather than targets**, and reproducing them is a failure rather than a
success.

A corpus of short-form posts is the common case, and it carries a cadence that does not transfer.
Aphoristic writing runs on short sentences: a clipped run of five or seven is the shape of the
form, and it lands. In long-form prose the same run reads as assertion without argument, hammering
a reader who was never given room to doubt.

**From a short-form corpus, expect long-form to run longer**: a higher median, and fewer
consecutive short sentences than the corpus shows. Take diction, moves and stance from the corpus.
Do not take its cadence.

No conversion factor is given, and none should be invented. Deriving one would need a long-form
corpus by the same author, and where that does not exist a computed target would be a fabricated
number scored against real prose. State the direction, and let the voicer judge the magnitude.

This is the same rule as **the corpus supplies a voice, never a format**, arriving by way of the
numbers rather than the layout.

## What the corpus is for

**The corpus supplies a voice, never a format.** This is the distinction the whole stage turns on,
and getting it backwards produces the most common failure: a long draft chopped into the shape of
the exemplars, so a design doc comes back reading like a thread of posts.

Two things live in any exemplar, and they travel differently:

| Read from the corpus | Fixed by the spine |
|---|---|
| sentence rhythm and the length spread within a paragraph | how long the finished piece is |
| diction, register, and the words this author reaches for | what the piece has to cover |
| how a paragraph opens, turns, and lands | how many paragraphs there are |
| punctuation habits, contractions, spelling conventions | how many words it has to spend |
| stance: hedged or blunt, first person or impersonal | which claims it has to land |
| the moves, such as a concrete case before the claim | the evidence and citations it rests on |

The left column is the voice and it projects onto any length. The right column is fixed by the
claims manifest and the spine, not invented by a section's voicer.

**A corpus of short pieces still teaches long-form voice.** Rhythm is a property of consecutive
sentences, not of a word count, so it is fully present in a 120-word post and it scales without
distortion. What a short corpus cannot teach is long-form architecture: how this author sustains an
argument over ten paragraphs, when they summarize, how they hand off between sections. The spine supplies it, invented at
spine time by an agent that has just read the corpus.

## Measure the corpus; do not imagine it

**Short-form pieces are not made of short sentences, and assuming they are is the failure mode this
section exists to stop.** A corpus of posts routinely averages 17 words a sentence with a fifth of
them past 26, which is an ordinary spread for any register. A voice pass that reads "short-form" and
reaches for clipped, punchy sentences is applying a stereotype of the format rather than the voice in
front of it.

So measure the pool before voicing anything: its typical sentence length, its share of sentences past
26 words, its longest, and its paragraph spread. Put those numbers in the profile's rates block and in every
subagent's prompt,
because a model cannot aim at a distribution it was never told.

`{skillDir}/weights.md` carries what to do with them, and every subagent receives it. It also carries the
second half of the problem, which no length target catches: sentences of varied length built to an
identical shape still read monotonously.

## Corpus resolution

Resolve the pool for the target form. First hit wins:

1. `$PATRONUS_VOICE_DIR/<genre>.md`, when the environment variable is set;
2. `~/.claude/patronus/voice/<genre>.md`, the default user-owned location;
3. the shipped stub, which contains no exemplars and triggers the empty-pool path.

Create neither the directory nor the files. On a first run with no corpus, print the resolved path
you looked for and what to put there, then continue in degraded mode. Corpus setup stays an explicit
user act, and upgrades stay incapable of touching it.

**The pool matching the target form wins whenever it has exemplars.** Long target with a populated
`long-form.md` uses `long-form.md`, and the short pool is not consulted. The reason is evidentiary,
not architectural: a matching pool shows how this author's sentences behave in sustained reasoning,
which is the part of the voice a short pool can only be projected for. Architecture comes from the spine in every case. The other pool is used only when the
matching one is empty, and then it is projected rather than copied.

| Target | long-form.md | short-form.md | Behaviour |
|---|---|---|---|
| long | populated | either | **use `long-form.md`.** Matching form, so it shows the voice at the target length |
| long | empty | populated | **use `short-form.md`, projected.** A supported path, not a degraded one, so it needs no permission. Voice from the corpus, length and structure from the spine |
| short | either | populated | **use `short-form.md`** |
| short | populated | empty | **use `long-form.md`, projected.** Same rule in reverse: take rhythm and diction, not the long piece's architecture |
| any | empty | empty | skip the voice stage, run the editorial tiers only, and say the pipeline ran in editorial-only mode |
| any | unreadable | unreadable | report the path and the error, treat as empty, do not fail the run |

Say which pool was used either way. Where the pools differ in what they can teach, the reader should
know which one shaped the output.

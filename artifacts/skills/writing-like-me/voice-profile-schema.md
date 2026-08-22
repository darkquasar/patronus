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
logical_signposts: 0.4 per paragraph
asides_self_correction: 1 per 300 words
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

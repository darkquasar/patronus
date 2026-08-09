# weights: rhythm calibration after the voice pass

Applying an author's voice can flatten rhythm on its own, so the variance calibration is reapplied
here, after the voice pass, even though tier-2 already carries it. This is shipped content, not your
corpus.

```
TARGET: variance, not a length.
  - sentence lengths should range roughly 3 -> 30+ words
  - a run where most sentences land 15-25 words reads robotic
  - paragraphs: some one sentence, some six or more
  - read-aloud test: if a TTS engine could read it without
    sounding odd, it is too uniform

ANTI-RULE: a short/long alternation is itself a pattern.
  Vary the variance.

ANTI-RULE: three or more same-shape fragments in a row is
  staccato drama, not variance. Keep the one fragment that
  earns its emphasis; fold the rest into ordinary sentences.
```

Structural regularity is a stronger signal than vocabulary. A draft in the author's words with the
rhythm flattened still reads as machine-made.

# weights: rhythm calibration after the voice pass

Applying an author's voice can flatten rhythm on its own, so the variance calibration is reapplied
here, after the voice pass, even though tier-2 already carries it. This is shipped content, not your
corpus.

**The corpus outranks every number below.** Where you have measured the author's own distribution,
that is the target, and these are the fallback for when you have not. An author whose median sentence
runs 15 words is not writing robotically; they are writing like themselves. Never push a draft away
from the corpus's centre to satisfy a generic band.

```
TARGET: the corpus's own distribution.
  - match its mean and median sentence length
  - match its share of sentences past 26 words
  - match its paragraph lengths, short and long
  - reproduce its full range, including the longest
    sentences. Those go first when a pass drifts.

FALLBACK, only where no corpus was measured:
  - sentence lengths ranging roughly 3 -> 30+ words
  - paragraphs: some one sentence, some six or more
  - read-aloud test: if a TTS engine could read it without
    sounding odd, it is too uniform

ANTI-RULE: uniformity is the fault, not length. A run of
  same-length sentences reads robotic at ANY length, and the
  repair is to widen the spread, never to shorten the run.

ANTI-RULE: a short/long alternation is itself a pattern.
  Vary the variance.

ANTI-RULE: three or more same-shape fragments in a row is
  staccato drama, not variance. Keep the one fragment that
  earns its emphasis; fold the rest into ordinary sentences.
```

Structural regularity is a stronger signal than vocabulary. A draft in the author's words with the
rhythm flattened still reads as machine-made.

The most common way to flatten it is to mistake brevity for style: chopping long sentences because
short ones feel punchier. That narrows the spread, which is the very signal this file protects, and
it erases the author's long sentences, which are usually where the reasoning lives.

# tier-2: machine tells and surface craft

**Gate:** live whenever the prose has a reader who will judge it, or carries a claim that reader will
act on. Length is not the test. The exception list is closed: a status ping, a one-line factual
answer, a code comment, a commit subject line.
**Operation:** local, and protect. Remove tells and mark the spans that carry the author's
fingerprint. Rewriting a flagged span is in scope, including where the replacement needs words the
original did not have; the word tables below are largely instructions to do exactly that.
Restructuring the piece is not in scope, and neither is adding material the author never had, which
is what tier-2.5 governs. Aim for the smallest edit that clears the tell.
**Emits:** an EDITS list and a **PRESERVE list**. Both travel with the draft to every later stage.

## The governing standard: cluster density

**One instance means nothing. Co-occurrence is the signal.** A single em-dash, a single
rule-of-three, a single fragment: all ordinary. The same four tells in one paragraph is the
confession.

This is the standard that protects good prose from this file. Report clusters, not per-entry hits,
for everything except the three **always-fire** classes: tier-0 phrasings, tier-1A words below, and
the tier-1.3 mirrored swap.

## tier-2.1 Trope catalogue

Flag these in clusters. These are the shapes tier-0 does not already cover.

Each entry is a named shape, a one-line description of what it does to the reader, and one or two
`Avoid:` examples. This is a catalogue, not a rule set, and it is the one section of the guide exempt
from the router's "rule, reasoning, worked examples" shape, because the shapes here are recognized
rather than reasoned about. A reader who can see the pattern needs the name and the example; the
judgment lives in the cluster-density standard above, which governs the whole catalogue at once and
is where the reasoning is written.

### Sentence and paragraph shapes

The X? A Y. A self-posed rhetorical question answered immediately, for drama nobody asked for.
Avoid: "The result? Devastating." / "The worst part? Nobody saw it coming."

Anaphora abuse. Three or more clauses opening on the same words, so cadence stands in for argument.
Avoid: "They could expose... They could offer... They could provide... They could create..."

Tricolon abuse. The rule of three overused, often stretched to four or five. One tricolon is
elegant; three back to back is a pattern the model cannot stop producing.
Avoid: "Products solve problems; platforms create worlds. Products scale linearly; platforms scale
exponentially."

False ranges. `from X to Y` where X and Y sit on no real scale, so the construction implies a
spectrum with nothing in the middle.
Avoid: "From innovation to implementation to cultural transformation."

Listicle in a trench coat. A list wearing paragraph punctuation, each point opening "The first...
The second... The third...".
Avoid: "The first wall is the absence of a free, scoped API... The second wall is the lack of
delegated access..."

One-point dilution. One argument restated ten ways across thousands of words, padded to feel
thorough.
Avoid: "Each section rephrases the thesis with a different metaphor but adds nothing new."

Fractal summaries. Every subsection summarizes itself, every section summarizes its subsections, and
the document summarizes all of it.
Avoid: "In this section, we'll explore... [3000 words later] ...as we've seen in this section."

Signposted conclusions. Announcing the close rather than landing it. Competent writing does not need
to tell a reader it is concluding.
Avoid: "In conclusion, the future of AI depends on..." / "To sum up, we've explored three key
themes..."

### Tone

Here's the kicker. False suspense promising a revelation, delivering an ordinary observation. The
family includes "Here's the thing", "Here's where it gets interesting", "Here's what most people
miss".
Avoid: "Here's the kicker." / "Here's where it gets interesting."

Think of it as. The patronizing analogy, reached for by reflex, often less clear than the concept it
replaces.
Avoid: "Think of it like a highway system for data." / "Think of it as a Swiss Army knife for your
workflow."

Imagine a world where. The invitation to futurism, followed by a list of good things that follow if
the reader accepts the premise.
Avoid: "Imagine a world where every tool you use has a quiet intelligence behind it..."

False vulnerability. Polished, risk-free self-awareness performing candor. Real vulnerability is
specific and uncomfortable.
Avoid: "And yes, I'm openly in love with the platform model." / "This is not a rant; it's a
diagnosis."

The truth is simple. Asserting that a point is obvious instead of proving it, including the dramatic
reveal variant that waves away everything preceding it.
Avoid: "The reality is simpler and less flattering." / "History is unambiguous on this point."

Let's break this down. The pedagogical voice assuming a reader needs hand-holding, applied even to
expert audiences. Includes "Let's unpack this", "Let's explore", "Let's dive in".
Avoid: "Let's break this down step by step." / "Let's unpack what this really means."

Grandiose stakes inflation. Every argument raised to world-historical significance, so a post about
API pricing becomes a meditation on civilization.
Avoid: "This will fundamentally reshape how we think about everything." / "will define the next era
of computing"

Despite its challenges. The rigid formula that raises problems only to dismiss them on the same
beat.
Avoid: "Despite these challenges, the initiative continues to thrive."

Aphorism formulas. A coined maxim in the shape of wisdom, most often `X is the language of Y`,
carrying no claim a reader can check.

Hedge-stacked predictions. Two or more hedges on one forecast, so the sentence commits to nothing
while sounding measured. `could potentially`, `may possibly`, `might eventually`.

Real and actual as inflation. `the real story`, `what actually happened`, `the real problem`, used to
claim privileged insight rather than to draw a distinction.

Generic future-narrative closers. An ending that gestures at what comes next without naming
anything. `only time will tell`, `the future looks bright`, `one thing is certain`.

False exclusivity. Claiming rarity or privileged access that the piece never establishes. `few
people realize`, `what nobody tells you`, `the secret that`.

Clichéd idioms. Ready-made phrases arriving instead of a thought: `at the end of the day`, `move the
needle`, `the elephant in the room`, `low-hanging fruit`.

Numbered phase labels. `Phase 1`, `Step 3`, `Stage II` imposed on work that has no phases, so a
sequence is asserted rather than found.

Gravitas words. Vocabulary reaching for weight the subject does not carry: `profound`, `stark`,
`sobering`, `staggering`, applied to ordinary facts.

### Composition

Invented concept labels. An abstract problem-noun (paradox, trap, creep, divide, vacuum, inversion)
welded to a domain word and then used as an established term. Naming a thing to skip the argument.
Avoid: "the supervision paradox" / "the acceleration trap" / "workload creep"

The dead metaphor. One image introduced and then beaten flat across a whole piece, where a human
would use it once and move on.
Avoid: "The ecosystem needs ecosystems to build ecosystem value." / "Walls and doors used 30+ times
in the same article"

Historical analogy stacking. Rapid-fire historical companies or technology shifts assembled to
manufacture authority. Especially common in technical writing.
Avoid: "Apple didn't build Uber. Facebook didn't build Spotify. Stripe didn't build Shopify."

Content duplication. Whole sections or paragraphs repeated within one piece, which happens when the
writer loses track of what is already there.
Avoid: "Paragraph 3 and paragraph 17 are the same sentence reworded."

### Formatting, as first-class entries

Bold-first bullets. Every list item opening on a bolded phrase. Almost nobody formats lists this way
by hand.
Avoid: "**Security**: Environment-based configuration with..." / "**Performance**: Lazy loading of
expensive resources..."

Title-case headings. Headings capitalized Like This Throughout, where the surrounding document uses
sentence case.

List-label periods. A bolded list label closed with a full stop where a human writes a colon:
`**Intros.**` for `**Intros:**`. The two quoted specimens here are the pattern under discussion, not
this file using it.

Curly quotes in plain-text contexts. Smart quotes and unicode arrows in output that a person typing
in an editor would produce as straight quotes and `->`.
Avoid: "Input → Processing → Output"

Emoji in headers. Decoration standing in for structure, most visible in README and documentation
output.

Excessive structure. Headings, bullets, and tables imposed where three paragraphs of prose would
carry the argument better.

## tier-2.2 Word tiers

Four bands. The band decides what a hit means, which is what makes the list usable.

### Tier 1A, machine frequency markers

**Always replace.** A cluster of these is evidence about how the passage was produced.

| Replace | With |
|---|---|
| paradigm | model, approach, framework |
| embark | start, begin |
| robust | strong, reliable, solid |
| comprehensive | thorough, complete, full |
| cutting-edge | latest, newest, advanced |
| meticulous / meticulously | careful, detailed, precise |
| seamless / seamlessly | smooth, easy, without friction |
| game-changer / game-changing | describe what changed and why it matters |
| watershed moment | turning point, shift |
| nestled | is located, sits, is in |
| thriving | growing, active (or cite a number) |
| deep dive / dive into | look at, examine, explore |
| unpack / unpacking | explain, break down, walk through |
| bustling | busy, active |
| intricate / intricacies | complex, detailed (or name the complexity) |
| enduring | lasting, long-running (or cite how long) |
| ever-evolving | changing, growing (or describe how) |
| daunting | hard, difficult, challenging |
| holistic / holistically | complete, full, whole |
| actionable | practical, useful, concrete |
| impactful | effective, significant (or describe the impact) |
| learnings | lessons, findings, takeaways |
| thought leader / thought leadership | expert, authority |
| best practices | what works, proven methods |
| at its core | cut, and state the thing |
| synergy / synergies | describe the actual combined effect |
| interplay | relationship, connection, interaction |
| symphony (metaphor) | describe the actual coordination |
| embrace (metaphor) | adopt, accept, use, switch to |
| load-bearing *(metaphor)* | essential, critical, or say what breaks without it |
| beacon | rewrite the sentence entirely |
| keen (as intensifier) | interested, eager, or cut |
| genuinely / genuine (as intensifier) | cut, and state the fact |
| hit differently / hits different | say what specifically changed, or cut |
| marking a pivotal moment | state what happened |
| the future looks bright | cut, and say something specific or nothing |
| only time will tell | cut, and say something specific or nothing |
| despite challenges… continues to thrive | name the challenge and the response, or cut |
| complexities | name the actual complexities, or use problems / details |

Entries tier-0 already carries as vogue words or puffery are not repeated here. tier-0 fires on them
first, ungated.

Each entry covers its morphological variants, so `meticulous` covers `meticulously` and `leverage`
covers `leveraging`, **unless a variant carries a separate honest sense**.

`load-bearing` needs the hyphen, because unhyphenated "load bearing" is ordinary English (`the load
bearing down on the bridge`). It does not fire before a literal structural noun: `wall`, `beam`,
`column`, `joist`, `truss`, `member`, `footing`, `slab`, `stud`, `partition`, `masonry`, `lintel`,
`pier`, `rafter`, `girder`, `capacity`, optionally with one material or position adjective in between
(`load-bearing structural wall`). Abstract-capable nouns are excluded from that carve-out on purpose,
so "the load-bearing structure of his argument" does fire.

### Tier 1B, clarity edits

**Always replace, and never evidence of machine authorship.** Report these separately. Presenting a
wordiness fix as authorship evidence is the error this split exists to prevent. The separation is
measured, not assumed: against 257 paragraphs of verified pre-2023 human prose, these entries fire on
ordinary professional writing at a meaningful rate. `in order to`, `utilize`, `commence`,
`ascertain`, and `endeavor` are simply the words some people reach for.

| Replace | With |
|---|---|
| utilize | use |
| in order to | to |
| due to the fact that | because |
| serves as | is |
| features (verb) | has, includes |
| boasts | has |
| presents (inflated) | is, shows, gives |
| commence | start, begin |
| ascertain | find out, determine, learn |
| endeavor | effort, attempt, try |

### Tier 2, flag in clusters

Legitimate alone. **Two or more in one paragraph** is the signal.

`harness`, `navigate`, `foster`, `elevate`, `unleash`, `streamline`, `empower`, `bolster`,
`spearhead`, `resonate`, `revolutionize`, `facilitate`, `underpin`, `nuanced`, `crucial`,
`multifaceted`, `ecosystem` (metaphor), `myriad`, `plethora`, `encompass`, `catalyze`, `reimagine`,
`galvanize`, `augment`, `cultivate`, `illuminate`, `elucidate`, `juxtapose`, `paradigm-shifting`,
`transformative`, `cornerstone`, `paramount`, `poised (to)`, `burgeoning`, `nascent`,
`quintessential`, `overarching`, `quietly`, `deeply` (in significance collocations only, so
`deeply integrated` counts and `deeply nested` does not), `underpinning`.

### Tier 3, flag by density

Normal words, flagged only at saturation, **roughly 3% of total words**.

`significant`, `innovative`, `effective`, `dynamic`, `scalable`, `compelling`, `unprecedented`,
`exceptional`, `remarkable`, `sophisticated`, `instrumental`, `world-class` and its family
(`state-of-the-art`, `best-in-class`).

Each wants the same fix: replace the adjective with the specific it is standing in for. A number, a
comparison, a benchmark, or the thing that actually makes it an exception.

`verbatim` sits in this band too, and takes a different fix. It is usually redundant with its verb
(`copies X verbatim` is `copies X`), so cut it. Where the exactness marks a real contrast, name the
contrast instead: byte-for-byte, word for word, unchanged. It is a term of art in legal, research,
and QA registers (`verbatim transcript`, `verbatim testimony`), so weigh density in that context
before flagging.

A caveat worth keeping visible: the claim that tier-1A words appear far more often in machine text is
inherited convention rather than a statistic measured here. Treat it as well supported and
unverified.

## tier-2.3 The variance rule

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

Structural regularity is a stronger detection signal than vocabulary. A draft with every flagged word
fixed and the rhythm untouched still reads as machine-made.

## tier-2.4 The inverse detector: signs of a human hand, protect these

Everything above says what to remove. This says what to keep, and what to record on the PRESERVE
list.

- Hoarded specifics: machines round off, and humans keep the odd exact detail (`nineteen winters of
  civil war`).
- Mixed or unresolved feelings, and a thought left open.
- Era-bound and personally-dated references.
- Self-interrupting asides and parentheticals.
- Deliberate fragments that land a point.
- Idiosyncratic repetition, where the same word recurs because it is the right word, as against
  thesaurus-driven synonym cycling.
- First-person stance, preference, and reaction where the genre carries a voice.
- Mixed registers, which signal a real person.

## tier-2.5 The never-inject list

Constraints on the editor, not detections on the text. **None of the following may be added** to
prose that did not already contain it:

fake first person, manufactured stakes, forced contrarianism, performed candor, em-dash theatrics,
staccato conversion (chopping sentences into fragments to manufacture rhythm), invented specifics.

**The provenance rule: you may subtract and sharpen; you may not add.** A first-person aside is not a
flag when the author wrote it, and is a failure when the tool inserted it. The difference is
provenance, which no pattern can see, so provenance is carried rather than inferred: each pass that
could add material receives the pre-editorial original as its reference. Where the original is
unavailable, the rule narrows to the scope the pass can check, which is its own input, and the output
says so.

Invented specifics earn their own emphasis. A fabricated number, name, or date is worse than the
vague phrasing it replaced, because specificity always reads better. Where a concrete detail is
missing, flag the gap and leave it.

## tier-2.6 Choose the simplest exact word, not merely the shortest

A long or technical word earns its place when it carries a distinction the shorter alternative
loses. The rule cuts both ways: prefer the plain word when the meaning is identical, and keep the
precise one when it is not.

- Use *use* rather than *utilize* when they mean the same thing.
- Use *latency* rather than *delay* when the technical distinction matters.
- Use *correlation* rather than *relationship* when statistical meaning matters, and *relationship*
  when statistical precision does not.

Plainness should never come at the cost of accuracy. The target is the exact word, and sometimes the
exact word is the longer one.

## tier-2.16 Use metaphor as an instrument of thought, not decoration

Pairs with the dead-metaphor entry in tier-2.1.

Imagery is welcome. The distinction is between a fresh image that assists thought and a worn one
that spares the writer from thinking. A metaphor earns its place when it reveals a useful
resemblance, makes a mechanism easier to grasp, stays coherent when examined, does not exaggerate
the subject, and is not immediately crossed by a second, conflicting image.

Decorative:
> The platform is a beacon that unlocks a new frontier and lays the foundation for an ecosystem.

That asks the reader to hold a beacon, a lock, a frontier, a foundation, and an ecosystem in mind at
once, and so pictures nothing. Functional:
> The template acts as scaffolding: it supports the first stages of the work and is meant to come
> down as the final structure takes shape.

One image, and it explains both the function and its limits.

## tier-2.17 Cut repetition that does not develop; keep repetition that builds structure

Pairs with synonym cycling in tier-2.4.

Restating an idea in slightly different words, then restating its importance, is static repetition:

> The policy makes ownership clearer. It creates greater clarity around responsibility. This
> highlights the importance of clearly defined ownership.

Nothing develops across those three sentences. But repetition can also do real rhetorical work,
building rhythm, contrast, or escalation:

> The first failure delayed a decision. The second delayed a launch. The third showed that the delay
> was part of the system.

The repeated structure is the point there. The test is not "can these words be removed" but "what
would be lost if they were: meaning, logic, emphasis, tone, rhythm, or nothing". Cut only when the
answer is nothing.

## tier-2.18 Vary sentence length according to the movement of the thought

Pairs with the variance rule, tier-2.3.

Uniformly short sentences turn abrupt and mechanical. Uniformly long ones hide weak logic in their
folds. Use length deliberately: a short sentence lands a conclusion, a medium one carries the main
explanation, a long one holds comparison, qualification, or accumulating detail as long as its
structure stays visible.

> The proposal looked cheaper. It was not. Once migration, support, and retraining were included, the
> apparent saving became a two-year cost.

The brief middle sentence creates the emphasis. It is not more factual than a longer version would
be; it simply controls the pace, and pace is part of the meaning.

## tier-2.19 Preserve voice, but do not confuse voice with verbal ornament

Pairs with the never-inject list, tier-2.5.

Voice comes from what the writer notices, the order in which details arrive, the confidence or
caution of the judgments, the rhythm of the sentences, restrained humour, the quality of the
comparisons, and the willingness to state a hard conclusion plainly. It does not come from a
constant layer of flourish. The standard to hold: the writing should sound like a particular mind
attending carefully to a particular subject, rather than a generic style applied to any subject at
all.

## Output contract: the PRESERVE list

This pass emits two lists, not one.

```
EDITS:    [spans changed, each with the rule that fired]
PRESERVE: ["forecast contract"          - coined term, load-bearing
           "It was the plants."         - earned fragment, lands the turn
           3x "pressure"                - deliberate repetition, not cycling
           "(my maths here are loose)"  - self-interrupting aside]
```

**PRESERVE is pipeline-scoped, not tier-scoped.** Once emitted it travels with the draft through
every later stage, including both voice models and the merge in `writing-like-me`. The voice stage is
where the risk is highest, because it is the one stage permitted to rewrite freely.

Entries are **advisory with disclosure**. A later stage may override one when it has a reason, and
must then say which entry it overrode and why. Silent removal is a failure; disclosed removal is a
judgment the reader can review. A preserved span can become wrong once the surrounding argument is
restructured, so a rule that could not be broken with disclosure would force a stage to leave prose
it knows is broken.

**An empty list is a valid output** and means the pass found nothing distinctive to protect. Report
it as `PRESERVE: (none)` rather than omitting it, so a later stage can tell "nothing to protect" from
"the pass did not run".

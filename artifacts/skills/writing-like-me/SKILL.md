---
name: writing-like-me
description: >
  Write or edit in the user's own voice from an exemplar corpus. Builds a voice profile and
  structural spine, generates independent faithful and route-blind drafts, synthesizes a third
  architecture, then runs a fresh developmental pass for intellectual fugues, literary depth,
  anti-axiomatic prose, inventories and entrances, then separates conceptual-coherence findings
  from fresh voice-owned repairs before fidelity and liveness audits. Routes post-final human
  feedback into a new immutable run at the earliest affected stage.
  Use whenever the user asks to make prose sound like them or supplies a draft to voice. Requires
  writing-editorial and a populated ~/.claude/patronus/voice/ corpus.
---

# Write like me

**Voice lives in conception, not in varnish.** Which scene opens the piece, which metaphor spans
it, where the author lets themselves be wry: a voice pass confined to diction and rhythm can only
polish an arrangement someone else already fixed. So this pipeline derives a spine before it writes
a sentence, and gives that spine authority over structure.

```
  sections/ + edits.yaml            from writing-editorial, cut and anchored
    |
    v
  profile: cached or fresh?          [ASK USER]
    |
    v  voice-profile.md
  derive TWO SPINES                  [CHECKPOINT]  independent whole-document derivations
    SPINE-ROUGH.md             <- original rough draft + profile
    SPINE-COMPARATIVE.md       <- user-named comparison draft + profile
    SPINE-COMPARISON.md        <- differences, risks, no silent merge
    |
    v  pressure + scene + running order + profile + draft   (NO claims manifest)
  TWO WHOLE-PIECE WRITERS            fresh, independent contexts
    |                                faithful arm + variance arm
    v  two complete drafts
  SYNTHESIS EDITOR                   third fresh context, rewrite-anything licence
    |
    v  synthesis draft + passage provenance
  JOURNEY EDITOR                     fourth fresh context, whole-piece developmental pass
    |                                fugues, literary depth, entrances, continuity, anti-axiom
    v  candidate draft + development record
  CONCEPT/COHERENCE AUDITOR          fifth fresh context, concepts only; no voice profile
    |
    v  COHERENCE.md findings only
  VOICE RESPONSE EDITOR              sixth fresh context, expression owner
    |
    v  revised candidate + COHERENCE-RESPONSE.md
  fidelity check                     manifest as ledger; material omissions only
    |                                -> back to the same writer, reconceived
    v
  audit                              per passage; flat -> rework once
    |
    v
  [ s01 ][ sNN ]  (optional)         refinement subagents, only where needed
    |
    v
  stitch                             only if the refinement pass ran
    |
    v
  codex advisory read                once, on the finished text
    |
    v  immutable final draft + codex notes taken and declined
  HUMAN FEEDBACK                     classify earliest changed authority
    |
    v  new run; rerun that stage and every dependent stage
```

## Entry modes

This skill is both an editor and a composer, so decide first which one you are. Choose from what the
user supplies:

| Mode | Input | What the voice stage may do |
|---|---|---|
| **Edit** | an existing draft | the author's claims and specifics carry authority; the source's sentence order, paragraph boundaries and section order do not. Rewrite and reorder freely where the argument improves, but **never** change a claim or inject one the author did not make |
| **Compose** | a request, no draft | write a faceless base draft **with no corpus in context**, run stage 0 over it, then voice: every sentence may be rewritten |
| **Voice-only** | a draft the user says is already edited | skip stage 0, say so, and voice with no edit records or sections, so no restore is available |

## The attractor

Compose mode still writes its base draft with **no corpus in context**. The reasoning holds: a
model told to sound like someone chases the sound and neglects the thinking.

What changes is its status. It is an **attractor and loose scaffolding**, not a text to be
preserved. It fixes the ideas, the evidence and the citations. **Its sentences carry no authority
at all**, and the voice stage is licensed to demolish and rebuild every one of them.

In edit mode the user's own draft plays this role. **Its claims and specifics carry authority; its
sentences and arrangement do not.** The never-inject rule binds, while the drafting arms and the
synthesis editor may replace, merge, split and reorder paragraphs to reach the same commitments by
a better route. A fourth whole-piece developmental editor then deepens earned intellectual fugues,
removes axiomatic accumulation and list-heavy prose, and repairs entrances and callbacks before
fidelity and liveness audits freeze the result.

## Stage 0: the editorial pass

Run the sibling `writing-editorial` skill over the draft, **supplying `trail-root`**, which is what
makes it emit section files and span-anchored edit records. Supply the attractor draft as its
`companion` input so each section's attractor slice is cut to the same ids.

What comes back, and what every later stage joins on:

| Artifact | Used by |
|---|---|
| `sections/NN-slug.md` | the voice subagent: post-tier-3 and authoritative |
| `sections/NN-slug.source.md` | restore resolution: the snapshot offsets resolve against |
| `sections/NN-slug.edits.yaml` | the citation rule, and PRESERVE, now advisory |
| `sections/NN-slug.companion.md` | the never-inject check |
| `sections/lineage.yaml` | the spine, when it reshapes |

The two schemas are defined in `{skillsDir}/writing-editorial/edit-record.md`. **They are an
interface, not an implementation detail**: read them there rather than inferring them from a file.

**The sibling resolves by path, not by name.** Where the host exposes a skill-invocation mechanism
by name, using it is equivalent and preferred, because it honours the router's own dispatch
question. If the tier files are absent from `{skillsDir}/writing-editorial/`, this is corruption
rather than a supported mode: the `requires:` edge means the skill is present under every supported
install path. Report the expected location, state that stage 0 was skipped and the draft is
unedited, and continue.

**If the sibling runs but returns no section files**, it is an older version that does not know
`trail-root`. Say so, naming the path that stayed empty, and continue on the whole draft as a
single section with no edit records: no restore is available, and the citation rule has nothing to
act on. Do not fabricate section files to keep the later stages tidy.

## Stage 1: the profile

Read `{skillDir}/voice-profile-schema.md` and follow it: ask cached or fresh, resolve the corpus,
extract from every corpus file whatever its language, keep evidence in its source language, and take
rhythm numbers from the English pool only.

With no corpus at the resolved path, **degrade**: print the path, say what to put there, and run
editorial-only.

## Stage 2: the spine

Read `{skillDir}/spine.md` and follow it. When the user supplies a comparison draft, derive two
spines independently: `SPINE-ROUGH.md` from the original rough draft and `SPINE-COMPARATIVE.md`
from the named comparison draft. Use the same profile and spine schema, but do not let either
derivation see the other source or output. The comparison draft is authorised as conception
evidence for this stage only; it is not silently promoted to source evidence or writer input.

Write `SPINE-COMPARISON.md` after both exist. Compare governing pressure, opening scene, metaphor,
argument order, definitions, ontology, evidence bounds and unresolved seams. Name source-derived
strengths, inherited defects and any claims present in only one input. Do not merge the spines or
choose a winner. Run checkpoint checks on each and **show all three files to the user for an
explicit governing-spine choice before any drafting fan-out**. Record the choice and whether any
elements were deliberately combined. Without a comparison draft, derive the usual single spine.

## Stage 3: two independent whole-piece drafts, then synthesis

One fresh writer improved continuity and still preserved the source's paragraph atoms. Permission
to change a plan was not enough while the same writer held a fixed `running_order`, section turns
and the source in front of it. The default is now **two complete drafts in parallel, followed by a
third fresh editor with licence to rewrite anything**.

### Stage 3a: the faithful arm

The faithful arm receives the governing pressure, opening scene, reference `running_order`, profile,
source draft and the licence below. It does not receive the claims manifest. It writes one complete
essay, and may depart from the reference route where the writing discovers a better one.

### Stage 3b: the variance arm

The variance arm receives the source, profile, governing pressure, register, unresolved seams,
citations and binding specifics. **It does not receive `running_order`, `per_section_assignment`,
`metaphor_locations`, section titles or the faithful draft.** Those fields are useful to the
faithful arm and poisonous to independent variation.

Before drafting, it privately tests materially different orders and granularities, then chooses the
one in which each boundary changes what the reader knows, expects or doubts. It writes a complete
essay, not an outline and not alternate paragraphs. Its job is not novelty for its own sake: every
source claim survives, but source paragraphs are raw material rather than units to preserve.

The variance arm may do **side research for expression**: physical processes, historical objects,
etymologies, spatial arrangements or other conceptual material that could make an abstraction
visible. Research supplies an image or analogy, never a new claim about the subject. A factual
statement not already in the source is omitted unless it is independently verified and cited, and
an analogy is discarded where explaining it costs more than it reveals.

### Stage 3c: the synthesis editor

A third fresh subagent receives both complete drafts, the source, profile, governing pressure and
the writer-safe spine without the claims manifest. It is **not a stitcher**. It may rewrite every
sentence, move any paragraph, discard either draft's opening, and invent a third arrangement. It
selects by function: which passage makes the claim concrete, which route keeps pressure alive, which
image changes the reader's understanding, and which plain sentence is stronger than both attempts.

The editor must not choose one draft wholesale and polish it. For each source section, it runs the
reversal test: if restoring that source section's paragraph sequence leaves pressure, dependencies
and reader state materially unchanged, the synthesis has preserved the source architecture and
must be reconceived. It writes `SYNTHESIS.md` beside the final draft, naming which arm supplied each
surviving passage, which passages were newly written, and every source-order block it deliberately
broke. This provenance is a checkable account, not proof; the drafts remain the evidence.

Before saving the final draft, scan every prose sentence that opens with `Imagine`, `Picture`,
`Consider`, `Suppose`, `Follow`, `Copy`, `Put`, `Give` or `Catalogue`. Record one of three findings in
`SYNTHESIS.md`:

1. **genuine advice** whose purpose is for the reader to perform the action;
2. **doubt-bearing invitation**, where `Suppose`, `Imagine` or `Picture` opens a possibility whose outcome the
   prose has not fixed and the thought experiment may change the author's position;
3. **staged proof**, where the action merely walks the reader to a conclusion already held.

The first two survive. **Rewrite every staged instance** as a condition shared by author and reader,
or begin on the concrete object itself. This is a functional preflight, not a lexical ban:
`Picture that the cheaper tool changes what we can afford to ask` may open inquiry, while
`Imagine twelve laptops; now you have a distributed system` performs a verdict.

### Profile-backed philosophical and mythic turns

When the profile evidences philosophical excursions, concept definition, coinage, mythic or occult
imagery, playful creatures, or a D&D-like register, treating the essay as literal technical
exposition is voice decay. The variance arm must explore at least one route outside the immediate
technical frame: a paradox, a philosophical concept, an older object or practice, a mythic figure,
or another strange-but-related idea. It returns only if the excursion changes how the technical
mechanism is understood.

The synthesis editor reports every explored excursion in `SYNTHESIS.md` and says why it took or
declined it. An earned turn may define a concept and then use it as working equipment, or leave the
technical frame and return with a new distinction. A named concept found through research, such as
Jevons paradox in an argument about AI efficiency and demand, is a subject-matter claim: verify it,
cite it, and show the mechanism. A kraken, archmage, amulet or trickster invoked only to satisfy the
register is costume and fails the removal test.

**An allusion is not yet an excursion.** Naming Ariadne without entering the labyrinth, or adding
`Deleuze would enjoy this` in parentheses, spends the cultural reference without giving the reader
its thought. Where the connection is taken, give it enough room to become intelligible to a reader
who does not already know it: explain the adjacent idea, allow it to disturb or illuminate the
technical frame, then return carrying a usable distinction. Chesterton's Fence can turn encoded
policy into inherited reasoning whose original problem may have vanished from view; Ariadne's thread
can make retraceability visible before the Minotaur tests what retraceability cannot guarantee.
These are examples of depth, never destinations to insert.

Mythic detail follows the same rule. Verify the named figure, object's history, paradox or
philosophical idea before use, and cite it where the passage relies on a factual account or
attributed principle. Research may widen the expressive territory; it may not smuggle in a new
claim about the essay's subject. A playful wink may remain brief, but it must be understandable
without prestige name-dropping and must alter the sentence it inhabits.

All drafting and developmental agents write to named files. The faithful and variance drafts are
durable artifacts even where synthesis uses none of a passage.

**The profile's `## Construction` block is handed to both writers and the editor as its own item,
not left to be found.** It
holds how this author wires two propositions together and what he does where they oppose, and it
applies to every sentence rather than to allocated places. Where the writer is about to set two
poles against each other, that block is what it reaches for; where the block is absent or marked
`Support: corpus-thin`, the writer states the relation in its own words and does not treat the
thinness as licence for the prohibited shape.

Two things forced the change, and both were measured on real runs rather than reasoned about.

**Fan-out removes the essay-level thought.** A voicer holding one section cannot decide that an idea
should germinate here, go doubtful two sections later, disappear for a page and return altered. The
pipeline permitted an assertion to open in one place and land in another on paper, but no agent
owned that movement, and the stitching pass could not supply it afterwards because it may not
rewrite inside a section. What came back was locally complete paragraphs: each one finished, each
one proving its own obligations, none of them thinking across a boundary.

**A visible checklist produces prose that proves rather than thinks.** Given a claim to land and a
row that checks the claim landed, a writer takes the shortest path, and the shortest path is
assertion. Three shapes recur because each is the cheapest available proof: a list performs
completeness, a mirrored opposition performs precision, a flat declarative performs confidence.
Prohibiting the shapes does not touch what generates them, which three runs of prohibition
demonstrated.

So the manifest becomes **a fidelity ledger checked afterwards, never a brief handed over in
advance**. What the writer receives instead is the pressure, the sources, the material and the
licence below.

### The licence

The prohibitions in this skill exist to catch known failures, and they are not a description of
how to write. This stage grants, explicitly, and these override any inference from the constraints
elsewhere:

- **Wander.** Take a strange route to a claim. Spend three sentences going somewhere before saying
  why, and trust the reader to follow.
- **Discover images the plan never named.** Local imagery belongs to the writer and needs no
  allocation. Only a second image competing to *govern* the whole piece is reserved.
- **Reconceive paragraph architecture.** Merge, split, reorder, subordinate, cut. The source's
  arrangement carries no authority beyond its claims and its specifics.
- **Arrive indirectly.** A claim may be approached, circled, doubted and reached late. It may land
  as a consequence of something else rather than as a stated proposition.
- **Let the writing change the plan.** Where a section finds something better than its assigned
  route, take it and say so. **The spine is a hypothesis about the piece, and the writer is the
  first agent in a position to test it.** Returning a spine as over-constraining is a legitimate and
  expected outcome, not a failure to comply.
- **Leave a section plain.** A section carrying the argument cleanly with no named move in it is
  finished. Manufacturing a move to have one is the ornamental compliance the audit exists to catch.

**Write it as one developing act of thought, not as a demonstration of coverage.**

### Concrete first, and either image or plainness

Where a section introduces an abstract rule, prefer to let the reader encounter the concrete case
that makes the rule necessary before naming it. A required-status check bound to a commit can earn
the abstraction that a verifier must not answer to what it verifies; leading with the abstraction
makes the reader decode the rule before they have anything to attach it to.

An abstraction gets one of two treatments: **make it visible, or say it plainly**. A gate may become
a passage into another space where that image changes how enforcement is understood. If no image
does real work, write `the central repository still exists`. Phrases such as `the thing nobody
proposes deleting` merely hide a plain noun and fail the removal test: they add effort without
adding a picture, mechanism or distinction.

### Questions make the reader participate

Carry the spine's `socratic_inquiry` across the whole essay. A Socratic question is not punctuation
variety and not uncertainty theatre. It asks the reader to inspect a familiar practice, hidden
premise or concrete situation, then makes a later distinction possible: why do we ask for peer
review after the code already runs; what is a colleague asking us to lend when they request
approval; what problem does a thread solve before it becomes a metaphor for memory? These are
illustrative functions, not sentences to copy.

Use the profile-derived ratio against the total prose sentence count and stay within its target
band. Questions may cluster where inquiry genuinely develops and disappear where the prose needs
to explain plainly. Do not place one in every section, turn declarative claims into cosmetic
questions, or immediately answer each question with the verdict the reader was marched toward.

### Break dry abstraction chains

Voice does not rescue prose that remains entirely conceptual. A paragraph about authority,
tentativeness, reversibility, evaluation, truth and surfaces can be rhythmic and vivid while still
giving the reader nothing they can see happening. When an inference begins accumulating abstract
nouns, return to the spine's `grounding_plan`: a colleague requesting review, a pull request whose
approval enables a new button, a failed deployment attached to a commit, an analyst meeting an
alert, or another source-bounded case that lets the reader derive the abstraction.

The concrete material must participate in the reasoning. An image appended after the conclusion is
decoration; a technical noun with no actor or event is still abstraction. Do not manufacture a case
that adds a subject-matter claim the source did not make. Expression research may supply a verified
illustration under the existing evidence rules.

### Keep coined referents recoverable

A coined shorthand saves repetition only while the reader can still recover what it denotes. After
a section boundary or a substantial excursion, re-expand it in apposition on first return: the
second surface, meaning the distributed agent capability across the workstation fleet. Do this when
needed for recovery, not on a cadence. A label whose referent lives only in an earlier heading or
paragraph has become private vocabulary.

### Four habits the author has named, with his own repairs

Each is a defect the last run reproduced, followed by how he writes it instead. **The repairs are
evidence of the register, not templates.** Reaching for the same phrase every time reproduces the
tic in new clothing, and a piece where every contrast turns on "on the flipside" is worse than the
version with the defect.

**Do not stage a demonstration for the reader.** "Catalogue it, and nothing changes." "Give that
stack to twelve engineers." The reader is being walked through steps toward a conclusion you already
hold, and it reads as dry and falsely confident. The author's repair puts the reader inside the
supposition rather than under instruction: an "if" clause, a "we", sometimes a tag question inviting
agreement. **Find your own phrasing for that.** In argumentative or explanatory prose, a bare
imperative that asks the reader to execute the author's proof is prohibited: `Follow the design`,
`Copy the workbench`, `Put one on every laptop`, `Catalogue it`. Put the condition around author and
reader (`if we follow...`, `if the tool is copied...`) or begin with the concrete case itself. A
genuine instruction remains available in a passage whose actual job is to advise action; the test
is whether obeying the verb is the recommendation or merely a staged route to a conclusion already
held. `Suppose`, `Imagine` and `Picture` also remain available where they create real doubt: the author does
not know what the thought experiment will reveal in advance, or risks changing position as it runs.

**Do not make yourself the measure of a difficulty.** "Which raises the thing I cannot get past"
makes the author the instrument. Leave the difficulty standing on its own, where the reader is free
to find it hard too.

**Say how two poles relate rather than leaving it as homework.** "It buys real speed. It costs real
governance." makes the reader work out whether that is a trade, a ranking or a concession. Concede
before turning, mark the concession as one, signal that a turn is coming. **The means are yours**,
and they may be loose or ungrammatical, since speech is. A piece where every contrast turns on the
same connective has reproduced the tic in new clothing.

**Do not set two poles side by side and let the full stop carry the opposition.** In long-form prose
this holds whatever the profile says, unless the profile names the shape as a move with long-form
corpus evidence. It is the defect this pipeline has produced most persistently, and it survives every rule
written against it because each instance looks locally fine. The exemption is that the prose
**travels** after the stop instead of landing: "An unmapped forest is not empty. It is just not
answerable, and a place becomes answerable at the moment somebody writes down enough about it that a
stranger can ask where something is and be told." The test the audit applies is whether a later
sentence can be **quoted** as supplying the relation. Where the sentence after the stop delivers a
verdict instead, rewrite it.

**Do not write inventories.** Not four items, not three, and full stops do not disguise one, nor do
invented labels: three batches of three under three headings is still a catalogue. The author's
instruction is "get rid of lists for good", and the default is that you do not write one.

The question is never whether you may keep a list, it is how to dismantle this one. Give the
principle that generates the items. Name the axis they vary on and show one instance. Subordinate
the minor ones into the sentence that needs them. Cut to the one that carries the argument and
develop it. **Where you find yourself justifying why each item earns its place, you are writing an
inventory and arguing for it**, which is the failure with extra steps.

The prose surface has a hard guard even where examples are useful: **a comma-delimited example run
stops after three objects and then says `etc.` only where the open remainder is genuinely meant.**
A section contains at most two such runs, and two runs need at least two ordinary paragraphs between
them. Semicolons, parenthetical batches and several short sentences do not evade the rule: count the
semantic inventory. `Etc.` is not a way to imply evidence the source never supplied.

An enumeration the spine named in `required_specifics`, or an inventory PRESERVE protects, no longer
earns a long prose string automatically. Where completeness really is the argument, move the full
set into a table, diagram or explicit structured list, or distribute interpreted instances far
enough apart that the reader encounters an argument rather than storage. Record the PRESERVE
override. Exact wording remains binding only where the protected item's function depends on it.

And one thing to verify rather than avoid: **when two propositions sit together, check the relation
the reader will take from them is one the piece has actually earned.** A sentence can have the shape
of an insight and the content of a non-sequitur, which is what "Not one of those is exotic, and that
is what makes them awkward" is: nothing about being ordinary makes a thing awkward.

### What binds regardless

The house mechanics (no em-dashes, punctuation outside closing quotes, British spellings), the
author's specifics in edit mode (a number, a name, a quotation, a citation), and the never-inject
rule: no claim the author did not make.

### Do not write like axioms

The most damaging remaining voice decay is not merely overclaiming. It is **axiomatic delivery**:
paragraph after paragraph resolves into a compact general law, spoken from nowhere, with the rhythm
of a textbook proposition. `As capability approaches action, authorisation narrows` may be defensible
and still exhaust the reader because it announces a law instead of letting a situated author and
reader reason towards a design judgement.

Do not repair this by sprinkling `perhaps`, `arguably` or `tends to` over unchanged verdicts. Choose
the operation the reasoning earns: locate the proposition in a case, scale or condition; state it as
the case for a design choice rather than a law of nature; let the author own it as a present reading
or wager; recruit the reader into a shared inference; carry a real concession inside the sentence;
or open uncertainty that the mechanism genuinely resolves or leaves standing.

`The closer a capability comes to being exercised, the stronger the case for tighter authorisation,
durable evidence and checks that are difficult to bypass` is gentler because it argues for a design,
not because it contains softer vocabulary. A few earned axiomatic landings may anchor an essay. A
run of them makes the prose a lecture, and the whole-piece audit blocks it.

### Section entrances and recoverable continuity

A heading resets the reader's working memory. The first paragraph must establish motion and recover
whatever it inherits; it cannot assume that a remote image, label or grammatical pointer remains
loaded. **Two short assertions at a section entrance are prohibited**: they deliver conclusions
before the reader has re-entered the thought. Begin with a scene in motion, a real question, a
condition, a tension, a developed image whose referent is re-established, or a sentence spacious
enough to carry the relation it asserts. These are available routes, not a rotation to satisfy.

Audit every `this`, `that`, `these`, `those`, `the former`, `the latter`, `above`, `below`, `here`,
`there`, `that sentence`, `this distinction` and coined label at paragraph and section boundaries.
The noun or proposition it denotes must be recoverable without rereading. `Completeness matters in
that sentence` fails when the preceding unit is a paragraph or when no sentence has been identified;
name the inventory, mechanism or proposition instead.

## Stage 3d: the journey and resonance pass

After synthesis and before the draft is frozen, a **fourth fresh whole-piece editor** receives the
synthesis draft, source, profile, both writer-safe briefs, `SYNTHESIS.md`, and the licence above. It
does not receive the claims manifest. This is a developmental pass, not copy-editing and not a
coverage repair. It may rewrite and reorder anything while preserving the source commitments.

The editor first maps the essay's conceptual hinges: a technical idea that could be seen differently
through philosophy, myth, literature, history, etymology, a physical process or playful theory. It
tests several adjacent territories privately and takes only those that return with explanatory
equipment. When the profile supports this register, **at least one taken fugue must be developed,
not merely named**, and a synthesis that already contains one is tested for whether it stopped just
as the useful story began. The pass may deepen an existing governing image, but it may not silently
replace the spine's governing image; a replacement is recorded as a proposed spine departure and
must survive the removal and competition tests.

Then it reads the whole piece for five failures that architectural synthesis can miss:

1. inventories exceeding the three-object/two-per-section/separation guard;
2. section entrances that begin with stacked verdicts or unrecovered imagery;
3. dangling demonstratives, coined terms and callbacks whose referents went cold;
4. successive paragraphs that land as axioms rather than a developing journey; and
5. an allusion that names an intellectual territory without entering and returning from it.

It writes the developed candidate and `DEVELOPMENT.md`. The record names candidate fugues taken and
declined, sources used to verify attributed ideas or mythic detail, inventories dismantled or moved
out of prose, every section entrance reconceived, every recovered referent, and each axiomatic run
reworked with the operation used. The pre-development synthesis remains durable evidence. The
developed candidate next receives the conceptual coherence stage; it is not yet frozen.

## Stage 3e: conceptual coherence and fresh voice response

Read `{skillDir}/coherence.md` and send it to a **fresh auditor without this orchestration file,
the voice profile or the liveness rubric**. Give it the raw source claims and evidence bounds,
citations, approved spine and definitions, and the whole developed candidate. Its sole deliverable
is `COHERENCE.md`: a short prioritised finding set about ideas and connections, never revised prose.

A different fresh voice-owning editor receives the candidate, source bounds, profile and only the
prioritised findings. It corrects accepted ideas and joins while preserving claims, citations and
profile-backed voice, and writes both a new named candidate and `COHERENCE-RESPONSE.md`. The response
maps every finding to an edit or a reasoned decline. The auditor never repairs its own finding. If
a finding changes the spine, manifest or synthesis authority, return there and rerun every
dependent stage.

Run fresh fidelity and liveness audits only after this response. A conceptual pass is not folded
into liveness: cadence, diction, register and voice remain outside `COHERENCE.md`, while domain
truth, definition, recoverable reader state, narrative situatedness and perceptual possibility
remain outside the voice score.

### Fidelity comes after, and repairs by reconception

Once the draft exists, a **separate** fidelity auditor maps it against the manifest and returns
**only material omissions and distortions**: an idea the piece needed and does not carry, or one it
now carries wrongly. An absent *formulation* is not an omission.

**Findings go back to the same writer**, who repairs by reconceiving the passage around the gap.
Inserting a sentence to discharge a missing claim reintroduces the proof surface this stage was
built to remove, and a coverage patch is visible in the prose as exactly what it is.

### Where the fan-out still earns its place

Section subagents remain available as a **refinement** pass over an existing whole-piece draft: a
named section that came back weak, a passage the audit flagged, a piece long enough that one context
cannot hold it. When used that way each subagent receives the seven items below **plus the finished
neighbouring prose**, so it can see the developing thought rather than only its own packet. It may
still reconceive paragraphs.

**The unsectioned contract below is now the default path, not the degraded one.**

## Stage 3f: the section refinement pass (optional)



**Only where the refinement pass is warranted**, spawn a subagent for the named sections, in
parallel. **The default is that it does not run**: a whole-piece draft that the audit did not flag
needs no refinement, and running it by habit reintroduces the section-compliance model this stage
was demoted to escape. Where it does run, the spine's `running_order` is what says which sections
exist, not the directory listing: the spine may have merged, split or cut
sections at Stage 2, and a cut section has no voicer. A piece with one section gets one subagent.

Each receives exactly seven items:

1. its section text, `sections/NN-slug.md`, post-tier-3 and authoritative;
2. its edit record, `sections/NN-slug.edits.yaml`, plus the `source.md` snapshot its offsets
   resolve against;
3. the voice profile, **including its `## Construction` block**, which is passed through whole and
   never allocated per section;
4. the spine, including its own `per_section_assignment` entry (its route: what the reader enters
   with, what turns, what they leave with, its `register_job`, its assigned `moves` and any `required_specifics`), the
   claims assigned to it **with their `evidence_bounds`**, the `governing_pressure` and this section's
   entry in its `progression`, and any `coinage_allocation` it holds;
5. the anti-slop techniques below;
6. its attractor slice, `sections/NN-slug.companion.md`, for the never-inject check. **Where that
   file is empty, fall back to the whole attractor draft**;
7. **its mode, compose or edit**, which decides how much licence it has over the prose.

Each subagent works on one section and returns its voiced text plus a restore log. It never sees
another section's text, which is what makes the fan-out parallel, and what the stitching pass and
the spine's assignments exist to compensate for.

### With no section files: the unsectioned contract

Two paths arrive here with no `sections/` directory: voice-only mode, and a `writing-editorial`
too old to honour `trail-root`. **Both still run**, on a reduced contract, and say which they are on.

| Item | Unsectioned form |
|---|---|
| 1. section text | the whole draft, treated as one section, id `00-preamble` |
| 2. edit record and snapshot | **absent.** No restore is possible, and the citation rule has nothing to act on |
| 3. profile | unchanged, `## Construction` included |
| 4. spine | unchanged: derived over the whole draft, with every claim assigned to `00-preamble` |
| 5. anti-slop techniques | unchanged |
| 6. attractor slice | the whole attractor draft, which is already the documented fallback |
| 7. mode | unchanged |

**One subagent, no fan-out, and the audit runs once over the piece.** Stitching has nothing to
join and adds nothing. The report says the run was unsectioned and that no restore was available,
so a reader can tell a run that found nothing to restore from one that could not look.

**Do not fabricate section files to satisfy the seven-item list.** An invented `edits.yaml` with no
upstream `source_rev` would offer restores that resolve against nothing.

### Optional section-refinement licence by mode

This subsection governs Stage 3f refinement only. It does not narrow the whole-piece drafting or
synthesis licence above.

*In compose mode*, rewrite every sentence. The section's prose came from a model told to have no
voice, so it carries no authority. What binds is the claims manifest: the ideas assigned to this
section must still be made.

*In edit mode*, the user's claims and specifics carry authority. A refiner may replace sentences,
merge or split paragraphs and reorder material inside its assigned section, while it must never
change a claim or introduce one the author did not make. Protecting source sentences wholesale
would turn refinement back into varnish; replacing the author's commitments would substitute the
model's argument for theirs.

**"Carries authority" is not "is finished".** The licence protects the author's *claims* and their
*specifics*, not the shape of every sentence that happens to carry one. A section left materially as
found, on the grounds that the input was already the author's, has done nothing: the input is a
draft the author brought **because** it does not yet sound like them.

So the licence in edit mode is this, and the distinction is the whole stage:

| May be rewritten freely | May not |
|---|---|
| how a claim is phrased, at any length | which claims the piece makes |
| the mood of a sentence: declarative into interrogative or concessive, subject to the landing rule | a claim's `type` obligation |
| paragraph architecture: merging, splitting, reordering, subordinating a minor point | a `type: evidence` specific: a number, a name, a quotation, a citation |
| an enumeration turned into prose, or a list's items chosen and the rest cut | the house mechanics, which bind every stage |
| a transition, a topic sentence, a summarising close | the author's conclusion, reached by another route or not |

**A sentence carrying a claim is not thereby protected.** Given the claim *"the questions that arise
are ordinary ones"*, the source sentence "The questions come up straight away, and they are not
exotic" is one rendering of it and the voicer owes the author a better one, in their voice. Passing
it through unchanged because it was already there is the decay this stage exists to catch.

### Render inside the claim's bound

Where a claim carries `evidence_bounds`, the rendering stays inside it. This is **scope**, and it is
not doubt: a scoped assertion is still an assertion, and the reader is told where it holds rather than
being asked to wonder whether it does.

**The claim is made, whatever its bound.** A bound states the claim's domain; it never turns the claim
into something the piece declines to assert. Where a bound would require you to say less than the
claim, that is a manifest defect rather than a rendering problem, and it goes back to the spine.

| The claim | Overstated | Scoped |
|---|---|---|
| approval is the usual seam, not the only one | "The seam between them is approval." | "The seam between them is usually approval." |
| a category error rather than a risk | "Not a risk, exactly. A category error." | "Not a risk, exactly, more like a category error." |
| the boundary has consequences | "So the boundary is real." | "So the boundary matters." |

Three different operations, and only the shape they share is worth naming: **each shows the reader a
range and places the author on it**, rather than handing down a point. "Usually" limits scope; "more
like" reclassifies; "matters" trades an ontological verdict for a consequence.

**This is not a quota and there is no target rate.** A claim whose evidence supports a universal is
rendered as a universal, and softening it produces vagueness, which is the opposite defect. What the
audit checks is whether a rendering asserts more than its claim and bound license, never whether
qualifiers are present.

**Where a section comes back materially unchanged, say so and say why.** "The author's phrasing
already carries the move" is a legitimate answer once or twice; it is not a legitimate answer for a
whole section, and the audit reports the section as inherited rather than voiced.

Every subagent is told which mode it is in, and every licence below is read against it.

### Restoring tier edits: the citation rule

A subagent may restore text the tiers removed, **but only by citing a named move in the profile. No
citation, no restore.**

The restore log is **this stream's own output, not an `editorial-edit-record/v1` document.** It has
two halves, and keeping them apart is what makes it checkable:

- an `edit:` block that **quotes the upstream record's own field names verbatim**, so a consumer can
  join back to it: `section_id` and `source_rev` from the record, `id`, `rule`, `span`, `offset`,
  `offset_rev`, `occurrence`, `removed` and `reversible` from the edit. **Carry `removed` and
  `reversible` even though they are not needed to locate the span**: `removed` is the text a restore
  puts back verbatim, and `reversible` is what lets a reader check the irreversible rule was honoured
  without re-opening the upstream record;
- restore-level fields this stream defines: `restored`, `citing`, `reason`.

```yaml
restores:
  - edit:                                     # verbatim from editorial-edit-record/v1
      section_id: 01-the-opening  # which record; a merged section carries two
      source_rev: 3f9a1c                      # the snapshot; resolve by occurrence + removed,
                                              # since only offset_rev: source offsets resolve
      id: e01
      rule: tier-1.3
      span: 01-the-opening/p03
      offset: 142
      offset_rev: source                    # only offset_rev: source is reproducible downstream
      occurrence: 1
      removed: "<the span the tier removed, verbatim>"
      reversible: true
    restored: "<the same span, put back>"
    citing: "Move: <the profile move that licenses it>"
    reason: "<why the corpus licenses this shape here>"
```

**Do not rename an upstream field on the way in.** `section_id`, not `from_record`; `id`, not
`edit_id`. A renamed field looks like a schema the upstream never defined, and the join silently
stops resolving.

`section_id` matters after a reshaping: a merged section carries **both** parents' records and a
split carries its parent's, so a restore that does not name which record it came from cannot resolve
its offset against the right snapshot.

The rule exists because the failure being guarded against is a voice stage rationalising its way
back to comfortable prose. A citation is cheap to demand and hard to fake against a profile carrying
quotations.

**Edits marked `reversible: false` are never restorable, whatever the citation.** Those are house
mechanics: no em-dashes, punctuation outside closing quotes, British spellings. They are the
author's stated rules rather than editorial judgement.

Resolving a restore depends on what the edit is anchored to, and the two cases are not the same:

- **An ordinary span** resolves by `span` plus `offset` plus `occurrence` against the record's
  `source_rev`. **A restore that cannot resolve all three against that snapshot is reported
  unresolvable and skipped**, never applied to a guess. Where `offset_rev` is anything but
  `source`, the offset is not reproducible downstream: treat it as advisory and resolve by
  `occurrence` and `removed` alone.
- **A `span: document` edit carries no `offset` at all.** It anchors to the reserved `document`
  id, names the sections it touched in `sections:`, and **is restored by reading its `reason` and
  the lineage**, per `{skillsDir}/writing-editorial/edit-record.md`. Demanding an offset here would
  reject every cross-section tier-3 edit as unresolvable. Such an edit is written once, into the
  first id in `sections:`, so a section reads its own record plus any `document` edit naming it.

### The four anti-slop techniques

Each lands in two places: this prompt, and the audit. A rule that exists only as an instruction is a
hope.

| # | Technique | Floor | Instruction | Audit criterion |
|---|---|---|---|---|
| 1 | Grounded reasoning | universal | concrete people, actions, events and cases must periodically carry the inference, not merely decorate it | dry-abstraction chains and grounded returns |
| 2 | No logical signposts | corpus-checked | match the corpus signpost rate; no "furthermore", "moreover", "consequently" bridging paragraphs; let the reader connect | signpost count against the profile rate |
| 3 | Asymmetric formatting | universal | vary paragraph shapes; an eight-sentence paragraph beside a one-sentence paragraph beside a fragment | paragraph sentence-count spread |
| 4 | Internal monologue | corpus-checked | asides, self-correction, doubt shown mid-sentence | asides present where the spine assigned them |

Techniques 1 and 3 are **universal floors**: no corpus is harmed by refusing uniform paragraph
blocks or by preferring a concrete verb.

Techniques 2 and 4 are **corpus-checked**, because they are stance choices that vary by author and
register. A flat ban on signposts misfires on a corpus that uses "however" and opens sentences with
"But" freely. The corpus rate is far below the machine default but is not zero.

**Technique 4 is a disposition, not a budget.** Three asides in one paragraph is a tic; a few
across an essay is voice. One writer holding the whole piece can see the difference, which is what
the per-section allocation existed to compensate for.

**Doubt is a disposition, not a rate and not a quota.** `hedge_rate_corpus` and `hedge_rate_target`
are **reported numbers and never scored**, joining the rhythm numbers in
`{skillDir}/audit.md` that are guard rails rather than
criteria. The spine names `unresolved_seams`, the places the piece **does not resolve**, as
**questions the piece is genuinely carrying rather than positions to fill**. They are not pinned to
sections: a seam assigned to one produces a set-piece paragraph about doubt, where a seam the writer
is actually holding surfaces wherever the argument touches it. **What the audit checks is that the
piece does not resolve everything it raised**, not that N seams appeared in N places.

| Seam kind | The piece leaves standing |
|---|---|
| `question-declined` | a question it raises and does not answer |
| `reconsideration-kept` | a change of mind shown, and not retracted into a tidy conclusion |
| `limit-unrepaired` | a limit of its own argument, stated and not repaired |

**An adverbial hedge does not discharge a seam.** "probably", "I think" and "perhaps" bolted onto a
declarative frame leave the frame declarative and the claim settled. Neither does a question the
next sentence answers, which is a rhetorical setup, nor a concession the paragraph immediately
reclassifies, which can read as more authoritative rather than less. **The test is whether something
is left standing open at the end of the piece**, not what shape it took on the page.

**A draft with no visible doubt anywhere is the failure this exists to prevent**: prose that knows
everything, states each claim as settled, and gives a reader no seam to think through. A rate cannot
catch it. One run met a target of 4.57 against 4.5, with the target chosen deliberately above the
corpus rate and approved by the author, and returned nine instances of the identical phrase "I
think", one per section. Cutting four improved the piece and put the rate below target. The
instrument was well-calibrated and correctly enforced, and it measured the wrong thing.

`type: hypothesis` and `type: question` claims are where seams live most naturally: the manifest has
already said this one may stay open, so leaving it open is conforming rather than a weakened claim.

Technique 1's floor is supplemented by the profile, so the replacement for an abstraction is not
generic concreteness but this author's, drawn from the images their own corpus reaches for rather
than from the first plain noun to hand.

## Stage 4: the audit

After writing, audit against `{skillDir}/audit.md`. Send
that file to the auditing subagent **without this one**: an auditor holding the orchestration body
inherits the frame of the stage it is meant to judge.

### Every auditor writes its findings to a file, and that file is the deliverable

**Name the output path in the auditor's opening instruction, at the run's own drop point beside the
drafts**, and say the file is the deliverable rather than the reply:

| Auditor | Writes |
|---|---|
| conceptual coherence | `<trail-root>/<slug>/COHERENCE.md` |
| liveness | `<trail-root>/<slug>/AUDIT.md` |
| fidelity | `<trail-root>/<slug>/FIDELITY.md` |

Tell it to **write the file even where the analysis is incomplete**, immediately, and to prefer a
partial file over a fuller reply. A findings file at eighty per cent is worth more than a perfect
report that never arrives.

**A returned report is a convenience; the file is the record.** A subagent's reply reaches the
orchestrator only as its final text, so an agent that spends its last turn on a tool call emits
nothing, and the caller receives a bare idle notification: the analysis happened and died with the
turn. A file is written mid-turn and survives however the agent ends. Instructing an auditor to
answer in reply text rather than write a file removes the durable path and leaves only the fragile
one.

**Read the file rather than waiting on the reply.** Where it is absent after the auditor goes idle,
re-dispatch a fresh auditor naming the path again. Where no file arrives across attempts, **report
that the audit did not run**, and never present the orchestrator's own checks as an audit: the stage
whose work is being judged cannot supply the judgement.

**The audit runs over the whole piece by default**, since the synthesis editor produced one piece
and the section boundaries are now the piece's own rather than the source's. Score section by section only where
the refinement pass ran. **A per-section score is a diagnosis of where to look, never a set of
obligations the writer owed**, and reading it the second way is what turned every section into a
deliverable that had to prove itself.

The auditor also receives the profile, the spine, and **the preceding section in `running_order`**,
which the metaphor row scores against. The first section has no predecessor: score its metaphor row
on whether the metaphor is present and doing work, and say the comparison was unavailable.

Flat fires one rework. A second flat result is **accepted and flagged**, naming the section and what
it failed.

## Stage 5: stitching

**This stage runs only where the refinement pass ran.** A whole-piece draft has no seams to join,
and its continuity is the writer's. Where refinement did run, the main agent reintegrates the
refined sections in the spine's `running_order`. **Continuity is a licence here, not an objective.** A seam between two sections is a legitimate finished state, and this stage
adds a connection only where its absence would lose the reader, never to make the piece feel whole.

**What it may add:** a callback, a returning image, the metaphor deepening across a boundary at a
`metaphor_locations` seam, the opening scene's character reappearing.

**Every addition is reported**, listing the bridge, callback or transition and the seam it sits on.
Stitching **may not rewrite sentences inside a section** to improve flow; a section's prose is the
voicer's, and a smoothing pass over it is an unreported second voice stage.

**Apply the removal test to everything this stage adds**, callbacks and returning images included,
not transitions alone. Delete it and ask what is lost. Where the only loss is smoothness, leave it
deleted. A piece where every seam is bridged reads as engineered, and
"too well threaded" is a failure mode of this stage specifically: three pressures compound here,
since the spine assigns the metaphor, the audit rewards carrying it, and stitching then reaches for
continuity on top of both.

**Hard cuts, unannounced pivots and changes of register are permitted output.** They are jitter, and
jitter is a property of writing by a person. Do not smooth them out, and do not report them as
defects.

**Check every deictic opener for a recoverable antecedent.** "Which...", "That...", "So..." and any
back-reference must point at something the reader still holds, and the relationship must be
necessary. Where the reference is to a frame the reader lost paragraphs ago (a title, an earlier
section's framing, "the argument above"), it fails: the reader cannot recover it, and the sentence is
performing continuity rather than supplying it. **Delete an opener that only performs coolness.**
Removing it is usually the whole fix, since the paragraph beneath it stands.

**Report paragraph-opener convergence.** Count the first word of every prose paragraph across the
finished piece. One word opening more than a quarter of them, or more than a third of paragraphs
opening on a back-reference to the one before, is a tic. Each instance is locally defensible, which is
why no voicer can see it and why the signposts row scores full marks throughout: the row counts
machine connectives, and "Which is why" performs the identical service while passing it. **A piece
where nothing starts cold has no joints a reader can feel**, which is what "too well threaded" is.

**Report repeated section architecture.** Where most sections share one shape, such as premise, then
enumeration, then categorical conclusion, name it. That regularity is what a reader registers as
machinistic, and no per-section audit can see it.

**What it may not add: logical signposts.** Technique 2 binds this pass as it binds the subagents.
"Furthermore" is banned at every stage; "she is still waiting on that runner" is the sanctioned way
to connect two sections.

**It must not regress the rhythm.** The risk here is smoothing section seams into uniformity,
flattening the spread the voicers just built. Check the stitched text against
`{skillDir}/weights.md`.

**Report runs of short sentences.** Count consecutive sentences of ten words or fewer in prose
paragraphs, and report any run of three or more, naming the paragraph. Runs that straddle a section
seam are only visible here, which is why the check sits at this stage rather than in the per-section
audit.

This is **reported, never blocking**, and it does not enter any score. A run of short sentences is a
fact about a text rather than a defect: a deliberately clipped passage may be the best paragraph in
the piece. Gating on it would teach the pipeline to pad sentences to clear a threshold, which is the
gradient `{skillDir}/audit.md` refuses for the rhythm numbers and refuses here for the same reason.
Exempt list items, headings and one-sentence paragraphs standing alone, none of which are the shape
this is looking for.

It also resolves duplicate coinages: where two sections independently reached for the same term
outside any allocation, keep one and say which.

**Then run the piece-level pass** in `{skillDir}/audit.md`
over the stitched text. It checks the spine's `signature_set` effects, the frame-break,
reader-address, aside and seam allocations, **the pressure progression, overstatement against
`evidence_bounds`, performed sameness, inventory guards, axiomatic accumulation, section entrances,
callback recovery, intellectual-fugue depth and post-voice compliance** (all
blocking), and reports route concentration, register modulation, paragraph-opener convergence, hedge
convergence, repeated section architecture and claims coverage across the finished piece. A claim assigned to a section that was later cut, and never reassigned,
surfaces there rather than silently vanishing, as does an `assertion` a voicer opened as a question
and no section ever landed.

A failure in that pass belongs to the spine or to this stage rather than to one section, so it does
not fire a per-section rework. Report it, naming the sections that could carry what is missing.

## Stage 6: one advisory codex read

Codex runs **once**, over the finished text, via the MCP server (`codex mcp-server`,
registered as `codex`). Call its `codex` tool with `sandbox: "read-only"` and a prompt naming the
paths of the text, the profile and the spine. It returns notes: where the piece reads flat, where a
move is absent, where the voice slipped.

**The main agent decides what to thread in, and reports both what it took and what it declined,
with reasons. Codex has no veto.**

Placing a different model in the judging seat rather than the writing seat follows from the
same-family bias argument: same-family judges favour low-perplexity text, which is the signature of
generic prose, so a foreign model earns more as a critic than as a co-author.

**Degrade, never block.** If codex is unregistered, errors or times out, continue and say which
happened, naming the error. Where the host allows a per-call timeout, allow around three minutes.

## Post-final human review and re-entry

Every accepted final is immutable. Human feedback creates a new run that references the prior
freeze and records the feedback verbatim, its classification, the chosen re-entry point, and why no
earlier stage was rerun. Never revise the prior final, its audits or its evidence files in place.

Classify by the earliest authority the feedback changes:

| Feedback changes | Re-enter at | Rerun downstream |
|---|---|---|
| claim, evidence or technical ontology | spine/manifest | both drafting arms when governing conception changes; otherwise synthesis, journey, coherence response and both audits |
| governing pressure, metaphor or argument logic | spine | both arms, synthesis, journey, coherence response and both audits |
| section order or argument placement | synthesis | journey, coherence response and both audits |
| opening depth, excursion, entrance or whole-piece resonance | journey | coherence response and both audits |
| conceptual join, definition, deictic, perspective or domain correction | conceptual coherence | fresh voice response editor and both audits |
| local voice or rhythm with no conceptual dependency | targeted fresh voice editor | both audits |

An auditor never silently fixes its own finding. Every prose edit belongs to a fresh editor with
the relevant voice authority. Preserve earlier freezes even when feedback proves them wrong: they
are evidence of what the previous pipeline accepted.

## What PRESERVE and the ledger bind now

- **PRESERVE is advisory to this stage.** It travels in the edit record marked
  `binding: advisory`, as tier-2's opinion. A subagent may override an entry, **reporting which
  entry and why**.
- **Override where transformation requires it, and disclose.** Tier-2 writes PRESERVE without ever
  reading the voice profile, so it protects spans on editorial grounds with no knowledge of whose
  voice the piece is meant to be in. Its own contract says a preserved span "can become wrong once
  the surrounding argument is restructured". Treat an entry as tier-2's opinion, formed before this
  stage existed. **Disclosure is the part that binds**, not the entry: say which entry, and what the
  override achieved.
- **Do not route the voiced draft back through the editorial tiers.** A tier pass cannot read the
  profile, so it would lawfully remove a corpus-backed move this stage lawfully introduced, and the
  piece would answer to two authorities over its own voice. What guards against reintroducing what
  the tiers removed is the regression check in the piece-level pass, which **reports to this stage
  and never rewrites**.
- **Advisory means the override path is expected to be used.** A run treating every entry as
  binding has silently converted tier-2's opinion into law, and the cost is concrete: one run left
  its closing entirely inherited because the passage was PRESERVE-protected, and a cold reader named
  that closing the least characteristic writing in the piece. Six long enumerations survived the
  same way. **Report per section which entries were overridden and what the override achieved**,
  naming the substantive operation from the transformation table in
  `{skillDir}/audit.md`. A raw count is provenance and not
  evidence of voicing: overriding one harmless entry to change a connective satisfies a counter
  while six cataloguing paragraphs survive intact. Zero overrides is a legitimate outcome, and a
  wholly inherited passage with zero overrides is reported as inherited.
- **The claims manifest protects ideas**, which are what must survive, rather than spans, which are
  what should be rewritten.
- **The contrast ledger is reconciled at the editorial merge and reported**, not enforced against
  this stage. Where contrast is a named move in the profile, contrast **as a habit of thought** is
  allowed, and the ledger does not constrain it. **This does not license contrast-by-adjacency**,
  which is prohibited by default whatever the profile names, on the terms above: the prose travels
  after the stop, or the profile evidences the shape in long-form. A move licenses the author's
  oppositions; it does not license one surface realisation of them.
- **Irreversible house mechanics remain binding on every stage**: no em-dashes, punctuation outside
  closing quotes, British spellings. Those are the author's stated rules.

## Reporting

Report in **this order**:

1. the finished draft;
2. the pre-development synthesis and `DEVELOPMENT.md`, including fugues taken and declined;
3. sections flagged as flat after rework, named;
4. **piece-level failures accepted after a second attempt**, named by check;
5. **the pressure progression**, section by section, naming any section that changed nothing;
6. **routes declared but not performed**, with the three surfaces that were missing;
7. **sections reported as inherited**: scored 0 or 1 on transformation, named, with what the voicer
   did and did not do to each;
8. claims from the manifest that no section made, if any, **including an `assertion` opened as a
   question and never landed**, and any rendering that overstated its `evidence_bounds`;
9. **the spine's counterfactual answer**, quoted, and what the spine reshaped;
10. **stitching additions**, each with the seam it sits on, and any that the removal test deleted;
11. **repeated section architecture and paragraph-opener convergence**, where the piece-level pass
    found them;
12. **regression findings** returned to the voicer, and what came back;
13. tier restores, with their citations;
14. PRESERVE overrides, with what each achieved;
15. codex notes taken and declined;
16. rhythm guard rails **and the hedge rates**, all **labelled as reported, never scored**;
17. the earned Socratic-question numerator, prose-sentence denominator, ratio and target band, plus
    every dry-abstraction chain returned for grounding.

**The ordering is the point: what the piece failed at comes before what it measured.** Never
present item 16 as evidence the voice landed. Items 4, 5, 6, 7 and 11 are the ones a passing audit
used to hide, so they are named even where everything else is clean: a run that reports nothing at 4 has
either done real editorial work everywhere or has not looked.

## The run trail

Every run writes its own directory named for the piece. **Where that directory goes depends on
whether the prose belongs to the project you are standing in**, because a trail is a work product
about the piece, not about the repository that happened to be open.

| The piece | Trail goes to |
|---|---|
| Belongs to this project: its README, an ADR, a design doc, a PR description | `docs/writing/<slug>/` in the repo |
| Does not belong to it, or there is no repo: an email, a message, a post, a personal essay | a scratch directory outside the repo, and say where |

When it is not obvious which, ask. Getting it wrong in the second direction is the one that stings,
since it commits private drafting into a project's history, so treat an unclear case as personal and
keep it out of the tree.

Inside a repo, whether the path is committed or ignored is the user's call: add it to `.gitignore`
for private drafting, leave it tracked where the prose is a team deliverable. Never edit
`.gitignore` yourself.

Ask once, at the start:

```
Keep full drafts for diffing? [Y/n]
```

Enter keeps them. Running unattended keeps them too, so the default is the same either way and a
trail is never silently thinner than it looks. Drafts are what make the trail checkable: without
them a reader has only each stage's own account of what it did, and a stage that fails to report an
edit is indistinguishable from one that made none.

```
<trail-root>/<slug>/
  manifest.yaml         always
  EDITS.md              always, per stage
  PRESERVE.md           always
  SPINE-ROUGH.md         when a rough/comparative spine pair is requested
  SPINE-COMPARATIVE.md   when a comparison draft is supplied
  SPINE-COMPARISON.md    when two spines are derived; no silent merge or winner
  COHERENCE.md           always, written by the concept/coherence auditor itself
  COHERENCE-RESPONSE.md  always; every finding taken or declined
  AUDIT.md              always, written by the liveness auditor itself
  FIDELITY.md           always, written by the fidelity auditor itself
  CODEX-NOTES.md        when the advisory read completes
  SYNTHESIS.md          always; passage provenance and source-order blocks deliberately broken
  draft-00-original.md  when drafts are kept
  draft-05-faithful.md  when drafts are kept
  draft-05-variance.md  when drafts are kept
  draft-0N-<stage>.md   when drafts are kept
```

**The auditors write their own files.** The orchestrator names the path and reads what lands; it
never transcribes an auditor's reply into the trail, because a transcription is the orchestrator's
account of the judgement rather than the judgement.

`manifest.yaml` records what each stage did and what it carried:

```yaml
run: deleuze-three-contributions
mode: compose
drafts_kept: true
stages:
  - stage: tier-1
    rules_fired: [tier-1.3]
    edits: 3
    contrast_ledger:
      retained: "actualization is not resemblance but invention"
      remaining: 0
  - stage: tier-3
    rules_fired: [tier-3.7, tier-3.8, tier-3.11]
    edits: 4
    preserve_overrides: []
    ledger_reallocations: []
    typical_sentence: 16      # median words
    longest_sentence: 41
    pct_past_26: 31
voice:
  profile: ~/.claude/patronus/voice/voice-profile.md
  profile_source: cached           # or: fresh
  corpus_files: [short-form.md, long-form.md]
  rhythm_source: english-pool      # or: unavailable
  spine_approved: true
  drafting_arms:
    faithful: draft-05-faithful.md
    variance: draft-05-variance.md
  synthesis: SYNTHESIS.md
  synthesized_sections: 6
  sections_reworked: [03-the-cost]
  sections_flagged_flat: []
  claims_unmade: []
  restores: 2
  preserve_overrides: 1
  metaphor_escalations: []         # sections that asked for a metaphor the spine did not assign
  unallocated_coinages: []         # terms coined outside an allocation; stitching says which it kept
  codex: read-completed            # or: unregistered, error, timeout
  guard_rails:                     # reported, never scored
    merged_median_sentence: 15
    merged_pct_past_26: 19
  model_reported_influences:       # the model's own account, not verified provenance
    - "consequence-first turn opening on a bare connective"
```

Two things about that shape are deliberate. **The drafts are what make it checkable**, not the
manifest: a stage's account of its own edits is a claim, and the draft beside it is the evidence.
Diff two stages and you see what actually changed, including what a stage failed to mention. The
manifest carries no checksums, because a model writing a hash it did not compute records a number
that proves nothing.

And **`model_reported_influences` is labelled as self-reported**, because a model cannot reliably say
afterwards which exemplar shaped a structure it synthesized across a dozen. The corpus path and the
exemplar count are knowable; the attribution is a claim.

## Why the two drafting arms are different

Two writers given the same spine are one experiment repeated twice. Independence comes from an
asymmetric brief: the faithful arm tests whether the approved conception can be written alive; the
variance arm tests whether the source's paragraph order was hiding a better conception. The
synthesis editor then has alternatives that fail in different places and explicit authority to
write a third answer rather than joining them at their seams.

The final stage's provenance is diagnostic. If nearly every surviving passage comes from the
faithful arm, the variance brief did not create useful alternatives. If nearly every passage comes
from the variance arm, the spine is over-specifying prose. If the editor rewrites most of both, the
two arms exposed useful material but neither found the piece. None of these distributions is a
target; each is evidence for the next iteration.

## Known limits, stated rather than hidden

- Register mismatch. Informal-register imitation verifies far less reliably than formal registers,
  so a corpus of casual posts gives weaker signal than one of worked prose. This is about how
  reliably the voice transfers, not about which lengths are allowed: a short corpus projecting onto
  a long piece is the supported path, and the constraint is that casual registers are harder to
  imitate at all.
- A short corpus teaches sentence construction and diction but not architecture. How a sentence is
  built scales from a 120-word post to a 4000-word essay; how densely short sentences are packed
  does not, per the format rule in `{skillDir}/voice-profile-schema.md`. Beyond both, how this
  author sustains an argument across ten paragraphs is not in the corpus to learn. The spine
  invents a running order the corpus cannot evidence, so a thin corpus produces a thin spine.
- Style strength and content preservation trade off against each other. They are competing
  objectives, not a tuning failure. This pipeline picks a point on that curve by putting
  content-shaping first and voice second.
- Scrubbing a widely circulated word list is itself becoming detectable, which is a reason the
  editorial tiers lean on structural rules rather than on lexical substitution alone.
- Parallel drafting arms may converge on the same unallocated image. Neither sees the other's text,
  so the synthesis editor is where a duplicate governing image is caught rather than prevented.

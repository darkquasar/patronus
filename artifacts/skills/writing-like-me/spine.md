# The spine

Derived once, by the main agent, after reading the cleansed draft and the profile, **before any
fan-out**. Two inputs, and the split matters:

- **from the attractor draft**: the claims manifest, and nothing else;
- **from the profile**: metaphor, opening scene, register and frame-breaks, all *invented* at
  spine time by an agent that has just read the corpus.

Voice lives in conception: which scene opens, which metaphor spans the piece, where the author
lets themselves be wry. A spine derived after the prose is fixed can only varnish it.

**The profile's `## Construction` block passes through untouched.** The spine allocates moves,
which appear in particular places and can be spent. Construction is the wiring under every
sentence, so there is nowhere to allocate it to, and a spine that assigns it to three sections has
turned it into ornament. Do not name it in `per_section_assignment`, do not cite it in a
`signature_set`, and do not paraphrase it: the writer receives the block as extraction wrote it.

## Shape

```yaml
governing_metaphor: >
  <ONE image the whole piece runs on, named as an image rather than as a term. State what it is
  and how it develops.>
metaphor_locations: ["00-preamble", "02-the-turn", "07-the-close"]
  # Where it appears, and nowhere else. NOT every section: an image touched eight times is
  # decoration, and a reader stops seeing it. Assign an opening, a deepening turn, and a return.

opening_scene: >
  <a specific moment, with a person in it, that the piece opens on. Name what later sections
  may call back to.>

register: >
  <the stance: hedged or blunt, first person or impersonal, wry or plain.>

# The effects this piece should CARRY, chosen for THIS essay. Each names the profile moves eligible
# to produce it. These are eligibility, NOT quotas: a count tells a writer to manufacture instances,
# which is how the moves row filled with ornament. The audit asks whether the effect is present and
# earned, not whether N of them are.
signature_set:
  epistemic_openness:   {eligible_moves: ["asks a run of open questions and declines to answer them"]}
  reader_turn:          {eligible_moves: ["breaks frame to address the reader mid-argument"]}
  stance_disruption:    {eligible_moves: ["undercuts its own seriousness with a parenthetical aside"]}
  figurative_grounding: {eligible_moves: ["lands an abstract point on a concrete, walkable image"]}

# Three DISTINCT acts, and one insertion never counts as two: an aside is not necessarily
# reader-facing, and a frame-break is not necessarily an aside. Name what each would interrupt or
# implicate; do NOT pin them to sections. A writer holding the whole piece places them where the
# argument turns, and pinning them produced insertions that arrived on schedule and did no work.
frame_breaks:     [{about: "<what the break interrupts>"}]
reader_addresses: [{about: "<what the reader is asked or implicated in>"}]
asides:           [{about: "<what it undercuts>"}]

# WHY THE READER KEEPS MOVING. Without this, routes and claims and register compose into a
# well-labelled tour of polished objects: each section true, each well made, none NECESSARY.
governing_pressure:
  question: >
    <the live problem the piece cannot answer immediately, and does not answer until late or at all.
    Not the thesis: the thesis is what the piece concludes, and the pressure is what makes a reader
    need the conclusion.>
  stakes: >
    <what changes for the reader depending on the answer. A pressure with no stakes is a topic.>
  progression:
    - section: 00-preamble
      changes: "<what becomes harder, stranger or more consequential here>"
    - section: 02-the-turn
      changes: "<what earlier answer stops being sufficient>"
    # EVERY section in running_order appears. A section that changes nothing is a polished object,
    # and the checkpoint says so.

# Places the piece does NOT resolve. Allocated, not left to a voicer's mood. See SKILL.md stage 3.
unresolved_seams:
  - section: 01-the-relocation-argument
    kind: question-declined        # question-declined | reconsideration-kept | limit-unrepaired
    destabilises: c6               # the claim or tension it opens; a seam floating free is decoration
    undecided: "<the proposition left standing>"
    why_material: >
      <what would change in the conclusion if this resolved. A seam whose resolution changes
      nothing is ornament, and the audit scores it 0.>

claims_manifest:
  - id: c1
    claim: "<a semantic commitment the piece would be wrong without, at commitment altitude>"
    type: assertion            # assertion | question | concession | hypothesis | evidence
    subsumes: ["<source proposition>", "<source proposition>"]   # what merging folded into this
    evidence_bounds: >         # ONLY where the source evidence has a domain worth stating. Omit otherwise.
      <the claim's DOMAIN and what it must not be extrapolated to. The claim itself is already
      evidence-safe: this does not repair an overstated claim, it tells a voicer where the claim
      holds so a rendering does not silently widen it. "holds for backends with a query compiler,
      and the source does not speak to the others" / "observed at twelve engineers, not asserted at
      any scale". Where the evidence supports the claim universally, omit the field rather than
      inventing a bound.>
    assigned_to: ["00-preamble"]
  - id: c2
    claim: "<another>"
    type: hypothesis
    assigned_to: ["02-the-turn", "04-the-case"]   # may land in more than one

# Full section ids throughout, never the bare ordinal: these join to the upstream
# edit records and lineage, which key on NN-slug.
running_order:
  - 00-preamble
  - 02-the-turn
  - 01-the-opening
  - 04-the-case

# Each section declares WHAT CHANGES for the reader, not only which moves land in it. A section
# that states a true claim well and leaves the reader's model exactly as it found it is exposition.
per_section_assignment:
  "02-the-turn":
    reader_enters_with: "<the live question or model the reader holds arriving here>"
    turn: "<what CHANGES. Not which claim is stated: what stops being sufficient, or becomes stranger>"
    reader_leaves_with: "<the new lens, tension or consequence they carry out>"
    route: derivation      # scenario | question | observation | proposition | example | contrast
                           # | excursion-and-return | concession | derivation
    register_job:
      load: "<what must become unmistakably clear here>"
      destabilise: "<what settled frame may be disturbed; empty where the section is plain>"
      device: "<image | aside | question | compression | hard cut; empty where plain>"
      grounding: "<the specific mechanism or example a departure must return to>"
    required_specifics: []   # inventories whose COMPLETENESS is argumentative. Empty by default.
    moves: "carries the metaphor's deepening: where the governing image is extended, not restated"
  "04-the-case":
    reader_enters_with: "..."
    turn: "..."
    reader_leaves_with: "..."
    route: concession
    register_job: {load: "...", destabilise: "...", device: "aside", grounding: "..."}
    required_specifics: []
    moves: "frame-break lands here, plus one aside per the monologue budget"

coinage_allocation:
  "02-the-turn": ["<the coined term>"]      # only this section may introduce this term
```

## The spine may reshape the piece

Draping a metaphor over a lifeless arrangement produces a live opening and a dead body. So the
spine owns architecture: it may reorder sections, merge two, split one, or cut a section whose
claims land elsewhere.

The constraint is the claims manifest. **Every claim survives somewhere in the finished
piece**, and the audit checks this.

**Reshaping mints new ids and records lineage.** Ids are identity, not sequence, and they are
append-only: a reshaping **never** reassigns an existing id, because an edit record written
before the reshaping must still find its text after one. Write the reshaping into
`sections/lineage.yaml`, in the `section-lineage/v1` schema that
`{skillsDir}/writing-editorial/edit-record.md` defines.

Full `NN-slug` ids throughout, never a bare ordinal: these join to the upstream edit records and
lineage, which key on `NN-slug`.

| Operation | New id | Lineage entry | Edit records that apply |
|---|---|---|---|
| reorder | unchanged | none; only `running_order` moves | unchanged |
| merge `02-the-turn` + `03-the-cost` | `02+03-the-turn` | `op: merged`, both parents in `derives_from` | both parents' records |
| split `04-the-case` | `04a-the-case`, `04b-the-objection` | `op: split`, parent in `derives_from` | the parent's record, resolved by span |
| cut `05-the-aside` | none | `05-the-aside` recorded `op: cut`, `edit_records: []` | none; restores against it are void |

**A reshaping writes the section files it mints.** A merge concatenates its parents' text in the
order the merge names; a split cuts the parent at a stated boundary. Write each new id to
`sections/<new-id>.md`, carry the parents' `.source.md`, `.edits.yaml` and `.companion.md` across
unchanged so offsets still resolve against the snapshot they were measured in, and leave a cut
section's files on disk. **Stage 3 reads a file per id in `running_order`**, so an id minted here
with no file behind it strands its voicer.

**Reorder writes no lineage entry**, because no id changes and `derives_from` would be the id
itself. It is listed here as the fourth thing the spine may do, not as a schema operation.

A restore names the parent record it came from, and its offsets resolve against that parent's
`source_rev` snapshot rather than against the reshaped text. Where a split leaves an edit's
span ambiguous between halves, **report the restore unresolvable and skip it**. Guessing a
location is worse than declining one.

## Pressure is why the reader keeps moving

A piece can carry a true claim in every section, land a named move in each, vary its register, and
still read as a tour: here is a reason, here is another reason, each polished, none of them
*necessary*. The reader can stop after any one of them and lose nothing. That is an enumeration of
facts wearing an essay's shape, and no per-section check can see it, because every section is fine.

**`governing_pressure` is the live problem the piece cannot answer immediately.** Not the thesis. The
thesis is what the piece concludes; the pressure is what makes a reader need the conclusion. It is
raised early, it gets worse before it gets better, and every section does something to it.

`progression` names, per section, **what becomes harder, stranger or more consequential**. Three
shapes count:

| | The section |
|---|---|
| intensifies | makes the problem worse, larger, or closer to the reader |
| complicates | shows an answer that looked sufficient is not |
| partially releases | resolves one part and leaves the rest standing |

**A section whose entry says "restates", "supports", "adds another reason" or "reinforces" is a
polished object.** It may be true and well written and it does not move the piece. The checkpoint
reports it, and the fix is to merge it into the section it supports, cut it, or find what it actually
changes.

Two tests, and they catch different things.

**The pressure exists:** could a reader stop after section 2 and feel finished? If yes, nothing is
pressing. A pressure released too early leaves the rest of the piece with nothing to do, which is the
same defect from the other side.

**Each entry is true**, which the first test cannot see. Requiring one entry per section invites a
generator to write "complicates the pressure by showing X is insufficient" for every section, since
that sentence can be written about almost anything after the fact. So, per section:

> Remove it. Carry its indispensable claims into the nearest surviving section as one flat sentence,
> and reconnect the seams. **Can every later progression entry, and the conclusion, stand unchanged?**

If they can, the section changed nothing and its entry is retrospective labelling. And for a
`partially releases` entry, the inverse: **name the exact uncertainty it retires**, and show no later
section depends on that uncertainty still being open. A release nothing was waiting on released
nothing.

## Sections travel, they do not simply contain

`per_section_assignment` says what each section does to a reader, not only which moves land in it.

**`reader_enters_with` is the model or question the reader actually holds on arrival**, given
everything before it. It is not invented so the section can defeat it: a straw belief the reader
never had makes the turn free, and the checkpoint asks where the reader picked it up.

**`turn` is what changes.** Not which claim is stated. A section that states its claim and leaves the
reader's model exactly as it found it has explained something without moving anything.

**`route` is how it travels.** The vocabulary is closed: `scenario`, `question`, `observation`,
`proposition`, `example`, `contrast`, `excursion-and-return`, `concession`, `derivation`. Vary it.
Eight sections sharing one route is the machinistic regularity a reader feels as a system meeting a
spec, and it is invisible to any single section's audit.

**`register_job` replaces a register label.** "Plain" and "mysterious" are impressions an auditor can
certify by pointing at vocabulary. A job is checkable: what must become clear (`load`), what settled
frame may be disturbed (`destabilise`), by what means (`device`), and the specific mechanism or
example a departure must return to (`grounding`).

**A departure without grounding is decoration.** The philosophical or figurative move earns its place
by changing how the reader sees the thing being discussed, then returning to it. Where a section is
plainly load-bearing, leave `destabilise` and `device` empty: that is the modulation working. A piece
that is mysterious throughout has no modulation at all, and its mystery reads as affectation.

**`required_specifics` is the narrow exception for inventories.** An enumeration earns its length only
where its COMPLETENESS is the argument (the point is that there are this many things). Empty by
default. Naming one here is a decision the checkpoint can see and count, which is what a per-span
protection written elsewhere in the pipeline cannot offer.

## A claim is a commitment, not a sentence

**A claim is something the piece would be wrong without.** It is not every assertion the attractor
makes, and this distinction decides whether the voice stage has any freedom at all.

A manifest written at paragraph granularity stops being a fidelity guarantee over *ideas* and
becomes a preservation system over *architecture*. Every source paragraph acquires an id, every id
must be "made", and the voicer's only conforming output is to re-say each source sentence more
nicely. It cannot compress two paragraphs into one, subordinate a minor point, reach the same
commitment by another route, or leave a question standing where the source asserted, because all
four read as a claim not made.

Worse, a claim phrased as a settled assertion **specifies certainty**. If the entry reads
`"the answer to every one of them is a repository with review, versioning and a reconciler"`, then
a conforming rendering is a settled assertion, and an author who would have written that with a
doubt cannot be voiced.

Two rules make this checkable:

- **Density bands, not a single ceiling.** Count only top-level commitments: `type: evidence`
  entries are children of the claim they support and do not count.

  | Density | The checkpoint |
  |---|---|
  | at or below 1 per 200 words | passes silently |
  | between 1/200 and 1/100 | **warns**, reports the ratio, names candidates to merge, and proceeds where the spine justifies the density |
  | above 1 per 100 words | **blocks**: a claim every other sentence is a paragraph map, whatever it is called |

  A ratio is a proxy for an altitude judgement and misfires at both ends. A dense technical argument
  may legitimately make several independent commitments in six hundred words; a reflective essay of
  two thousand may make three. The bands exist to catch the threefold-over failure, so the warn band
  is a conversation and only the block band is a gate. **Below four hundred words, the ratio is not
  computed**: it produces a ceiling of one or two, which is an artefact of the arithmetic.

- **Merge, never delete, and record what was merged.** Sibling assertions serving one commitment
  become one claim, which lists them in `subsumes`. Specifics become `type: evidence` children where
  they must survive verbatim, or they become the voicer's material.

- **The vagueness guard.** Compression is gameable in the opposite direction: thirty-three precise
  claims can be merged into six that commit to almost nothing, clearing every band while losing the
  argument. `subsumes` is what makes this checkable. **The test is entailment, not count**: does the
  merged commitment still carry every decision-relevant distinction its children drew? Where two
  subsumed propositions would lead a reader to different conclusions, the merge has destroyed one
  and the claims split again. "The agent stack needs governance" subsumes nothing usefully; it is a
  topic wearing a claim's clothes.

**Phrase claims as commitments, not as sentences.** A manifest entry that could be pasted into the
draft unchanged is written at the wrong altitude, and the checkpoint says so.

| At the wrong altitude (a sentence) | At commitment altitude |
|---|---|
| "The centralised surface does not shrink under the agentic design. It splits in two." | "The agentic design does not reduce the centralised surface; it produces a second one." |
| "Every agent-stack question is a distribution and versioning problem whose answer is a repository with review, versioning and a reconciler." | "The agent stack needs the same governance the detection content needs." |

The right-hand column commits to the same thing and leaves the voicer every choice about how it
lands, including whether it lands as a question that the piece answers later.

## Claims carry a type, and the type governs how they may land

One word, "claim", was carrying five different argumentative objects. Typing them is what makes
doubt **conforming** rather than tolerated:

| `type` | The finished piece must | The voicer may |
|---|---|---|
| `assertion` | affirm it declaratively somewhere | qualify, reframe, or concede-then-reclassify; open it as a question that lands later |
| `question` | raise it | leave it unanswered, which is the point of the type |
| `concession` | acknowledge it | grant it fully; it must **not** become the conclusion |
| `hypothesis` | carry it | hedge it, without the audit reading the hedge as a weakened claim |
| `evidence` | keep the specific | not promote it into a thesis |

**An untyped entry is an `assertion`**, so a manifest written before this rule keeps its meaning.

### Interrogative treatment opens a claim; it never discharges one

A voicer may render an `assertion` interrogatively. `"Is X really Y?"` **suspends** the proposition,
so a question alone never marks the claim made, however clearly it is declared: that would make the
coverage check a matter of trusting the voicer's own log.

What binds for `assertion`:

- the question may open or destabilise the claim anywhere in the piece;
- **the claim still lands declaratively somewhere**, and the landing may be qualified, reframed or
  conceded and reclassified. **Declarative is a grammatical mood, not a register**: it does not
  license the clipped verb-first delivery, and a landing phrased as something the reader is invited
  to agree with is still a landing. "This raises something that is hard to get past" lands; so does
  "we can both agree the lamp has not changed, right?";
- the voicer records `question_surface` and `landing_surface` against the claim id;
- **the whole-piece coverage pass evaluates the landing**, never the declaration.

**`evidence` is not subject to the landing rule.** A number, a name, a quotation or a citation is
not a proposition that can land declaratively; what it owes is presence and correct use. It is made
when the specific survives and is not promoted into a thesis, whatever the mood of the sentence
carrying it.

`question` and `hypothesis` claims are exempt too: staying open is their specification. Where the
author's genuine intent is to decline an answer, the manifest entry is typed `question` at spine
time rather than an exception being punched through fidelity downstream.

## Claims are assigned, and the assignment is tracked

The audit checks that claims assigned to a section were all made, so the assignment exists
before the audit runs. Three rules make it checkable:

- **Every claim is assigned somewhere.** A manifest entry with an empty `assigned_to` is a
  spine defect, caught at the checkpoint before fan-out.
- **Reshaping carries assignments with it.** A merge unions its parents' assignments; a split
  carries the parent's to both halves, and the audit accepts the claim as made if either half
  makes it; a cut reassigns its claims to another section. **A cut that would orphan a claim is
  rejected at the checkpoint.**
- **Coverage is verified after stitching, not only per section.** The per-section audit checks
  its own assigned claims; a final pass checks that every claim in the manifest was made
  somewhere in the finished piece. A claim assigned to a section that was later cut, and never
  reassigned, surfaces there rather than silently vanishing.

## The checkpoint

**Show the spine to the user for approval before the fan-out runs.** It is the
highest-leverage artifact in the pipeline: a bad governing metaphor poisons every section, and
it is cheap to fix here and expensive later.

Check before showing it, and report any that fire:

- **a claim the attractor draft makes that the manifest does not carry.** Re-read the attractor
  and enumerate its claims independently of the manifest, then compare. A spine that simply never
  enumerated a claim passes every other check here, because every other check reasons about the
  manifest rather than about the draft it was derived from;
- a claim with an empty `assigned_to`;
- a cut section whose claims are assigned nowhere else;
- a `running_order` naming an id that no section carries;
- a bare ordinal anywhere an id belongs;
- **claim density**, reported as a ratio, warning in the 1/200 to 1/100 band and blocking above
  1/100, naming which claims to merge;
- **a merged claim whose `subsumes` children draw a distinction it no longer carries**: the
  vagueness guard, and the failure mode of compression;
- **untyped claims**, counted and reported: "N claims defaulted to `assertion`; classify or
  confirm before fan-out." Silence here re-specifies certainty for every question, concession and
  hypothesis the manifest was carrying untyped;
- **a claim stronger than its own evidence.** `evidence_bounds` states a domain; it never licenses a
  claim the source does not support. Where the manifest asserts more than the evidence, **the claim
  is rewritten**, because a claim the voicer is instructed not to make as written puts two
  authorities in one record: the audit requires the assigned claim to be made, and the bound tells
  the voicer to narrow it;
- **an `opening_scene` the running order does not open on.** A spine specifying a concrete scene
  while the piece opens on an abstract evaluation has already lost the opening;
- **a second image competing with the governing metaphor.** List every image the plan carries. Where
  two are load-bearing, resolve to one before fan-out: a reader who meets four images finds none of
  them governing;
- **an effect in `signature_set` whose `eligible_moves` are not in the profile**, or a profile move
  cited for an effect its evidence does not show;
- **a section missing from `governing_pressure.progression`**, or one whose `changes` reads as
  "restates", "supports", "reinforces" or "adds another reason": a polished object, named;
- **a pressure a reader could stop caring about after section 2**, or one released so early the rest
  of the piece has nothing to do;
- **a `reader_enters_with` naming a belief the reader was never given.** Say where they picked it up.
  A straw entry makes the turn free;
- **a `turn` that only restates its assigned claim**, which is exposition wearing a route's label;
- **one route used by more than half the sections.** Eight sections sharing a route is the regularity
  a reader feels as a system meeting a spec, and no single section can see it;
- **a `register_job` with a `device` and no `grounding`**: a departure with nothing to return to;
- **every section carrying a `device`**, which is not modulation but a uniform register, and is how a
  piece becomes mysterious throughout and therefore mysterious nowhere;
- **`required_specifics` naming an inventory whose completeness is not the argument.** Report the
  total across the piece: several individually-defensible inventories compose into a catalogue, and
  the count is only visible here.

### The freedom check: does this plan leave room to write?

Every check above asks whether the plan is complete, consistent and well formed. **A plan can pass
all of them and still be unwritable**, because a spine fails in two directions and only one of them
has ever been instrumented. An underspecified spine produces a tour of polished objects. An
**overspecified** spine produces prose that reads as compliance: the writer arrives at each paragraph
holding a claim to land, a move to demonstrate, a route to perform and an allocation to discharge,
and the cheapest way to satisfy all four at once is a flat declarative, a mirrored contrast or a
list. Those three defects are what an over-planned section looks like from the outside.

The audit cannot see this. It scores conformance **to** the spine, so a restrictive spine that is
perfectly executed reports green on every row. **This is the only stage where over-constraint is
visible**, so run these before showing the plan:

- **Necessity, not quantity.** For each section, list the instructions it carries and ask of each
  one: **would removing this widen the writer's available routes without risking a lost claim, a lost
  specific, or whole-piece incoherence?** Remove or defer every instruction that fails that test.
  **The checkpoint blocks where the planner cannot say why a retained instruction had to be fixed
  before drafting rather than discovered during it.** Counting obligations instead would set a
  target: a planner can always split a section, merge two claims into one baroque commitment, or move
  a requirement into a prose field, and clear any threshold while constraining the writer exactly as
  much. Report the counts afterwards as observability; never let them decide passage.
- **Invariants and open choices, per section.** Say which of the section's properties the spine
  actually decides. **Invariants** are the indispensable commitments, the evidence bounds and the
  dependencies later sections rely on. **Everything else stays open unless the spine records a
  specific reason to close it**: sentence shape, local imagery, paragraph architecture, the order in
  which the thought develops, and the route by which a claim becomes credible. A spine that records
  no open choices anywhere has decided the prose, and blocks.
- **Where literal exposition is unlikely to be enough.** Name the places where the argument probably
  needs a conceptual transformation, and **leave the destination undetermined**. This is a
  permission, not an obligation: the writer may find plain exposition does the work, and saying so is
  a legitimate outcome. Naming both the departure and its destination writes the excursion in
  advance, and what comes back is fancy dress.
- **The metaphor's monopoly.** `metaphor_locations` governs the GOVERNING image only. **Local imagery
  is permitted anywhere and is never scored as an unassigned metaphor.** What the allocation prevents
  is a second image competing to govern the piece, not a figure that serves one passage and is done.
- **Claim shape.** Read each claim aloud as a sentence. **Where the claim is already shaped like the
  sentence that will carry it, it has pre-written the prose**, and the entry is rewritten. A claim
  phrased as a mirrored contrast ("X does not shrink; it produces a second one") licenses one
  rendering, and it is the rendering this pipeline keeps being asked to stop producing. This subsumes
  the sentence-altitude check above: apply it once, here.

**Density measures what must be SAID; the necessity test measures what must be PERFORMED.** Report
both, and let only the necessity test block: a piece can carry many claims and still read as writing,
where a piece whose every paragraph is pre-decided cannot.

### The counterfactual: has this plan reconceived, or varnished?

Ask it explicitly, and record the answer beside the spine:

> **Strip every named move out of this plan. Is this still the attractor's argument, in its
> original order, at its original granularity?**

**If yes, the spine has varnished and goes back.** Every other check here reasons about claim
completeness and identifier hygiene, and a plan can pass all of them while proposing to leave the
source's architecture exactly where it found it, decorated. The spine holds the power to merge,
split, cut and reorder precisely so it can answer no.

**"The sections changed function" is not an answer.** A spine that returns the source's section
count, in the source's order, under new titles has varnished, whatever it claims about what the
sections now *do*. Function is a statement of intent and nothing downstream can check it, so it is
the escape hatch this question exists to close: one run answered the counterfactual with "the order,
yes, the granularity, no" and shipped seven sections in, seven out, in identical sequence.

**No operation is required, because a required operation is satisfied by a cosmetic one.** Splitting
a section at a paragraph break it already had, or merging two and keeping every paragraph in order,
clears any quota and reconceives nothing. That is the same trap the necessity test above refuses,
arriving one level up.

**What is required is the reversal test, applied to whatever the spine did or did not do:**

> For each architectural choice, **undo it** and ask what changes. Where pressure, dependency, claim
> hierarchy and the reader's state at each boundary all survive the undo materially unchanged, the
> choice was cosmetic.

Applied to an operation performed, it asks whether the merge or cut did any work. Applied to an
order retained, it is stronger: **name the reorder, merge and cut that were tested, and what broke
in each.** "The source order is right" is a finding only where something was tried and failed; left
untested it is the question going unasked.

The check that catches varnishing is downstream of this and is about the reader, not the operation:
**at each section boundary, name what the reader knows, expects or doubts that they would not under
the source architecture.** A spine that cannot answer that at any boundary has re-titled the source,
whatever operations it performed or declined.

**A claim dropped at spine time is invisible to every later stage**, because the per-section audit
checks only claims that were assigned and the final coverage pass checks only the manifest. This
check is the one place it can be caught.

## How the spine's authority is enforced

Every subagent holds the same profile, so left to themselves several will reach for the same
coinage and each will invent its own metaphor:

- **The governing metaphor is the spine's, not the subagent's.** A writer may deepen it and
  **must not** introduce a second image competing to govern the piece. A section that wants a
  different governing metaphor reports that rather than adopting it. **Local imagery that serves one
  passage is not a competing metaphor and needs no permission.**
- **The GOVERNING metaphor appears where `metaphor_locations` says, and nowhere else.** A section outside
  that list leaves the image alone, and doing so is compliance rather than a missed opportunity.
  Requiring every section to touch the image is what produces decorative recurrence: the returns
  stop being returns, and a reader stops registering the image as an image at all.
- **Coinages are allocated.** `coinage_allocation` names which section may introduce a new
  term; the others use terms already established. A subagent coining outside its allocation
  records it, and the stitching pass resolves duplicates.
- **The audit checks compliance, not just quality.** A section carrying a metaphor the spine
  did not assign fails the metaphor row regardless of how well it is written, which is what
  makes the constraint real rather than advisory.

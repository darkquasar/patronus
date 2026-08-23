---
name: writing-like-me
description: >
  Write in your own voice, editorially clean first. Runs the writing-editorial tiers over a draft
  (or over a faceless base draft it composes), derives a voice profile of named moves from your
  exemplar corpus, then a spine that owns the piece's metaphor, opening scene, register and running
  order, voices each section in parallel against both, audits every section for liveness, stitches
  them with narrative continuity, and takes one advisory read from a second model. Use WHENEVER the
  user asks to "make this sound like me", "write this in my voice", "draft this the way I would", or
  hands over a draft and asks to have it voiced. Ships with EMPTY exemplar files by design: it does
  nothing useful until you supply a corpus at ~/.claude/patronus/voice/. Requires the
  writing-editorial skill, which its manifest pulls in automatically.
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
  derive SPINE                       [CHECKPOINT]  main agent, whole-document
    metaphor, scene, register  <- profile
    claims manifest            <- attractor draft
    running order, per-section assignments
    |
    v  spine + profile + section[i] + edit record[i]
  [ s01 ][ s02 ][ sNN ]              voice subagents, parallel
    |
    v  voiced section[i] + restore log
  audit each section                 flat -> rework once -> accept and flag
    |
    v
  stitch                             main agent: narrative continuity, no signposts
    |
    v
  codex advisory read                once, on the finished text
    |
    v  final draft + codex notes taken and declined
```

## Entry modes

This skill is both an editor and a composer, so decide first which one you are. Choose from what the
user supplies:

| Mode | Input | What the voice stage may do |
|---|---|---|
| **Edit** | an existing draft | the prose is the author's and carries authority: diction, rhythm, paragraph shape, cutting and reordering within a section. **Never** replace their sentences wholesale, and **never** inject a claim they did not make |
| **Compose** | a request, no draft | write a faceless base draft **with no corpus in context**, run stage 0 over it, then voice: every sentence may be rewritten |
| **Voice-only** | a draft the user says is already edited | skip stage 0, say so, and voice with no edit records or sections, so no restore is available |

## The attractor

Compose mode still writes its base draft with **no corpus in context**. The reasoning holds: a
model told to sound like someone chases the sound and neglects the thinking.

What changes is its status. It is an **attractor and loose scaffolding**, not a text to be
preserved. It fixes the ideas, the evidence and the citations. **Its sentences carry no authority
at all**, and the voice stage is licensed to demolish and rebuild every one of them.

In edit mode the user's own draft plays this role, and **its sentences do carry authority**: the
never-inject rule binds, and the voice stage may not introduce claims the author did not make.

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

Read `{skillDir}/spine.md` and follow it. Derive the spine once, run its checkpoint checks, and
**show it to the user for approval before any fan-out**.

## Stage 3: the voice subagents

**Spawn one subagent per section named in the spine's `running_order`, in parallel.** That list, not
the directory listing, is what says which sections exist: the spine may have merged, split or cut
sections at Stage 2, and a cut section has no voicer. A piece with one section gets one subagent.

Each receives exactly seven items:

1. its section text, `sections/NN-slug.md`, post-tier-3 and authoritative;
2. its edit record, `sections/NN-slug.edits.yaml`, plus the `source.md` snapshot its offsets
   resolve against;
3. the voice profile;
4. the spine, including its own `per_section_assignment` entry, the claims assigned to it, and any
   `coinage_allocation` it holds;
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
| 3. profile | unchanged |
| 4. spine | unchanged: derived over the whole draft, with every claim assigned to `00-preamble` |
| 5. anti-slop techniques | unchanged |
| 6. attractor slice | the whole attractor draft, which is already the documented fallback |
| 7. mode | unchanged |

**One subagent, no fan-out, and the audit runs once over the piece.** Stitching has nothing to
join and adds nothing. The report says the run was unsectioned and that no restore was available,
so a reader can tell a run that found nothing to restore from one that could not look.

**Do not fabricate section files to satisfy the seven-item list.** An invented `edits.yaml` with no
upstream `source_rev` would offer restores that resolve against nothing.

### Licence differs by mode, and that difference is the whole point of the attractor

*In compose mode*, rewrite every sentence. The section's prose came from a model told to have no
voice, so it carries no authority. What binds is the claims manifest: the ideas assigned to this
section must still be made.

*In edit mode*, the prose is the user's own and it does carry authority. Work on diction, rhythm
and paragraph shape, and cut or reorder within the section, but **never** replace the author's
sentences wholesale, and **never** introduce a claim they did not make. A voice pass that rewrites
a user's draft from scratch has substituted its voice for theirs, which is the opposite of this
skill's purpose.

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
| 1 | Low-perplexity diction | universal | concrete plain-spoken verbs over grand abstractions; "he grabbed the keys", not "he initiated his journey" | abstraction rate per section |
| 2 | No logical signposts | corpus-checked | match the corpus signpost rate; no "furthermore", "moreover", "consequently" bridging paragraphs; let the reader connect | signpost count against the profile rate |
| 3 | Asymmetric formatting | universal | vary paragraph shapes; an eight-sentence paragraph beside a one-sentence paragraph beside a fragment | paragraph sentence-count spread |
| 4 | Internal monologue | corpus-checked | asides, self-correction, doubt shown mid-sentence | asides present where the spine assigned them |

Techniques 1 and 3 are **universal floors**: no corpus is harmed by refusing uniform paragraph
blocks or by preferring a concrete verb.

Techniques 2 and 4 are **corpus-checked**, because they are stance choices that vary by author and
register. A flat ban on signposts misfires on a corpus that uses "however" and opens sentences with
"But" freely. The corpus rate is far below the machine default but is not zero.

**Technique 4 is budgeted at the spine, not per section.** Three asides in one section is a tic;
three across a 2000-word essay is voice. The subagent is told whether its section carries one.

**The budget is set from a rate, not from taste.** The profile carries `hedge_rate_target`, and the
spine allocates against it across the running order the way it allocates asides and frame-breaks:
target rate times draft length, distributed over sections, recorded in the spine so each voicer
knows what its section owes. Where the profile carries no target, fall back to `hedge_rate_corpus`.

Doubt is a stance the corpus can be measured for, so it is allocated rather than left to the
voicer's mood. **A draft with no visible doubt anywhere is the failure this budget exists to
prevent**: prose that knows everything, states each claim as settled, and gives a reader no seam to
think through. `hedge_rate_target` may sit deliberately above `hedge_rate_corpus`, because long-form
has room for reconsideration that a short post does not.

Technique 1's floor is supplemented by the profile, so the replacement for an abstraction is not
generic concreteness but this author's, drawn from the images their own corpus reaches for rather
than from the first plain noun to hand.

## Stage 4: the audit

After voicing, audit each section against `{skillDir}/audit.md`. Send that file to the auditing
subagent **without this one**: an auditor holding the orchestration body inherits the frame of the
stage it is meant to judge.

The auditor also receives the profile, the spine, and **the preceding section in `running_order`**,
which the metaphor row scores against. The first section has no predecessor: score its metaphor row
on whether the metaphor is present and doing work, and say the comparison was unavailable.

Flat fires one rework. A second flat result is **accepted and flagged**, naming the section and what
it failed.

## Stage 5: stitching

The main agent joins the voiced sections in the spine's `running_order` and adds continuity.

**What it may add:** a callback, a returning image, the metaphor deepening across a boundary, the
opening scene's character reappearing.

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

**Then verify claims coverage across the finished piece**, not only per section. A claim assigned to
a section that was later cut, and never reassigned, surfaces here rather than silently vanishing.

## Stage 6: one advisory codex read

Codex runs **once**, over the finished stitched text, via the MCP server (`codex mcp-server`,
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

## What PRESERVE and the ledger bind now

- **PRESERVE is advisory to this stage.** It travels in the edit record marked
  `binding: advisory`, as tier-2's opinion. A subagent may override an entry, **reporting which
  entry and why**.
- **The claims manifest protects ideas**, which are what must survive, rather than spans, which are
  what should be rewritten.
- **The contrast ledger is reconciled at the editorial merge and reported**, not enforced against
  this stage. Where contrast is a named move in the profile, it is allowed.
- **Irreversible house mechanics remain binding on every stage**: no em-dashes, punctuation outside
  closing quotes, British spellings. Those are the author's stated rules.

## Reporting

Report in **this order**:

1. the finished draft;
2. sections flagged as flat after rework, named;
3. claims from the manifest that no section made, if any;
4. tier restores, with their citations;
5. PRESERVE overrides, with reasons;
6. codex notes taken and declined;
7. rhythm guard rails, **labelled as guard rails**.

**The ordering is the point: what the piece failed at comes before what it measured.** Never
present item 7 as evidence the voice landed.

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
  draft-00-original.md  when drafts are kept
  draft-0N-<stage>.md   when drafts are kept
```

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
  sections_voiced: 6
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
- Parallel subagents may converge on the same unallocated image. Each holds the same profile and
  none sees another's text, so two sections can independently reach for the same figure. Only
  stitching sees both, which is where the duplicate is caught rather than prevented.

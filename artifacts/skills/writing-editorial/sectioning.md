# Sectioning, fan-out and merge

This file governs how the tiers are applied to a long draft. The tiers themselves are
unchanged: what changes is how many agents apply them, and at what altitude.

```
 draft
   |
   v
 sectioning            main agent, one pass: cut on headings, assign stable ids
   |
   v  section[i]
 [ s00 ][ s01 ][ sNN ] tiers 0-2, one subagent per section, in parallel
   |
   v  edited section[i] + edit record[i]
 merge                 main agent: concatenate in id order, whole-document tier-2,
   |                   ledger reconcile, consistency normalisation
   v
 tier-3                whole-document, single agent, the only pass that may restructure
   |
   v
 emit                  only when trail-root is supplied
```

## Cutting

**The cut level.** The first heading in the draft establishes the cut level. Every heading at
that level starts a section; deeper headings stay inside their parent. An `h1` standing alone
at the top of the file, with all other headings at `h2`, is the document title rather than a
section boundary, and the cut level is `h2`. Where headings appear at mixed levels below the
cut level, only the cut level divides.

A section is a heading at the cut level and everything under it up to the next heading at
that level. Text before the first heading is section `00`. **Section `00` exists only when there
is such text.** Where the draft opens directly on a heading at the cut level, the first section is
that heading's, and a document title above it belongs to no section: it is carried through
unedited rather than filed under a `00` invented to hold it.

**Ids.** Each section gets an id of the form `NN-slug`: `NN` is its ordinal at cut time,
`slug` is a kebab-case reduction of its heading truncated to six words. **`00` is reserved for the
preamble**, so a draft that opens directly on a heading numbers its first section `01`. Numbering
the first heading `00` reads as a preamble that is not there. A section with no
heading takes the reserved slug `preamble`, so the headless case is `00-preamble` rather than
a bare `00-`.

Ids are **identity, not sequence**. A downstream stage may present sections in any order, and
the id it joins on never moves. Ids are append-only: a later reshaping mints a new id and
**never** reassigns a parent's.

A draft with no headings is one section, `00-preamble`. **Do not invent headings to create
sections.** A 2000-word unheaded essay is a legitimate single section, and cutting it on
paragraph boundaries would produce subagents reasoning about fragments.

The same judgement runs the other way. **Headings that label rather than divide are not cut
points.** A commonplace book, a numbered post series and a collection of aphorisms carry headings
that work as timestamps, and cutting on them yields dozens of sections a few sentences long. A
section that short gives a subagent nothing to judge: tier-2's structural rules need a paragraph or
more of runway, and a document cut this way spends its cost on fragments while inviting exactly the
cross-section divergence this file exists to prevent. **Where the cut would produce many sections
whose text runs shorter than a few paragraphs, say so and run whole-document instead**, at option 1
or 2. Report the cut you declined and why, rather than fanning out silently.

**Spans.** At cut time, split each section into paragraph-level spans, numbered in document
order, giving span ids of the form `<section-id>/pNN`. Every edit anchors to one. The schema
and the resolution rules are in `{skillDir}/edit-record.md`.

**The source snapshot.** The main agent takes each section's pre-tier text as its `source_rev`
snapshot at cut time, before any subagent runs, and every offset in that section's record resolves
against it. Under `trail-root` the snapshot is written to `sections/NN-slug.source.md`.

## Fan-out: tiers 0, 1 and 2

One subagent per section, in parallel. Each receives its section text, the tier files for 0
through 2, its section id, and the shared decision sheet below. It returns the edited section
and its edit record.

**What a per-section subagent does not apply.** Four tier-2 entries have the whole piece as
their evidence rather than a span, and no prompting makes a subagent holding one section able
to judge them:

- one-point dilution
- fractal summaries
- the dead metaphor, where one image is beaten flat across a whole piece
- title-case headings, which are judged against what the surrounding document does

These run at the **merge pass** instead, applied by the main agent over the concatenated
document. The per-section subagents apply the span-local remainder of tier-2. No rule's
judgement changes; where four of them run does.

Tier-2.3's paragraph-size bullet stays with the section subagents. Where the flattening runs
wider than one section, tier-2.3 already says to report it as an unresolved finding, and
tier-3 reads the whole document.

**The contrast ledger under fan-out.** Tier-1.3's ledger is a document-level budget, which a
per-section subagent cannot hold. Each subagent **reports** the contrastive constructions it
found and removed in its section rather than enforcing a global allowance. The main agent
reconciles at merge.

**Tier-2 emits PRESERVE entries per section**, each carrying the section id.

## Keeping parallel sections consistent

Independent subagents make independent choices. Left alone they diverge on terminology, on
which of two honest repairs a rule offers, on heading case, and on whether a construction is a
machine tell or the author's habit. Concatenating divergent sections produces a document that
reads as several people wrote it, which is its own machine tell.

**The shared decision sheet.** Before fan-out, extract the draft's recurring terms, its
heading shapes and its spelling conventions, and pass that sheet to every subagent as binding.
A subagent needing a decision the sheet does not cover records it in its edit record rather
than deciding silently.

**Normalisation at merge.** Read the recorded decisions, pick one where they conflict, apply
it across the merged document, and report the conflicts resolved.

## Merge

Concatenate the edited sections in id order, then, as the main agent:

1. **Apply the four whole-document tier-2 entries** named above.
2. **Reconcile the ledger.** Count the contrastive constructions surviving across the merged
   document. Where the reconciled count exceeds the allowance, **select which instance is
   retained, rewrite the others in the positive per tier-1.3, and record the reconciliation.** A
   count alone cannot do this: three sections may each independently keep the one allowance, and
   counting afterwards discovers three where one was allowed. The surplus is repaired the way
   tier-1.3 repairs it, by naming the mechanism the mirror gestured at; it is not restored by
   reverting an edit, which would raise the count rather than lower it. This is the one place the
   merge edits text rather than concatenating it, and the selection is reported.
3. **Normalise consistency** per the decision sheet, reporting the conflicts resolved.

## Tier-3

Runs once, over the merged document, by a single agent.

Tier-3 is whole-document because it is **state-dependent and compositional**. Most of its
rules are sentence- and paragraph-local. What makes it whole-document is that it is the only
pass permitted to restructure, and that it reads two document-level inputs, the PRESERVE list
and the contrast ledger, which no per-section subagent holds. A fanned-out tier-3 would
compose against an allowance it could not see.

Tier-3's edits anchor the same way as every other edit: to a span id, with `source_rev` and
offset. An edit spanning sections, such as a reordering, anchors to `document` and records the
section ids it moved.

## Projecting tier-3 back into sections

Tier-3 operates on the merged document, so project its results back into section files before
anything downstream reads them:

- Text tier-3 **moved** between sections lands in the destination section's file, and
  `lineage.yaml` records the move with both section ids.
- Text tier-3 **cut** disappears from the section file and survives only in its edit record.
- Text tier-3 **added**, such as a repaired ending, is appended to the section it falls in
  with a fresh span id, recorded `origin: tier-3` since it has no `source_rev` span.
- A tier-3 edit **spanning sections** anchors to the reserved span id `document` and lists
  every section id it touched in `sections:`, per `{skillDir}/edit-record.md`.

Where tier-3 restructures so heavily that a section's text no longer corresponds to its
source, record it in `lineage.yaml` with `derived: true` and `offsets_usable: false`. The
edit record is kept, because `removed` still says what the text held; only the offsets are
unusable.

## Emitting the trail

**Only when the caller supplies `trail-root`.** The caller owns the path: never invent one,
never resolve one from the repository, and never ask the user where a trail should live. That
decision belongs to the calling skill.

**Fan-out and emission are two separate switches, and this file governs only what happens when
the first one is on:**

| Dispatch option | Fan-out | `trail-root` | Result |
|---|---|---|---|
| 1 or 2 | no | either | this file does not apply; the tiers run whole-document as the router describes |
| 3 | yes | absent | cut and fan out, write **nothing** to disk, return the inline report |
| 3 | yes | supplied | cut, fan out, and write the artifacts above |

Option 3 is what turns fan-out on, and the router says when it is chosen: the caller picks it, and
it is the default only when `trail-root` is supplied. **A caller who took option 1 or 2 gets no
sectioning at all**, which is what keeps a direct caller's cost unchanged.

```
<trail-root>/<slug>/
  sections/
    00-preamble.source.md      pre-tier text; the source_rev snapshot offsets resolve against
    00-preamble.md             post-tier-3 text; authoritative, and what downstream consumes
    00-preamble.edits.yaml     span-anchored records
    ...
  sections/lineage.yaml      one entry per section: what it derives from and whether its offsets resolve
  <draft>-edited.md            the merged, tier-3'd draft: the standalone output
  EDITS.md                     human-readable summary
  manifest.yaml                plus the sections block below
```

**Write all four run-level files on every run, not only when something interesting happened.** A
run with no moves still emits `lineage.yaml`, with `op: origin` for each section: that is what tells
a consumer the sections were left in place, which an absent file cannot say. The same holds for
`EDITS.md` and `manifest.yaml`. A downstream stage joins on `section-lineage/v1` and reads the
manifest's `merge:` block to interpret the records, so a trail missing either is a trail it cannot
use.

Both section files are needed. Without the post-tier-3 file, a consumer reading section files
reads text the final draft no longer contains; without the source snapshot, no offset in any
edit record resolves.

**The manifest's sections block.** One entry per section, plus the run-level facts a reader needs
to interpret the fan-out. This is where a companion's absence is recorded, and where the merge
reports what it reconciled:

```yaml
sections:
  cut_level: h2                       # the heading level that divided the draft
  count: 3
  companion: supplied                 # supplied | absent
  entries:
    - id: 00-preamble
      heading: null                   # null for the reserved preamble
      edits: 4
      companion: absent               # this section has no counterpart in the companion text
    - id: 01-the-relocation-argument
      heading: "The Relocation Argument"
      edits: 7
      companion: present
  merge:
    ledger_retained: "not a productivity setup, but a distributed system"
    ledger_rewritten: 2               # surplus instances rewritten positive so the allowance holds
    whole_document_rules_fired: [one-point dilution]
    consistency_conflicts_resolved: 1
```

`companion: absent` on an entry is the recorded absence: the `.companion.md` file is written
empty **and** the entry says so, because an empty file alone cannot distinguish "no counterpart"
from "a counterpart that was empty".

## A companion text

The skill accepts an optional `companion` text alongside the draft, for a caller whose
downstream stage needs a second text sliced the same way.

When supplied, cut it with the **same section ids as the draft**, matched by heading text, and
write it to `sections/NN-slug.companion.md`. Where the companion has no counterpart for a
section, write the file empty and record the absence. Heading text is the only join available,
so a companion whose headings diverge from the draft's yields empty counterparts rather than a
guessed alignment.

**No tier runs over the companion.** It is carried, not edited.

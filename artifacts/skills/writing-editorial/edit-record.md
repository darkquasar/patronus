# Edit records and section lineage

Two schemas. A downstream stage joins on both, so their field names are an interface rather
than an implementation detail.

## editorial-edit-record/v1

One file per section, at `sections/NN-slug.edits.yaml`.

```yaml
schema: editorial-edit-record/v1
section_id: 01-the-relocation-argument
heading: "The Relocation Argument"
source_rev: 3f9a1c                    # content hash of NN-slug.source.md
edits:
  - id: e01                           # unique within the section
    tier: 1
    rule: tier-1.3
    span: 01-the-relocation-argument/p03
    offset: 142                       # character offset into the span, as of offset_rev
    offset_rev: tier-0                # which text this offset was measured against
    removed: "not a productivity setup, but a distributed system"
    replaced_with: "a distributed system whose nodes happen to be partially synchronised workstations"
    occurrence: 1                     # which match, when removed appears more than once in the span
    reason: "mirrored swap; ledger allowance spent at document level"
    reversible: true

  - id: e02
    tier: 2
    rule: tier-2.2/1A
    span: 01-the-relocation-argument/p07
    offset: 31
    offset_rev: tier-1
    removed: "genuinely"
    replaced_with: ""
    occurrence: 2                     # the second "genuinely" in this span
    reason: "vogue intensifier"
    reversible: true

preserve:
  - span: 01-the-relocation-argument/p05
    offset: 12
    text: "It splits in two."
    reason: "the piece's thesis turn"
    binding: advisory

decisions:
  - term: "workstation"
    chose: "workstation"
    over: ["machine", "box"]
    reason: "the draft's own dominant term"
```

### decisions

A subagent holding one section decides terminology, heading case and spelling for that section
alone. `decisions` records each choice the shared decision sheet did not already cover, so the
merge pass can read them, resolve conflicts across sections and normalise the merged document.
**A choice made silently is a choice the merge cannot reconcile**, which is how independently
edited sections drift into reading as several authors.

### Span ids

At cut time, each section's text is split into paragraph-level spans, numbered in document
order. A span id is `<section-id>/pNN`. An edit anchors to a span id rather than to quoted
text, because an edit record must survive two later rewrites: tier-3 restructuring the merged
document, and a downstream voice stage rewriting every sentence. Quoted text survives
neither.

### Locating an edit takes four fields, not one

`span` plus `offset` plus `occurrence`, resolved against `source_rev`, identify exactly one
position. Quoted text alone does not: `replaced_with: ""` carries no location, and a word
like "genuinely" may appear several times in one span. A consumer that cannot resolve all
four reports the edit as unresolvable rather than guessing at a match.

### An edit that spans sections

Tier-3 may reorder or move text across a section boundary, and such an edit belongs to no one
span. It anchors to the reserved span id `document`, carries no `offset`, and names the
sections it touched:

```yaml
  - id: e07
    tier: 3
    rule: tier-3.7
    span: document                      # reserved: this edit belongs to no single span
    sections: [02-the-second-house, 04-the-argument]
    removed: ""
    replaced_with: ""
    occurrence: 1
    reason: "moved the cost paragraph under the argument it supports"
    reversible: true
```

`sections:` appears **only** on a `span: document` edit. **An edit anchored to `document` is
not resolvable by offset**, and a consumer restores it by reading `reason` and the lineage
rather than by computing a position.

**It is written once, into the first section id in `sections:`, and never duplicated.** Writing
it into every section it touches would make one edit look like several, and a consumer counting
edits or replaying restores would apply it twice. A consumer looking for the edits affecting a
section therefore reads that section's own file **plus** any `span: document` edit naming it in
`sections:`.

`removed` holds the text verbatim, and is what makes a restore possible: a downstream stage
puts back exactly what the draft had, rather than paraphrasing what a count implies.

### Four tiers edit in sequence, so an offset needs to say which text it saw

Tier-1 edits text tier-0 already changed, and tier-2 edits what tier-1 left. **An offset recorded
by a later tier is not an offset into `NN-slug.source.md`**, so a single record-level `source_rev`
cannot locate every edit in the file.

Two fields resolve this, and both are required on every edit:

- **`occurrence` is the primary locator, and it survives earlier edits.** "The second `genuinely`
  in this span" stays true whether or not tier-0 removed a phrase ahead of it. A consumer resolves
  by scanning the span for the nth match of `removed`, and uses `offset` only to disambiguate.
- **`offset_rev` names the text the offset was measured against**: `source` for the pre-tier
  snapshot, or the tier that produced the text this tier saw (`tier-0`, `tier-1`, `tier-2`).

```yaml
    offset: 142
    offset_rev: tier-1        # this offset was measured in the text tier-1 left behind
    occurrence: 2             # the locator that survives earlier edits
```

**A consumer that cannot reproduce an `offset_rev` treats the offset as advisory and resolves by
`occurrence` alone.** Only `offset_rev: source` is reproducible from the emitted artifacts, since
the intermediate texts are not kept. Recording which text an offset saw is what stops a consumer
applying it to the wrong one, which is the silent-corruption case this whole schema exists to
prevent.

`occurrence` is counted in the text its tier saw, so an earlier tier that added or removed another
match of the same string shifts it. Where the ordinal is ambiguous against the `source_rev` snapshot,
a consumer **reports the edit unresolvable rather than applying it to a guessed match**: a wrong
match is a silent corruption, and an unresolved edit is a visible one.

Where `occurrence` cannot resolve either, because the text the edit removed is itself gone from the
snapshot, the edit is **reported unresolvable**. That is the honest outcome, and it is why
`removed` holds the text verbatim: a human reading the record can still see what was taken.

### Offsets are valid only at their source_rev

A consumer that has rewritten the text does **not** apply an offset computed against an
earlier revision. It joins on the span id, which is stable, and treats the offset as a hint
for locating the original span in the `source_rev` snapshot. This is why full section
snapshots are kept: without them, an offset into rewritten text is a silent corruption.

### reversible

`reversible: false` marks edits that are **never** undone: house-rule mechanics such as
em-dash removal, quote punctuation, and British spellings. These are the author's stated
rules rather than editorial judgement, and no downstream licence extends to them.

`binding: advisory` on a PRESERVE entry restates tier-2's contract. A later stage may
override an entry when it says which one and why.

## section-lineage/v1

One file per run, at `sections/lineage.yaml`.

```yaml
schema: section-lineage/v1
sections:
  - id: 00-preamble
    derives_from: [00-preamble]
    op: origin

  - id: 02-the-second-house
    derives_from: [02-the-second-house]
    op: moved                           # tier-3 moved text into or out of this section
    moved_from: [04-the-argument]       # the other section ids the move touched
    offsets_usable: true

  - id: 03-what-it-costs
    derives_from: [03-what-it-costs]
    op: origin
    offsets_usable: false               # tier-3 rewrote it wholesale: this section is derived
    derived: true

  - id: 02+03-the-second-house          # merge: ids joined with +, in source order
    derives_from: [02-the-second-house, 03-what-it-costs]
    op: merged
    edit_records: [02-the-second-house, 03-what-it-costs]

  - id: 04a-the-argument                # split: parent id plus a letter suffix
    derives_from: [04-the-argument]
    op: split
    edit_records: [04-the-argument]
  - id: 04b-the-objection
    derives_from: [04-the-argument]
    op: split
    edit_records: [04-the-argument]

  - id: 05-the-aside
    derives_from: [05-the-aside]
    op: cut
    edit_records: []                    # text is gone; restores against it are void
```

`op` takes one of five values: `origin`, `moved`, `merged`, `split` and `cut`. The editorial
stage writes only `origin` and `moved`; a downstream voice stage writes the rest, and both
write this schema.

### Spans tier-3 created

Tier-3 may add text, such as a repaired ending, that no `source_rev` span contains. Those spans
are listed per section, so a consumer knows which span ids have no snapshot behind them:

```yaml
    new_spans:                          # spans with no source_rev text behind them
      - id: 01-the-relocation-argument/p09
        origin: tier-3
```

**A span in `new_spans` has no restorable history.** An edit anchored to one carries no usable
offset, and a consumer reads it rather than resolving it.

Every entry carries `id`, `derives_from` and `op`. Three fields are conditional:

| Field | On | Meaning |
|---|---|---|
| `edit_records` | `merged`, `split`, `cut` | which sections' records apply to this id |
| `moved_from` | `moved` | the other section ids the move touched |
| `offsets_usable` | any | whether this section's edit offsets still resolve |
| `derived` | any | tier-3 rewrote the section so heavily its text no longer corresponds to its source |

**`derived: true` always accompanies `offsets_usable: false`.** A derived section keeps its
edit record, because `removed` still says what the text once held, but no offset in it
resolves. Saying so is better than emitting offsets that silently do not.

The rules a consumer applies:

- **Moved**: the section's own record still applies, and `moved_from` names where the text
  came from. Offsets in the destination section are usable only if `offsets_usable` is true.
- **Merged**: both parents' edit records apply. A restore names its parent record, and
  offsets resolve against that parent's `source_rev` snapshot.
- **Split**: the parent's record applies to both halves. An edit whose span falls in only one
  half resolves there; a consumer that cannot tell reports it unresolvable.
- **Cut**: the section's edit records are void. Restores against cut text are rejected,
  because there is nowhere to put the text back.
- **Ids are append-only.** A new id is minted for a merge or split; parent ids are **never**
  reused and **never** silently reassigned. This is what lets an edit record written before a
  reshaping still find its text after one.

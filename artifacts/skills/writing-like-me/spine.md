# The spine

Derived once, by the main agent, after reading the cleansed draft and the profile, **before any
fan-out**. Two inputs, and the split matters:

- **from the attractor draft**: the claims manifest, and nothing else;
- **from the profile**: metaphor, opening scene, register and frame-breaks, all *invented* at
  spine time by an agent that has just read the corpus.

Voice lives in conception: which scene opens, which metaphor spans the piece, where the author
lets themselves be wry. A spine derived after the prose is fixed can only varnish it.

## Shape

```yaml
governing_metaphor: >
  The pipeline as a house you keep the deeds to. The agentic turn builds a second house and
  moves half the furniture across, so there are two sets of deeds to keep honest. Each
  section touches this at least once, and it gains depth rather than repeating.

opening_scene: >
  The Tuesday afternoon: a four-line rule, ninety seconds to write, ninety minutes waiting on
  a runner image whose credential lives in a vault she cannot reach. Later sections may call
  back to her.

register: >
  Wry, occasionally mythic. First person present and unhedged where the author has a view.
  Breaks frame to address the reader twice across the piece.

claims_manifest:
  - id: c1
    claim: "The repository is a state-management system, not an authoring workflow"
    assigned_to: ["00-preamble"]
  - id: c2
    claim: "The centralised surface does not shrink, it splits in two"
    assigned_to: ["02-the-second-house", "04-the-argument"]   # may land in more than one

# Full section ids throughout, never the bare ordinal: these join to the upstream
# edit records and lineage, which key on NN-slug.
running_order:
  - 00-preamble
  - 02-the-second-house
  - 01-the-relocation-argument
  - 04-the-argument

per_section_assignment:
  "02-the-second-house": "carries the metaphor's deepening: this is where the second house gets furnished"
  "04-the-argument": "frame-break lands here, plus one aside per the monologue budget"

coinage_allocation:
  "02-the-second-house": ["second house"]      # only this section may introduce this term
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
| merge `02-the-second-house` + `03-what-it-costs` | `02+03-the-second-house` | `op: merged`, both parents in `derives_from` | both parents' records |
| split `04-the-argument` | `04a-the-argument`, `04b-the-objection` | `op: split`, parent in `derives_from` | the parent's record, resolved by span |
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
- a bare ordinal anywhere an id belongs.

**A claim dropped at spine time is invisible to every later stage**, because the per-section audit
checks only claims that were assigned and the final coverage pass checks only the manifest. This
check is the one place it can be caught.

## How the spine's authority is enforced

Every subagent holds the same profile, so left to themselves several will reach for the same
coinage and each will invent its own metaphor:

- **The governing metaphor is the spine's, not the subagent's.** A subagent may deepen it and
  **must not** introduce a competing one. A section that wants a different metaphor reports
  that to the main agent rather than adopting it.
- **Coinages are allocated.** `coinage_allocation` names which section may introduce a new
  term; the others use terms already established. A subagent coining outside its allocation
  records it, and the stitching pass resolves duplicates.
- **The audit checks compliance, not just quality.** A section carrying a metaphor the spine
  did not assign fails the metaphor row regardless of how well it is written, which is what
  makes the constraint real rather than advisory.

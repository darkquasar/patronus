---
status: accepted
date: 2026-07-12
---

# A spec folder is one research effort with many streams; a stream is one spec and one plan

## Context

`docs/specs/NN-slug/` was designed as though it held **one** feature: one `spec.md`, one `plan.md`,
one `research.md`, and a `meta.yaml` whose `completeness:` block carried a **scalar flag per document
type**:

```yaml
completeness:
  research: false
  spec:     false
  plan:     false
  tasks:    false
```

The first real folder broke that model immediately. `01-lifecycle-and-test-surface` is **one
investigation that produced two independent work streams** — a lifecycle-skills rewrite and a
test-surface refactor. They share a research doc (the same evidence motivates both) and share nothing
else: different files, different reviewers, different order, different definitions of done. Two specs
were written. Two plans were written.

The scalar schema could not say that. It said:

```yaml
completeness:
  spec:     true      # both lifecycle-skills-spec.md and test-surface-spec.md
  plan:     false
```

— a boolean carrying a **hand-written comment about the two things it was actually summarizing**. And
the moment one stream got planned and the other did not, `plan:` had no honest value to take. A
flattened bit plus a prose annotation is not a record; it is a note explaining why the record is
wrong.

**That is the defect this whole branch exists to eliminate:** *the thing that REPORTS the state had
diverged from the thing that IS the state, and nothing reconciled them.* Here the schema itself
guaranteed the divergence — it had no shape in which the truth could be written down.

The naming convention (`<slug>-spec.md` rather than `spec.md`, established alongside this) is a
symptom of the same pressure: you only need to prefix a filename when there is more than one of it in
the folder.

## Decision

**The folder is one research effort. A stream is one spec and one plan. The folder has many streams.**

```
docs/specs/NN-slug/                          ← ONE research effort
├── meta.yaml
├── <slug>-research.md                       ← ONE per folder. It is what makes the folder a folder.
├── <stream>-spec.md   ┐
├── <stream>-plan.md   ┘                     ← ONE stream: exactly one spec, exactly one plan
├── <stream2>-spec.md  ┐
└── <stream2>-plan.md  ┘                     ← another stream
```

1. **Research is folder-level and singular.** One investigation; it is the thing the folder *is*. If
   there is a second, unrelated investigation, that is a second folder.

2. **A stream is the unit of independent work**, and it is where a spec forks into something that can
   be planned, reviewed, ticketed, and shipped on its own.

3. **Spec ↔ plan is one-to-one.** A spec with two plans has no meaning. A plan implementing two specs
   is a spec whose boundary was never drawn. Where you want to split a plan, split the *spec* — that
   is what a stream is.

4. **Every stream has a spec. The user may override.** The default is a spec even when the work looks
   small enough to plan directly, because `spec-review` and `plan-review` are **different lenses** —
   one asks *"is this the right thing to build"*, the other *"is this buildable as written"* — and
   collapsing them silently discards the cheaper one. A user who explicitly declines the spec gets a
   plan-only stream; the skill records that it was declined rather than pretending the spec exists.

5. **Completeness is per stream, not per folder.** Each stream carries its own state, so a folder with
   one stream planned and one not can *say so*.

6. **A file's presence IS its completeness flag.** `meta.yaml` names the file
   (`spec: test-surface-spec.md`); it does **not** carry a redundant `spec: true` beside it. Naming a
   file and asserting it exists are the same act, and keeping both is how you end up with `spec: true`
   next to a file nobody wrote.

7. **`tickets:` is per stream, and it records the tk EPIC ID, not a boolean.** Seeding is one epic per
   plan, so "seeded?" is a per-stream question. And an id (`epic: pat-a1b2`) is **checkable** —
   `tk show pat-a1b2` either resolves or it does not — where `tickets: true` can only be believed.
   This is the same move as the drift guard: record the thing, then actually read it back.

```yaml
slug: 01-lifecycle-and-test-surface
intent: "One line: what this investigation was."
created: 2026-07-12
updated: 2026-07-12

research: lifecycle-and-test-surface-research.md   # ONE per folder; presence = done

streams:
  - slug: lifecycle-skills
    intent: "Playbook, tk scaffolding, team protocol, artifact drift guard."
    spec: lifecycle-skills-spec.md      # omit ONLY if the user declined a spec (see below)
    plan: lifecycle-skills-plan.md
    epic: null                          # the tk epic id once seeded, e.g. pat-a1b2
  - slug: test-surface
    intent: "Fixture catalog, R-SEC, the archive-SKIP security hole."
    spec: test-surface-spec.md
    plan: test-surface-plan.md
    epic: null
```

A stream whose spec was declined records the decision rather than the absence:

```yaml
  - slug: quick-fix
    intent: "One line."
    spec: null
    spec_declined: "User opted to plan directly (2026-07-12) — scope judged too small for a spec."
    plan: quick-fix-plan.md
    epic: null
```

**The stream `slug` is the one name that ties everything together:** `<slug>-spec.md`,
`<slug>-plan.md`, and `--tags <slug>` on that stream's tickets. One name, three places, all
mechanically checkable.

## Consequences

- **The manifest becomes checkable rather than believable.** Two invariants:
  1. every filename `meta.yaml` names **resolves to a file on disk**;
  2. every `*-spec.md` / `*-plan.md` in the folder **is named in `meta.yaml`**.

  A scalar `spec: true` supported neither. (A folder-lint that enforces both is worth building; it is
  the same "read the record back" discipline as `patronus scan`'s drift guard. **Parse the YAML — do
  not grep the file.** A `grep -oE '\S+\.md'` also matches filenames inside *comments*, and will
  report a false violation for a comment that cites this ADR. Read the `research:` and
  `streams[].{spec,plan}` **values**; ignore everything else.)

- **`completeness.tasks` is dead, and it does not become `completeness.tickets`.** It becomes
  `streams[].epic` — an id you can resolve, not a bit you can only trust. The old `tasks:` flag meant
  *"a `tasks.md` exists"*, and `tasks.md` no longer does.

- **Four skills now read and write this schema**: `brainstorming` (creates the folder and its first
  stream), `team-research` (writes the research doc and N streams), `writing-plans` (fills in its
  stream's `plan:`), and `team-implement` (fills in `epic:` when it seeds the graph). Each owns exactly
  the field it produces — no skill sets a flag for work it did not do.

- **"One feature per folder" is abandoned, deliberately.** `01-lifecycle-and-test-surface` is a
  conjunction, and nobody would call it one feature. Forcing folders to be features would have split
  a **shared research doc** across two folders — duplicating the evidence, and guaranteeing that the
  two copies drift. The research is precisely what the two streams have in common, so it is what the
  folder is organized around.

- **The cost:** a folder no longer has one obvious "am I done?" bit. That is the point. It never
  honestly had one; it had a bit plus a comment explaining why the bit was wrong.

## The general rule this is an instance of

> **A schema that cannot express the truth will be filled with a claim instead.**

`spec: true  # both lifecycle-skills-spec.md and test-surface-spec.md` is not a record — it is a
scalar field, a prose apology for the scalar field, and a state nobody can query. When the shape of
the record cannot hold the shape of the world, the gap gets papered over with a comment, and the
comment is exactly the kind of thing that goes stale silently.

Prefer a record that **names the artifact** (a filename, a ticket id) over one that **asserts a fact
about it** (`true`). The first can be verified against the thing. The second can only be believed.
See also [ADR-0002](0002-tests-assert-behavior-not-artifacts.md), which is the same rule applied to
tests.

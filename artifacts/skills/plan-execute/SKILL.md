---
name: plan-execute
description: Use when you have a written implementation plan to execute and the tasks do not split into disjoint file-owning boundaries
---

# Plan Execute

Execute a written plan. Choose the execution mode by proportionality: does independent
per-task review earn its cost for this plan?

**Announce at start:** "I'm using the plan-execute skill to implement this plan."

## Where this sits

`plan-review` forks first, on parallelism. If the plan's tasks split into disjoint
file-owning boundaries, that fork sends the work to `plan-execute-parallel` and this
skill never runs. You are here because the tasks share files or must land in order.

Inside that arm, this skill forks again, on proportionality:

```
  plan-execute
  |
  +-- hard trigger, or 2+ soft signals with 2+ implementation tasks --> sdd.md
  +-- otherwise                                                      --> solo.md
```

Both modes end with an independent whole-branch review. That is what makes two modes
enough: there is no "solo but reviewed" gap to fill, because solo is already reviewed at
the end.

## Step 1: Load the plan

Read the plan file. Review it critically and identify questions or concerns about the
plan itself, before deciding how to run it. If you find concerns, raise them with your
human partner before starting.

## Step 2: Resolve the mode

Apply these rules **in order**. The first one that fires decides.

**Rule 1: an explicit user mode request wins outright.** If the invocation names a mode,
use it and skip the assessment entirely. There is no flag to parse: match on the mode
words in what the user actually said. "Execute this plan solo", "run it in sdd mode",
"just do it solo" all count. When Rule 1 fires, the record says the mode was requested,
not assessed.

**Rule 2: any hard trigger selects `sdd`**, regardless of how many tasks the plan has. A
one-task schema migration goes to `sdd`.

Hard triggers:

- security or trust boundary (auth, secrets, crypto, sandboxing, untrusted input,
  permission expansion)
- irreversible or hard-to-recover state change (schema migration, destructive data
  operation, deployment cutover, public API removal)
- concurrency or distributed correctness (locking, retries, idempotency, ordering,
  transactions, caches)
- compatibility contract (public API, wire format, persisted format, CLI compatibility)
- weak verification: important behaviour the plan cannot cover with deterministic
  automated tests
- high blast radius: shared framework, installer, profile resolver, or code with several
  independent consumers

**Rule 3: two or more soft signals select `sdd`**, but only if the plan has at least two
implementation tasks. An implementation task changes source; docs-only and scaffolding
tasks do not count toward the floor. With fewer than two, SDD's startup cost cannot
amortize: select `solo`.

Soft signals:

- introduces a new architectural pattern rather than following an existing one
- correctness depends on an invariant spanning three or more modules
- changes both producer and consumer sides of a contract
- acceptance criteria use qualitative terms ("appropriate", "robust") with no exact
  oracle
- relies on negative requirements ("must never", "no behaviour change") that tests
  commonly miss
- implementation requires choosing among several plausible designs

**Rule 4: otherwise `solo`.**

The floor in Rule 3 binds soft-signal routing only. It never overrides Rule 2.

Raw task count is not a criterion beyond that floor. Twenty mechanical tasks with no
trigger is a `solo` plan.

### Reading the plan, not the words in it

A trigger fires on what the plan **does**, not on vocabulary that appears in it. A docs
task that mentions "schema migration" while implementing nothing is not a hard trigger.
Qualitative wording that another section of the same plan pins to an exact oracle is not
a soft signal: read the whole plan before scoring it.

### Guarding against your own bias

Under cost pressure you will drift toward `solo`. Primed by "sdd is higher quality" you
will drift toward `sdd`. The citation requirement is the check. A mode chosen with no
nameable, quotable trigger is a preference, not a decision, so name the plan section or
choose the other mode.

## Step 3: Emit the decision record

State the choice, then continue. Do not stop for an answer: an unattended run would
stall at the prompt.

```
Mode: sdd
Hard triggers: Task 3 alters the persisted lockfile format (compatibility contract)
Soft signals: none needed
Proceeding. Say "run this solo" to override.
```

Cite the plan sections that drove the decision, by task number or heading. Every citation
must point at a section that exists and must actually support the trigger it is named
for. A record whose citation does not hold up is a gate failure, not a valid record: go
back to Step 2 and score the plan again.

When Rule 1 fired, say so:

```
Mode: solo (requested)
Gate skipped: you asked for solo.
Note: this plan has a hard trigger (Task 2 expands file permissions). Proceeding solo.
Proceeding.
```

The record names one mode for the whole plan. There is no per-task mode switching.

## Step 4: Load the sidecar and run it

- `sdd` → read [sdd.md](sdd.md) and follow it.
- `solo` → read [solo.md](solo.md) and follow it.

Read one. The mode you did not choose does not enter your context.

## Integration

- **plan-writing** — creates the plan this skill executes.
- **plan-review** — the gate before this one; it forks on parallelism and hands the
  non-parallel arm here.
- **requesting-code-review** — ships the `code-reviewer.md` prompt both modes use for
  the final whole-branch review. The `requires:` edge installs it alongside this skill.
- **finishing-a-development-branch** — takes over once the whole-branch review is clean.

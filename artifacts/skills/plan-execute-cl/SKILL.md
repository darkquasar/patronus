---
name: plan-execute-cl
description: Use when you have a written implementation plan to execute and the tasks do not split into disjoint file-owning boundaries
---

# Plan Execute

Execute a written plan. The coordinator chooses the execution mode by ownership, risk and context needs.
SDD uses fresh implementer leaves, not automatic per-task reviewers.

**Announce at start:** "I'm using the plan-execute-cl skill to implement this plan."

## Where this sits

The coordinator, not the plan reviewer, selects execution mode. Disjoint isolated
file owners may use `plan-execute-parallel-cl`; shared files or ordered tasks use
this serial route. Valid existing grants remain valid within scope.

Inside that arm, this skill forks again, on proportionality:

```
  plan-execute-cl
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
- a named safety/compatibility invariant with uncertain verification before dependent
  consumption, or a named irreversible acceptance boundary

Weak tests, framework use, several consumers or task count alone do not create an
earlier review gate. A hard execution trigger selects fresh context; it does not
admit a task reviewer. Earlier review requires the named changed invariant or
irreversible boundary, and never replaces aggregate review.

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
Mode assessment skipped: you asked for solo. Safety and aggregate review remain.
Note: this plan has a hard trigger (Task 2 expands file permissions). Proceeding solo.
Proceeding.
```

The record names one mode for the whole plan. There is no per-task mode switching.

## Step 4: Load the sidecar and run it

- `sdd` → read [sdd.md](sdd.md) and follow it.
- `solo` → read [solo.md](solo.md) and follow it.

Read one. The mode you did not choose does not enter your context.

## Integration

- **plan-writing-cl** — creates the accepted plan this skill executes.
- **plan-review-cl** — the independent plan-review subject; the coordinator chooses execution mode.
- **requesting-code-review-cl** — ships the `code-reviewer.md` prompt both modes use for
  the final whole-branch review. The `requires:` edge installs it alongside this skill.
- **finishing-a-development-branch-cl** — consumes the accepted aggregate evidence within its separate grant.

Only the coordinator admits leaves and successors. No mandatory numerical root
budgets, per-task reviewer chains or automatic model escalation. Use the sole
selected progress authority. Read the chosen sidecar's bounded correction,
settlement, TDD and aggregate acceptance contract. Explicit solo does not waive
those safety and review obligations.

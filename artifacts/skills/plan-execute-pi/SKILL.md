---
name: plan-execute-pi
description: "Execute approved plans with proportionate solo or fresh task review"
---

# Plan execute

Coordinator routing only. Leaves execute their assigned task and never delegate. Read the approved plan, exact source/root/base and current native state before selecting a mode. For independent disjoint lanes use plan-execute-parallel-pi with serial consumption/integration barriers; task count alone never justifies fanout.

## Scope and recovery

Act only within the task's current grant. Research, authorship, review, planning, execution, integration, publication and cleanup are separate actions. Existing explicit approval remains valid within its recorded root, outputs and revalidation conditions; do not ask again merely because the stage changed. Changed scope or destructive actions need revalidation. Before work and after compaction, reread the mandatory brief, native mission state, latest decisions and project instructions, then inspect current artifacts. A leaf never delegates, mutates native mission state or writes lessons. Only the coordinator does so when authorized. Use contact_supervisor need_decision/interview_request for a material unresolved choice and wait for the actual reply; steering delivery is not consent. Do not switch model, provider, protocol or isolation on failure.
## Select testing before implementation

Record `critical-tdd` or `focused-postcheck` in the existing task brief, separately
from execution mode. New or changed auth/secrets, migration/removal, ownership,
concurrency/settlement/isolation behavior and reproducible behavior bugs require
`critical-tdd` and an actual read of tdd-pi. Use short public-behavior slices:
observe the intended invented-data failure, implement minimally, then run green
and relevant negative/legacy regressions. Explicit test-first requests and
project-required checks govern.

Routine prose/manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`: exact diff inspection, positive/negative cases and approved
focused checks after editing. Preserve existing safety tests. Do not invent red
cycles or prose-substring tests. Missing or ambiguous strategy, including a legacy
brief without one, stops for a coordinator decision before implementation.

Neither strategy waives required fresh independent whole-change implementation
review. Mode selection grants no additional checks or mutation authority.

## Step 2: Resolve the mode

Apply these rules **in order**. The first one that fires decides.

**Rule 1: an explicit user mode request governs within its recorded authority.** If the invocation names a mode,
record it without pretending it was an assessed choice. If its required capabilities or grants are missing, stop; do not silently change mode. There is no flag to parse: match on the mode
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
- high blast radius: a changed safety/compatibility invariant in a shared framework,
  installer, profile resolver or code with several independent consumers. Using an
  existing shared framework alone is insufficient

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

## Record and execute

Cite the actual plan sections that triggered each risk, selected mode, authority and budgets. Read [solo.md](solo.md) or [sdd.md](sdd.md), not both by default. An existing execution grant permits continuation; a mode decision does not grant delegation, worktree deletion or integration. Both modes retain required fresh independent whole-change implementation review under requesting-code-review-pi, with budgets allocated before launch. Small change size is no exemption; only the owner may explicitly waive this gate. Hybrid means isolated independent lanes plus serial contract/integration gates, not overlapping writers or task-by-task silent mode switching.

Before launch read using-git-worktrees-pi. Bind exclusive report paths; record run/session/mission/attempt lineage, source/output hashes, owners, open asks and budgets. Every promise must settle before transition or cleanup; await dependent runs.run and ordered runs.all results. Child async receipts are not completion. A missing tool/output or infrastructure failure preserves evidence and stops the lane for same-protocol recovery, never inline/model/provider fallback.

## Declared-target delivery barrier

After any mutation-producing task, do not admit review, validation, integration, publication, or another dependent stage from a receipt, report, verdict, structured claim, or `outputReference`. The coordinator must independently read and hash every exact declared target from its authorized location, bind each target to its pre-run identity and required post-run change or deterministic postcondition, verify syntax and task semantics against actual bytes, and compare the complete authorized path set and source status before/after for unauthorized additions, modifications, or deletions. Compare the child report with actual target bytes and record divergence.

A missing target, unchanged target when change was required, malformed target, wrong-target write, report-target divergence, preidentity drift, or unauthorized sibling mutation produces blocking delivery evidence and zero dependent launches. Preserve the report, actual hashes, path-set observations, and unknown ownership state. This is a parent gate at the existing workflow seam, not permission to build a generic delivery framework or silently repair the child result.

Helpers [task-brief](scripts/task-brief), [review-package](scripts/review-package) and [sdd-workspace](scripts/sdd-workspace) perform only local extraction/Git reads/file writes when separately permitted. They are not authority gates. Defaults share a plan workspace: coordinator serializes use or passes explicit attempt-qualified OUTFILE paths. Copy/hash authoritative brief/native-state/reports into the approved durable location before temporary cleanup. A nonempty helper output is not acceptance. The numbered [fixtures](fixtures/README.md) and RESULTS are dated source provenance, not new Pi behavior or runtime proof.

## Review disposition and authority

Normalize by consequence to **Critical / Major / Medium / Low**: catastrophic security/data-loss impact; substantial requirement/correctness/safety failure; bounded material defect; or presentation/low-impact improvement, respectively. Preserve original severity and every source finding ID under a canonical defect ID, with location, evidence, consequence rationale, proposed correction and unresolved status. Never mechanically convert Important to Major or Minor to Low, or downgrade severity to fit a threshold. A reduction needs concrete consequence evidence and a recorded parent disposition.

Use one fresh independent review and at most one bounded correction/disposition, not automatic two-cycle rituals. Unresolved Critical/Major findings block; Medium residuals need explicit owner acceptance. Preserve source finding IDs, consequences and old/new hashes. Required tests, authority, evidence and requirement coverage independently block. Corrected bytes are not retroactively independently reviewed; stop at the bound and return unresolved decisions to the parent.

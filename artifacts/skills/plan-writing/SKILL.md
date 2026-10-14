---
name: plan-writing
description: Use when authoring or revising an authorized requirement-covered implementation plan from an approved stream spec
---

# Write a buildable implementation plan

This optional Codex/OpenCode identity remains available. Author plans, not specs or
production code. Read approved brief/spec/decisions, effective instructions and
actual source/interfaces initially and after resume. Existing grants remain valid
within scope; changed targets, destructive actions or material unresolved choices
need owner direction, not another consent request merely for a stage label.

## Spec and folder contract

Use ADR-0003 `docs/specs/NN-slug/`: one effort/synthesis and one spec/plan pair per
stream. Resolve the approved stream spec from parsed `meta.yaml` and confirm the
named file exists. Missing spec redirects to installed own-host spec authoring guidance
(`{skillsDir}/spec-writing-cx/SKILL.md` for Codex,
`{skillsDir}/spec-writing-oc/SKILL.md` for OpenCode),
not a guessed spec from research or an orphan plan. The only plan-only exception
is an explicit owner-declined spec, retaining `spec: null` and `spec_declined:` and
disclosing that spec coverage cannot be checked. Preserve reviewed/user-edited
plans; an existing valid revision grant suffices, otherwise ask revise or supersede.

Save `<stream>-plan.md` beside its spec unless the owner selected another path.
Respect actual repository ignore/tracking policy; do not assume `docs/specs/` or
`.tickets/` is tracked or ignored, change ignore rules, or manufacture Git authority.
Only the coordinator updates this stream's `plan:` filename and `updated:` after
saving the file, preserving other streams/specs/research. `plan: null` remains until
bytes exist. Validate references in both directions with authorized project checks:
every named artifact exists and every stream spec/plan is named. Missing tooling
blocks that check honestly, without installation or a fabricated pass.

## Plan shape and coverage

Start with goal, approved spec/hash (or explicit spec decline), architecture,
tech stack, exact global constraints and source anchors. Map every requirement ID
to at least one task and acceptance check. Split independent streams only where
ownership/interfaces and deliverables are genuinely separable. Keep scaffolding,
configuration and docs with the deliverable needing them, not separate gate tasks.
Task count alone adds neither reviewers nor progress surfaces.

## Slicing

Record the strategy once in the plan header: `Slicing: horizontal (default)` or
`Slicing: vertical tracer (user-selected)`. Horizontal (tasks by responsibility
or component) is the default; an absent value means the default and adds no
approval stop. A vertical tracer (thin end-to-end capabilities) needs the user's
choice; reuse a valid prior choice instead of asking again. Offer the tracer with
one simple same-feature example only when useful (CSV export: horizontal query,
formatter and button tasks plus a composing endpoint, versus one thin
query-to-button path, then denial, then quoting and empty results). A strategy
change, or a chosen slicing found unsafe or infeasible, returns to the owner with
the exact conflict; never swap silently. Keep plan slicing (how tasks divide),
test slicing (red-green-refactor loops inside each task, owned by the TDD skills)
and parallel ownership (exact disjoint paths and accepted inputs) separate.
Neither strategy makes overlapping writers safe. Tasks stay behavior-sized.

## Execution graph

Every plan has `## Execution graph`, rendered from existing task predecessor
fields, not a second task list, runtime admission, status, colors, YAML ledger or
engine input. A one-task or purely serial plan uses one line, such as
`T1 -> J -> C -> R`. Otherwise draw a compact ASCII graph (at most 100 columns,
spaces only, node IDs matching task headings) showing serial sections, candidate
parallel branches, `=>` start and `~>` join edges, named shared-file hazards with
their resolution (one owner, serialization or an authorized seam split), J
(serial integration), C (fresh composed checks) and R (one aggregate review):

```text
legend: => start (accepted bytes needed first)   ~> join (needed at integration)
SERIAL     [H0 contracts] => [H1 query], [H2 csv], [H4 UI]   (candidate parallel)
DEPENDENT  [H1 query], [H2 csv] => [H3 endpoint] ~> [H4 UI]
HAZARD     server/export.ts wanted by H1 and H2: owned by H1 only
SERIAL     [J integrate H0..H4] -> [C fresh checks] -> [R one review]
```

Reuse an accepted graph and revise only changed seams under existing revision
authority.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer. This is file organization; it doesn't forbid horizontal plan slicing and isn't the TDD "horizontal slices" anti-pattern.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Task structure

Each task is an independently testable deliverable with:

- exact create/modify/test paths and current source anchors;
- owned/excluded files, input/predecessor barriers and build order, each
  predecessor labelled `start: <task>` (accepted bytes needed before work) or
  `join: <task>` (needed only at integration or verification);
- consumed and produced interfaces, exact signatures/types/names and invariants;
- testing strategy/rationale, exact public behavior/negative paths and acceptance;
- concrete one-action steps with actual code/content, commands and expected oracle;
- required regressions, migration/rollback, applicable host/resource limits,
  evidence return and durable handoff references.

Show real code/signatures and known commands, not speculative approved decisions.
A behavioral RED must expose the intended missing behavior, not an import/syntax/
fixture/tool error. State exploratory unknowns as blockers rather than plausible
code. Commits are optional steps only under a separate grant, not frequent-commit
rituals. Checkboxes may organize intent; they are not another mutable status owner.

## Testing defaults

Use the installed own-host TDD defaults, `tdd-cx` for Codex or `tdd-oc` for
OpenCode. Explicit test-first requests and required project checks govern.
Critical auth/secrets, ownership, concurrency/isolation/settlement, migration/
removal and reproducible bugs require public-behavior TDD. Other features/behavior
changes default to TDD; exceptions need concrete reason and owner agreement unless
already approved. Use `critical-tdd` where a field exists for short invented-data
intended RED, minimum GREEN and relevant negative/legacy regression slices.
Pure behavior-preserving refactors use green coverage and characterize uncertain
legacy behavior first. Nonbehavioral prose/manifests/mechanical wiring without a
changed safety invariant uses `focused-postcheck`, not invented RED or prose-
substring tests. Omitted strategy applies these defaults and records selection in
the existing brief/handoff. Genuine classification/conflict/exception ambiguity
needs owner direction; omission alone does not block or require a new ledger.

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** : never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code : the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task

## Inline author check and independent review

Check coverage, placeholders, types/names/paths, interface/predecessor consistency,
verification feasibility and plan shape inline. Fix assigned defects without
launching an author “self-review” worker. Every written plan, including a single
task, needs one independent plan review unless explicitly owner-declined for that
subject and retained in the brief. Spec review is a separate accepted subject,
not repeated by plan writing. Only the coordinator admits reviewers, persists
complete returned findings and authorizes one bounded batched correction.

Use own-host installed `{skillsDir}/plan-writing-cx/SKILL.md` or
`{skillsDir}/plan-writing-oc/SKILL.md` for native planning guidance, and own-host
`{skillsDir}/plan-review-cx/SKILL.md` or `{skillsDir}/plan-review-oc/SKILL.md` for
an admitted native review. The retained optional `plan-review` entry and its rubric
are also available when selected. These are optional own-host routes, not a
cross-host requires edge or substitute for a missing explicitly selected route.
No automatic second-model discovery, permanent reviewer-choice config, task-level
review chains, CLI model fallback or repeated consent follows from this skill.

Unresolved Critical/Major or more than two distinct Medium block. Up to two Medium
residuals require explicit owner acceptance, Low receives disposition and missing
authority/requirements/tests independently block. Preserve finding IDs/severities.
The coordinator resolves corrected exact bytes with sufficient evidence against
the cited criterion, labelled parent-verified, not independent re-review. Changed
interface/safety invariant/acceptance or inadequate oracle needs owner direction
or an additional independent-review grant, never automatic successor waves.

## One progress authority and handoff

At plan acceptance the coordinator records one mutable progress authority in the
existing brief: qualified native state that actually owns delegated runs first;
without it, explicit project selection, otherwise existing tracker or one durable
plan-associated record. UI todo alone does not outrank the tracker. Specs/plans
hold intent, metadata indexes artifacts and logs retain observations, not mirrored
status. Seed selected tickets once under one owner only when that tracker is
selected, using exact task/plan pointers and actual tracking policy. There is no
mandatory tk mirror, ledger, ticket commit or automatic checkpoint reconciliation.

Return complete plan/output hashes, coverage, actual validation commands/exits/
logs, decisions, limitations and next authorized action. Read-only leaves return
complete text through native delivery; coordinator persists shared outputs.
Only the coordinator chooses execution mode after plan acceptance and execution
authorization, through own-host `{skillsDir}/plan-execute-cx/SKILL.md` or
`{skillsDir}/plan-execute-oc/SKILL.md` (serial or parallel-ready tasks; ready
means accepted `start` inputs and disjoint owned paths). Disjoint native writers
require checked isolated roots/ownership, settlement and serial integration;
ordered/shared-file tasks remain serial.
Final composed verification and independent aggregate implementation review remain.
Plan delivery grants no worker launches, commits, integration, publication or
cleanup. Local handoff is sufficient when publication is ungranted; then stop.

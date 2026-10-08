---
name: plan-writing-pi
description: "Write authorized, requirement-covered implementation plans"
---

# Plan writing

Read the approved spec and actual source seams before planning. A plan promises one spec, not a substitute spec invented from research. Missing spec blocks coverage claims unless the owner explicitly declined it and the limitation is recorded. An existing reviewed plan requires revision/supersession authority, never blind overwrite.

## Scope and recovery

Act only within the task's current grant. Research, authorship, review, planning, execution, integration, publication and cleanup are separate actions. Existing explicit approval remains valid within its recorded root, outputs and revalidation conditions; do not ask again merely because the stage changed. Changed scope or destructive actions need revalidation. Before work and after compaction, reread the mandatory brief, native mission state, latest decisions and project instructions, then inspect current artifacts. A leaf never delegates, mutates native mission state or writes lessons. Only the coordinator does so when authorized. Use contact_supervisor need_decision/interview_request for a material unresolved choice and wait for the actual reply; steering delivery is not consent. Do not switch model, provider, protocol or isolation on failure.
## Shape (ADR-0003)

Use the project's actual layout/tracking policy; never add a specs ignore rule. One effort has one synthesis; each independent stream has one spec and one plan. Record filenames in parsed meta.yaml only after authorized files exist; before then plan is null. Link actual upstream mission/receipt references rather than ticket seeds; metadata is not a task ledger. Coordinator alone updates this shared metadata. A spec-declined stream records the explicit decision. Verify both directions: every named file exists; every spec/plan is named.

## Write a buildable plan

Start with goal, architecture, tech stack and exact global constraints. Map every requirement to a task and acceptance check. Each task names owned files, source anchors, consumed/produced interfaces and predecessor dependencies. Fold scaffolding and documentation into the smallest independently testable deliverable. Split only when ownership and evidence can be independent. Record a testing strategy and rationale in each existing task brief. Select `critical-tdd` for new or changed auth/secrets, migration/removal, ownership, concurrency/settlement/isolation behavior and reproducible behavior bugs. Plan an actual tdd-pi read and short public-behavior slices: an invented-data test, observed intended failure, minimal implementation, green and relevant negative/legacy regressions. Routine prose/manifests/mechanical wiring without a changed safety invariant use `focused-postcheck`: exact diff inspection, positive/negative cases and approved focused checks after editing. Explicit test-first requests and project-required checks govern. Preserve existing safety tests; never manufacture red cycles per file/task or prose-substring tests. Missing or ambiguous strategy, including legacy briefs without one, blocks implementation for a coordinator decision. Include a scoped commit only if authorized. Include real code/commands/expected outcomes where known, not “handle edge cases”, “similar to task N” or undefined signatures. State remaining decisions rather than pretending invented code is approved.

## Behavior-sized tasks and briefs

Each implementation task delivers exactly one observable outcome at one interface seam. Record its stable task key, one owner, changed or exercised seam, acceptance point, exact owned paths, predecessor inputs and identities, bounded initial context, selected testing strategy and authorized initial checks, stop and non-prescriptive return conditions, and durable handoff consumed by successors.

A task is not one schema field, function, file, unit test, manifest bump, or documentation edit. Group those details when they jointly deliver the same behavior and acceptance point. Split only when ownership, predecessor inputs, and acceptance evidence are independently useful. Merge tasks that share a seam, unresolved decision, paths, or repeated partial-state exchange, or whose coordination cost exceeds useful parallelism. A parallel-ready set contains only predecessor-complete tasks with disjoint ownership and no unresolved shared design choice. Reject both microtask fragmentation and a task that hides independently ownable outcomes.

Initial context is limited to the approved spec section, predecessor interfaces, relevant source anchors, project instructions, acceptance point, and authorized commands. It is not an accumulated transcript or a static prediction of every future read. Keep plan-generated child briefs simple: outcome and acceptance point; exact source/base and subject identity; owner and owned paths or read-only subject; predecessor interfaces; bounded initial context; selected strategy; authorized commands and budgets; output binding; stop and non-prescriptive return conditions; and upstream mission/run/receipt references when applicable. Reference shared approved evidence rather than duplicating it.

Do not introduce packet loaders, global read/skill binding tables, per-child `requiredSkills` arrays, read-key graphs, rubric hash tables, or perfect-context prediction. Children still read current mandatory briefs, native state, project instructions, and selected guidance required by their effective role.

When a task stops, it returns only observations, evidence attempted, unresolved questions or shortfalls, newly discovered requirements, confidence, and consequences or risk if unresolved. It must not suggest, request, name, initiate, or semantically prefer paths, commands, context, budget, scope, specialist, model, provider, or escalation/remedy/package. The parent owns any later decision.

Include negative paths, migration/rollback, authority boundaries, resource budgets and a durable handoff. Retain required fresh independent whole-change implementation review even for small deliveries; only the owner may explicitly waive it. Preserve Pi's ban on task-by-task two-cycle review rituals. Task count does not justify fanout. Single-stream risk assessment belongs to plan-execute-pi; disjoint lanes may use plan-execute-parallel-pi after explicit execution/delegation authority. Bind stable workflow keys and dependency-ready stages in native mission state, without ticket mirroring or a second ledger. A plan saved locally is not a claim of remote availability.

## Review and handoff

Self-check coverage, placeholders and type/name consistency. Request authorized fresh independent plan-review-pi through the coordinator; do not silently substitute inline review or another model. A valid pre-existing planning grant continues; plan acceptance does not authorize execution. Report exact hashes, open decisions and the next authorized stage.

## Review disposition and authority

Normalize by consequence to **Critical / Major / Medium / Low**: catastrophic security/data-loss impact; substantial requirement/correctness/safety failure; bounded material defect; or presentation/low-impact improvement, respectively. Preserve original severity and every source finding ID under a canonical defect ID, with location, evidence, consequence rationale, proposed correction and unresolved status. Never mechanically convert Important to Major or Minor to Low, or downgrade severity to fit a threshold. A reduction needs concrete consequence evidence and a recorded parent disposition.

Use one fresh independent review and at most one bounded correction/disposition, not automatic two-cycle rituals. Unresolved Critical/Major findings block; Medium residuals need explicit owner acceptance. Preserve source finding IDs, consequences and old/new hashes. Required tests, authority, evidence and requirement coverage independently block. Corrected bytes are not retroactively independently reviewed; stop at the bound and return unresolved decisions to the parent.

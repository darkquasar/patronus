---
name: plan-writing-cx
description: "Codex only: write a requirement-covered implementation plan from an approved stream spec, before execution."
---

# Plan writing for Codex

Use the owner's selected destination and established project layout throughout.
Any `docs/specs/NN-slug/` path below is a default example, not a location
requirement; preserve the selected effort's metadata and stream relationships.

Read the approved spec, task brief, decisions, project instructions and actual source seams first. After context loss, reread inputs and the latest saved outputs, verify their hashes and reconcile completed work before retrying. Plan writing authors plans, not replacement specs or production code.

## Stage grant and spec gate

Research, authorship, review, planning, execution, integration, publication and cleanup have separate grants. Continue within existing approval's root, files and applicable resource limits; ask the lead or owner for material unresolved choices or changed targets and wait for the reply. Skill completion is not permission to execute, commit or publish.

Find this stream's spec through parsed `meta.yaml` and read the actual file. No spec named or a missing referenced file blocks coverage claims: route to `{skillsDir}/spec-writing-cx/SKILL.md`. Do not derive a spec from research or reverse-engineer it from an orphan plan. An explicit owner-declined spec is the only plan-only exception; preserve `spec: null` and `spec_declined:` and disclose that coverage against a spec cannot be checked. If a plan already exists, obtain revision/supersession authority before editing reviewed or user-modified bytes.

## Folder layout

Use `docs/specs/NN-slug/`, one research effort with one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. Multiple independent changes share the research but get separate stream/spec/plan pairs. A stream's plan is `docs/specs/NN-slug/<stream>-plan.md`, co-located with `<stream>-spec.md`. Follow project tracking policy without editing `.gitignore` or mirroring work into another ledger.

Only the lead writes shared outputs and metadata. A read-only Codex worker returns complete draft text or findings, source anchors and hashes to the lead; it never writes shared files, changes workflow state or delegates. Check exposed native tools, read-only scope, authentication and complete delivery before coordinator-admitted use. Missing capability blocks the lane without provider/model/host substitution.

After saving the plan, the lead updates only this stream's `plan:` and `updated:`. `plan: null` stays until the file exists. Preserve `research:`, other streams, `spec:` and recorded work references. Name the file, never `plan: true`. Validate both directions: every named research/spec/plan file exists and every `*-spec.md` / `*-plan.md` in the folder is named. Use available project tools to inspect parsed YAML values and actual
files, recording what was checked. No specific language, compiler, dependency
cache or bundled validator is required. Missing project-required checks remain
visible; do not install tools or claim an unrun check passed.

## Solution ownership and semantic traceability

Consume the actual accepted spec and current source. Choose the technical
solution within inherited constraints; distinguish provenance-bearing owner or
consumer restrictions from planner choices and explain consequential tradeoffs
near their rationale. An approved legacy internal design remains consumable as a
constraint; do not infer missing behavior from its implementation proposal or
require a cosmetic migration. Do not treat delegated routine choices as missing
requirements or invent a mandatory count of alternatives. A changed required
outcome needs an explicit spec revision. Required public formats/signatures stay
exact even when private mechanisms are open.

Link every requirement to contributing tasks and completion checks; inspect the
meaning, not just ID presence. Preserve obligation, permission, prohibition,
actor/scope, triggers, conditions, bounds and exceptions, retaining stable IDs
through harmless edits. Coverage expresses intended contribution, not verified
satisfaction. Each task has an independently checkable outcome, owned surfaces
and relevant consumed/produced interfaces or deliverables.

Explain each dependency using consumed deliverables: `start` inputs must be
accepted before authoring; `join` inputs are needed only at integration or
verification. Runtime behavior order alone is not build order. Parallel readiness
needs accepted inputs, disjoint ownership and checked isolation/settlement, not
merely separate task names. Graph views project these same relationships without
becoming a second mutable status authority; preserve serial J/C/R barriers.

Preserve and show the spec's resulting vocabulary, boundary and invariant
clarifications, and identify new clarification introduced by planning. Do not
merge concepts merely because they share a word in different contexts: finalized
payment settlement and terminated worker/process settlement remain distinct.
For a small prose correction, preserve why no tactical modeling is warranted.

Resolve routine choices from source and record compatible low-impact reversible
observable defaults as agent choices. Reuse valid decisions; consequential
unresolved goals, scope, external contracts, security, preservation, irreversible
commitments or material cost need evidence, recommendation, consequences, a
concrete distinguishing example and an actual owner answer. Expose unavailable
required inputs and affected claims rather than inventing approval.

Surface new or materially expanded verification machinery or hard gates not
settled in the spec as consequential owner choices: protected purpose/value,
evidence, scope/useful granularity, time/cost/maintenance burden, exact blocked
stage/condition and simpler alternative (including informational comparison).
Recommend, show a concrete example, ask one question and wait before adoption;
keep the proposal unresolved while continuing unrelated authorized work. Never
silently turn a proposed evaluator, run matrix or subjective threshold into a
requirement. Reuse approved policy and required project checks without repeated
permission; ordinary relevant test choices remain agent-owned.

## Write a buildable plan

Start with the goal, source spec/hash, architecture, tech stack and exact global constraints. Map every requirement ID to a task and acceptance check. Each task names owned files/source anchors, consumed and produced interfaces, predecessor dependencies labelled `start: <task>` (accepted bytes needed before work) or `join: <task>` (needed only at integration or verification) and an independently testable deliverable. Fold scaffolding and docs into the deliverable that needs them. Split only where ownership and evidence can be independent.

Use the installed tdd-cx defaults and record strategy/rationale in the existing
brief/handoff. Explicit test-first requests and required project checks govern.
Critical auth/secrets, ownership, concurrency/isolation/settlement, migration/
removal and reproducible behavioral bugs require public-behavior TDD. Other
features/behavior changes default to TDD; a different strategy needs a concrete
reason and owner agreement unless already approved. Use `critical-tdd` for
short test-first slices: invented-data intended RED, minimum GREEN and relevant
negative/legacy regressions. Pure behavior-preserving refactors work from green
coverage and characterize uncertain legacy behavior first. Nonbehavioral prose/
manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`, not manufactured reds or prose-substring tests. Omission
alone applies these defaults and records selection, without another approval or
ledger; genuine classification, conflict or exception ambiguity needs owner
direction. Keep existing safety tests. Tests do not admit successor reviewers.

Include real code/signatures and known commands/results, never vague "handle edge cases" steps or references to undefined interfaces. Record consequential unresolved decisions as affected-outcome blockers; own delegated routine design instead of inventing owner approval. Commits are optional steps only under a separate Git grant.

Include negative paths, migration/rollback, ownership, applicable host/resource limits and a durable handoff. Retain required fresh independent whole-change implementation review even for small deliveries; only the owner may explicitly waive it. Earlier boundary Codex review requires a named changed safety/compatibility invariant or irreversible boundary with its own acceptance point and consequence. Using an existing shared framework alone is insufficient. A saved local plan is not evidence that another checkout can read it. The lead binds source/base, output paths, authority and dependencies in the existing project work record; there is no mandatory foreign-host mission or ticket API here.

## Slicing

Record the strategy once in the plan header: `Slicing: horizontal (default)` or `Slicing: vertical tracer (user-selected)`. Horizontal (tasks by responsibility or component) is the default; an absent value means the default and adds no approval stop. A vertical tracer (thin end-to-end capabilities) needs the user's choice; reuse a valid prior choice instead of asking again. Offer the tracer with one simple same-feature example only when useful (CSV export: horizontal query, formatter and button tasks plus a composing endpoint, versus one thin query-to-button path, then denial, then quoting and empty results). A strategy change, or a chosen slicing found unsafe or infeasible, returns to the owner with the exact conflict; never swap silently. Keep plan slicing (how tasks divide), test slicing (red-green-refactor loops inside each task, owned by tdd-cx) and parallel ownership (exact disjoint paths and accepted inputs) separate. Neither strategy makes overlapping writers safe. Tasks stay behavior-sized.

## Execution graph

Every plan has `## Execution graph`, rendered from the existing task predecessor fields: not a second task list, runtime admission, status, colors, YAML ledger or engine input. A one-task or purely serial plan uses one line, such as `T1 -> J -> C -> R`. Otherwise draw a compact ASCII graph (at most 100 columns, spaces only, node IDs matching task headings) showing serial sections, candidate parallel branches, `=>` start and `~>` join edges, named shared-file hazards with their resolution (one owner, serialization or an authorized seam split), J (serial integration), C (fresh composed checks) and R (one aggregate review):

```text
legend: => start (accepted bytes needed first)   ~> join (needed at integration)
SERIAL     [H0 contracts] => [H1 query], [H2 csv], [H4 UI]   (candidate parallel)
DEPENDENT  [H1 query], [H2 csv] => [H3 endpoint] ~> [H4 UI]
HAZARD     server/export.ts wanted by H1 and H2: owned by H1 only
SERIAL     [J integrate H0..H4] -> [C fresh checks] -> [R one review]
```

Reuse an accepted graph and revise only changed seams under existing revision authority.

## Progress authority

At plan acceptance the coordinator records the sole mutable progress authority
once in the existing plan/brief. Native state that actually owns delegated runs
remains authoritative, even when a project tracker is selected. Otherwise use
the explicit project/owner selection, then the existing tracker, or one existing
durable plan-associated record. A UI todo list alone does not outrank the tracker.
Specs/plans hold intent, metadata indexes artifacts and logs retain observations;
do not mirror progress into checklists, native todos, tk or another ledger.
Respect actual ignore/tracking policy; seed selected tickets once under one owner,
not automatically. Recovery reads this authority and retained results without
replaying settled work.

## Review and handoff

Check requirement coverage, placeholders and name/type consistency inline;
do not dispatch an author self-check. Every written plan, including a single-task
plan, needs one independent plan review unless the owner explicitly declines
that subject and the decision is retained in the existing brief. Task count
neither skips the review nor adds another reviewer. Under the review grant,
request one fresh independent `{skillsDir}/plan-review-cx/SKILL.md` with the exact plan AND spec, source evidence, checks and hashes. Inline author checks cannot replace required independence. No automatic second
model, discovery loop or repeated consent for already valid grants. Read-only reviewers return complete findings to the lead; the lead writes the report and dispositions them.

Review acceptance does not authorize implementation. Once the gate is settled and execution is granted, the coordinator selects the
execution mode, not the reviewer. Use `{skillsDir}/plan-execute-cx/SKILL.md` for serial and parallel-ready tasks: ordered/shared-file tasks stay serial, and disjoint writers whose `start` inputs are accepted use its parallel-ready section with native subagents and checked allocated worktrees. The plan's `start`/`join` labels separate authoring dependencies from integration barriers; read-only scouts may prepare later waves while admitted writers work. A plan without a usable split returns to plan writing for a revision under existing revision authority, not ad hoc decomposition at execution time. No CLI supervisor or extra tracker is required; native delegation alone does not prove isolation. Return saved file hashes, actual validation, decisions, residuals and the next authorized action.

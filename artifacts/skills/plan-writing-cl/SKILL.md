---
name: plan-writing-cl
description: Use when you have a spec or requirements for a multi-step task, before touching code
---

# Plan Writing

Use the owner's selected destination and established project layout throughout.
Any `docs/specs/NN-slug/` path below is a default example, not a location
requirement; preserve the selected effort's metadata and stream relationships.

Read the approved spec, task brief, decisions, project instructions and actual
source seams first. After context loss, reread inputs and the latest saved
outputs, verify their hashes and reconcile completed work before retrying. Plan
writing authors plans, not replacement specs or production code: the plan fixes
what each task owns, produces and consumes, and the implementer writes the code
under tdd-cl.

## Stage grant and spec gate

Research, authorship, review, planning, execution, integration, publication and
cleanup have separate grants. Continue within existing approval's root, files and
applicable resource limits; ask the owner for material unresolved choices or
changed targets and wait for the reply. Skill completion is not permission to
execute, commit or publish.

plan-writing-cl is **plan-only**. It does not author specs. Before planning
anything, confirm a spec exists for this stream: a plan is a promise to build a
spec, and there is nothing to promise without one, except an explicitly
owner-declined spec retained in the brief.

Check for a `<stream>-spec.md` that is **named in the folder's `meta.yaml`**
(`streams[].spec:`) **and resolves on disk.** A folder can have a `meta.yaml`
and a `research.md` but no spec, and that still redirects.

- **Spec explicitly declined by the owner**: retain the decision and plan against
  the accepted requirements. Preserve `spec: null` and `spec_declined:` and
  disclose that coverage against a spec cannot be checked.
- **No spec named, or the named file is absent (without explicit decline)**:
  **STOP.** Suggest spec-writing-cl for this stream. Do not synthesize a spec,
  infer one from `research.md` or reverse-engineer one from an orphan plan.
- **A `<stream>-plan.md` already exists for this stream**: do not silently
  overwrite reviewed or edited work. Report it and obtain revision or
  supersession authority first. With a spec present, ask revise-vs-supersede;
  without one, apply the two spec cases above.
- **Spec present (no plan yet)**: plan normally (the rest of this skill).

Spec ownership stays with spec-writing-cl. plan-writing-cl sets `streams[].plan:`
only, never `streams[].spec:`.

## Folder layout

Save plans to `docs/specs/NN-slug/<stream>-plan.md`, alongside the spec they
implement, one plan per spec: the plan takes the same `<stream>` prefix as the
`<stream>-spec.md` it is planned from. After writing it, fill in **your stream's**
`plan:` in the folder's `meta.yaml`, name the file (`plan: <stream>-plan.md`),
never assert `plan: true`, and bump `updated:`. If the folder has no `meta.yaml`
yet, create one (see the spec-writing-cl skill's "Folder layout and metadata"
for the shape). Preserve `research:`, other streams, `spec:` and recorded work
references. Validate both directions: every named research/spec/plan file exists
and every `*-spec.md` / `*-plan.md` in the folder is named. Respect actual
repository tracking/ignore policy; authoring grants no commit, ignore-rule change
or publication. User preferences for plan location override this default.

## Scope Check

If independently specifiable subsystems need separate plans, surface a stream/spec
split under revision authority first. One stream retains one spec and one plan;
do not silently produce multiple plans for the same spec.

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

## File Structure

Before defining tasks, map out which files will be created or modified and what
each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file
  should have one clear responsibility.
- Prefer smaller, focused files over large ones that do too much: an
  implementer reasons best about code it can hold in context at once.
- Files that change together should live together. Split by responsibility,
  not by technical layer. This is about file organization: it does not forbid
  horizontal plan slicing (see Slicing) and is not the TDD "horizontal slices"
  anti-pattern.
- In existing codebases, follow established patterns. If the codebase uses
  large files, do not unilaterally restructure, but if a file you are modifying
  has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task produces self-contained
changes that make sense independently.

## Write a buildable plan

Start with the goal, source spec and its hash (or the explicit owner-declined
exception), chosen architecture with its consequential rationale, tech stack, the
`Slicing:` value and exact global constraints copied verbatim from the spec:
version floors, dependency limits, naming and copy rules, platform requirements.
Every task's requirements implicitly include the global constraints.

Map every requirement ID to a task and acceptance check. Each task names its
owned files with exact paths and source anchors, the interfaces it consumes from
predecessors and produces for successors (exact names, parameter and return
types), each predecessor labelled `start: Task M` (accepted bytes needed before
work) or `join: Task M` (needed only at integration or verification), and an
independently testable deliverable. A task's implementer sees only their own
task; the consumed/produced block is how they learn the names and types
neighboring tasks rely on. Fold setup, configuration, scaffolding and
documentation into the deliverable that needs them. Split only where interfaces,
ownership and evidence can be checked independently, not to manufacture reviewer
gates. A task is the smallest independently verifiable behavior-sized deliverable.
Head each task `### Task N: <name>`; plan-execute-cl's `task-brief` script selects
a task's brief by that heading. Execution graph node IDs reference the task number
(`T1`) or a short label stated in the heading, such as `### Task 1 (H1): query`.

Use the installed tdd-cl defaults and record strategy and rationale in the plan.
Explicit test-first requests and required project checks govern. Critical
auth/secrets, ownership, concurrency/isolation/settlement, migration/removal and
reproducible behavioral bugs require public-behavior TDD. Other features and
behavior changes default to TDD; a different strategy needs a concrete reason
and owner agreement unless already approved. Use `critical-tdd` for short
test-first slices: invented-data intended RED, minimum GREEN and relevant
negative/legacy regressions. Pure behavior-preserving refactors work from green
coverage and characterize uncertain legacy behavior first. Nonbehavioral
prose/manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`, not manufactured reds or prose-substring tests. Omission
alone applies these defaults and records the selection, without another approval
or ledger; genuine classification, conflict or exception ambiguity needs owner
direction. Keep existing safety tests. Tests do not admit successor reviewers.

Name real signatures, paths, commands and expected results, never vague "handle
edge cases" steps, "similar to Task N" or references to interfaces no task
defines. Describe each test by the behavior it checks and the invented data it
uses; the implementer writes the test and the implementation body under tdd-cl.
Include code only where it is already known and binding, such as a required
public signature, format or command. Record consequential unresolved decisions
as affected-outcome blockers; own delegated routine design instead of inventing
owner approval. Commits are optional steps only under a separate Git grant.

Include negative paths, migration/rollback, ownership, applicable host/resource
limits and a durable handoff. Retain the required fresh independent whole-change
implementation review even for small deliveries; only the owner may explicitly
waive it. Earlier boundary review requires a named changed safety/compatibility
invariant or irreversible boundary with its own acceptance point and
consequence. Using an existing shared framework alone is insufficient. A saved
local plan is not evidence that another checkout can read it. The coordinator
binds source/base, output paths, authority and dependencies in the existing
project work record; there is no mandatory foreign-host mission or ticket API.

## Bite-Sized Task Granularity

Each step inside a task is one action of two to five minutes, named by what the
implementer does and checks, not by code the plan pre-writes:

- "Write the failing test for <behavior> with <invented data>" is a step
- "Run it and confirm the intended public-behavior failure" is a step
- "Implement the minimal code to make it pass" is a step
- "Run the focused checks and confirm they pass" is a step
- "Capture commands, exits and changed-byte evidence" is a step; commit only if
  separately authorized

Each step states the exact command where one exists and the expected outcome.

## Slicing

Record the strategy once in the plan header: `Slicing: horizontal (default)` or
`Slicing: vertical tracer (user-selected)`. Horizontal (tasks by responsibility
or component) is the default; an absent value means the default and adds no
approval stop. Use a vertical tracer (thin end-to-end capabilities) only when the
user chooses it, and reuse a valid prior choice instead of asking again. Offer
the tracer alternative with one simple same-feature example only when useful,
for example CSV export: horizontal is query, formatter and button tasks plus an
endpoint that composes them; tracer is one thin query-to-button path, then
denial, then quoting and empty results. A strategy change, or a chosen slicing
that proves unsafe or infeasible, returns to the owner with the exact conflict.
Never swap silently.

Keep three meanings separate: **plan slicing** (how tasks divide), **test
slicing** (red-green-refactor loops inside each task, owned by tdd-cl and
unchanged) and **parallel ownership** (exact disjoint paths and accepted inputs).
Neither slicing strategy makes overlapping writers safe. Tasks stay behavior-sized.

## Execution graph

Every plan has a `## Execution graph` section rendered from the tasks' existing
`start`/`join` labels. It is not a second task list, runtime admission, status,
colors, a YAML ledger or input to any engine. A one-task or purely serial plan
uses one line, such as `T1 -> J -> C -> R`. Otherwise draw a compact ASCII graph
(at most 100 columns, spaces only) with node IDs matching task headings: serial
sections, candidate parallel branches, `=>` start edges, `~>` join edges, each
named shared-file hazard with its resolution (one owner, serialization or an
authorized seam split), then J (serial integration), C (fresh composed checks)
and R (one aggregate review):

```text
legend: => start (accepted bytes needed first)   ~> join (needed at integration)
SERIAL     [H0 contracts] => [H1 query], [H2 csv], [H4 UI]   (candidate parallel)
DEPENDENT  [H1 query], [H2 csv] => [H3 endpoint] ~> [H4 UI]
HAZARD     server/export.ts wanted by H1 and H2: owned by H1 only
SERIAL     [J integrate H0..H4] -> [C fresh checks] -> [R one review]
```

Reuse an accepted graph and revise only changed seams under existing revision
authority.

## Inline author check

Check spec/requirements coverage, placeholders and cross-task type/interface
consistency yourself. Fix gaps within the authoring grant. This is not a dispatched
review and does not replace independent plan review.

## One progress authority

At plan acceptance the coordinator records the sole mutable progress authority
once in the existing brief/plan. Native mission/workflow state that actually owns
delegated runs stays authoritative; a project tracker receives references only.
Without run-owning state use explicit project/owner selection, otherwise the
existing project tracker or, if absent, one durable plan-associated record.
Native UI todos alone do not outrank a project tracker.

No checklist-to-todo expansion, tk mirror or competing ledger. Metadata indexes
documents; evidence logs record observations, not mirrored progress. If tk is
selected, seed once under its authorized owner after acceptance, respecting actual
ignore/tracking policy. Never commit ignored .tickets/ or require a tracker install.
Existing ticket pointers name the actual plan file and exact task heading; follow
them, do not reconstruct missing requirements from a stale title.

## Review and execution handoff

One independent plan review is required for every written plan, including one
task, unless explicitly owner-declined and retained in the brief. The coordinator
uses plan-review-cl once for this subject. No dispatched Self-Review, automatic
second-model discovery or consent cycle. A spec, when present, keeps its separate
spec-review-cl subject. Review exceptions do not waive required tests or authority.

Consume full findings and the one bounded correction/disposition through
receiving-code-review-cl. Corrected bytes are parent-verified, not independently
re-reviewed; material changes or insufficient proof require an owner decision.
Use the existing review/evidence routes, not another ledger or automatic wave.

The coordinator selects execution mode from concrete ownership/dependencies in
plan-execute-cl: its parallel-ready section admits disjoint isolated owners whose
`start` inputs are accepted; shared/ordered work stays serial, and the solo/SDD
choice never creates automatic task reviews. A plan without a usable split
returns here for a revision under existing revision authority.
Pass relevant requirements, source/interface anchors, paths, settled predecessors,
strategy, acceptance and evidence references, not a growing transcript.
Verify real tool/auth/read-only/isolation/settlement requirements before launch.
Missing tools stop that lane without provider fallback.

On resume read effective instructions, accepted brief, latest handoff and exact
source/results; use the one authority and never replay settled work. Existing valid
grants remain valid within scope. Handoff does not self-admit implementation,
publication or cleanup. Ask only for a real changed scope or unresolved choice.

---
name: spec-writing-pi
description: "Author behavior-first requirements and linked acceptance from evidence, with a clear research/spec/plan folder layout."
---

# Spec writing

Read project instructions, the brief, recorded decisions, actual source and all
supplied research/findings before drafting. Reuse settled evidence; research is
an input, not permission to repeat it. Separate unknown-domain research from
splitting known work into independent streams. After context loss, reread the
brief, current decisions, selected progress authority and latest saved outputs;
check actual bytes before retrying. Do not restart completed work from a title.

Research, spec authorship, review, planning, implementation, integration,
publication and cleanup have separate grants. Existing approval remains valid
within its recorded scope. Changed targets, destructive actions or consequential
unresolved choices require the lead/owner's decision and an actual answer. Skill
completion grants no next stage, commit or publication. Steering is not consent.

Only the authorized lead/coordinator writes shared outputs and `meta.yaml`.
Read-only workers return complete draft text/findings and evidence; a filename or
completion notification alone is not delivery. Delegated authors act only within
their assigned write grant, never changing shared workflow state or admitting
successors. A delegated leaf returns unresolved interpretations and consequences;
the coordinator owns recommendations and owner-facing questions/waits. Honor the
native role's return/stop conditions. Do not delegate from a leaf or silently
change model, provider, protocol or isolation when capability is missing.

## Native integration

Use the available Pi supervisor/return route and qualified native runtime only
within its current grant. Delegated leaves return complete artifacts, evidence
and unresolved interpretations without prescribing remedies or successor stages.
The coordinator owns owner-facing recommendations, questions and actual waits.
Missing runtime/tool/authentication blocks that lane without substitution.

## Optional own-host offers

The coordinator may offer `{skillsDir}/research-team-pi/SKILL.md` for
unknown-domain research under its own grant. On owner request or explicit
acceptance, offer `{skillsDir}/grilling-pi/SKILL.md` for a bounded stress-test.
Verify the selected own-host companion is installed/readable. Missing optional
companions are reported, never silently installed or substituted. Offers grant
no stage and do not automatically start research or an additional interview.

## DDD consultation before drafting

Read `{skillsDir}/ddd-distilled-pi/SKILL.md` before drafting. Consultation is mandatory;
apply particular guidance only where domain complexity and risk justify it.
Briefly record applicability or why no additional modeling is warranted. Show
resulting vocabulary, boundary and invariant clarifications in the spec so planning
can preserve them. The same word in different contexts need not mean the same
concept; a generic internal model is not an external contract. If the required
guidance is unavailable, disclose the gap and affected claims; never claim an
unread consultation, silently install it or substitute another host's workflow.

## Behavior-first document contract

Specify outcomes at the declared system/API/library/subsystem boundary, scope and
non-goals. Preserve required public signatures, formats, compatibility, security,
reliability, performance and deployment constraints, with their owner/source or
consumer and why they bind. Advisory research and current code structure alone
do not fix a solution. An explicitly approved legacy design remains a constraint;
missing behavior cannot be reconstructed from its implementation proposal.

Leave replaceable internal architecture, algorithms, private signatures, file
layout, test implementation and task/build order to planning. Inspect source for
facts, constraints and feasibility; do not convert its present structure into an
automatic compatibility restriction. Compare real outcome choices only when
needed; require no count of alternatives or design approval for delegated details.

Use clear, consistent headings with intent/scope, requirements and acceptance
criteria readily identifiable. Each requirement has a stable stream-local ID,
a clear statement and linked acceptance criteria/examples. Preserve IDs through
reordering and nonsemantic edits; identify changed obligations explicitly.
Preserve actor/scope, trigger, condition, bound, exception, obligation, prohibition
and permission. Distinguish illustrative examples from normative rules: finite
examples neither narrow a universal guarantee nor create unrequested universals.
An engineering refactor can require a verifiable structural outcome plus behavior
preservation; do not invent product changes or mandate tactical DDD patterns.

Supporting content covers domain terms when needed, constraints/compatibility,
assumptions, risks, evidence, unresolved owner decisions and delegated planning
choices. Combine or omit supporting sections only with a reason when genuinely
inapplicable; keep the requirement identity/statement/acceptance core strict.
No exact heading syntax, empty boilerplate or additional primary document is
required. State failure, rollback/preservation and evidence limits at the required
boundary without choosing private recovery mechanisms. Contradictory sources,
missing required inputs and consequential unknowns stay visible with affected
requirements and consequences; do not claim unsupported acceptance.

## Decisions and focused questioning

Resolve questions answered by source or valid prior decisions yourself. Own routine
technical choices within scope. Distinguish facts, assumptions, owner-fixed
constraints and agent choices; label low-impact, reversible observable defaults
as agent-selected. Escalate unresolved consequential goals, scope, external
contracts, security, preservation, irreversible commitments or material costs
with evidence and a recommendation. Explain consequences and show a concrete
distinguishing example when interpretations compete; wait for the owner's answer.
Necessary clarification needs no separate interview grant.

An additional stress-test interview runs only on owner request or explicit
acceptance, within a declared finite question/time bound, one consequential
question at a time, with recommendation, consequences and an actual wait.
Stop when decisions are settled or the bound is reached. Manufacture no minimum
question count, extend no bound automatically and never rerun a valid approval.
Completion grants no next stage.

Before adopting new or materially expanded blocking verification criteria or
substantial test/evaluation machinery, surface a consequential owner choice.
Explain the protected goal and value, evidence, scope and useful granularity,
time/cost and maintenance burden, exact blocked stage/condition and a simpler
alternative (including informational comparison). Recommend an option, show a
concrete distinguishing example, ask one question and wait. Keep an undecided
proposal unresolved; continue unrelated authorized development. Reuse settled
policy and existing required checks without repeated permission. Ordinary relevant
test selection remains agent-owned.

## Folder layout and metadata

Use the owner's selected destination and the target project's established layout.
When neither specifies another structure, use `docs/specs/NN-<effort>/`, with the next
available sequential number and a short effort name. Inspect existing folders
before allocating a new one. Reuse an existing effort instead of creating a
second folder for related work.

One folder is one research effort. It holds one shared research synthesis when
research exists, one `meta.yaml`, and one spec/plan pair for each independently
specifiable stream. Separate unrelated investigations into separate folders.
A second independent plan needs its own stream/spec, not a second plan for the
same spec. The layout is:

```text
docs/specs/NN-<effort>/
|-- meta.yaml
|-- <effort>-research.md        shared synthesis, when research exists
|-- <stream>-spec.md            this stream's requirements
|-- <stream>-plan.md            authored later under a planning grant
|-- <other-stream>-spec.md      another independent stream, if needed
`-- <other-stream>-plan.md
```

The stream name connects its spec, plan and any optional work references. Before
writing, resolve the actual paths from existing metadata and project instructions;
do not infer document content or approval from names. User-selected destinations
override the example path without changing the spec/plan responsibility boundary.

The authorized lead/coordinator owns shared metadata. Create it for a new effort
only within the authorship grant; preserve unrelated streams and project fields.
For example, after the research and first spec files have actually been written:

```yaml
slug: NN-<effort>
intent: "One line describing the research effort"
created: <actual YYYY-MM-DD>
updated: <actual YYYY-MM-DD>
research: <effort>-research.md
streams:
  - slug: <stream>
    intent: "One independently specifiable change"
    spec: <stream>-spec.md
    plan: null
    epic: null
  - slug: <other-stream>
    intent: "Another change, if needed"
    spec: null
    plan: null
    epic: null
```

Starting without research is allowed: keep `research: null` or omit it until the
synthesis exists. Here `<effort>` is the unnumbered effort name; `NN-<effort>`
is the numbered folder and metadata slug. Name only files that exist; unfinished
research/spec/plan fields stay null or absent. If an authorized new stream is not listed, append its
entry with its slug/intent and null unfinished document fields, preserving
all existing streams. After saving the authorized spec, update that stream's
`spec:` and the effort's `updated:`. Planning owns `plan:`. Preserve
existing work references; `epic:` is optional and gets a real tracker ID only when
that tracker is selected and seeding is separately authorized. Metadata indexes
artifacts, not task status, semantic approval or stage authority.

A spec is the default. If the owner explicitly declines it, retain the decision
in the brief and record `spec: null` with `spec_declined:` (decision and date).
Do not fabricate a spec or claim spec review for a declined document.

Check both directions: every research/spec/plan filename named by metadata exists,
and every stream spec/plan in the effort is named. Inspect parsed YAML values;
comments are not references. Use available project tools and record what was
actually checked. No particular language, compiler, dependency cache or bundled
validator is required. Respect the target project's tracking/ignore policy; do
not assume these documents are gitignored, change `.gitignore`, force-add files,
commit or publish under an authorship grant.

## Author and check

Write the authorized `<stream>-spec.md`, using the strict requirement/acceptance
core and proportionate supporting content above. Check placeholders,
contradictions, missing requirements and uncertain claims inline; do not launch
an author self-check worker. Distinguish checks performed from unverified claims.
A compact ASCII boundary/behavior diagram can clarify a non-trivial contract;
it does not require private architecture. Use labeled `+---+` boxes, annotated
`=>` sync or `~>` event edges, spaces only and at most 100 columns.

## Review and handoff

When a spec exists, require one independent spec review unless the owner
explicitly declines that review subject in the existing brief. An owner-declined
spec is a separate plan-only exception. Under the review grant, the coordinator
uses `{skillsDir}/spec-review-pi/SKILL.md` with exact spec, brief, evidence/source
paths and hashes. Include `{skillDir}/spec-document-reviewer-prompt.md` as a
supplement to that same subject, not another review cycle. Preserve skipped
lenses, complete consequence-based findings and the review skill's existing
one-batch correction/disposition rules. Corrected bytes are parent-verified,
not independently re-reviewed. Material scope/interface/invariant changes or
insufficient proof return to the owner; no automatic second model or review wave.

Read returned findings completely and check consequential source claims. Inline
author checks cannot replace required independence. After accepted disposition,
`{skillsDir}/plan-writing-pi/SKILL.md` is the planning destination and must consume
the actual written spec. An existing planning grant remains valid; completion
alone does not start planning or coding. Return saved paths/hashes, actual checks,
assumptions/defaults, decisions, limits and the next authorized action.

At plan acceptance the coordinator records one mutable progress authority in the
existing brief/plan: native state actually owning runs first; otherwise explicit
project selection, the existing tracker or one durable plan-associated record.
UI todos alone do not outrank it. Metadata indexes artifacts and evidence records
observations; neither is a competing task ledger. No checklist-to-todo/tk mirror,
automatic seeding or mandatory root numerical form. Commit, publication and
cleanup retain their separate grants.

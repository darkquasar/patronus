---
name: plan-review-cl
description: Use at the end of planning for one independent requirements-coverage, build-order and engineering/design/DevEx/strategy review before implementation
---

# Plan Review

Review the written plan before anyone builds it. Coverage is the sharpest check:
every accepted requirement must map to an executable task and verification.

One independent review is required for every written plan, including a single
task, unless explicitly owner-declined in the existing brief. An inline author
check is not independent review. No automatic second-model discovery, provider
probing, permanent preference file or repeated consent. A separately requested
extra reviewer needs explicit coordinator admission.

## Identify and review

Read the exact plan and source spec. If the owner explicitly declined a spec,
read accepted requirements and retained decline instead; report coverage against
those requirements without pretending a spec was reviewed. Otherwise missing
spec/source anchors block the affected comparison.

Only the coordinator admits the read-only reviewer with
[plan-reviewer.md](plan-reviewer.md). Pass relevant context, exact document/source
identities and actual evidence references, not session history. Verify real native
tools, permissions/auth and delivery/wait controls; missing prerequisites stop the
lane without provider/model fallback. Read the complete returned report.
The reviewer remains a leaf, with no fixes, new gates or successor admission.

## Findings and acceptance

Retain stable source IDs and original severities. Emit consequence-based
Critical/Major/Medium/Low. Critical/Major or more than two distinct Medium block;
up to two Medium residuals require explicit owner acceptance. Low gets disposition.
Missing requirements, authority or tests independently block. Skipped lenses
are unknown, not clean.

Use receiving-code-review-cl for at most one batched correction/disposition:
exact defect/bytes and sufficient positive/negative/regression oracle permit
parent-verified resolution, not independent re-review. Changed public interface,
safety invariant or acceptance scope, or insufficient proof, returns to the owner
for a decision/additional grant. No automatic reviewer/fixer chain.

## Rubric and scoped handoff

The preserved reviewer rubric checks requirement coverage, bite-sized actions,
placeholders, type consistency, project idioms and right-sized tasks, plus
engineering/build order, applicable UI design, DevEx and strategy lenses. Its
interfaces and ownership bullet checks the recorded or defaulted `Slicing:`
value and the execution graph: agreement with task text, consistent task IDs,
no missing edges or cycles, correct `start`/`join` labels, shared-file hazards
named and resolved by one owner, serialization or an authorized seam split, no
unaccepted owned collaborators hidden behind mocks, and J/C/R barriers present.

The coordinator, not this reviewer, selects execution mode from ownership and
dependencies; the reviewer reports feasibility only. plan-execute-cl runs serial
and parallel-ready tasks. Risk or task count does not add task reviewers.
Return the accepted subject, complete findings/disposition and exact evidence
references; do not automatically invoke implementation.

At acceptance record the sole progress authority once in the existing brief.
Run-owning native state wins; otherwise explicit project selection, existing
tracker or one existing durable plan-associated record. UI todos alone do not
outrank a project tracker. Do not mirror tk, checkboxes or todos. Respect actual
tracking/ignore policy and retain settled work on resume.

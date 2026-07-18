---
id: pat-nlf8
status: closed
deps: [pat-06p3, pat-xdxv]
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 2
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# Specify the plan -> tk mirror's shape in writing-plans

Plan Task 7 (R3 cont.). Files: artifacts/skills/writing-plans/SKILL.md (:154-173). Skip -p and tk ready — which SORTS BY PRIORITY — hands back your work in no meaningful order. Skip --acceptance and 'one ticket = one verifiable outcome' is unenforceable. Skip tk dep and the build order exists only in prose you just gitignored.

## Acceptance Criteria

writing-plans specifies: one epic per plan, one ticket per plan TASK, --acceptance copied from the plan's verification step, -p from the build order, tk dep for the edges


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 7: The plan→tk mirror's shape' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 7: The plan→tk mirror's shape". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-12T04:59:27Z**

SCOPE ADDED (2026-07-12): the writing-plans mirror must (a) name the PLAN FILE in --external-ref, (b) put the task's VERBATIM section heading in -d — copied via `grep -n '^## Task' <plan>`, never retyped, and never a bare 'Plan Task N' (a number forces the reader to count into a document that may be reordered), (c) state that docs/specs/ is gitignored, and (d) VERIFY every pointer resolves against the plan file. EVIDENCE: on the first real seeding of this graph the headings were retyped from memory and 9 of 23 were WRONG. A pointer you have not followed is a claim, not a reference.

**2026-07-18T23:47:13Z**

DONE (Task 7). writing-plans now specifies the plan->tk mirror: one epic to group, one ticket per PLAN TASK (reviewable unit, not per step), plan's verification step copied verbatim into --acceptance, build order as tk dep edges, -p from the ordering (tk ready SORTS BY PRIORITY). The RESOLVE discipline: --external-ref names the PLAN FILE not the folder (a folder holds many plans, ADR-0003 — folder ref ambiguous by construction); -d names the SECTION HEADING verbatim (grep '^## Task' and COPY, never 'Plan Task 4'); say docs/specs/ is gitignored. Plus the verification loop 'every cited heading must exist in the plan it cites'. Replaced the older 'Optional: Mirror Tasks Into Ticket' section. VERIFIED against the LIVE graph: ran the resolver over all .tickets/ -> 23 resolve, 0 broken; tk ready sorts P0>P2>P3; no dep cycles. This is the shape team-implement Phase 2 + plan-review's Next both assume. Clean-body rule applied.

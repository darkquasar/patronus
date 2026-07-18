---
id: pat-27x6
status: closed
deps: []
links: []
created: 2026-07-12T04:48:22Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# The playbook: pointers not gates — grilling gets an exit, brainstorming permits spec-review

Plan Task 4 (R4). Files: artifacts/skills/{grilling,brainstorming,team-research,spec-review,writing-plans,plan-review}/SKILL.md. grilling has an INBOUND pointer and ZERO outbound — it interviews and dead-ends. brainstorming:61,179-180 FORBIDS the very hop spec-review:79 depends on. brainstorming is vendored; ADR-0001 authorizes re-coupling it — PRESERVE THE NOTICE. Also reconcile its duplicate review mechanisms: keep the cheap inline placeholder scan, delegate the real review to spec-review's subagent.

## Acceptance Criteria

Every lifecycle skill names its successor; grep -rn 'ONLY skill you invoke after brainstorming' -> 0 hits; NOTICE files still present on the vendored skills


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 4: The playbook — pointers, not gates' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 4: The playbook — pointers, not gates". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-18T23:40:14Z**

DONE (Task 4). Through-line closes: grilling now has an OUTBOUND exit (team-research | brainstorming, offer-not-gate) it never had. brainstorming's contradiction resolved — terminal state is the WRITTEN SPEC, next hop is spec-review (it forbade that hop before via 'the ONLY skill you invoke after brainstorming is writing-plans'; the process-flow doublecircle + Implementation block both fixed). Duplicate review mechanisms reconciled: the inline 'fresh eyes' self-review is demoted to a cheap PLACEHOLDER SCAN (author cannot have fresh eyes on own work), the real review delegated to spec-review's fresh subagent; spec-document-reviewer-prompt.md kept as that review's PROMPT, not a 2nd path. Successor 'Next:' pointers added to team-research, spec-review, plan-review, writing-plans. NOTICE files preserved (ADR-0001). Clean-body rule applied. Acceptance: all 6 skills name a successor; contradiction grep = 0; NOTICEs present.

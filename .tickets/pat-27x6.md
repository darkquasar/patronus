---
id: pat-27x6
status: open
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


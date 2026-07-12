---
id: pat-54y7
status: open
deps: []
links: []
created: 2026-07-12T04:48:22Z
type: epic
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
tags: [lifecycle]
---
# Lifecycle skills — playbook, tk scaffolding, team protocol, drift guard

Stream 'lifecycle-skills' of docs/specs/01-lifecycle-and-test-surface. Plan: lifecycle-skills-plan.md (10 tasks, R1-R7). Root cause: the thing that REPORTS the state diverged from the thing that IS the state, and nothing reconciles them. Groups only; tk ready reads deps, never parent.

## Acceptance Criteria

patronus scan reports an UNMANAGED SHADOW (the defect that started this); every lifecycle skill names a predecessor and successor; grep for TeamCreate/tasks.md/'graph is flat'/'If your platform supports' all return 0


## Notes

**2026-07-12T04:49:12Z**

NOT ACTIONABLE — this is a GROUPING epic. It appears in 'tk ready' because ready/blocked read ONLY deps, never parent. Epics group and display; tk dep orders. Work the children.

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md (10 tasks, R1–R7). SPEC: lifecycle-skills-spec.md. See also docs/adr/0003-spec-folder-is-a-research-effort-with-many-streams.md (COMMITTED). ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

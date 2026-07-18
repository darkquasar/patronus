---
id: pat-xdxv
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
# Folder schema: one research effort, many streams (ADR-0003)

Plan Task 3 (R1), implements docs/adr/0003-spec-folder-is-a-research-effort-with-many-streams.md. Files: artifacts/skills/{brainstorming,writing-plans,team-research,team-implement}/SKILL.md. A STREAM is one spec + one plan; the folder has many. A file's PRESENCE is its completeness flag (name the file; never assert 'true' beside it). tickets: becomes streams[].epic — an id you can resolve with tk show, not a bit you can only believe. Every stream gets a spec by default; the user may decline, and the decline is RECORDED.

## Acceptance Criteria

grep -rn 'completeness:' artifacts/skills/ -> 0 hits (the scalar block is gone); every skill writes only the field it produces; THIS folder's meta.yaml validates under the new schema


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 3: The folder schema — one research effort, many streams (ADR-0003)' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 3: The folder schema — one research effort, many streams (ADR-0003)". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-18T23:30:43Z**

DONE (Task 3, ADR-0003). Schema rewritten in brainstorming/team-research/team-implement + writing-plans (fills its stream's plan:) + NOTICE, spec-document-reviewer-prompt, plan-review. completeness:{} scalar block GONE; a file's PRESENCE is its flag (name the file, never spec: true); tasks:/tickets: -> streams[].epic (a resolvable tk id). Names prefixed everywhere: <slug>-research.md, <stream>-spec.md, <stream>-plan.md, <stream>-<who>-findings.md. gitignore is now a check (git check-ignore) not an assertion. Two checkable invariants stated in brainstorming+team-research. DOGFOOD: this folder's meta.yaml was already in the new schema; BOTH invariants verified against it and both epic ids (pat-54y7, pat-zr8z) resolve via tk show. Clean-body rule applied — all 'old scalar schema said X' archaeology kept OUT of the skill bodies (it lives in ADR-0003 and this commit).

**2026-07-18T23:30:43Z**

RESIDUE FOR pat-2dgf (Task 2, still blocked on pat-06p3 which is CLOSED + this pat-xdxv now closing): team-implement/SKILL.md Phase 2 (line ~51) still says 'Write tasks.md / set tasks: true / TASKS-TEMPLATE.md'. That's Task 2's verifiable outcome (kill tasks.md + delete the template), not Task 3's. My schema edits made Phase 0/6 streams-based; pat-2dgf must reconcile Phase 2 and delete TASKS-TEMPLATE.md. With pat-06p3 closed and pat-xdxv closing, pat-2dgf should be READY next.

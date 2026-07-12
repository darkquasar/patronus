---
id: pat-xdxv
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
# Folder schema: one research effort, many streams (ADR-0003)

Plan Task 3 (R1), implements docs/adr/0003-spec-folder-is-a-research-effort-with-many-streams.md. Files: artifacts/skills/{brainstorming,writing-plans,team-research,team-implement}/SKILL.md. A STREAM is one spec + one plan; the folder has many. A file's PRESENCE is its completeness flag (name the file; never assert 'true' beside it). tickets: becomes streams[].epic — an id you can resolve with tk show, not a bit you can only believe. Every stream gets a spec by default; the user may decline, and the decline is RECORDED.

## Acceptance Criteria

grep -rn 'completeness:' artifacts/skills/ -> 0 hits (the scalar block is gone); every skill writes only the field it produces; THIS folder's meta.yaml validates under the new schema


## Notes

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 3: The folder schema — one research effort, many streams (ADR-0003)' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 3: The folder schema — one research effort, many streams (ADR-0003)". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

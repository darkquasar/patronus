---
id: pat-uxdx
status: open
deps: []
links: []
created: 2026-07-12T04:48:22Z
type: task
priority: 2
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle]
---
# Reviews run in a fresh subagent — stop hedging

Plan Task 5 (R5). Files: artifacts/skills/writing-plans/SKILL.md (:144, :177), executing-plans/SKILL.md (:14, :72). The three DEDICATED review skills already mandate a subagent WITH the rationale. The gap is the two skills that review their OWN output inline under the name 'fresh eyes' — and writing-plans says the quiet part out loud: 'a checklist you run yourself.' THE AUTHOR CANNOT HAVE FRESH EYES ON THEIR OWN WORK.

## Acceptance Criteria

grep -rn 'If your platform supports' artifacts/skills/ -> 0 hits; grep -rn 'checklist you run yourself' -> 0 hits


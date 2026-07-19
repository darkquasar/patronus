---
id: pat-tcem
status: open
deps: [pat-4v2t, pat-nfi7]
links: []
created: 2026-07-19T01:56:53Z
type: task
priority: 3
assignee: darkquasar
external-ref: profiles/core.yaml
parent: pat-7i2i
tags: [codex-flavour]
---
# Wire team-{implement,research}@codex into profiles/core.yaml

Wire the two codex-flavoured artifacts into the core profile (mirrors how team-{implement,research}@claude are wired). Depends on both artifacts existing.

## Acceptance Criteria

grep 'team-implement@codex' profiles/core.yaml AND grep 'team-research@codex' profiles/core.yaml both hit; patronus build validates


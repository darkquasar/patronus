---
id: pat-kwwi
status: open
deps: []
links: []
created: 2026-07-19T02:01:20Z
type: task
priority: 3
assignee: darkquasar
external-ref: internal/adapter/builtin/codex.yaml
parent: pat-7i2i
tags: [codex-parity]
---
# DISCOVERY: map the Codex runtime and the core-profile parity gaps

The gate for the whole epic. Understand how Codex actually works before authoring anything. Sources: internal/adapter/builtin/codex.yaml (what's already mapped + the hook:null TODO), profiles/core.yaml (the @claude-only items), and the real codex runtime/docs. May want /team-research or grilling the user on the codex exec spawn model + hook surface. Output feeds the acceptance criteria of every downstream child. NOTE: docs/specs/ is GITIGNORED.

## Acceptance Criteria

A written parity map exists (docs/specs stream or research doc) covering: (1) the codex spawn/isolation model (codex exec + git worktree + supervisor); (2) the codex hook/event surface that fills adapter codex.yaml's hook:null TODO, or a finding that none exists; (3) a per-item verdict for every @claude-only core.yaml entry — needs @codex flavour, or N/A with reason


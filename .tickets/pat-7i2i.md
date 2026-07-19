---
id: pat-7i2i
status: open
deps: []
links: []
created: 2026-07-18T22:05:05Z
type: epic
priority: 3
assignee: darkquasar
external-ref: profiles/core.yaml
tags: [codex-parity]
---
# Codex-runtime parity for the core Patronus profile

EPIC. Bring the **core Patronus profile to parity with the Codex runtime**, so a user on Codex gets an equivalent capability set to a user on Claude — not just the two team skills. The team-skills codex flavour (deferred from the lifecycle plan, Task 6 Step 6) is ONE instance of this; the epic is the whole parity surface.

**The parity gaps, from the real code (not guesses):**
- `internal/adapter/builtin/codex.yaml` maps skill/agent/command/mcp/setting/instruction/output-style, but **`hook: null` with a TODO** ("confirm settings surface before authoring hook artifacts"). Every hook artifact is therefore Claude-only today.
- `profiles/core.yaml` has `@claude`-only items with NO codex counterpart: `skills-heartbeat@claude` + `work-state-reground@claude` (both hooks — blocked on the hook gap above), `team-research@claude` + `team-implement@claude` (need the codex spawn/isolation flavour), `ccusage-statusline@claude` (a setting — may be N/A on codex).
- The cross-cutting rule (shipped in pat-jl1r): do NOT branch on capability inside one skill — a markdown file cannot execute if(hasNativeWorktrees). Patronus flavours PER TARGET at BUILD time. So parity = a separate codex-flavoured artifact per capability, wired per target — never one skill introspecting its runtime.

**Discovery gates the build.** Nothing is authored until the codex runtime is actually understood — its spawn model (codex exec + git worktree + a supervisor, since it has no native isolation:worktree), its hook/event surface (the adapter TODO), and, per core item, whether it needs a @codex flavour or is legitimately N/A.

Grouped children (`--parent pat-7i2i`; ordering via `tk dep`):
- DISCOVERY: map the Codex runtime + the core-profile parity gaps (blocks the rest)
- team-implement-codex artifact
- team-research-codex artifact
- profile wiring (team-{implement,research}@codex, + any other @codex items discovery surfaces)
- (further children as discovery scopes the hook-surface parity, etc.)

NOTE: docs/specs/ is GITIGNORED. A fresh session should run DISCOVERY first (it may want /team-research or grilling the user on the codex spawn + hook model) before any authoring.


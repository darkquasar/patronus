---
name: isolated-writer-supervisor
description: "Explicit-only CLI worker supervision when the user chooses subprocess automation. Native workflows may use the worktree-only utility without activating this skill."
---

# Optional CLI worker supervision

## Mode gate, before setup or launch

Use this orchestration skill only by explicit user choice. First ask whether the
user wants native subagents (recommended when the active harness can do the work)
or specifically wants CLI workers. Reuse an already explicit CLI decision for
the exact scope; do not ask again. Merely requesting parallel work, naming this
skill or encountering an unavailable native tool is not CLI-launch consent.

For native mode, stop here and return to the matching installed native execution
skill, `{skillsDir}/plan-execute-parallel-cx/SKILL.md` for Codex or
`{skillsDir}/plan-execute-parallel-oc/SKILL.md` for OpenCode. No subprocess agent,
request scheduler, CLI result contract or subprocess qualification is needed.

Only after explicit CLI selection, read
`{skillDir}/references/cli-workers.md`. Bind the exact worker command, harness
and grants before launching; do not change model/provider or cross harnesses
without that separate authorization. This skill grants no source edits,
integration, publication or cleanup.

## Worktree-only utility, not CLI worker mode

Native workflows use the matching parallel executor's bundled allocator. This
skill's compatibility flag is only a proxy to that installed helper:

```sh
python3 {skillDir}/scripts/supervisor.py run ALLOCATION.json --worktrees-only
```

The proxy selects the installed Codex/OpenCode allocator only when all discovered
copies are byte-identical; absence or divergence blocks. It launches no agents.
Its allocation input binds the repository, base, brief, output roots and disjoint
task claims; no worker command, timeout or result receipt is accepted. It returns
allocated roots/branches/base, not a
completed run. It does not apply predecessor patches, integrate, retry or remove
worktrees. It requires Git-allocation authority, not CLI-agent consent.

Prefer the harness's native allocation mechanism when suitable. A leaf may
execute only the coordinator's exact approved allocation, never choose new
roots/claims, spawn successors or take over scheduling. Worktrees provide
cooperative isolation, not an OS sandbox. Partial allocations are retained and
reported, not silently cleaned up.

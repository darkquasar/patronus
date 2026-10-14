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
skill, `{skillsDir}/plan-execute-cx/SKILL.md` for Codex or
`{skillsDir}/plan-execute-oc/SKILL.md` for OpenCode, which prepare writer roots
through `{skillsDir}/using-git-worktrees/SKILL.md`. No subprocess agent,
request scheduler, CLI result contract or subprocess qualification is needed.

Only after explicit CLI selection, read
`{skillDir}/references/cli-workers.md`. When the explicitly selected worker
harness is OpenCode, also read `{skillDir}/references/opencode-cli-workers.md`.
Bind the exact worker command, harness and grants before launching; do not
change model/provider or cross harnesses without that separate authorization.
This skill grants no source edits, integration, publication or cleanup.

## Worktree-only utility, not CLI worker mode

Native workflows use the allocator of the shared `using-git-worktrees` skill.
This skill's compatibility flag is only a proxy to that installed helper:

```sh
python3 {skillDir}/scripts/supervisor.py run ALLOCATION.json --worktrees-only
```

The allocator lives only in the installed shared skill, at
`using-git-worktrees/scripts/prepare-worktrees.py` beside this skill. There is
one copy, so there is no multi-copy agreement check. If it is absent, the proxy
returns `blocked` and names the skill to install; it never falls back to a
retired executor path. It launches no agents. Its allocation input binds the repository, base, brief, output roots and disjoint
task claims; no worker command, timeout or result receipt is accepted. It returns
allocated roots/branches/base, not a
completed run. It does not apply predecessor patches, integrate, retry or remove
worktrees. It requires Git-allocation authority, not CLI-agent consent.

Prefer the harness's native allocation mechanism when suitable. A leaf may
execute only the coordinator's exact approved allocation, never choose new
roots/claims, spawn successors or take over scheduling. Worktrees provide
cooperative isolation, not an OS sandbox. Partial allocations are retained and
reported, not silently cleaned up.

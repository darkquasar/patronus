---
name: using-git-worktrees-cx
description: "Codex only: Use when feature work needs an isolated Git workspace or an approved plan assigns worktree ownership."
---

# Choose worktree ownership before work

Isolation, source authoring, integration, publication and cleanup are separate
grants. This skill grants none of them. A leaf uses its allocated root and never
allocates another worktree. Worktrees provide cooperative Git isolation, not an OS
sandbox or a restriction on tools that can reach other paths. Static route and
placement checks do not establish native runtime behavior.

## Inspect before allocating

Read project instructions and the task's recorded scope, source root, base commit,
branch, file claims and output binding. Inspect with read-only Git commands:

```sh
git rev-parse --show-toplevel
git rev-parse --git-dir --git-common-dir
git rev-parse --show-superproject-working-tree
git rev-parse HEAD
git branch --show-current
git status --porcelain=v1
git worktree list --porcelain
```

Different Git/common directories suggest a linked worktree, but a submodule is
not proof of worktree allocation. Reuse an existing allocated root only when its
identity and ownership match the grant. Detached HEAD does not require branch
creation if the workspace is externally managed. Do not switch its branch.
Dirty files must remain visible and owned. Never stash source, discard it or
allocate from dirty source as an implicit fallback.

## Explicit allocation, if authorized

The lead records the exact root/base, branch, disjoint file claims, output paths,
owner, test/resource limits and capture/cleanup consent before allocation. Use the
project's approved mechanism. An authorized manual Git worktree is an intentional
choice, never a failed-engine fallback. For a project-local destination verify
it is ignored before creation. If it is not, ask for an approved destination or
an explicit ignore-file edit grant; do not edit or commit automatically.

Missing permission, unavailable isolation, path collision, overlap or an unknown
owner blocks the lane. No in-place fallback. Do not install dependencies or alter
settings to make allocation succeed. Run only configured setup/baseline checks
under their lock and budgets. A failing baseline needs an explicit disposition.

## Capture and retention

Before integration or cleanup, confirm every started worker and descendant has
settled. Unknown settlement retains the workspace and blocks retry/integration.
Capture the complete tracked and untracked patch, base/head, file/mode hashes,
result bytes and test logs. A pathname under `worktrees/` is not ownership proof.
Cleanup requires exact pre-recorded ownership and explicit cleanup authority;
publication and merge do not imply deletion. Never run blanket prune or remove
another owner's root. Preserve failed attempts and evidence.

For approved execution, read `{skillsDir}/plan-execute-cx/SKILL.md` only after
confirming that exact companion is installed. It does not authorize a stage hop.
For completion decisions use `{skillsDir}/finishing-a-development-branch-cx/SKILL.md`.

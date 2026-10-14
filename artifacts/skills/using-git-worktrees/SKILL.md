---
name: using-git-worktrees
description: "Use when feature work needs an isolated Git workspace, or an approved plan assigns writer roots, prepared baselines or worktree ownership."
---

# Choose worktree ownership before work

Isolation, source authoring, integration, publication and cleanup are separate
grants. This skill grants none of them. A leaf uses its allocated root; it may
execute only an exact allocation already authorized by the coordinator, never
choose new roots, claims or successors. Worktrees provide cooperative Git
isolation, not an OS sandbox or a restriction on tools that can reach other
paths. Static route and placement checks do not establish native runtime behavior.

## Inspect before allocating

Read project instructions and the task's recorded scope, source root, base
revision, branch, file claims and output binding. Inspect with read-only Git
commands:

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
Dirty files must remain visible and owned. Never stash, discard or commit owner
work to manufacture a clean base, and never allocate from dirty source as an
implicit fallback.

## Explicit allocation, if authorized

The lead records the exact root/base, branch, disjoint file claims, output
paths, owner, test/resource limits and capture/cleanup consent before
allocation. Allocate through suitable native isolation, authorized Git, or the
bundled allocation-only script. An authorized manual Git worktree is an
intentional choice, never a failed-engine fallback. For a project-local
destination, verify it is ignored (`git check-ignore`) before creation. If it is
not, ask for an approved destination or an explicit ignore-file edit grant;
never edit the ignore file or commit it automatically.

Missing permission, unavailable isolation, path collision, overlap or an unknown
owner blocks the lane. There is no in-place fallback. Do not install
dependencies, run package setup or alter settings to make allocation succeed.
Run only configured setup/baseline checks under their lock and budgets. A
failing baseline needs an explicit disposition.

The bundled allocator lives at `scripts/prepare-worktrees.py`, relative to this
skill's directory:

```sh
python3 scripts/prepare-worktrees.py ALLOCATION.json --validate
python3 scripts/prepare-worktrees.py ALLOCATION.json
```

Its input and exit contract is in `references/worktree-allocation.md`. It
allocates from the named base only. It launches no agents, needs no supervisor,
activates no CLI skill and applies no predecessor bytes. CLI supervision needs
explicit user choice.

## Prepared baselines

When a writer needs accepted predecessor bytes that the base lacks (dirty,
untracked or relevant ignored files), carry exactly those bytes into each
writer root after allocation and before launch. Bind every carried path and its
hash, then record the prepared-baseline identity: root, base, carried paths with
hashes and the resulting status. The writer verifies root, base and those hashes
before its first write. Writers never run concurrently in the lead checkout.

## Ownership, settlement and capture

Verify that claims are disjoint before launch and again after settlement. Before
capture, integration or cleanup, confirm that every started worker and every
descendant it started has stopped. Unknown settlement retains the workspace and
blocks retry, capture and integration.

Capture against the prepared baseline, not the bare base, so inherited
predecessor bytes are not credited to the worker as its own delta. The capture
holds the complete tracked and untracked delta, with deletions, modes and binary
bytes preserved, plus base/head, file/mode hashes, result bytes and test logs.
A change outside the owned paths, a drifted input hash or a failed capture stops
acceptance of that work and keeps its root and evidence. A pathname under
`worktrees/` is not ownership proof.

## Retention and cleanup

Cleanup requires exact pre-recorded ownership and explicit cleanup authority;
publication and merge do not imply deletion. Never run blanket prune or remove
another owner's root. Preserve failed attempts and evidence.

## Host notes

- **Claude:** a writer may use the Agent tool's native `isolation: "worktree"`
  only when every required accepted input is already in the revision that
  isolation starts from; record the returned path and branch. Otherwise prepare
  the root here and pass it to the writer as an absolute root.
- **Codex:** native workers do not get a separately allocated or enforced writer
  root. Prepare the root here and bind it in the brief.
- **OpenCode:** Task subagents share the lead cwd, and the `patronus-writer-oc`
  role denies `external_directory`. Check the writer's actual access to the
  assigned root before a parallel write. Missing access blocks the lane; never
  bypass it with Bash.

## Next stage

Execution and completion stay with the host's own skills: `plan-execute-cl`,
`plan-execute-cx` or `plan-execute-oc`, then the matching
`finishing-a-development-branch-cl`, `-cx` or `-oc`. Read one only after
confirming it is installed. This skill does not authorize a stage hop.

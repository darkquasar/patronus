# Allocate worktrees only

Use only for an exact coordinator-approved allocation with Git authority. Native
launch, admission, wait/stop and integration stay with the coordinator. The
script is `scripts/prepare-worktrees.py` (in this skill's directory):

```sh
python3 scripts/prepare-worktrees.py ALLOCATION.json --validate
python3 scripts/prepare-worktrees.py ALLOCATION.json
```

Example input (replace the full base and brief hash with actual checked values):

```json
{
  "schemaVersion": 1,
  "runId": "first-wave",
  "repository": "/work/project",
  "baseCommit": "<full checked commit id>",
  "brief": {"path": "/work/brief.md", "sha256": "<actual brief SHA256>"},
  "worktreeRoot": "/work/allocated-roots",
  "runDir": "/work/allocation-output/first-wave",
  "tasks": [
    {"key": "registry", "ownedPaths": ["internal/registry/"]},
    {"key": "skills", "ownedPaths": ["artifacts/skills/"]}
  ]
}
```

Roots must be absolute canonical paths outside registered checkouts/Git storage;
output and worktree roots are disjoint. Claims are normalized repository-relative
paths, disjoint across tasks; a trailing slash claims a directory. Ancestor/child
claims conflict even without a trailing slash, so `src` and `src/b` cannot belong
to different writers. Existing roots,
branches, symlink destinations/claims, predecessors, integration and unknown
fields are refused. No worker command, concurrency, deadline, retry or result
contract is accepted.

The helper writes one allocation.json with task/root/branch/base bindings. This
is allocation output, not another progress tracker or execution receipt. Exit 0
means allocated (or valid with --validate), not implemented or verified. Invalid
requests exit 2; partial failures exit 3 and retain roots/diagnostics. Inspect
those before any new allocation. There is no cleanup, capture or agent launch.
Git checkout can invoke repository-configured hooks/filters; the helper is not an
OS sandbox. It starts at the named commit only. Carry accepted dirty/untracked
inputs separately before native writers start.

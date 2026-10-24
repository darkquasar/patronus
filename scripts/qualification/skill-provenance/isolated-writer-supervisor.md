# Source and invocation inventory

Neutral isolated-writer supervisor, version 1.0.0. The helper, schema and
fixtures are Patronus-authored (first introduced in `plan-execute-parallel-cx`
1.1.0, settlement corrections in 1.1.1) and were moved here unchanged from
`artifacts/skills/plan-execute-parallel-cx` 1.1.1 at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. `SKILL.md` is authored for this
artifact; the original attribution (`NOTICE`, `LICENSE`) is copied. The
supervisor accepts a worker argv already; no harness-specific code was added.

## Moved files (byte-identical)

| File | SHA-256 |
|------|---------|
| `scripts/supervisor.py` | `fde4bf79ead669fd871e16b4e0b6b21ae8e732fba53d1ce6ca418d1ee7d0d85c` |
| `fixtures/fake_worker.py` | `7fa9273b817a6743998ffd29b09b0e6995be0ac2de4deccf86fd46e4562edbeb` |
| `fixtures/test_supervisor.py` | `717c6bb911e60e4b939082b90c75d77212576715ef601be30664869293c5bc57` |
| `references/request.schema.json` | `d5a83f1c5960e8e6f0083c1e4e0939cd9e3463f98fbd07c640f4f38365123d32` |
| `NOTICE` | `f06979caadf3af300ba27c87dee139bedad4f0a86caf2cd5b37884ef1b955714` |
| `LICENSE` | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |

Only the entry and the manifest-declared sidecars are distributed. The
fake-worker suite runs invented local processes only; it is not native
qualification of any worker, harness, authentication or sandbox.

## Historical fixture result

Run once from this location on 2026-10-09 (UTC), not part of any development
gate:

```text
python3 artifacts/skills/isolated-writer-supervisor/fixtures/test_supervisor.py
Ran 12 tests in 10.789s
OK
exit 0
```

This is historical evidence of the fixtures at the recorded revision. It is
NOT RUN in the application or catalog gates and does not mark any native row
passed.

## Historical 1.1.x to 1.2.0 migration

Existing 1.1.x installs keep ownership of the old sidecar paths; `patronus update` reports
`obsolete owned path retained; explicit removal required` for them. Run
`patronus remove plan-execute-parallel-cx` then reinstall, or remove the listed paths
explicitly, before relying on the shared supervisor.

## Version 2.0.0 (historical)

CLI orchestration is explicit-only, with an immediate native/CLI choice gate.
Codex's declared agents/openai.yaml disables implicit invocation; OpenCode's
explicit-only boundary is instructional, not a claimed native invocation switch.
Full CLI procedures move to references/cli-workers.md, read only for that mode.
The public --worktrees-only compatibility flag delegates to the installed
plan-execute-parallel-cx/oc allocator after verifying any discovered copies are
byte-identical. It contains no second allocation implementation. The strict
allocation request accepts only repository/base/brief/output roots and task
names/claims; execution, scheduling, predecessor, retry/result and integration
fields are rejected. Allocation output is not completed execution evidence.
The historical moved-file hashes above describe 1.0.0, not the amended payload.

## Current version 2.1.0

Native routes now name `plan-execute-cx` and `plan-execute-oc`, which prepare
writer roots through the shared `using-git-worktrees` skill. The retired
`plan-execute-parallel-cx/oc` executors are no longer routes.

The `--worktrees-only` proxy looks only for the installed
`using-git-worktrees/scripts/prepare-worktrees.py` beside this skill. That
script is the single allocator copy, so the two-candidate list and the
"allocators differ" check are removed. An absent allocator returns `blocked`
with an actionable reason; retired executor paths are ignored. Two fixtures
cover this: `test_worktrees_only_proxy_uses_shared_worktree_skill` and
`test_worktrees_only_proxy_ignores_retired_executor_path`. Both first failed
against the 2.0.0 proxy (blocked on a missing executor; allocated from the
retired path), then passed after the change, with the 21 existing cases green.

OpenCode CLI-only material moved here from `plan-execute-parallel-oc`
(Patronus contributors, GPL-3.0, same `LICENSE` bytes). It is read only after
the explicit CLI gate and only when the selected worker harness is OpenCode.
The generic `references/cli-workers.md` is unchanged.

| New path | Prior path | Prior SHA-256 | Change |
|----------|------------|---------------|--------|
| `references/opencode-cli-workers.md` | `plan-execute-parallel-oc/references/cli-workers.md` | `ab7199fe9a7374cf9b683c6a0519236779722aabad7f46c75b7e984f5196c1ec` | precheck invocations name `scripts/opencode-worker-precheck.sh` in this skill; read-only route names `dispatching-parallel-subagents` |
| `scripts/opencode-worker-precheck.sh` | `plan-execute-parallel-oc/scripts/worker-precheck.sh` | `72d0fde1be4869280adb151c973888d5ab6fa3e932a7a4f06062500f1ae60753` | none, bytes unchanged |
| `fixtures/test_opencode_worker_precheck.py` | `plan-execute-parallel-oc/fixtures/test_worker_precheck.py` | `d8279261a52202345755e67630ac187ee049937efec9000116c3ae37b7440626` | script path and docstring name the new file |

The precheck stays CLI-only; native Task writers do not use it. Its fixtures
run invented scratch data only and do not qualify native OpenCode workers or
the permission merge. `requires` now names `using-git-worktrees` (the proxy's
allocator) and `dispatching-parallel-subagents` (the read-only route). The
skill stays optional and explicit-only, outside every core and TDD closure.

## Active reads

- `references/cli-workers.md`: only after explicit CLI selection.
- `references/opencode-cli-workers.md` and `scripts/opencode-worker-precheck.sh`:
  only after explicit CLI selection with OpenCode as the worker harness.
- `using-git-worktrees/scripts/prepare-worktrees.py`: the `--worktrees-only` proxy target.
- `plan-execute-cx/SKILL.md`, `plan-execute-oc/SKILL.md`: native-mode return routes.
- `dispatching-parallel-subagents/SKILL.md`: separately authorized read-only investigation.

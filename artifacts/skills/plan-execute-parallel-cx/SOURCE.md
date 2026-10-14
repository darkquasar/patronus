# Source and invocation inventory

Codex adaptation at source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.

- `artifacts/skills/plan-execute-parallel/patronus.yaml`: SHA-256 `cd3f699dd3ba242f939a3757f1db094879335f24a1654bb16c5ce1ceb45541d6`.
- `artifacts/skills/plan-execute-parallel/SKILL.md`: SHA-256 `5367d1c3bfcce1dffcbd2725407c4b387205e0127a997336f4568db9d9961989`.

Bodies are behaviorally authored for Codex, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Historical supervisor helper

`scripts/supervisor.py`, `references/request.schema.json`,
`fixtures/test_supervisor.py` and `fixtures/fake_worker.py` were new
Patronus-authored files for this Codex skill (introduced in version 1.1.0), not
ports of a source helper. Version 1.1.1 corrects supervisor fault/interruption
settlement and adds stop-unknown, supervisor-error and input-apply-failed fixtures.
Version 1.2.0 moves those four files byte-identically to the required neutral
skill `isolated-writer-supervisor` 1.0.0 (hash table in its SOURCE.md); this
skill no longer distributes them and references the installed sibling path
`{skillsDir}/isolated-writer-supervisor/...`.
The fake-worker suite runs invented local processes only; it is not native Codex
qualification. The captured predecessor and its failed/partial evidence remain
upstream provenance, not a new qualification claim.

## Historical 1.1.x to 1.2.0 migration

Existing 1.1.x installs keep ownership of the old sidecar paths; `patronus update` reports
`obsolete owned path retained; explicit removal required` for them. Run
`patronus remove plan-execute-parallel-cx` then reinstall, or remove the listed paths
explicitly, before relying on the shared supervisor.

## Active reads

- `dispatching-parallel-agents-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `plan-execute-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `using-git-worktrees-cx/SKILL.md`: exact allocated-root companion.
- `{skillDir}/scripts/prepare-worktrees.py` and `{skillDir}/references/worktree-allocation.md`: bundled allocation-only utility, no supervisor dependency or agent launch.
- `isolated-writer-supervisor/SKILL.md`: optional explicit-user CLI mode only, not read for native execution.

## Current version 2.0.0

Native Codex delegation replaces the mandatory CLI helper route. First-wave
decomposition asks reuse existing approval; read-only next-wave preparation
overlaps writers without admitting unaccepted dependencies. The shared utility
is no longer in requires or the resolved core closure. The standalone allocation
script here is canonical; OpenCode packages a byte-identical mirror checked by
the allocation fixtures. Allocated roots, dirty-input preservation, settlement,
serial integration, TDD and aggregate review remain. Source checks and a native
allocation trial do not qualify all installed harness/runtime combinations.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-cx owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.

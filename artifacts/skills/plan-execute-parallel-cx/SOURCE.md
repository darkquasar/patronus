# Source and invocation inventory

Codex adaptation at source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.

- `artifacts/skills/plan-execute-parallel/patronus.yaml`: SHA-256 `cd3f699dd3ba242f939a3757f1db094879335f24a1654bb16c5ce1ceb45541d6`.
- `artifacts/skills/plan-execute-parallel/SKILL.md`: SHA-256 `5367d1c3bfcce1dffcbd2725407c4b387205e0127a997336f4568db9d9961989`.

Bodies are behaviorally authored for Codex, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Bundled helper

`scripts/supervisor.py`, `references/request.schema.json`,
`fixtures/test_supervisor.py` and `fixtures/fake_worker.py` are new
Patronus-authored files for this Codex skill (introduced in version 1.1.0), not
ports of a source helper. Version 1.1.1 corrects supervisor fault/interruption
settlement and adds stop-unknown, supervisor-error and input-apply-failed fixtures.
The fake-worker suite runs invented local processes only; it is not native Codex
qualification. The captured predecessor and its failed/partial evidence remain
upstream provenance, not a new qualification claim.

## Active reads

- `dispatching-parallel-agents-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `plan-execute-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-cx owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.

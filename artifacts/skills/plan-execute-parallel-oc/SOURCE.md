# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `port-oc`.

- `artifacts/skills/plan-execute-parallel-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/plan-execute-parallel-cx/NOTICE`: SHA-256 `f06979caadf3af300ba27c87dee139bedad4f0a86caf2cd5b37884ef1b955714`.
- `artifacts/skills/plan-execute-parallel-cx/SKILL.md`: SHA-256 `19a5de1fc780dce551cbd2287b8165c0a4ed904c8e40dbffdf1547568db08a71`.
- `artifacts/skills/plan-execute-parallel-cx/SOURCE.md`: SHA-256 `a50df9b9cccd5557111deb914a7103696faa0cac3d6f577d8ce7796d970acb94`.
- `artifacts/skills/plan-execute-parallel-cx/patronus.yaml`: SHA-256 `18a0f575a2b3534783bda4568ec02b3e1bc93bdf255ac45889ba02fa05c153e6`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Manifest `files:` adds `scripts/worker-precheck.sh` and `fixtures/test_worker_precheck.py`
  (both new, Patronus-authored for this artifact); `requires` keeps `isolated-writer-supervisor`
  unsuffixed and names `plan-execute-oc` and `dispatching-parallel-agents-oc`.
- Body ported: title and Partial statements name OpenCode; new "Working set precheck"
  section (OC-D6 discoverability: tracked at the base or installed at global scope);
  new "Worker argv" section with the amended OC-D6 argv
  (`env OPENCODE_CONFIG_CONTENT=<inline JSON> opencode run --format json --auto --agent
  patronus-writer-oc -m <provider/model> --dir <attempt worktree> <brief reference>`), the
  exact inline JSON granting `agent.patronus-writer-oc.permission.external_directory."<runDir>/*"`,
  the `python3` and `jq` renderers, the deny/allow merge marked runtime-pending, the
  evidence-root trade, `--auto` flipping `ask` only, the result file as the only contract;
  new "Resource protocol" section (OC-O7 defaults when the brief is silent).
- Dropped the sibling's "Migration from 1.1.x" section: this artifact is new at 1.0.0 and
  never distributed the supervisor files.
- Request binding, Writers, Integration and Limits paragraphs kept in substance; Limits adds
  that `external_directory` is an OpenCode tool rule, not a kernel boundary, and that the
  precheck fixture does not qualify the permission merge.
- Authorized alternative adds the `opencode` executable, the writer role and the guards to the
  blocked-prerequisite list.

## Worker precheck (bundled, Patronus-authored)

`scripts/worker-precheck.sh <base-ref> [--global]` is POSIX sh (`#!/bin/sh`, `set -eu`) using
`git ls-tree -r --name-only`, `git show` and `python3 -c` (json module) only; no jq dependency.
It exits 0 only when all six identities are present (agent with `mode: subagent`, hook runner
plugin, `hooks.json` with `patronusId` second segments `block-secrets-oc` and `gitleaks-guard-oc`,
both guard scripts executable, `opencode.json` `permission.edit` with keys `*`, `*.env`, `.env.*`),
names every missing identity on stderr, exits 1 otherwise and 2 on usage or unknown base.

`fixtures/test_worker_precheck.py` (Python stdlib unittest, invented scratch repositories and
invented agent/hooks/guard/config bytes, no real guard script) is historical and not run by the
development gates, the same status as the supervisor fake-worker suite. Historical run at
`1a9b283` with the dirty Lane B tree: `python3 artifacts/skills/plan-execute-parallel-oc/fixtures/test_worker_precheck.py -v`,
16 tests, OK (log `laneB-2/t10-precheck-fixture.log`). That run proves the script's behaviour on
invented data only; it is not native OpenCode worker qualification.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `dispatching-parallel-agents-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-execute-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `isolated-writer-supervisor/scripts/supervisor.py` (and its schema/fixtures): required shared helper, verify installed/readable before use.
- `{skillDir}/scripts/worker-precheck.sh`: bundled precheck, run before writing a request.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.

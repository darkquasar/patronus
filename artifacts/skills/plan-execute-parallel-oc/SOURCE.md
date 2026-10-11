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

## Fail-closed correction (artifact 1.1.0)

Bounded correction for OC-WC-002 from source base
`76f8f5166e85d7b55c2a855401f8caae922a9cbb`: permission containers are type-checked
before accessing `edit`. Both JSON validators now refuse every nonzero checker exit,
including failure with empty stdout. Known missing identities and malformed JSON keep
named diagnostics; unexpected checker failures use a generic diagnostic, with checker
stderr suppressed to avoid exposing configuration or traceback details. Valid tracked
and env-resolved global working sets retain their previous admission behavior.
No runner, guard program, native permission rule or supervisor behavior changed.

Observed local invented-data run on 2026-10-11: from this artifact directory,
`flock -x /tmp/patronus-gates.lock python3 fixtures/test_worker_precheck.py -v`,
22 tests, OK. New regressions cover scalar/null/list permission containers, malformed
JSON, and controlled invented executable failures without stdout for both JSON
validators (statuses 1, 2, 7 and 127). Existing complete/partial sets, named missing
identities, tracked-base, executable-bit and global-root precedence cases remain.
Fixture subprocesses have 10-second timeouts. Scalar-container, hook-checker and
permission-checker tracer bullets each first observed exit 0 instead of refusal, then
passed after minimal fixes. Full RED/GREEN logs are retained in the source-authoring
handoff at `correction-final/precheck-evidence/` (not distributed artifact files).
The earlier 16-test result above remains historical, not current acceptance.
These fixtures are still outside development gates and prove invented-data script
behavior only. Native OpenCode workers and permission merging remain runtime-pending.

Corrected source identities (SHA-256):

- `scripts/worker-precheck.sh`: `72d0fde1be4869280adb151c973888d5ab6fa3e932a7a4f06062500f1ae60753`.
- `fixtures/test_worker_precheck.py`: `d8279261a52202345755e67630ac187ee049937efec9000116c3ae37b7440626`.

## Active reads

- `dispatching-parallel-agents-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-execute-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `isolated-writer-supervisor/scripts/supervisor.py` (and its schema/fixtures): required shared helper, verify installed/readable before use.
- `{skillDir}/scripts/worker-precheck.sh`: bundled precheck, run before writing a request.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.

# Failure and resource procedures

Operator/coordinator guidance only: no admission daemon, repair helper or runtime
policy interceptor is delivered. Follow [readiness](readiness.md), [writer
bootstrap](writer-bootstrap.md) and [teardown](teardown.md) under separate grants.

## FM-1 — One readiness attempt, then retain and report

Default budget is **one attempted readiness check** for the selected identity,
not one retry per alias, transport or failing tool. An attempt is the bounded
readiness sequence, including initialization/root verification before symbols;
stop the affected sequence on failure. Record a missing tool/provider as an
unavailable attempt, not an invitation to launch another client. Readiness is
not a listening socket. It needs actual initialization, matching root, live
representative definition/caller, graph query/provenance and restrictions.

For each failure retain: run/owner/grant, UTC start/end/deadline, attempt number,
expected **and observed** endpoint/namespace/transport, canonical cwd/root,
revision/dirty/untracked inventory, config source/path/digest and alias, exact
host/adapter/server/LSP/provider versions, stage/Pi-trust/MCP/tool approvals,
operation/error/log references, affected claim and permitted fallback. Use
`unknown` for unavailable observations; never replace expected identity with the
unexpected one. Preserve partial evidence without tokens, secret values or broad
environment dumps. Source reads may continue only within the task's existing
roots/authority and when its required outcome does not depend on live tools.
Otherwise block that outcome and escalate.

A retry needs a new explicit coordinator/owner budget tied to that failure and
identity: reason, maximum additional attempts, deadline, resource reservation,
allowed operations and stop condition. Revalidate changed inputs and approvals;
never loop indefinitely. No automatic package repair/install, process/LSP launch,
graph refresh, project/mode/model/protocol switch or alternate provider fallback.

## FM-2 — Diagnose without repairing

| Failure case | Expected versus observed evidence | Safe disposition after the attempt |
|---|---|---|
| Missing endpoint / socket-only | Approved endpoint and namespace versus connection/initialize error or only a listening socket; service owner/identity unknown if unavailable | Fail affected readiness, report namespace/loopback limits; read local source, do not start a replacement. |
| Missing provider / unloaded or foreground MCP | Required effective role gateway/provider/extensions versus actual registry/load error and launch mode | Infrastructure failure, not task success. Retain actual registry/agentConfig; no skill/tool-name substitution or alternate runner. |
| Missing/broken LSP | Required language, executable/toolchain pin and known definition/caller versus initialization error or unsupported language | Report missing navigation; source reads permitted, no automatic language-server download/start/upgrade. Empty callers alone cannot prove absence. |
| Wrong root / writer cwd mismatch | Canonical selected root and writer cwd/revision versus initialized shared server root | Stop worktree-local symbol claims; never switch the shared project or present main-root symbols as writer-local. Use granted source reads and escalate the mismatch. |
| Stale alias / effective config / reload | Approved source inventory/digests/definition and reload acknowledgement versus cached alias/path/transport/tools/root | Hold MCP dispatch, preserve both disk/effective evidence. Owner resolves/reloads with all-client accounting, not repeated writes. Same-name static conflicts still refuse. |
| Missing/malformed graph | Expected graph format/hash/provenance versus absent/unreadable/invalid data | Do not query as valid evidence or infer code absence. Read source and report limitation, no rebuild/repair. |
| Historical graph / untracked input / excluded docs | Expected root/revision/included inventory versus snapshot origin/hash and actual changed/untracked/excluded paths | Label historical/coverage-limited; verify consequential EXTRACTED and INFERRED edges in current source. Unchanged HEAD/tracked diff cannot establish freshness. No child refresh. |
| Denied Pi project trust | Required exact project resource trust versus actual Pi trust decision | Record Pi trust denial distinctly; no trust relaxation, setup-hook prompt/approval or headless bypass. Source reads only if already allowed. |
| Denied/absent MCP approval | Required approval of exact effective server versus missing/denied adapter decision, independently of Pi trust | No MCP-dependent launch; Pi invocation trust does not approve MCP. Interactive preapproval belongs to operator, not child repair. |
| Version/schema mismatch | Selected host/adapter/server/LSP/provider pins and supported schema versus observed versions/errors | Withhold exact-pin readiness; no silent downgrade, upgrade or mode/model/protocol fallback. Return delta to Q/operator. |

An explicitly selected static declaration that is malformed, unreadable or unsafe
still blocks static admission. Runtime-only registrations are instead visibly
unverified; extension presence alone does not require deactivation or mandatory
`PI_OFFLINE`. This procedure does not widen the inspected static inventory.

## FM-3 — Record independent resource budgets before expensive admission

One coordinator owns reservations and resamples immediately before admission;
record measurement time/source, host namespace and effective cgroup hierarchy.
Measure both host free and available RAM and remaining cgroup headroom (tightest
applicable ancestor limit minus current usage). Unknown/stale/unreadable headroom
blocks expensive work. If no finite cgroup limit applies, explicitly record how
that was verified and use host available bytes as the conservative numeric
headroom; `max` is not a guessed infinite budget. Account for already running work
in usage and additionally reserve its not-yet-used peaks, without double counting.

Example **operations record**, not an installer schema or an executable policy.
Nulls deliberately mean unknown; concurrency limits alone cannot admit it. Byte
units distinguish 500 MB (500000000 bytes) from 750 MiB (786432000 bytes). Limits
are deployment-selected, not assumptions about the machine size. The example's
concurrency/worker settings are conservative starting budgets, not host capacity.

```json
{
  "owner": "<coordinator>",
  "grant": "<hash-bound-action-grant>",
  "rootRun": "<exact-root-and-run>",
  "measurementSource": "<host-and-effective-cgroup-evidence>",
  "observedAt": null,
  "freeBytes": null,
  "availableBytes": null,
  "cgroupHeadroomBytes": null,
  "minimumFreeBytes": 786432000,
  "minimumAvailableBytes": 786432000,
  "reserveBytes": 500000000,
  "outstandingReservedBytes": null,
  "incrementalPeakBytes": null,
  "activeSpawns": null,
  "maxConcurrentSpawns": 1,
  "totalSpawns": null,
  "maxTotalSpawns": 4,
  "activeWorkers": null,
  "requestedWorkers": null,
  "maxWorkers": 1,
  "nowMs": null,
  "deadlineMs": null,
  "requestedDurationMs": null,
  "separatelyAdmittedCoordinatorGraphWork": false,
  "result": "unverified"
}
```

Admission checklist (all predicates independent; unknown is blocked):

1. Verify stage/action/owner/root grant and fresh complete nonnegative measurements
   and reservations. Free **and** available must meet their thresholds. This effort
   requires each >=750 MiB and reserve >=500 MB. Adjust concurrency down to measured
   headroom; the execution coordinator's lower cap overrides any planning maximum.
2. Conservatively require `min(freeBytes, availableBytes, cgroupHeadroomBytes)`
   minus `outstandingReservedBytes` minus `incrementalPeakBytes` >= `reserveBytes`.
   An idle concurrency counter cannot prove reserve. An unknown incremental peak
   is not zero. Keep reserve during the action, not just at its start.
3. Admit one new spawn only if `activeSpawns + 1 <= maxConcurrentSpawns` AND
   `totalSpawns + 1 <= maxTotalSpawns`. Count retries in the total; ending a spawn
   releases concurrency, not total usage. Worker count must independently satisfy
   `activeWorkers + requestedWorkers <= maxWorkers` with requestedWorkers >=1.
4. Require positive bounded requested duration and `nowMs + requestedDurationMs <=
   deadlineMs`. Record actual deadline enforcement owner/mechanism and cancellation/
   settlement observations. An example clock or cooperative cancellation is not
   hard enforcement. Retain unknown descendants under TD-1, not guessed cleanup.
5. Graph operations requiring subprocess/provider work are **coordinator-only**,
   separately admitted with explicit indexing/mode/provider/cost authority even
   when memory fits. Children may only use already-approved query surfaces; no
   automatic CLI extraction, watch or graph build. Serialize expensive tests/builds/
   indexing per deployment policy; on the small implementation host no simultaneous
   expensive actions. Recheck during work; pressure stops new admissions and triggers
   owner-directed bounded cancellation/settlement, never arbitrary process killing.

Low/unknown memory blocks the expensive action but does not grant or revoke
ordinary bounded source reads already permitted by the task. The adjacent fixture
models these decisions in memory for **C-class checklist consistency only**, not
M enforcement or measured host admission. Unknown runtime limits remain unknown.

## FM-4: interrupted recovery preserves evidence

On interrupted profile inverse or package removal, retain Patronus state, original
and current config bytes, package-manager output, service records and per-step
results. A partially restored role or MCP leaf is not safe to replay blindly.
The owner reconciles against TD-1 through TD-4 under a renewed exact-input grant,
then observes reload and base-role behavior. Edited config, missing state, unknown
processes or failed reload preserve work rather than claiming cleanup.

Profile removal only inverses unchanged Patronus ownership. Service processes,
Pi/npm and uv package internals, auth, snapshots, caches, worktrees, outputs and
evidence remain under their recorded owners. Archive, export and deletion need the
owner retention policy and separate authority. A file sentinel or test double
cannot prove teardown, shared-server survival or containment. Return actual
observations and withheld claims using [qualification](qualification.md).

---
name: code-intel-operations-pi
description: Operator-owned MCP readiness, manual lifecycle, failure/resource procedures and qualification handoff for optional Pi code intelligence.
---

# Pi code-intelligence operations

This skill delivers inert documentation/examples, not an executable helper,
installer, service supervisor or configuration policy writer. Installing the
optional `code-intel-pi` profile is **placed, runtime-unverified**. Core requires
neither these services nor this adapter. Operator provision of the selected
pi-mcp-adapter 3.0.0, server pins, language tools and auth is separate.

## Ownership and static boundary

- Operator owns binaries/dependencies/auth/containment and existing global policy.
- Coordinator owns shared services, service descriptors, snapshots and expensive
  indexing. No child starts/stops/retargets a shared server or changes its modes.
- Research/spec/review/plan children own query clients and assigned outputs only.
- Execution writers use local source reads unless private Serena/LSP has separately
  approved exact-worktree ownership, resource admission and interactive Pi/MCP
  approvals. Shared Graphify remains a main snapshot. Follow the manual procedures
  below under their own grants; this static profile grants no lifecycle actions
  and does not alter core role Markdown/settings.

Static admission checks observable files/configuration/manifests/declarations and
resolvable structured imports, not runtime callbacks or exhaustive registration.
Extension entrypoints are not blanket blockers and need not be deactivated merely
for dynamic uncertainty. Known static collisions, unsafe paths, malformed data,
unreadable required static sources, ownership/drift and mixed-context consent
still refuse the whole selection before writes. No npm/git resolver, Pi, extension,
helper or MCP connection runs during static apply. Runtime-only global npm agent
discovery stays visibly unverified: never run `npm root -g` for admission.
`PI_OFFLINE` is not a production admission requirement or network sandbox.

## Shared recipe/config contract

The independent wire-only recipes `serena-shared-pi` and `graphify-shared-pi` merge
only `mcpServers.<recipe-name>` HTTP `{url: ...}` records using D-04. No delivery,
EXEC, credential header, server binary or resolver is supplied. Default endpoints
are loopback ports 9121 and 9122 respectively, path `/mcp`. `wire.tools: [pi]`
selects the default route for empty/all mechanism requests; it is **not** a hard
allowlist against an explicit target. CLI selections still need explicit scope/
target according to the delivery contract. Endpoint customization requires a
reviewed local recipe/config selection; no magic environment interpolation.

Global destination: effective `PI_CODING_AGENT_DIR/mcp-adapter.json`, otherwise
`~/.pi/agent/mcp-adapter.json`. Local destination: the selected workspace's
`.pi/mcp-adapter.json`. Patronus does not manage standard `.mcp.json`, user-global
`.config/mcp/mcp.json` or legacy Pi `mcp.json` on their behalf. Compare canonical
root and actual adapter override path with the preview before apply/readiness.

D's effective-source parser is `internal/scan/pi_discovery.go` (with command-layer
preflight); it owns safe admission, not this skill. At the inspected adapter pin,
normal file precedence is standard user config, `.agents/mcp.json`, nested
`.agents/mcp/mcp.json`, Pi global override, opted-in ancestor standard/Pi configs,
project `.mcp.json`, project `.pi/mcp-adapter.json`. Imports are expanded within
active sources. Opted-in host discovery is a lower-precedence fallback. Package
MCP defaults are below agent-plugin defaults, which are below explicit config;
Claude-plugin defaults fill otherwise absent entries. Inventory every active
source, not only the final winning value. Exclusive `PI_MCP_CONFIG_MODE` uses the
selected global override plus imports and configured Claude-plugin defaults;
it skips normal package/agent-plugin/host discovery and project sources. D refuses
active unqualified plugin schemas/host discovery rather than executing them.
Exclusive mode is operator-owned, not a way to waive known conflicts; a local
managed destination inconsistent with it refuses. See [readiness](references/readiness.md).

Same normalized name in another active source conflicts even when identical.
Unreadable required sources, unknown precedence, symlink aliases or runtime config
path mismatch prevent managed-write readiness. Owner resolves them externally;
normal install must not adopt or overwrite them. Unknown runtime registrations
are a separate visible limitation, not invented static names or a success claim.

## Inert hardening example

[shared-mcp.example.json](references/shared-mcp.example.json) uses 3.0.0 keys
checked against the installed pin's `types.ts`, `config.ts` and consuming source;
[SOURCE.md](SOURCE.md) records hashes. This is source/schema validation, **not** a
runtime security qualification. Only the operator may preview and apply selected
settings after accounting for existing users; never replace a whole existing
config with the example. Template placement neither edits policy nor connects.

| Setting | Intended prerequisite |
|---|---|
| `allowInstall: false` | No agent install action persisting endpoints |
| `hostConfigDiscovery: "off"`, `ancestorConfigRoots: []` | No broad host/ancestor discovery |
| `projectServers: "ask"` | User-global policy; no blanket project-server auto-allow |
| `sampling: false`, `samplingAutoApprove: false` | No model sampling |
| `jev: false`, `scriptMode: false` | No external semantic search/script evaluation |
| `autoAuth: false`, `elicitation: false` | No automatic authentication/elicitation flow |
| `agentPluginPaths: []` | No plugin discovery from that setting |

These are not a complete environment sandbox. Inspect effective environment,
imports, `claudePlugins`, package defaults, higher-precedence settings and built-in
server defaults before connecting. No `requestHeadersCommand`, leading-`!` secret
command values, credentials/headers, connect-time `npx`/`uvx` or other resolver is
allowed in the selected shared definitions. There is no invented `disableSecrets`
key: absence must be checked across effective sources. Omitted sources do not
become disabled merely because the example omitted them. Do not log secret values.
Project policy can only be set in user-global config; a local copy is insufficient.

Recipes intentionally contain only URL leaves; policy belongs to the operator.
Changes inside a Patronus-owned server leaf require the normal reviewed recipe/
update ownership flow, not a manual edit disguised as unchanged ownership.
The server must enforce a fixed read/navigation or query-only tool surface where
supported. Modes and client filters are not an OS sandbox. Discover exact names;
never assume a wildcard filter, skill name or cached alias enforces the boundary.

## Manual role augmentation and private writers

- [Role overrides](references/role-overrides.md), RO-1 through RO-5: one owner per
  effective scope; inventory/preview/quiesce/record before manual changes; complete
  base tools and mandatory skill union, provider loading, reload and child smoke.
- [Role example](references/role-agentOverrides.example.json): inert field-level
  `subagents.agentOverrides` examples, not ready-to-apply settings. Substitute
  reviewed absolute extension paths before use. At the selected pins, retain the
  `mcp` gateway in role tools; discover underlying server/tool names through that
  gateway, not as direct role-tool substitutions. Read RO-2's query-only limits.
- [Writer bootstrap](references/writer-bootstrap.md), WB-1 through WB-4: execution
  grant, retained exact cwd allocation, RAM admission, separate interactive Pi
  resource/MCP approvals and child-local initialization. Native automatic worktrees
  remain source-read-only for symbols until prelaunch bootstrap is qualified.
- [Private MCP example](references/private-mcp.example.json) and
  [ownership descriptor](references/private-ownership.example.json): inert stdio
  definition and external operator evidence, not installer state or deletion grant.
- [Teardown](references/teardown.md), TD-1 through TD-4: actual settlement first,
  unchanged record-backed field restoration, unrelated-data preservation, reload
  and explicit research admission. Missing record/drift/unknown runs retain data.

C-class consistency only (no actual config writes or subprocess/server launch):

```sh
node --test artifacts/skills/code-intel-operations-pi/fixtures/manual-lifecycle.test.mjs
```

Run from a source checkout; the distributed fixture can also run by its installed
path. Its [independent models](fixtures/manual-lifecycle.test.mjs) are not shipped
lifecycle enforcement or runtime qualification. I-T3 bootstrap is integration-only;
I-T4/I-T6/I-T7 actual loading, restore and settlement observations remain QP-03.

## Failure, resources and qualification handoff

- [Failure matrix](references/failure-matrix.md), FM-1 through FM-4: one attempted
  readiness sequence, expected/observed identity and disclosed source fallback;
  bounded retries need owner budget. Pi trust and MCP approval failures remain
  distinct. No automatic install, service launch, refresh or project/mode switch.
- FM-3 defines the operator resource record: measured free/available RAM and
  cgroup headroom, reserve/outstanding peaks, independent concurrent/total/worker/
  deadline budgets. Low or unknown headroom blocks expensive work, not authorized
  bounded source reads. A concurrency counter is not reserve. Subprocess/provider
  graph work stays separately admitted and coordinator-only.
- [Qualification handoff](references/qualification.md), QUAL-1 through QUAL-4:
  exact-input evidence and all I-T1..I-T7 cases linked to OP-CODEINTEL, including
  base/enabled/restored roles, private cwd/approval/settlement, shared-client exit,
  interrupted reset and optional static removal. It separates generic D inverse
  tests, C example consistency and later actual Q runtime observations, and lists
  selected candidate gaps. Q owns the operational runbook and real execution.

The fixture adds C resource/admission and interrupted-recovery models, not shipped
policy or M/I proof. Operational release remains blocked until fresh selected
OP evidence, independent review and owner acceptance. No test double establishes
process settlement, shared-server survival, isolation or hard resource enforcement.

## Records and acceptance

- [Service descriptor](references/service-descriptor.example.json): one record per
  service with root/revision/dirty state, pin, owner, endpoint, transport, observed
  tools, restrictions and lifecycle/readiness evidence. Values are deliberately
  unverified; substitute reviewed observations, not invented success.
- [Snapshot provenance](references/snapshot-provenance.example.json): inventory,
  hashes, untracked treatment, exact build commands/mode/model/tool versions,
  exclusions and warnings. Copying a graph must retain its originating identity.
- [Readiness procedure](references/readiness.md): initialize, root verification,
  live definition/caller, graph query/provenance and effective restrictions.

Loopback is local trust, not tenant isolation; container/remote namespaces may
not reach it. No wildcard bind or remote auth design is authorized here. Missing
endpoint/provider/LSP, wrong root, stale aliases or malformed/historical graph
mean one attempted check then disclosed source reads/escalation. No install,
refresh, process launch, project switch or silent mode/model/protocol fallback.

Profile/item removal owns only unchanged Patronus static files and recipe leaves
under D's inverse/drift rules. It does not kill services, remove operator binaries,
credentials or caches, delete worktrees/outputs, or erase manual role overrides.
Prototype aliases/configs remain externally owned. Settle dependent work and
acknowledge client reload before separately authorized lifecycle changes. Unknown
ownership/process state is retained and escalated, never treated as cleanup grant.

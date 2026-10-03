# Manual role augmentation (operator-owned)

These numbered steps are a checklist, not shipped enforcement. The inert
[role example](role-agentOverrides.example.json) shows complete field replacements
under supported `subagents.agentOverrides.<role>` in **effective settings.json**.
Select only the granted roles/fields; never copy the whole file over existing
settings. No agent inheritance, Markdown rewrite, runtime helper or new per-launch
`tools`/`extensions` field is supplied. Call-level skills do not widen strict tools.
Base role context, output, acceptance, no-nesting and write-scope limits stay intact.
Bash and mode guidance are not an OS sandbox.

## RO-1 — Inventory and claim one scope

Record one operator owner and a matching stage/action/root/files grant. Inventory
active user-global settings in the effective agent directory (`PI_CODING_AGENT_DIR`
or the default Pi agent directory), exact project/worktree `.pi/settings.json`,
role source definitions, user/project `agentOverrides`, provider-scoped overrides,
per-run selections, extension defaults, required runtime extensions and inherited
capability ceilings (`excludeTools`, denied tools/extensions included). Verify the
selected pins' actual resolution; do not assume destination wins. At subagents
0.72.1 project settings override user fields; lists replace, not concatenate.
Provider-scoped overrides and call-level replacements can supersede the plan.

Refuse competing owners, higher-precedence definitions for selected fields,
unknown precedence, unreadable/malformed sources and path/symlink aliases. An equal
conflicting definition is still not this owner's field. Do not widen a ceiling to
make a tool available. Global changes must identify **all affected clients** and
owners; unknown consumers block. Writers always use their exact allocated worktree
scope, never a global switch between concurrent writers. Research/spec/review/plan
roles refuse execution-configured worktrees pending [TD-1 through TD-4](teardown.md).

## RO-2 — Preview complete role plans

The example includes the base tools/skills for each delivered C role and adds
`pattern-mcp-pi`, `graphify-pi`, `code-intel-operations-pi`. It also preserves the web
role's mandatory web tools/skill/provider. Reconcile with the actual base definition
on every update; do not use this example as a permanent role inventory oracle.
The selected route adds the adapter's **`mcp` gateway** to each complete base tool
list, not individual registered MCP tool names. Keep its explicit child extension
below. Discover underlying server/tool names and schemas through the gateway;
those names are call arguments and exact server filters, NOT replacements for
`mcp` in the role list. Researchers/authors/reviewers/planners select shared
read/navigation Serena plus query-only Graphify. The writer selects its approved
private Serena server; Graphify still describes the **main snapshot**.

At subagents 0.72.1, `src/agents/agents.js:splitToolList` creates direct selections
only for `mcp:` entries. `src/runs/shared/child-launch.js:childProcessEnv` otherwise
sets `MCP_DIRECT_TOOLS=__none__`; adapter 3.0.0 `index.ts:resolveCurrentDirectTools`
suppresses direct registration even if a server says `directTools:true`.
`index.ts:registerProxyTool`/`syncProxyTool` still supplies `mcp` when direct tools
are absent. Thus raw registered names in role tools cannot work at these pins.
Do not blindly prefix them with `mcp:`: subagents' direct selector resolver reads
legacy `mcp.json` paths, not the delivered `mcp-adapter.json` source set. This route
uses the adapter's qualified effective config through the gateway, with no legacy
config copy, environment override, resolver patch or capability-ceiling bypass.
Changed pins require renewed source review and runtime qualification.

Before granting the gateway, inventory **every** reachable effective server and
its exposed tool surface. Require only approved query/navigation operations, with
server-side restrictions where supported and reviewed exact nonempty `includeTools`
filters; an empty list is not a deny-all list. No wildcards, admin, build, edits,
project/mode switching or unapproved servers. Preserve operator/Patronus ownership
rules: do not silently edit managed shared leaves to add filters. If the approved
surface cannot be established, hold MCP use and report source-read fallback.
The private candidate uses `directTools:false`; this disables direct exposure,
not proxy calls. Reload and observe allowed and denied calls before dispatch.

Authorized leaf operations are server-scoped lexical search, describe and calls
to the approved discovered query tools, within task budgets. For example, after
substituting reviewed server/name values (not executing these placeholders):

```json
{"server":"<approved-server>","search":"symbol","searchMode":"lexical","limit":5}
{"server":"<approved-server>","describe":"<discovered-query-tool>"}
{"server":"<approved-server>","tool":"<discovered-query-tool>","args":{}}
```

Use the tool's discovered argument schema, not the illustrative empty args.
Initialize the selected Serena before symbols; use the same server-scoped route
for the approved Graphify query. Implicit connection is allowed only to the exact
already-approved definition under the bootstrap grant. No leaf gateway install,
auth, configuration or service-management action, semantic search or script mode
is granted. The gateway itself advertises broader operations and can reach other
configured servers: these usage limits are **cooperative**, not a per-call policy
interceptor or OS sandbox. Hardening and negative runtime observations remain
mandatory; `mcp` in a tools list alone proves neither security nor readiness.

The explicit `extensions` list **disables ambient extension discovery**: substitute
reviewed absolute paths for ALL required model-provider and runtime extensions,
not just the illustrated slots. The web role also needs its qualified web provider.
`subagentOnlyExtensions` loads the reviewed MCP adapter into that role's child;
retain any other required child-only extensions when replacing that list. Both
lists replace prior lists. Empty lists are not an inheritance shortcut. Provider
paths, tool names, effective MCP source definitions and selected skills require
inspection/smoke evidence, not assumptions from a frontmatter name. The child must
actually register `mcp` with the approved config and server restrictions. Keep the
approved model/provider/protocol; no fallback or model pin is implied.

Preview only selected field changes: `tools`, `skills`, `extensions`,
`subagentOnlyExtensions`. Save expected original absence versus null versus value,
full original file bytes and SHA-256 in a protected operator evidence location,
plus planned values/digests and all affected consumers. Do not put secrets in logs
or source control. A call-level skill replacement must contain the full C-role plus
three-code-intel-skill union; a missing mandatory skill blocks launch. Include the
technical reviewer's full dual-review skill union, not just its selected rubric.

## RO-3 — Quiesce and record before mutation

Settle all affected runs/clients successfully; cancellation, an empty status list
or a missing row is not settlement. Record outputs, run/session/descendant identity
and shutdown evidence. Unknown/active/failed consumers retain configuration.
Re-read the full source inventory and selected fields immediately before writing.
A competing editor or changed managed field/digest blocks; no blind overwrite.

Use the [ownership descriptor](private-ownership.example.json) record pattern for
settings too: separate record per config file, field path such as
`["subagents", "agentOverrides", "<selected-role>", "tools"]`, original presence/
value, desired managed value/digest, original bytes, root/run/owner and matching
hashed grant. This is external operator evidence, **not installer state** and not
a generic trust ledger. Grants follow the deployment's QP-01 record format:
issuer, action, roots, exact input/output hashes, expiry or revalidation trigger,
and owner verification binding the same scope. The record is not authorization.
No marker alone confers deletion rights. Example nulls/placeholders are unverified.

For digests, hash original file **bytes** exactly; for each JSON field, use UTF-8
compact JSON with object keys recursively sorted, arrays ordered, no trailing
newline, and SHA-256. Record that algorithm alongside local evidence. Distinguish
`original.present:false` from `original.present:true,value:null`. Record the
preview's input hashes and desired digest before the manual change. Retain the
first baseline on same-owner updates; append history of prior/new managed digests,
grants, reload and effective-plan observations. Never make a prior managed value
the new original. New fields need their own baseline before first mutation.

## RO-4 — Apply selected fields manually and verify reload

Only after RO-1 through RO-3, manually change the approved fields, preserving all
unrelated entries. Save resulting bytes/digests. Reload/restart every affected
session; acknowledge actual reload, effective role definition, complete tool,
extension/provider and skill plan. Read selected skill contents explicitly;
discovery/inheritance is not reading. Missing tools/provider/skills or a ceiling
conflict is a held launch, not permission for a builtin or model/protocol fallback.

MCP-dependent launches require qualified **background native sessions** at the
selected pin; foreground MCP rejection is an infrastructure failure, not a reason
to silently switch launch mode. Before fanout, coordinator runs one separately
authorized child capability smoke: Serena initialization and expected root, known
definition/caller, Graphify query with provenance, and effective restrictions,
all through server-scoped `mcp`. Observe the gateway under the normal child
`MCP_DIRECT_TOOLS=__none__` environment; do not change that variable to repair it.
Include disallowed-tool/server and management-action boundary observations without
claiming the cooperative instructions technically block arbitrary gateway callers.
Unloaded/unknown tools block MCP use. Source reads may continue only under the
task's existing allowance and with the limitation reported. Exact schema/effect,
base/enabled/restored child behavior and loading remain QP-03 integration evidence.

## RO-5 — Update, restore and uninstall boundary

Same-owner updates repeat RO-1 through RO-4 with the original baseline carried
forward. Restore through [TD-1 through TD-4](teardown.md) only after successful
settlement, unchanged recorded fields and a matching reset grant. Restore original
values (including null); delete only fields originally absent. Preserve unrelated
settings, even those added after augmentation. Empty parent objects may remain;
never delete a containing object/file just because the selected field is absent.

Profile uninstall removes Patronus-owned artifacts only. Manual settings survive
until explicit operator cleanup; plan cleanup before removing required skills.
Retain the original backup and digest history after uninstall or an interrupted
reset. Noncooperating editor races remain a manual-procedure limitation; rereads
and ownership conventions are not a lock or hard enforcement.

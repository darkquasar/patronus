# Private-writer bootstrap (manual, separately authorized)

This procedure delivers no bootstrap mechanism and performs no approval or launch.
Use local source reads by default. Private MCP examples are inert and must never
be copied into active config without reviewed substitution and the following gates.

## WB-1 — Bind the grant, exact worktree and budget

Require an execution grant and a **coordinator-retained exact cwd/worktree**
allocation with one writer owner. Record canonical cwd/root, repository identity,
named base ref/full commit, current full commit/branch, owner, task, run/session,
server identity and allowed files/actions. Confirm root is the allocated worktree,
not main or a symlink alias. The coordinator retains the allocation through
settlement/reset; the child neither allocates nor removes it. Research authority
is not execution authority. Native automatic worktrees use source reads until a
separately qualified exact-path prelaunch bootstrap exists; no new automatic
bootstrap is planned and a postlaunch hook is not a substitute.

Before any approved server/LSP startup measure free AND available RAM and container
headroom; unknown values block expensive admission. This deployment requires both
free and available >=750 MiB and >=500 MB reserve **after** the estimated additional
private server/LSP load. Record concurrent writers, process/deadline budgets and
observation time; a concurrency count alone is not reserve evidence. Stop/escalate
when the budget cannot be established; source reads remain available.

## WB-2 — Preview the exact private definition and ownership

Read [shared readiness](readiness.md) and [RO-1 through RO-3](role-overrides.md).
Inventory all effective MCP sources/imports, settings overrides, package/plugin
defaults, qualified exclusive-config mode and ceilings. Known normalized-name
collision in another source, higher precedence, unvalidated selected static data,
path escape or competing owner blocks. Dynamic/runtime-only registration is
visibly unverified, not a blanket extension-deactivation requirement. Do not run
npm/git resolvers or import extensions during this inspection.

[private-mcp.example.json](private-mcp.example.json) selects one stdio server leaf
in the exact worktree `.pi/mcp-adapter.json`. Replace
`/ABSOLUTE/APPROVED/PROJECT`, executable/cwd and tool-name placeholders with
reviewed absolute local paths and discovered names. `cwd` and `--project` must
identify that same canonical allocation. The command must be an already
provisioned pinned local executable, never `npx`, `uvx`, a shell bootstrap or a
connect-time resolver. Verify the selected Serena pin's CLI/context/mode arguments
before launch; this candidate `ide`/`editing` invocation is not universal support.
Modes are guidance, not security. The example restricts MCP tools to navigation;
source edits still use the writer's granted native tools.

The adapter 3.0.0 `ServerEntry` surface supports `command`, `args`, `cwd`,
`inheritEnv`, `directTools` and `includeTools` (source inspection, not runtime
qualification). Set `directTools:false` for this gateway route. Server filters
use exact discovered server-tool names; the role tool list retains `mcp`, NOT
registered direct-tool names or `mcp:` selectors. Apply RO-2's selected-pin route
and inspect all effective server surfaces, including shared Graphify. Nonempty
`includeTools` is required; an empty list allows all tools at this pin. No
wildcard/admin tools, legacy config copy or child environment override.
`inheritEnv:false` is not an OS sandbox: SDK platform defaults still apply. Review
any narrowly required language-tool environment separately; do not log secrets,
inherit an entire home/environment or put credential/header commands in config.
Operator-owned global hardening from the shared runbook still applies; a project
file cannot supply global project-server approval policy.

Prepare [private-ownership.example.json](private-ownership.example.json) outside
installer state, in a protected operator evidence root. Record original full file
bytes/base64 and byte hash (or explicit file absence), each selected field's
original presence/value, exact substituted managed definition/hash, root/run,
matching hashed execution and reset grants, inventory and unknown readiness.
Use RO-3's digest convention. Retain restricted backups, never commit secrets.
For an existing file, `originalFile.present:true` requires real bytes/hash; null
is not a backup. A prior null leaf is present, not absent. Same-owner updates keep
the first baseline and append history. Never replace an entire config file or
unrelated server entries with this example. Re-read, quiesce and refuse drift
before the operator manually writes only the approved leaf and role fields.

## WB-3 — Interactive preapproval before headless dispatch

In a trusted interactive Pi session at that **exact** worktree, the operator must
preapprove Pi project resources AND the exact MCP server definition through their
supported independent paths. Record both decisions and evidence. A Pi trust flag,
project resource approval or descriptor alone is not MCP approval. No global
auto-allow, approval bypass or invocation-specific bypass is assumed. Setup hooks
neither prompt nor approve. Changed root/definition/executable invalidates prior
approval until revalidated. If interactive approval is unavailable, do not launch
the private service; report source-read fallback.

After the separately authorized interactive connect/check, close its client and
prove its private runtime/LSP settlement before dispatch to avoid duplicate LSPs.
Reload approved config/role plans and verify full tools/skills/provider loading as
RO-4 requires. Coordinator dispatches a qualified background native child with the
retained exact cwd, existing context/output/acceptance contract and run identity;
no invented per-launch tools/extensions fields and no provider/model fallback.

## WB-4 — Child proves locality before symbols

Verify the explicit child adapter registers the `mcp` gateway under its normal
no-direct-tools environment. Use server-scoped lexical discovery/describe and
calls through `mcp` to the approved private Serena; raw direct names in the role
allowlist are not a working substitute. Call its discovered `initial_instructions`
before symbols, and compare its initialized canonical project root to actual cwd
and the retained record. Observe a representative known definition and caller.
Wrong root stops symbol use; never retarget a shared service to repair it. Missing,
unapproved or unavailable local service means disclosed **local source reads**,
never shared-main symbols labeled worktree-local. Do not have the child start a
service or change modes as recovery. Shared Graphify remains a labeled main
snapshot; verify important edges against changed local source, never refresh it.

Record expected/observed identities, actual tool plan, evidence hashes and limits.
I-T3 is **integration-only**: the independent fixture model cannot prove approval,
exact cwd loading, native child tool registration or shutdown. QP-03 must observe
these under its own grant; static template delivery is placed, runtime-unverified.
Follow [TD-1 through TD-4](teardown.md) before reset, research or removal.

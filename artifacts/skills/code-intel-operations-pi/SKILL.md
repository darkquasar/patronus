---
name: code-intel-operations-pi
description: Shared Serena and Graphify runtime delivery, coordinator lifecycle, role exposure, failure handling and qualification for Pi code intelligence.
---

# Pi code-intelligence operations

`code-intel-pi` is an overlay on `core-profile-pi`. It keeps every core role
definition and workflow selection, then adds the pinned Pi adapter, shared HTTP
MCP wiring and narrow field-level role overrides. Its companion `code-intel-pi-runtime` profile delivers
Serena and Graphify through uv. The profiles are separate because a Pi-targeted
static selection intentionally refuses package-manager EXEC intent. Installation
places declarations and can install packages when the operator grants package
execution. It does not start services, build a graph, approve project resources or
prove runtime readiness.

## Delivered components

| Item | Pin | Owner after deployment |
|---|---|---|
| `pi-mcp-adapter` | `npm:pi-mcp-adapter@3.0.0` | Pi/npm |
| `serena-runtime-pi` | Serena commit `7a2968335f2198b966864de1ce3655c8e485a653` | uv |
| `graphify-runtime-pi` | `graphifyy[mcp]==0.9.31` | uv |
| `serena-shared-pi` | `http://127.0.0.1:9121/mcp` | Patronus config leaf |
| `graphify-shared-pi` | `http://127.0.0.1:9122/mcp` | Patronus config leaf |

The adapter is a Pi package, not a copied extension file. Child roles leave their
`extensions` and `subagentOnlyExtensions` overrides unset, so Pi's normal package
discovery remains active. Because `tools` is a strict pi-subagents allowlist,
adding `mcp` also makes it a required child tool. Every overlaid role must run as
a background child with `async: true`; foreground children do not load ambient
extensions and fail before their first model turn. This is a deliberate
fail-closed constraint, not a source-read fallback.

The overlay changes only each core role's `tools` and `skills` leaves. Removal can
therefore restore those leaves without replacing the role definition or containing
`agentOverrides` object. These fields are complete replacement lists, not additive
patches. Any core role tool or skill change requires a matching overlay update and
SemVer bump; catalogue tests compare each list with current core frontmatter.

Install and review `code-intel-pi-runtime` without a target first. Then install
`code-intel-pi` with `--target pi`. Use `--deploy --allow-package-installs` only
after approving each plan. The delivered executables are prerequisites, not
activation. Pi/npm and uv retain their native package lifecycle. Patronus must not
delete their internal files directly.

## One shared service pair

The coordinator owns exactly one Serena process and one Graphify MCP process for
the selected root. Follow [shared service lifecycle](references/shared-services.md).
Serena starts with an explicit project, loopback Streamable HTTP transport and a
non-editing planning mode. Graphify starts against one explicit, provenance-bound
`graph.json` using its HTTP transport. Both bind only to loopback by default.

```
  +-------------+   HTTP /mcp   +-------------------+
  | coordinator | ============> | Serena :9121     |
  +-------------+               +-------------------+
          |       HTTP /mcp      +-------------------+
          +====================> | Graphify :9122   |
                                  +-------------------+
          |
          | native child launch
          v
  +-------------+   mcp gateway  +-------------------+
  | core roles  | =============> | shared pair only |
  +-------------+                +-------------------+
```

Children are query clients only. They never run `serena`, `graphify`,
`graphify-mcp`, `uv`, `uvx`, package installers, graph builders, watchers or
service stop/restart commands. Once a background child has successfully loaded
the adapter, a missing or wrong shared endpoint produces a reported source-read
fallback or a blocked MCP-dependent outcome. A missing adapter or foreground
launch is an infrastructure failure before the child starts. Neither case produces
a child-owned replacement.

## Pi configuration and core-role integration

The wire recipes merge only these named URL leaves:

- global: effective `PI_CODING_AGENT_DIR/mcp-adapter.json`, otherwise
  `~/.pi/agent/mcp-adapter.json`
- local: `<workspace>/.pi/mcp-adapter.json`

They do not write `.mcp.json`, Pi's built-in `mcp.json`, credentials, headers or
adapter-wide policy. The adapter's full effective-source precedence still applies.
Inventory all active sources and imports before deployment. A same normalized
name in another active source conflicts even when its value is identical.
Malformed, unreadable or unsafe selected sources block the whole managed change.
Readiness must also reject every active stdio or command-based Serena/Graphify
entry, alias or unresolved import that could create a child-private process. The
inert hardening example is not enforcement by itself.

Sixteen setting artifacts augment the eight roles inherited from
`core-profile-pi`: one `tools` leaf and one `skills` leaf per role. Tool lists keep
the complete core list and add only `mcp`. Skill lists keep the complete core list
and add `pattern-mcp-pi`, `graphify-pi` and this skill. The web role keeps all four
web tools and `web-research-pi`. The technical reviewer keeps its verification skill and selects review rubrics by task mode.
See [role integration](references/role-overrides.md). A completed readiness record
plus explicit task authority satisfies the core-role clause requiring separately
qualified configuration before optional MCP use.

At the selected pins, roles use the generic `mcp` gateway. Do not replace it with
raw server tool names or legacy `mcp:<server>` selectors. Discover the server and
tool schemas through the gateway. Keep direct tool registration disabled for this
route. The role allowlist is cooperative capability selection, not an operating
system sandbox or a server-side authorization layer.

Writers may query the shared Graphify snapshot and may query Serena only when its
reported root and revision match the claim being checked. A shared main-root
Serena never represents an unmerged writer worktree. Writers use local source
reads for worktree-local symbols. This profile does not create private Serena or
Graphify processes for writers.

## Adapter hardening and readiness

[shared-mcp.example.json](references/shared-mcp.example.json) is an inert policy
example, not an installer input. Preserve unrelated policy and review effective
imports before applying any values. The intended baseline disables connect-time
installation, host and ancestor discovery, sampling, auto-auth, elicitation,
external semantic execution and script mode. No leading-`!` secret commands,
request-header commands, `npx`, `uvx` or other connect-time resolvers belong in
the selected shared definitions.

Use [readiness](references/readiness.md) after package installation, service
startup and Pi reload. A pass requires:

1. exact package and executable identities;
2. MCP initialization for both shared entries;
3. Serena initial instructions and canonical root match;
4. one current definition and caller checked against source;
5. one Graphify query bound to completed snapshot provenance;
6. observed query-only or navigation-only child use; and
7. a cold background core-role smoke with the effective `mcp` gateway.

A listening socket, installed package, config leaf or tool count does not satisfy
readiness. Pi trust, MCP approval, task authority and service ownership are
independent gates.

## Failures, resources and teardown

Follow [failure and resource procedures](references/failure-matrix.md). Default to
one readiness attempt. Record expected and observed identity, endpoint, root,
revision, tool schema and error. Missing endpoint, provider, LSP, graph, approval
or matching root does not authorize repair, retargeting or a private server.

Graph construction and language-server startup can be expensive. The coordinator
measures free and available RAM plus cgroup headroom, reserves 500 MB, and admits
an expensive action only with at least 750 MiB headroom. Children never perform
indexing or service lifecycle work even when resources are available.

Follow [teardown](references/teardown.md). Settle every client before stopping the
shared pair. Profile removal owns only unchanged Patronus role-setting leaves,
MCP URL leaves and authored resources. It does not itself prove service exit,
remove uv/npm package internals, delete graphs, credentials, caches, logs, outputs
or worktrees. Package-manager removal and data deletion need their own reviewed
plans and authority.

## Qualification boundary

The adjacent fixture checks delivered document and example consistency only:

```sh
node --test artifacts/skills/code-intel-operations-pi/fixtures/manual-lifecycle.test.mjs
```

It does not execute Pi, import the adapter, install packages, start services or
qualify runtime behavior. [Qualification handoff](references/qualification.md)
keeps actual cold-child, lifecycle, graph provenance, failure and removal evidence
separate. Candidate dossiers still withhold full dependency closure, provenance,
advisory, currency and platform claims. Static catalogue success must remain
labeled `placed, runtime-unverified` until those observations exist.

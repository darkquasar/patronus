# MCP Server Catalog

> **Purpose:** Registry of remote MCP (Model Context Protocol) servers for use with Claude Code and other MCP clients. Only includes servers with verified remote HTTP/SSE endpoints — no local stdio-only servers.
>
> **Setup guide:** See [`claude/mcp-server-configuration.md`](../claude/mcp-server-configuration.md) for how to add, authenticate, and manage these servers in Claude Code.

---

## Quick setup

```bash
# Add any server from this catalog
claude mcp add --transport http <name> <url>

# Then authenticate via /mcp inside Claude Code
```

---

## Cloudflare

Cloudflare splits its MCP surface into separate servers per product area for security through precise permission scoping. All use Cloudflare's own OAuth (`https://dash.cloudflare.com/oauth2/auth`) with PKCE.

- **Source:** [github.com/cloudflare/mcp-server-cloudflare](https://github.com/cloudflare/mcp-server-cloudflare)
- **Docs:** [developers.cloudflare.com/agents/model-context-protocol/mcp-servers-for-cloudflare](https://developers.cloudflare.com/agents/model-context-protocol/mcp-servers-for-cloudflare/)
- **Blog:** [Thirteen new MCP servers from Cloudflare](https://blog.cloudflare.com/thirteen-new-mcp-servers-from-cloudflare/)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `cloudflare-docs` | `https://docs.mcp.cloudflare.com/mcp` | None | Search Cloudflare documentation |
| `cloudflare-workers-bindings` | `https://bindings.mcp.cloudflare.com/mcp` | OAuth | Manage Workers, KV, R2, D1, Hyperdrive |
| `cloudflare-observability` | `https://observability.mcp.cloudflare.com/mcp` | OAuth | Query Workers Logs and metrics |
| `cloudflare-builds` | `https://builds.mcp.cloudflare.com/mcp` | OAuth | View and debug Workers Builds (CI/CD) |
| `cloudflare-ai-gateway` | `https://ai-gateway.mcp.cloudflare.com/mcp` | OAuth | Manage AI Gateway, view logs |
| `cloudflare-containers` | `https://containers.mcp.cloudflare.com/mcp` | OAuth | Manage container lifecycle, exec commands |
| `cloudflare-logpush` | `https://logs.mcp.cloudflare.com/mcp` | OAuth | Manage Logpush jobs |
| `cloudflare-radar` | `https://radar.mcp.cloudflare.com/mcp` | OAuth | Internet traffic analytics and insights |
| `cloudflare-browser-rendering` | `https://browser-rendering.mcp.cloudflare.com/mcp` | OAuth | Browser automation and rendering |
| `cloudflare-autorag` | `https://autorag.mcp.cloudflare.com/mcp` | OAuth | AutoRAG management |
| `cloudflare-audit-logs` | `https://audit-logs.mcp.cloudflare.com/mcp` | OAuth | Query account audit logs |

**Note:** Most servers require calling `accounts_list` then `set_active_account` before other tools become usable.

---

## Axiom

Observability platform for logs, traces, and metrics. Query data using APL (Axiom Processing Language).

- **Docs:** [axiom.co/docs/console/intelligence/mcp-server](https://axiom.co/docs/console/intelligence/mcp-server)
- **Free tier:** 500 GB/month ingest, 30-day retention

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `axiom` | `https://mcp.axiom.co/mcp` | OAuth | Query datasets, list dashboards, check monitors |

---

## GitHub

GitHub's official MCP server for Copilot-integrated workflows.

- **Docs:** [docs.github.com/en/copilot/concepts/context/mcp](https://docs.github.com/en/copilot/concepts/context/mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `github` | `https://api.githubcopilot.com/mcp` | OAuth | Repos, issues, PRs, code search, actions |

---

## Atlassian

Jira, Confluence, and Compass access via Rovo MCP.

- **Docs:** [support.atlassian.com/atlassian-rovo-mcp-server](https://support.atlassian.com/atlassian-rovo-mcp-server/)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `atlassian` | `https://mcp.atlassian.com/v1/mcp` | OAuth | Jira issues, Confluence pages, Compass services |

**Note:** The legacy `/v1/sse` endpoint is still supported but Atlassian recommends using `/v1/mcp` instead.

---

## Sentry

Error tracking and performance monitoring.

- **Docs:** [docs.sentry.io/product/sentry-mcp](https://docs.sentry.io/product/sentry-mcp/)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `sentry` | `https://mcp.sentry.dev/mcp` | OAuth | Query errors, issues, performance data |

---

## Stripe

Payment processing APIs.

- **Docs:** [docs.stripe.com/mcp](https://docs.stripe.com/mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `stripe` | `https://mcp.stripe.com` | OAuth | Manage payments, customers, subscriptions |

---

## Linear

Project management and issue tracking.

- **Docs:** [linear.app/docs/mcp](https://linear.app/docs/mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `linear` | `https://mcp.linear.app/mcp` | OAuth | Issues, projects, cycles, teams |

---

## Notion

Workspace and documentation platform.

- **Docs:** [developers.notion.com/docs/mcp](https://developers.notion.com/docs/mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `notion` | `https://mcp.notion.com/sse` | OAuth | Pages, databases, search |

**Note:** Uses SSE transport (`--transport sse`), not HTTP.

---

## Vercel

Frontend cloud and deployment platform.

- **Docs:** [vercel.com/docs/mcp/vercel-mcp](https://vercel.com/docs/mcp/vercel-mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `vercel` | `https://mcp.vercel.com` | OAuth | Projects, deployments, domains, logs |

---

## Supabase

Open-source Firebase alternative (Postgres, Auth, Storage, Edge Functions).

- **Docs:** [supabase.com/docs/guides/getting-started/mcp](https://supabase.com/docs/guides/getting-started/mcp)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `supabase` | `https://mcp.supabase.com/mcp` | OAuth | Database, auth, storage, edge functions |

---

## Neon

Serverless Postgres.

- **Docs:** [neon.com/docs/ai/neon-mcp-server](https://neon.com/docs/ai/neon-mcp-server)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `neon` | `https://mcp.neon.tech` | OAuth | Databases, branches, queries, migrations |

---

## Hugging Face

ML model hub and inference platform.

- **Docs:** [huggingface.co/docs/hub/en/hf-mcp-server](https://huggingface.co/docs/hub/en/hf-mcp-server)

| Name | URL | Auth | Description |
|------|-----|------|-------------|
| `huggingface` | `https://mcp.huggingface.co` | OAuth | Models, datasets, spaces, inference |

---

## Directories

For discovering additional MCP servers:

- **Official MCP Registry:** [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io)
- **PulseMCP:** [pulsemcp.com/servers](https://www.pulsemcp.com/servers) — community directory with 8,600+ servers
- **Remote MCP Servers:** [remote-mcp-servers.com](https://remote-mcp-servers.com) — curated remote-only list

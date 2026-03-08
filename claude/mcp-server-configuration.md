# Claude Code MCP Server Configuration Guide

> **Purpose:** Definitive guide for configuring remote MCP (Model Context Protocol) servers in Claude Code. Covers transport selection, authentication, scoping, and common failure modes. Born from hard-won debugging across multiple sessions.

---

## Core rule

**Use Claude Code's native HTTP transport for remote MCP servers. Do not use `mcp-remote` as an intermediary.**

```bash
# Correct — native HTTP transport
claude mcp add --transport http <name> <url>

# Wrong — stdio wrapper via mcp-remote
# DO NOT DO THIS:
# Add entry to ~/.claude/.mcp.json with "command": "npx", "args": ["-y", "mcp-remote", "<url>"]
```

---

## Why not mcp-remote?

`mcp-remote` is a stdio-to-HTTP bridge that many MCP docs recommend. It introduces multiple failure modes that Claude Code's native HTTP transport avoids entirely:

1. **Build-version mismatch bug:** The compiled bundle can hardcode a stale version string (e.g., bundle says `0.1.37` while npm reports `0.1.38`), causing OAuth tokens to cache under the wrong directory and never be found on subsequent launches.
2. **npx caching bug:** `npx` has a known, long-standing issue where it doesn't always fetch the latest version of a package, leading to stale binaries.
3. **Extra process overhead:** Each `mcp-remote` instance spawns a Node.js process that acts as a proxy between Claude Code's stdio and the remote HTTP server — unnecessary when Claude Code speaks HTTP natively.
4. **Token storage mismatch:** `mcp-remote` stores OAuth tokens in `~/.mcp-auth/mcp-remote-{version}/`. Claude Code's native transport stores tokens internally. The two systems don't share state.

---

## Adding servers

### Via CLI (recommended)

```bash
claude mcp add --transport http <server-name> <server-url>
```

This is the canonical method. It writes the config to `~/.claude.json` under the appropriate scope.

### Scopes

| Flag | Scope | Storage location | Use case |
|------|-------|-----------------|----------|
| `--scope local` (default) | Per-project per-user | `~/.claude.json` → `projects["/path/to/project"].mcpServers` | Project-specific servers (most common) |
| `--scope user` | All projects for this user | `~/.claude.json` → `mcpServers` | Servers you want everywhere (e.g., documentation search) |
| `--scope project` | Shared with team via repo | `.mcp.json` in repo root | Servers the whole team should use (committed to git) |

### Important: config file roles

| File | Purpose | Server types |
|------|---------|-------------|
| `~/.claude.json` | User and project-scoped config | HTTP, SSE (via CLI) |
| `~/.claude/.mcp.json` | Legacy / stdio servers | stdio (command-based) only |

**Do not manually write `"type": "http"` entries into `~/.claude/.mcp.json`.** They will be ignored. Always use `claude mcp add --transport http`.

---

## Authentication (OAuth)

Most remote MCP servers (Cloudflare, Axiom, GitHub, etc.) use OAuth 2.1 with PKCE for authentication.

### First-time auth flow

1. Add the server: `claude mcp add --transport http <name> <url>`
2. Restart Claude Code (Ctrl+C → `claude --resume` or fresh launch)
3. Run `/mcp` inside Claude Code — servers needing auth will show `needs-auth`
4. Select the server to authenticate — a browser tab opens for OAuth consent
5. Complete the auth in the browser
6. The server should connect. If it shows "reconnection failed", restart Claude Code once more — the token is cached and will work on next launch.

### Subsequent sessions

Cached tokens are reused automatically. No re-auth needed unless tokens expire and refresh fails.

### Checking status

Run `/mcp` at any time to see:
- Which servers are connected
- Which need authentication
- Which failed to connect (with error details)

---

## Removing servers

```bash
claude mcp remove <server-name>
```

This removes from the default scope (`local`). To remove from a specific scope:

```bash
claude mcp remove <server-name> --scope user
```

---

## Lifecycle rules

1. **MCP servers load at session startup.** Adding or removing servers mid-session requires a restart.
2. **`/mcp` can reconnect servers** that failed during startup or need auth, but first-time OAuth sometimes requires a full restart to take effect.
3. **`claude --resume`** preserves your conversation context across restarts.
4. **Auth one server at a time** if multiple servers need OAuth — authenticating them all simultaneously can cause browser tab confusion.

---

## Server catalog

For a catalog of known MCP servers organized by vendor, see [`mcp/README.md`](../mcp/README.md).

---

## Troubleshooting

### "No MCP servers configured"
- Check that servers were added with `claude mcp add`, not by hand-editing config files
- Verify the project path matches: servers added with `--scope local` only appear when Claude Code is launched from that project directory

### Server shows "needs-auth" repeatedly
- Complete the OAuth flow via `/mcp`
- Restart Claude Code after authenticating
- If it persists, remove and re-add the server: `claude mcp remove <name> && claude mcp add --transport http <name> <url>`

### Server connected but tools not available
- Some servers require you to set an active account first (e.g., `accounts_list` → `set_active_account`)
- Run `/mcp` to verify the connection is healthy

### Duplicate server configs causing conflicts
- Check for duplicates across scopes: a server defined in both `--scope local` and `--scope user` can cause unpredictable behavior
- Use `claude mcp list` to see all configured servers
- Remove duplicates from the scope you don't want

---

## Anti-patterns

| Don't | Do instead | Why |
|-------|-----------|-----|
| Use `npx -y mcp-remote` in config | `claude mcp add --transport http` | Avoids version mismatch bugs, npx caching issues, extra process overhead |
| Hand-edit `~/.claude/.mcp.json` for HTTP servers | Use the CLI | `~/.claude/.mcp.json` is for stdio servers only; HTTP entries are ignored |
| Authenticate all servers simultaneously | Auth one at a time via `/mcp` | Multiple concurrent browser OAuth tabs cause confusion |
| Assume servers reconnect after mid-session auth | Restart Claude Code after first auth | MCP servers load at startup; first-time tokens often need a restart |
| Use `--scope user` for project-specific servers | Use `--scope local` (default) | Keeps server configs scoped to the projects that need them |

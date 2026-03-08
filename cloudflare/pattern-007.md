# Pattern 007: MCP Containers for Remote Debugging and Troubleshooting

## Pattern header

- **Pattern ID:** 007
- **Name:** MCP Containers — Ephemeral Sandboxes for Live Service Debugging
- **Status:** Draft
- **Scope:** Remote troubleshooting, wire protocol inspection, network-level diagnostics against deployed Workers
- **Primary goal:** Use Cloudflare MCP Container instances as disposable sandboxed environments to debug deployed Workers — inspect exact wire formats, eliminate browser-side variables, and iterate on diagnostic scripts without polluting local state.
- **Non-goals:**
  - Using containers as production compute (they are ephemeral and session-scoped)
  - Replacing local development tooling for routine work
  - Long-running services or background jobs (containers are destroyed after session)
- **Key Cloudflare products involved:** MCP Server (cloudflare-containers), Workers (as the target being debugged)
- **Primary constraints to respect:**
  - Containers are ephemeral — all state is lost when the session ends
  - The container runs Ubuntu 20.04 with Node.js, Python 3, and standard build tools pre-installed
  - Additional packages can be installed but must be re-installed each session
  - The container has internet access and can reach any deployed Worker
- **Related patterns**
  - **See also:** Pattern 005 (Testing Architecture) — containers complement local Vitest tests by enabling network-level debugging against deployed services that Vitest cannot reach

---

## Executive summary

Cloudflare's MCP Container server provides ephemeral sandboxed Linux environments accessible via Claude Code's tool system. When debugging deployed Workers — especially WebSocket protocols, streaming responses, or cross-service interactions — local tools often fall short: browsers cache aggressively, curl cannot speak WebSocket, and Node.js clients may behave differently depending on local environment. MCP Containers provide a clean, neutral environment with internet access where diagnostic scripts run against live endpoints and capture exact wire-level output. This is particularly valuable when the protocol between client and server is the unknown — the container acts as a controlled probe that eliminates browser-side and local-environment variables.

---

## Context and forces

### Debugging deployed services is different from debugging locally

- Local `wrangler dev` may not reproduce issues that only appear in production (Durable Object hibernation, real DNS, TLS, edge routing).
- Browser DevTools show WebSocket frames but require manual inspection and cannot be automated or scripted.
- `curl` handles HTTP well but cannot establish WebSocket connections or speak custom protocols.

### Browser-side variables obscure server-side issues

- Browsers cache aggressively — even hard-refresh may serve stale local files or maintain stale WebSocket connections.
- Browser WebSocket APIs have implicit behaviors (auto-reconnection, CORS preflight, cookie attachment) that make it hard to isolate whether the problem is client or server.
- When debugging a streaming protocol, you need to see exactly what bytes arrive in what order — browser rendering logic obscures this.

### Local Node.js has its own quirks

- Local network conditions (firewalls, DNS, proxies) can differ from cloud-to-cloud communication.
- Installing debug dependencies locally can pollute the project's `node_modules` or conflict with workspace dependencies.
- The `ws` library in Node.js may receive HTTP 400 from `routeAgentRequest` due to missing upgrade headers — a container reproduces the same behavior consistently without local environment variables.

### Containers provide neutral ground

- A clean Ubuntu environment with no cached state, no browser quirks, no local network interference.
- Scripts can be written, executed, and iterated on within the same Claude Code session.
- Output is captured and returned directly for analysis — no manual copy-paste from DevTools.
- Dependencies installed in the container never affect the project.

---

## Decision triggers

- If your deployed Worker uses **WebSocket protocols** and you need to inspect the exact frame sequence → **use this pattern**.
- If the bug only reproduces **against the deployed service** (not `wrangler dev`) → **use this pattern**.
- If you need to **eliminate browser-side variables** (caching, JS client behavior, rendering) to determine whether the problem is client or server → **use this pattern**.
- If you need to **install temporary debug tools** without polluting your project → **use this pattern**.
- If you're debugging **HTTP-only endpoints** that `curl` can handle → **don't use this pattern** — use curl or existing test scripts instead.
- If the bug is in **local development** → **don't use this pattern** — use Vitest or local debugging tools.

---

## Solution (architecture in words)

### The debugging workflow

The container serves as a controlled probe between Claude Code and the deployed Worker:

1. **Initialize** the container (creates a fresh Ubuntu sandbox with internet access)
2. **Install** any needed diagnostic libraries (e.g., a WebSocket client) into a scratch directory to avoid project dependency conflicts
3. **Write** a diagnostic script that connects to the deployed service and logs raw wire data — typeof, byte length, raw content, parsed structure
4. **Execute** the script with a timeout — output streams back to Claude Code
5. **Analyze** the captured output to identify the mismatch between what the server sends and what the client expects
6. **Iterate** if needed — modify the script and re-run without re-installing dependencies

### Three container tools and their roles

- **`container_exec`** — Run shell commands, capture stdout and stderr. This is the primary tool for both setup and execution.
- **`container_file_write`** — Write files into the container filesystem. Alternative to heredoc for longer scripts.
- **`container_file_read`** — Read files from the container. Useful for inspecting output files or installed package structures.

### What the container reveals that browsers cannot

The key value is seeing the **exact wire format** before any client-side interpretation:

- Whether response chunks are top-level JSON objects or nested inside a wrapper envelope
- The exact field names on each chunk type (e.g., `delta` vs `textDelta`)
- The ordering and count of messages in a streaming sequence
- Whether a message body is a JSON string that requires double-parsing or a direct object
- HTTP status codes, headers, and TLS handshake details for non-WebSocket debugging

---

## Invariants (must always hold)

- **Ephemeral by design:** Never store secrets, credentials, or persistent state in the container. It will be destroyed. Pass sensitive values as inline environment variables if needed.
- **Diagnostic only:** The container observes and probes — it should not mutate production state unless that is the explicit intent (e.g., testing a write endpoint).
- **Install in scratch directories:** When installing npm packages, use `/tmp` or another scratch path — not the container's working directory, which may inherit a `package.json` with workspace dependencies that will fail outside a monorepo.
- **Always set timeouts:** Every diagnostic script must include a self-terminating timeout. Every `container_exec` call should include a timeout parameter. A hung connection without a timeout blocks the entire debugging session.
- **Capture raw before parsing:** Log raw type, byte length, and string content before calling JSON.parse — the whole point is to see what is actually on the wire, not what you assume it is.
- **Import from absolute paths:** When packages are installed in `/tmp`, import them using the full absolute path to the module entry point — relative imports will fail.

---

## Implementation guidance (LLM-facing)

### Container lifecycle

- **Do** call `container_initialize` before any other container operation.
- **Do** call `container_initialize` again if you encounter connection errors — this restarts the container.
- **Avoid** assuming the container persists between separate conversations — it does not.
- **Because** containers are session-scoped and may be evicted between conversations.

### Dependency installation

- **Do** install packages in a scratch directory (e.g., `/tmp`) to avoid workspace `package.json` conflicts.
- **Do** use ES module syntax (`.mjs` extension) when writing diagnostic scripts that use `import`.
- **Avoid** running `npm install` in the container's default working directory — it may contain a `package.json` with `workspace:*` dependencies that fail outside the monorepo.
- **Because** the container's working directory may inherit project files that cause install failures.

### Writing diagnostic scripts

- **Do** use heredoc syntax via `container_exec` for scripts — write the file and execute it as separate steps.
- **Do** include a self-terminating timeout in every script (e.g., close the connection and exit after 30 seconds).
- **Do** log structured output: message count, raw type, raw length, parsed keys, and inner structure for nested payloads.
- **Do** enable stderr capture (`streamStderr: true`) on all exec calls — many diagnostics (Node.js warnings, TLS errors) go to stderr.
- **Avoid** scripts that run indefinitely or rely on graceful shutdown signals.
- **Because** the goal is maximum observability with minimum assumptions about what the server will send.

### WebSocket protocol debugging

This is the most common use case. The approach is:

1. Connect to the Worker's WebSocket endpoint using the correct path (agent namespace must be kebab-case for partyserver)
2. Wait briefly for initialization messages (identity, MCP state) — log each one
3. Send a test payload matching the expected client protocol
4. Log every received message: raw string, parsed outer structure, and if the body field is itself a JSON string, parse and log the inner structure too
5. Close after a timeout and report the total message count

The critical diagnostic output is the **relationship between the outer wrapper and inner payload** — this is what browser clients typically get wrong when they handle only one layer of the protocol.

### HTTP debugging

For non-WebSocket debugging, `curl` is pre-installed in the container and is often sufficient. Use verbose mode to see headers, TLS handshake details, and response timing. The container's network path (cloud-to-cloud) may also reveal latency or routing differences compared to local requests.

### Analyzing output

After the script runs, look for:

- **Message wrapping:** Is the payload at the top level or nested inside a wrapper envelope? If nested, is the body field a string (requires JSON.parse) or an object?
- **Field names:** Are they what the client expects? Common mismatches: `delta` vs `textDelta`, `done` vs `finish`.
- **Message ordering:** Do lifecycle events arrive in the expected sequence?
- **Missing messages:** If expected chunks (e.g., text deltas) are absent, the server may be returning empty content or erroring silently.
- **Error fields:** Look for error flags or error-type messages in the stream.

---

## Failure modes and mitigations

| Failure | Impact | Mitigation |
|---------|--------|------------|
| Container evicted mid-session | Script output lost | Re-run `container_initialize`; keep scripts short and re-runnable |
| npm install fails due to workspace deps | Cannot install debug packages | Install in `/tmp`, not the working directory |
| Script hangs (no timeout) | `container_exec` blocks indefinitely | Always include a timeout in scripts and on exec calls |
| WebSocket gets HTTP 400 | Cannot connect to Worker DO | Verify agent namespace is kebab-case (partyserver convention); check path format |
| Container cannot resolve DNS | Scripts fail to connect | Verify Worker URL; use `dig` to check DNS resolution from within the container |
| Large output truncated | Miss critical debug info | Write output to a file in the container, then read specific sections with `container_file_read` |

---

## Observability and operations

- **Container health:** Use `container_ping` to verify the container is alive before running scripts.
- **Stderr is essential:** Always enable stderr capture — TLS errors, deprecation warnings, and unhandled promise rejections go to stderr, not stdout.
- **Iterative debugging:** The container persists within a session. Install once, run many scripts. Modify and re-run without re-installing.
- **Clean exit:** Scripts should explicitly exit after completing — do not leave connections open or the container in an ambiguous state.

---

## Anti-patterns to avoid

- **"Debug by guessing in the client"** — If you don't know what the server sends, don't guess and iterate on the client. Probe the server directly from a controlled environment first.
- **"Install in the working directory"** — Running `npm install` in the container's default directory can fail due to workspace dependency resolution. Always use a scratch directory.
- **"Skip the timeout"** — A script without a timeout will hang forever, blocking the debugging session with no output.
- **"Store secrets in the container"** — API keys, tokens, or credentials should be passed as inline environment variables, never written to container files that persist (even briefly).
- **"Use containers for things curl can do"** — If you're debugging a simple HTTP endpoint, curl is faster and simpler. Containers are for when you need a full runtime: WebSocket, streaming, custom protocol parsing.
- **"Assume the container persists across sessions"** — It does not. Re-initialize and re-install at the start of each new conversation.
- **"Run one mega-script"** — Break diagnostics into small, focused scripts. One script per unknown. This makes output easier to analyze and re-runs faster.

---

## Acceptance checklist

- [ ] Container is initialized before any exec calls
- [ ] Dependencies installed in a scratch directory (not the working directory)
- [ ] All scripts include a self-terminating timeout
- [ ] Stderr capture is enabled on all exec calls
- [ ] Raw wire data is logged before any parsing or interpretation
- [ ] Script output identifies the specific unknown being investigated
- [ ] No secrets or credentials are persisted in the container filesystem
- [ ] Findings are applied back to the actual client or server code

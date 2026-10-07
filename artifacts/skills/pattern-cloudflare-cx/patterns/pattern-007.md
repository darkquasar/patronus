# Pattern 007: Remote protocol diagnosis, local-first

## Codex local use and handoff

This is design guidance, not a live platform action grant. Read
`{skillDir}/SKILL.md`, `{skillDir}/patterns/pattern-005.md` for test fidelity,
`{skillsDir}/diagnosing-bugs-cx/SKILL.md` for the reproduction loop and
`{skillsDir}/pattern-mcp-cx/SKILL.md` for any separately approved MCP capability.
A leaf never delegates. The lead owns shared outputs and stage decisions.
Static placement is not observed native Codex or Cloudflare behavior.

## Purpose and triggers

Use when a streaming/WebSocket client misreads message wrapping, field names or
ordering, or a deployed-only symptom cannot be reproduced locally. A neutral
probe can separate browser cache/rendering from server wire behavior. Start with
sanitized captured traces and local replay, not automatic container creation.
A local fixture cannot establish real DNS, TLS or deployed Durable Object behavior.

## Local evidence first

1. Identify the exact symptom, boundary, expected protocol and allowed data.
2. Replay a sanitized capture through the real parser/handler seam. Capture type,
   byte length, envelope keys, ordering and terminal status before interpretation.
3. Compare outer and inner payloads. An object and a JSON-encoded string require
   different parsing. Check lifecycle messages and missing chunks explicitly.
4. Use a pinned timeout, maximum messages/output bytes and deterministic fixture.
   Await asynchronous work and close connections. Never log credentials or raw
   personal payloads. Retain failed evidence within the approved output binding.
5. Turn the confirmed mismatch into a local regression, then run the configured
   checks. Separate the observed local result from the unresolved deployed claim.

## Optional remote probe prerequisites

Live web, endpoint, auth, service/platform calls, package installation and
container creation each need separate explicit authority and qualified tools.
Record exact endpoint/environment, allowed read/write behavior, egress/data
policy, resource/spend limits, maximum attempts, timeout, outputs and lifecycle
owner first. Do not infer availability from the legacy provider's tool names.
No container API is assumed to exist on Codex. Missing capability or auth blocks
without installing tools, restarting services or changing provider/protocol.

Prefer already available tools; installation is not part of this pattern. Never
transfer credentials from another CLI or put secrets in command arguments, logs
or scratch files. An approved secret channel remains the owner's responsibility.
"Ephemeral" does not mean safe: retention, network reachability and isolation
must be qualified for the selected environment, not assumed from source claims.

An authorized probe should observe, not mutate production. Testing writes needs
explicit selected-target consent and idempotency controls. Set a process deadline,
close sockets and confirm settlement. On interruption or unknown container state,
stop, retain evidence and ask the lead. Cleanup, recreation and retries need their
recorded grants; failure is not permission to restart or install dependencies.

## Failure modes

| Symptom | Evidence to seek, within approved scope |
|---------|----------------------------------------|
| Nested JSON or double parsing | raw type plus outer/inner envelope keys |
| WebSocket upgrade rejected | expected path/namespace and sanitized status |
| Missing chunks | message count, ordering and terminal lifecycle event |
| Local success but deployed failure | disclose environment gap, request authorized capture |
| Truncated output | preserve bounded log file and read exact relevant sections |
| Hung connection | bounded stop and settlement result, retain unknown work |

## Acceptance and handoff

Report the unknown investigated, source/capture hashes, exact commands/exits,
case names, sanitized output, timeout/settlement and residual environment limits.
Distinguish local reproduction, remote observation and production correction.
A proposed probe or placed pattern is not a passed native test. Integration,
publication, production deployment and cleanup remain separate grants.

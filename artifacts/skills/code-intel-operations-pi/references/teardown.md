# Shared-service settlement and overlay removal

Installation and removal do not silently control service processes. Use these
steps under explicit lifecycle and package-manager grants.

## TD-1: settle every consumer

Account for all parent sessions, background children, external clients, outputs
and descendants. Require terminal status and readable hash-bound results. Active,
unknown or failed consumers retain the shared pair and block update or shutdown.
Closing an HTTP client does not stop Serena, Graphify or Serena's language-server
descendants.

## TD-2: identify owned state

Inventory the exact profile lock/state, each role tools/skills setting leaf, the
two MCP URL leaves, Pi package declaration, uv package records, service process
records, logs and graph provenance. Compare Patronus-owned leaves with their last
managed values. Drift blocks inverse for that leaf. Do not adopt or erase same-name
entries from another source.

Patronus owns authored resources and unchanged config leaves. Pi/npm owns the
adapter package files. uv owns Serena and Graphify tool environments. The
coordinator owns service processes and runtime records. Snapshot and cache owners
remain as recorded by the task.

## TD-3: restore config and remove packages separately

Preview normal Patronus removal first. It restores or deletes only the unchanged
managed role and MCP leaves, preserving sibling role fields, unrelated server
entries and settings added later. Apply after review, then reload Pi and verify the
base core role definitions no longer request `mcp` or overlay skills.

Package removal is a separate operation. Delegate the adapter removal to Pi and
Serena/Graphify removal to uv only after all consumers are settled and the package
plan names the exact selected identities. Never delete npm or uv internal
directories directly. A package removal does not delete credentials, logs, graphs
or project files.

## TD-4: stop services and retain evidence

Stop each service through its recorded supervisor identity after all clients have
settled. Do not kill by guessed PID or port. Verify process start identity, exit,
Serena language-server descendants and endpoint closure. Unknown descendants or a
PID identity mismatch retain the process record and block cleanup.

Preserve install plans, exact package observations, service descriptors, logs,
snapshot provenance, outputs and removal results according to the owner retention
policy. Graph deletion, cache deletion, worktree cleanup and log deletion each
need separate authority. Profile removal alone grants none of them.

The adjacent fixture checks that this contract remains present in distributed
content. It cannot observe real process settlement, package removal, reload or
retention.

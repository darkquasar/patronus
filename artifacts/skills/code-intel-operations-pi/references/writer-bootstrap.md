# Writer use of shared code intelligence

The profile does not bootstrap a private MCP process in a writer worktree. The
writer receives the same coordinator-owned HTTP entries as every other core role.
This avoids one Serena and Graphify process per subagent.

## WB-1: bind the writer allocation

Record the writer's exact cwd/worktree, base and head revisions, owner, run and
allowed files. Compare that identity with the shared Serena root and Graphify
snapshot provenance before using either result.

## WB-2: classify shared evidence correctly

Shared Graphify is a labeled snapshot of the coordinator-selected root. It can
answer architecture and where-consumed questions, but every consequential edge
that intersects writer changes must be checked against local source.

Shared Serena may be used only for claims about the root and revision it reports.
If the writer's worktree differs, Serena results are main-root navigation hints,
not worktree-local symbol evidence. Use `read` and `bash` within the granted
worktree for local definitions and callers.

## WB-3: forbid child lifecycle recovery

A writer never starts `serena`, `graphify`, `graphify-mcp`, `uv` or `uvx`; never
activates another Serena project; and never rebuilds Graphify. Missing tools,
wrong root, stale graph or a transport failure is reported once under FM-1. The
coordinator decides whether to repair shared infrastructure under a separate
grant. The writer continues with source reads only when the implementation grant
permits that fallback.

## WB-4: settle the query client

The child closes its client as part of normal run settlement. That must not stop
the shared services. Before the coordinator stops or updates them, every writer
and other consumer must be settled and all outputs retained. Worktree cleanup is
independent of shared-service cleanup and still needs its normal owner authority.

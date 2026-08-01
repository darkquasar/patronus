# Query the knowledge graph first

This project can be mapped into a queryable knowledge graph by graphify
(`graphify-out/graph.json`). When it exists, consult it before reading files
one by one or grepping raw source.

- **Ask a scoped question:** `graphify query "how does auth get wired?"` returns
  a focused subgraph rather than a wall of files.
- **Trace a connection:** `graphify path "FastAPI" "ModelField"` shows how two
  concepts connect, hop by hop.
- **Explain one node:** `graphify explain "APIRouter"` lists its connections,
  each tagged `EXTRACTED` (explicit in the source) or `INFERRED` (resolved by
  graphify).
- **Broad architecture review:** read `graphify-out/GRAPH_REPORT.md`.

Build or refresh the graph with `graphify .`. If no graph exists yet, fall back
to normal reading — the graph is an accelerator, not a gate.

# Query, path and explain: child read-only procedure

1. Read the service descriptor and snapshot provenance supplied by the coordinator.
   Match canonical root, revision, graph digest, tracked dirty state and relevant
   untracked coverage to the question. If they differ, label the graph historical;
   use current source for changed regions. Do not refresh it.
2. Discover the effective adapter query tool names and schemas. Use approved
   read-only MCP operations; do not assume upstream CLI names are MCP aliases.
   A separately qualified, already installed CLI may be used only when the task
   permits subprocess queries against the identified snapshot. No resolver,
   interpreter auto-detection/install or alternate provider fallback is implied.
3. Match question terms to actual node labels/vocabulary. Substring matching can
   miss synonyms, morphology and other languages. Choose at most 12 terms actually
   observed in the graph, record the expansion, and report no match honestly.
   Never invent vocabulary or write a vocabulary sidecar to the shared checkout.
4. Use bounded breadth-first traversal for nearby context, depth-first for a
   dependency chain, path for two identified nodes, or explain for one node.
   Set the supported output/depth limit explicitly; report truncation. Output
   character/token estimates do not bound server CPU, network or provider cost.
5. Cite node IDs, edge direction/relation, confidence class and source locations.
   Check the live definition/caller or current file for every consequential edge,
   including EXTRACTED edges. Undirected paths do not establish call direction;
   collapsed or missing edges make absence claims unsound.
6. Keep the result in the assigned report only. Do not call save-result, reflect,
   rebuild or any cache-writing flow. Corrections go to the coordinator with
   snapshot identity and source evidence, not a silent edit to the shared graph.

Missing tools, malformed graphs and insufficient coverage permit disclosed source
reads, not automatic repair. One readiness attempt is the default; further retry
requires coordinator authority and budget. Neither a query response nor successful
initialization alone establishes all service readiness conditions.

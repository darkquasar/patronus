# Interpreting extraction evidence

This is an adapted evidence reference, **not** a prompt authorizing a child
extraction team. Graph construction and semantic/provider work are coordinator-only
and require explicit stage and resource/model budget approval.

- EXTRACTED denotes a source-explicit relation (for example import, call or
  citation). It can still be stale, incomplete or structurally misinterpreted.
- INFERRED denotes a heuristic or semantic relationship, not an observed call.
- AMBIGUOUS flags uncertainty; do not silently omit it or upgrade its confidence.
- Confidence scores are extractor annotations, not measured correctness. Verify
  consequential EXTRACTED and INFERRED edges in current source.

When consuming a node, retain its stable ID, label, repository/source path,
location and available capture/author provenance. For an edge, retain source,
target, relation, confidence class/score and source location. A calls edge points
from caller to callee; an undirected path alone does not establish this direction.
Semantic similarity and community membership do not prove runtime coupling.

Coordinator validation of an admitted build must reject dangling endpoints,
source-ID collisions and accidental edge collapse, disclose missing/failed chunks,
and preserve provenance for hyperedges and merged repositories. Record the exact
extraction prompt/tool/model versions and hashes, including cache inputs. Do not
replace real usage/cost observations with placeholder zeros or describe excluded
documents as indexed. Children report defects without editing the shared graph.

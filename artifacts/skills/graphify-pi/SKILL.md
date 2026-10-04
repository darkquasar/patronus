---
name: graphify-pi
description: Query an existing coordinator-owned graph snapshot; verify architectural edges in current source without child indexing.
---

# Graphify snapshot queries for Pi

Use graph snapshots for architecture and file relationships; use initialized,
root-verified Serena for live definitions/callers. Neither replaces source checks.
Read `code-intel-operations-pi` for service readiness and provenance records.

## Query-only child contract

Discover the effective Graphify MCP query/path/explain schemas. Confirm endpoint,
source root, snapshot hash/revision, tracked dirty state, relevant untracked inputs
and exclusions before relying on results. Follow [query guidance](references/query.md).
The shared graph is the main-checkout snapshot even for worktree writers. Compare
against current local source; never describe it as indexing unmerged changes.

If no graph/tool is available, report the missing evidence and use source reads.
Do not install/upgrade, build, refresh, watch, clone, transcribe, export, start a
server, change a shared project or invoke a provider. No child writes vocabulary,
reflection, saved-answer, cache or graph files as a query side effect. Missing
nodes and absent edges are not proof of absent code. EXTRACTED and INFERRED are
different evidence classes; consequential edges in **both** need source checks.

## Reference inventory

All eight adapted references remain distributed, with their responsibilities made
explicit rather than retaining upstream automatic-build instructions:

- [Query/path/explain](references/query.md): bounded, read-only traversal.
- [Freshness/update](references/update.md): child diagnostics; coordinator-only refresh.
- [Hooks](references/hooks.md): no child hook installation or automatic rebuild.
- [Exports](references/exports.md): coordinator-only output/publication/server actions.
- [Add/watch](references/add-watch.md): coordinator-only ingestion and watchers.
- [Transcription](references/transcribe.md): coordinator-only media/provider work.
- [GitHub/merge](references/github-and-merge.md): coordinator-only acquisition/merge.
- [Extraction semantics](references/extraction-spec.md): evidence interpretation,
  not an extraction-subagent launch prompt.

The hidden `.graphify_version` is preserved byte-for-byte from the source skill
(`0.9.31`, no newline). It records adaptation provenance, **not** the version of
any connected server. [SOURCE.md](SOURCE.md) inventories all original files and
[NOTICE](NOTICE) preserves attribution. There is no bundled Graphify runtime.

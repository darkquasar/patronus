# Shared MCP readiness: operator/coordinator procedure

This is a separately authorized deployment check, not a static install action.
Use the inert [service descriptor](service-descriptor.example.json) for each
service; change implementation/entry/endpoint to the observed Graphify selection
for its record. Empty tool lists, null observations and placeholder strings are
unverified, not a valid readiness pass. Retain failed evidence without credentials.

1. **Select exact inputs.** Record Pi, adapter (candidate 3.0.0), server immutable
   versions/refs and language-tool versions, source root, repository commit,
   tracked dirty state/diff hash, untracked treatment and client namespace. Confirm
   one coordinator owner and scope/stage authority. Check RAM/free/available and
   container headroom against deployment reserve, concurrency and deadline policy
   before expensive work. Unknown headroom is not admission.
2. **Preview effective sources.** Use Patronus's D discovery/preflight and inspect
   qualified adapter resolution, without importing/executing extensions. Include
   normal user/`.agents`/Pi/ancestor/project sources, statically resolvable imports,
   package defaults and plugin declarations; record exclusive-mode and explicit
   config-path overrides and effective agent root. Confirm path/precedence agrees
   with the intended target. Same normalized names in any other active source
   conflict even if equal. Inventory every active MCP entry and reject stdio/command
   Serena or Graphify entries, alternate aliases, host-discovered entries and
   unresolved imports that could start a private copy in a child. Malformed or
   unreadable required sources and symlink aliases stop managed writes. Runtime-only
   sources remain visibly unverified, not a blanket extension-deactivation or
   `PI_OFFLINE` gate.
3. **Check hardening before connecting.** Operator previews the selected settings
   from [the inert example](shared-mcp.example.json), preserving unrelated policy.
   Check effective imports/environment and higher-precedence overrides: install
   action off, discovery off, project policy ask, sampling and external semantic
   search off, no broad auto-allow, secret/header commands or connect-time resolver.
   Inspect built-in/package defaults even when absent from a single config file.
   Do not print tokens/environment values. Project trust, MCP approval, tool
   approval and stage grant are distinct. None follows from static file placement.
4. **Reload and initialize.** After approved changes, acknowledge client reload/
   restart and discover effective server entries/tool names and schemas. Confirm
   `serena-shared-pi` and `graphify-shared-pi` map to the intended endpoints; do not
   substitute prototype aliases or infer readiness from cached metadata. Perform
   MCP initialize, then call Serena's discovered `initial_instructions` before
   symbol queries and follow its live manual. Compare canonical observed root with
   the expected root. Wrong root stops symbol use: never activate another shared
   project or change modes to make it pass. Socket-only success is insufficient.
5. **Observe live navigation.** Query a representative current definition and its
   callers with discovered `find_symbol`/`find_referencing_symbols` schemas. Verify
   locations and relationship in current source and record the actual observation.
   No caller result alone proves there are no callers; use a known representative
   relation for the readiness case. Shared symbols describe the main root, not a
   writer's unmerged worktree. Missing language support is an explicit limitation.
6. **Observe a graph query.** Bind the response to a completed
   [snapshot provenance record](snapshot-provenance.example.json). Require included
   inventory with relative paths/modes/sizes/hashes, root/revision, tracked dirty
   state, untracked policy/inventory, exact commands/cwd/mode/tool/model versions,
   provider policy, generated and protected served paths, identical pre/post-copy
   graph hashes, read-only served copy, graph time, exclusions, failed inputs and
   warnings. The service must read the protected digest-named copy, never mutable
   `graphify-out/graph.json`. Record origin when copied. Inspect a representative
   relation and verify consequential
   EXTRACTED and INFERRED edges in current source. A historical snapshot may aid
   navigation but fails a claim of current matching readiness. Missing graph or
   excluded docs cannot prove code absence. Children never refresh it.
7. **Verify effective restrictions and child access.** Record observed tool names
   and fixed server read/navigation versus Graphify query-only surfaces, then the
   actual client/role tools, skills, provider and extension plan. Mode labels alone
   do not enforce read-only. No admin, build or project-switch operation belongs
   in a child plan. Do not invoke a forbidden tool as a probe. Every overlaid role
   must launch with `async: true`: `mcp` is a strict required tool and foreground
   children cannot load ambient extensions. A qualified native background child
   capability smoke is needed before MCP-dependent fanout; frontmatter/skill names
   and extension inheritance alone prove nothing. Preserve
   core output, acceptance and no-nesting limits. Verify the profile-managed
   field-level tools and skills overrides against the inherited core definitions.
8. **Decide and retain.** Readiness passes only with MCP initialization, correct
   root, live definition/caller, graph query/matching provenance and effective
   restrictions, with timestamps/evidence and reload acknowledged. Missing endpoint,
   approval, provider/LSP, wrong root, stale alias, mismatched version or malformed
   graph fails the affected claim. Default is one attempt then disclosed source
   reads/escalation; retry needs explicit coordinator budget. A foreground or
   missing-adapter failure occurs before child fallback and remains infrastructure
   failure. Never silently install, switch protocol/model/mode, refresh or launch a
   replacement service.

HTTP teardown closes the client's connection only. Stopping/restarting a shared
service requires coordinator accounting for dependent clients. Profile removal
restores or removes only unchanged owned leaves and static files under D's inverse
rules. Unrelated manual overrides, services, credentials, caches, outputs and
worktrees survive.
Static catalog validation and invented mechanism fixtures do not qualify these
runtime observations; preserve that distinction in every acceptance record.

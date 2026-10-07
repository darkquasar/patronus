# Codex core unchanged-reuse audit

Source: `46247c13c3de38008c59995c5b1b697f78d3f9e5`. This is a static content audit,
not a native acceptance result. `reuse-audit.yaml` hashes every distributed file
of the reused artifacts and the entire selected recipe bodies. The historical catalog test
`TestCodexCoreReuseAuditMatchesBytes` rejected missing/stale audit bytes; current
structural checks belong to the catalog gate (`scripts/tests/catalog-contract.sh`). Changes
require a fresh body, sidecar, dependency and handoff audit, not a hash refresh
alone.

## Accepted unchanged

| Item | Entire audited surface | Rationale and residual |
|---|---|---|
| domain-modeling 1.0.1 | patronus.yaml, SKILL.md, ADR-FORMAT.md, CONTEXT-FORMAT.md, NOTICE | File-based glossary/ADR discipline; local sidecar links, no host APIs, requires or lane-specific skill handoffs. Source Git-grant instructions still control when files may be authored/committed. |
| ddd-distilled 1.0.0 | patronus.yaml, SKILL.md, NOTICE | Domain-modeling decision/trigger rules; no sidecars, dependencies or worker protocol. Preserve the LLM-derived, inspired-by-book caveat and MIT attribution. |
| refactoring-distilled 1.0.0 | patronus.yaml, SKILL.md, NOTICE | Behavior-preserving change rules; no sidecars, dependencies or workflow handoffs. Preserve the source-content caveat and MIT attribution. |
| context7 1.0.0 | recipes/context7.yaml | Wire-only hosted HTTP URL; Codex in tools; no package scripts, secrets, sidecars or workflow routes. Native initialization and optional key/rate limits remain pending. |
| github 1.0.0 | recipes/github.yaml | Wire-only hosted HTTP URL; Codex in tools; no secret copying or package execution. Authentication is operator-owned and unqualified. |
| gitleaks 1.0.0 | recipes/gitleaks.yaml | Target-agnostic fetch-only CLI with six explicit OS/arch assets and SHA256 pins. No hook activation. Pins were read, not downloaded/revalidated. Native/platform and commit-guard acceptance remain pending. |

The three recipes stay in one Codex profile. Pi's runtime split responds to its
static selection rejecting generic package-manager EXEC intent. These recipes
need no Codex-specific split on the inspected contract; the MCP recipes merge
config and gitleaks fetches a pinned binary. They establish neither credentials
nor native server/guard readiness.

## Rejected or deferred unchanged reuse

- `codebase-design`: all body/sidecars read. `DESIGN-IT-TWICE.md` explicitly uses
  the Claude Agent tool. Port the design route before reuse.
- `tdd`: all body/tests/mocking/refactoring sidecars read. Its `/codebase-design`
  handoff reaches that foreign tool call, so TDD and design need a coherent port.
- `diagnosing-bugs`: body, manifest, HITL script and NOTICE read. The final
  `/improve-codebase-architecture` handoff is absent from the catalog; the NOTICE
  acknowledges a no-op suggestion. A missing route is not a successful handoff.
- `using-git-worktrees`: entire body read. It can install dependencies, commit an
  ignore edit and fall back in place on sandbox failure. Those defaults do not
  preserve stage/allocation grants.
- `finishing-a-development-branch`: entire body read. It invokes `git pull`,
  infers cleanup ownership from directories and prunes worktrees. A scoped port
  must separate integration, publication and cleanup grants.
- `writing-editorial`: router and tier-2 handoff read. It carries PRESERVE through
  voice models and merge in `writing-like-me`, which is outside the selected
  closure. Full sidecar/host audit and Codex route adaptation are deferred.
- `writing-skills`: manifest inspected; it declares subagent-testing, Claude
  examples and upstream-only tool-mapping references. Full body/sidecar audit and
  a coherent Codex authoring route are deferred.
- `pattern-cloudflare`: manifest/index inspected; seven sidecars still need a
  complete audit. Multi-target metadata alone is insufficient.
- `ticket`: manifest inspected; requires tk and assumes committed `.tickets`.
  Work-graph integration is outside this foundation's port set.

These omissions remain Partial in the compatibility ledger. No source artifact
was renamed or edited, and no unexamined body was admitted just because a grep
found no foreign tool token.

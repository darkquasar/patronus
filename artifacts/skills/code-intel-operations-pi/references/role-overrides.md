# Declarative core-role augmentation

The overlay does not replace the role Markdown delivered by `core-profile-pi`.
It installs one setting artifact for the `tools` leaf and one for the `skills`
leaf of each inherited role. This keeps output, acceptance, context inheritance,
no-nesting and every unrelated override under its existing owner.

## RO-1: inventory one effective scope

Preview the exact global or project Pi settings target and all higher-precedence
sources. Record existing values for each selected leaf. A conflicting owner,
unreadable source, malformed settings, symlink alias or changed preview input
blocks the whole selected install. Do not copy a complete `agentOverrides` object
over user settings.

The managed paths are:

```text
subagents.agentOverrides.<core-role>.tools
subagents.agentOverrides.<core-role>.skills
```

There are eight `<core-role>` values: plan author, plan reviewer, researcher, spec
author, technical reviewer, web researcher, workflow security reviewer and
writer. Patronus's ordinary setting inverse records and restores each leaf.

## RO-2: verify complete replacement lists

Pi-subagents treats these arrays as replacements, not additions. Each delivered
value therefore repeats its role's complete core list and adds only the overlay
members.

- Every tools list adds `mcp`.
- Every skills list adds `pattern-mcp-pi`, `graphify-pi` and
  `code-intel-operations-pi`.
- The web role retains `web_search`, `fetch_content`, `get_search_content`,
  `source_check` and `web-research-pi`.
- The technical reviewer retains `spec-review-pi` and
  `requesting-code-review-pi`.
- Author and writer roles retain their existing write tools. Read-only roles do
  not gain write tools.

The overlay does not set `extensions` or `subagentOnlyExtensions`. Leaving those
leaves untouched preserves Pi package discovery and any separately owned provider
configuration. The exact `pi-mcp-adapter` package is a profile dependency. Since
`tools` is a strict allowlist, `mcp` is required and overlaid roles must run as
background children with `async: true`. A foreground child cannot load ambient
extensions and fails before its first model turn. A child that lacks `mcp` after
reload likewise fails preflight. Neither failure is a source-read fallback. Do not
add raw discovered tool names or legacy `mcp:` selectors as a repair.

These are frozen copies of the core lists at the overlay release. Every core role
tool or skill change requires a matching overlay update and SemVer bump. The
catalogue test compares current core frontmatter with each replacement list, but
an already installed older overlay still needs an ordinary previewed upgrade.

## RO-3: preserve the shared-only boundary

Before MCP-dependent dispatch, inspect every effective server reachable through
the gateway. Reject active stdio/command Serena or Graphify entries, alternate
aliases, host-discovered entries and unresolved imports that could start a private
copy in a child. The approved profile entries are `serena-shared-pi` and
`graphify-shared-pi`. Children may search, describe and call approved navigation
or query tools on those entries. They may not install, authenticate, reconfigure,
retarget, build, refresh, start or stop services.

The gateway can expose more operations than the role should use. Prompt and tool
lists are cooperative boundaries, not server authorization. Serena's planning
mode, Graphify's query surface, exact adapter policy and runtime negative tests
must supply the effective restriction evidence.

## RO-4: reload and smoke a cold background child

After deployment, restart or reload Pi. Inspect the effective role definition and
confirm the complete tools, skills, package extensions and provider plan. Run one
qualified background child smoke before fanout:

1. confirm the `mcp` gateway is registered;
2. initialize `serena-shared-pi` and verify the expected root;
3. query one known definition and caller;
4. query `graphify-shared-pi` and bind the result to snapshot provenance; and
5. verify a lifecycle or mutation operation is outside the child's grant.

Missing package, provider, skill or tool is a launch/infrastructure failure.
After successful background startup, a missing endpoint, approval or root stops
MCP use; source reads may continue only when the task permits that fallback.

## RO-5: update and remove without replacing siblings

A profile update re-previews each tools and skills leaf against the recorded prior
and current managed value. Drift blocks the affected inverse. Removal restores the
original leaf value or deletes the leaf when it was originally absent. It preserves
other role fields, sibling role overrides and settings added later.

Settle affected children and acknowledge reload before update or removal. Removing
role settings does not stop shared services, remove packages or delete snapshots.
Follow TD-1 through TD-4 for service settlement and retained data.

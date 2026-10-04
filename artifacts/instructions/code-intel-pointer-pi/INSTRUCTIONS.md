# Optional Pi code intelligence

Before code-intelligence work, read the installed `pattern-mcp-pi`, `graphify-pi`
and `code-intel-operations-pi` skills. `code-intel-pi` extends
`core-profile-pi`: it preserves the core role definitions and workflow
selections, installs the exact MCP adapter, wires one shared HTTP service pair,
and adds field-level `mcp` and skill overrides to the inherited roles. The separately installed
`code-intel-pi-runtime` companion profile delivers the pinned Serena and Graphify
executables without mixing package-manager execution into the static Pi plan.

Installation is still **placed, runtime-unverified** until package identity,
service startup, Pi reload, approvals and cold-child access are observed. Do not
infer readiness from an install plan or listening socket. The overlaid roles have
a strict `mcp` tool allowlist and must run with `async: true`; foreground launches
fail before the first model turn because they cannot load ambient extensions.

The coordinator owns one Serena service for the selected root and one Graphify
service for the selected snapshot. Subagents use those shared entries only. They
never start, stop, retarget, rebuild, refresh or replace either service, and never
run `uv`, `uvx`, `serena`, `graphify` or `graphify-mcp` as recovery.

Use the adapter's discovered `mcp` gateway and live schemas. Initialize Serena
through its discovered `initial_instructions` operation before symbol work and
verify its canonical root. Prefer symbol and reference queries for live
main-root definitions. Never activate a different shared project or mode.

Graphify is a labeled snapshot. Inspect its root, revision, dirty and untracked
treatment, coverage and graph hash. Verify consequential EXTRACTED and INFERRED
edges against current source. A missing node or graph cannot prove code absence.
Children never build or refresh it.

A writer may use shared results as main-root navigation. It must not represent
those results as evidence about an unmerged worktree unless the recorded service
root and revision match that worktree. Known file and line reads remain valid.
Unavailable or mismatched services mean disclosed source reads or a blocked
MCP-dependent outcome, not a child-owned server.

Role augmentation changes only `tools` and `skills` leaves. It retains the core
authority, output, acceptance, context and no-nesting limits. The complete lists
replace, rather than extend, those fields, so a core list change requires a matching
overlay release. A setting or skill name does not prove the adapter, provider or
service loaded. Inspect the complete effective role and run a qualified background
child smoke before MCP-dependent fanout. That readiness record plus explicit task
authority satisfies the base role's requirement for separately qualified optional
MCP configuration.

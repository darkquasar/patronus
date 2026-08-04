# Consult the code-intel tools first

This project can be navigated by two complementary code-intel tools. When they
are available, reach for them before searching or reading files one by one.

- **At session start, before the first search or read on a code file, invoke
  `mcp__serena__initial_instructions`** and follow that manual for symbol-level
  work. Serena delivers its own tool guidance server-side — read it there rather
  than from a copy that can drift.

Division of labour:

- **serena** — live, LSP-backed: find a symbol, its callers, its definition;
  rename across the codebase; anything that must reflect the code as it is right
  now.
- **graphify** — a snapshot knowledge graph: architecture, "where is X wired",
  "how do these connect", impact across the corpus. Build or refresh it with
  `graphify .`.

Neither replaces reading a known `file:line` — when you already have the anchor,
open it. If a tool is unavailable, fall back to normal reading; these are
accelerators, not gates.

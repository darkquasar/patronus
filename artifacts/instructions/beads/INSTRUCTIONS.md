# Work-graph discipline (Beads / `bd`)

This project uses **Beads** (`bd`) as a durable, git-native work-graph: the
canonical record of what work exists, what it depends on, and what is ready to do
next. Prefer it over keeping the plan only in conversation — `bd` survives across
sessions, context compaction, and hand-offs between agents, and it lives in the
repo so the work-graph travels with the code.

**Finding the binary.** Patronus installs `bd` to `~/.patronus/bin/bd`. That
directory may not be on your `PATH`, so a bare `bd` can fail with "command not
found". Resolve it once at the start of a session: if `command -v bd` succeeds,
use `bd` directly; otherwise use the full path `~/.patronus/bin/bd` (or, for
convenience, prepend `export PATH="$HOME/.patronus/bin:$PATH"`). The `bd …`
examples below mean "the resolved bd". If the binary is at neither location, fall
back to plain in-context tracking and note that `bd` is unavailable rather than
failing.

## When to use it

- **Multi-step or multi-session work** — anything that won't finish in one go, or
  that another agent/session may pick up. Capture it as issues, not as a mental list.
- **Discovered work** — when you find a follow-up, a bug, or a blocked path mid-task,
  record it as an issue and link the dependency instead of silently deferring it.
- **Skip it for trivial one-shots** — a single throwaway edit does not need a graph.

## The loop

1. **Init once per repo:** `bd init` (creates the local work-graph; commit it).
2. **Capture work as issues**, smallest useful unit first:
   - `bd create "Short imperative title" -p 1` — create an issue (priority 0–3).
   - `bd q "Quick capture"` — quick-capture, prints only the new id.
3. **Express order with dependencies**, don't encode it in prose:
   - `bd dep add <issue> <depends-on>` — `<issue>` is blocked until `<depends-on>` closes.
   - `bd dep tree <issue>` / `bd dep cycles` — inspect structure; keep it acyclic.
4. **Pull the ready set** — the work with no open blockers — instead of guessing:
   - `bd ready` — list issues that are open and unblocked. Start from here.
5. **Keep state honest as you go:**
   - `bd update <issue> ...` — change fields; `bd note <issue> "..."` — append a finding.
   - `bd close <issue>` when done (this can unblock dependents); `bd reopen` if it regresses.
6. **Review:** `bd status` for the database overview, `bd list` / `bd show <issue>` to inspect.

## Conventions

- **One issue = one verifiable outcome.** If you can't state how it's checked, it's
  probably an epic — split it and link children with `bd dep add`.
- **Link, don't narrate, blockers.** A dependency edge is checkable; a sentence in a
  description is not. Let `bd ready` compute what's next.
- **Close eagerly.** An open issue means unfinished work; closing it is what unblocks
  the next ready item. Don't leave done work open.
- **Commit the graph.** `bd` state is git-native — commit it with the work it tracks so
  the next session (or agent) resumes from the same picture.

Run `bd help` (or `bd <command> --help`) for the full, version-accurate flag set.

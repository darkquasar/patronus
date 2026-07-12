# Work-graph discipline (Ticket / `tk`)

This project uses **Ticket** (`tk`) as a durable, git-native work-graph: the
canonical record of what work exists, what it depends on, and what is ready to do
next. Prefer it over keeping the plan only in conversation — the graph survives
across sessions, context compaction, and hand-offs between agents.

The graph is **plain markdown in `.tickets/`**, one file per ticket, committed like
any other file. There is no database and no separate sync step: `git add .tickets/`
is how the work-graph travels with the code.

**Finding the binary.** Patronus installs `tk` to `~/.patronus/bin/tk`. That
directory may not be on your `PATH`, so a bare `tk` can fail with "command not
found". Resolve it once at the start of a session: if `command -v tk` succeeds, use
`tk` directly; otherwise use the full path `~/.patronus/bin/tk` (or, for
convenience, prepend `export PATH="$HOME/.patronus/bin:$PATH"`). The `tk …` examples
below mean "the resolved tk". If the binary is at neither location, fall back to
plain in-context tracking and note that `tk` is unavailable rather than failing.

`tk` is a POSIX shell script: it runs on Linux and macOS, not on native Windows.

## When to use it

- **Multi-step or multi-session work** — anything that won't finish in one go, or
  that another agent/session may pick up. Capture it as tickets, not as a mental list.
- **Discovered work** — when you find a follow-up, a bug, or a blocked path mid-task,
  record it as a ticket and link the dependency instead of silently deferring it.
- **Skip it for trivial one-shots** — a single throwaway edit does not need a graph.

## The loop

1. **No init step.** `.tickets/` is created on the first `tk create`. Commit it.
2. **Capture work as tickets**, smallest useful unit first:
   - `tk create "Short imperative title"` — creates a ticket and **prints its id**
     (e.g. `pat-a1b2`). Capture that id; you need it for the dependency edges.
3. **Express order with dependencies**, don't encode it in prose:
   - `tk dep <id> <depends-on-id>` — `<id>` is blocked until `<depends-on-id>` closes.
   - `tk dep tree <id>` — show the dependency tree.
   - `tk dep cycle` — find cycles; keep the graph acyclic.
4. **Pull the ready set** — the work with no open blockers — instead of guessing:
   - `tk ready` — tickets that are open and unblocked. **Start from here.**
   - `tk blocked` — what is waiting, and on what.
5. **Keep state honest as you go:**
   - `tk start <id>` — claim it (status → in_progress).
   - `tk add-note <id> "..."` — append a finding.
   - `tk close <id>` when done (this can unblock dependents); `tk reopen <id>` if it
     regresses.
6. **Review:** `tk ls` to list, `tk show <id>` to inspect one, `tk closed` for done work.

> **Careful: `tk status` is not a status report.** It is a *setter* —
> `tk status <id> <status>` changes one ticket's status. To see where the project
> stands, use `tk ready` + `tk blocked`.

## Conventions

- **One ticket = one verifiable outcome.** If you can't state how it's checked, it's
  too big — split it and link the pieces with `tk dep`.
- **Link, don't narrate, blockers.** A dependency edge is checkable; a sentence in a
  description is not. Let `tk ready` compute what's next.
- **Close eagerly.** An open ticket means unfinished work; closing it is what unblocks
  the next ready item. Don't leave done work open.
- **Commit the graph.** `.tickets/` is tracked — commit it with the work it tracks so
  the next session (or agent) resumes from the same picture.

Run `tk help` for the full, version-accurate command list.

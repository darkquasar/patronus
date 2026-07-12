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
2. **Capture work as tickets**, smallest useful unit first. **Use the full create
   surface** — the defaults will quietly cost you:

   ```sh
   tk create "Short imperative title" \
     -t task \
     -p 1 \
     --tags <concern> \
     --acceptance "The ONE check that closes this ticket" \
     -d "PLAN: <the exact file> → '<the exact section heading>'. What to build.
         Files expected to change: path/a.go, path/b.go" \
     --external-ref <the exact file the work is specified in>
   ```

   It prints the new ticket's id (e.g. `pat-a1b2`). **Capture that id** — you need it
   for the dependency edges.

   **Why each flag is not optional:**
   - **`-p` (priority, 0–4, 0 = highest).** **`tk ready` SORTS BY PRIORITY.** Leave it
     at the default and every ticket lands at `2` — so `tk ready` hands your work back
     in no meaningful order, and the one signal it exists to give you is dead.
   - **`--acceptance`.** "One ticket = one verifiable outcome" is the rule this
     instruction preaches. `--acceptance` is where that outcome is *recorded*. A rule
     with no field to hold it is a wish.
   - **`--tags <concern>`.** tk generates opaque ids (`pat-a1b2`); there is **no `--id`
     flag**. The concern — the thing a human or a teammate filters on (`tk ready -T
     <tag>`) — survives only in the tags.
   - **`-d` with the file list.** `tk query` reads **only frontmatter**, so no field
     holds a file list and none is machine-queryable. Writing the expected files into
     the description is what lets a lead check, by *reading*, that no two agents own
     the same file.
   - **`--external-ref` + a `PLAN:` line that RESOLVE.** A ticket whose reference an
     agent cannot follow is a ticket nobody can do. Point at **the file the work is
     specified in** — never at a directory holding several candidates — and name **the
     section, verbatim**, copied rather than retyped. If the target sits in a
     **gitignored** directory, **say so in the ticket**: a fresh clone gets a pointer to
     a path that is not there, and should ask rather than improvise. **A pointer you
     have not followed is a claim, not a reference.**
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
- **Epics group; only `tk dep` orders.** tk *does* have `-t epic` and `--parent` — but
  they are **grouping and display only**: `tk ready` and `tk blocked` read **only
  `deps`**, never `parent`. An epic never blocks and never unblocks anything, and
  (having no deps of its own) it will sit in `tk ready` looking actionable forever. So
  use `--parent`/`--tags` to group work, and use `tk dep` to order it. A dependency edge
  is checkable; a parent link is not.
- **Close eagerly.** An open ticket means unfinished work; closing it is what unblocks
  the next ready item. Don't leave done work open.
- **Commit the graph.** `.tickets/` is tracked — commit it with the work it tracks so
  the next session (or agent) resumes from the same picture.

Run `tk help` for the full, version-accurate command list.

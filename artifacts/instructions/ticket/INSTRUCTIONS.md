# Work-graph discipline (Ticket / `tk`)

Ticket (`tk`) is an available durable work-graph, not an automatic status owner.
At plan acceptance the coordinator records one authority in the existing brief
or plan. Qualified native state that actually owns delegated runs wins (Pi today),
even with a project tracker selected; tickets receive references only, not mirrored
mutable progress. Without such state use explicit owner/project selection, then
the existing tracker, or one durable plan-associated record if absent. UI todos
alone do not outrank a project tracker. Do not create another selection ledger.

The graph is plain Markdown in `.tickets/`, one file per ticket, without a database
or separate sync step. Inspect actual repository ignore/tracking policy before
staging it. Commit graph changes only when tracked by that policy and a commit is
authorized. Never force-add ignored tickets or change ignore rules to satisfy this
instruction. A local ignored graph is not promised to travel with a fresh clone.

**Finding the binary.** Patronus installs `tk` to `~/.patronus/bin/tk`. Resolve it
once: use `command -v tk` if available, otherwise inspect the installed path.
The examples mean the resolved binary. If neither exists, report it unavailable
and preserve findings in the existing handoff. Do not silently switch the selected
authority to in-context tracking or install a replacement. An optional tracker
failure does not invalidate product work or authorize unlimited retries.

`tk` is a POSIX shell script: it runs on Linux and macOS, not on native Windows.

## When to use it

- **Selected Ticket authority:** the coordinator seeds accepted plan work once,
  at useful outcome granularity. Do not expand every checklist into tickets/todos.
- **Discovered work:** the coordinator records authorized follow-ups in the selected
  authority. Leaves return findings and stop, without tracker mutations.
- **Unselected Ticket:** do not seed or update a competing graph merely because
  `.tickets/` or the binary exists. Reference native run state when applicable.
- **Skip it for trivial one-shots** — a single throwaway edit does not need a graph.

## The loop

1. **No init step.** When Ticket is selected and graph writes are authorized,
   `.tickets/` is created on the first `tk create`. Tracking policy and separate
   commit authority govern whether it is staged or committed.
2. **Capture work as tickets**, smallest useful unit first. **Use the full create surface** — the defaults will quietly cost you:

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

   It prints the new ticket's id (e.g. `pat-a1b2`). **Capture that id** — you need it for the dependency edges.

   **Why each flag is not optional:**
   - **`-p` (priority, 0–4, 0 = highest).** **`tk ready` SORTS BY PRIORITY.** Leave it at the default and every ticket lands at `2` — so `tk ready` hands your work back in no meaningful order, and the one signal it exists to give you is dead.
   - **`--acceptance`.** "One ticket = one verifiable outcome" is the rule this instruction preaches. `--acceptance` is where that outcome is *recorded*. A rule with no field to hold it is a wish.
   - **`--tags <concern>`.** tk generates opaque ids (`pat-a1b2`); there is **no `--id` flag**. The concern — the thing a human or a teammate filters on (`tk ready -T <tag>`) — survives only in the tags.
   - **`-d` with the file list.** `tk query` reads **only frontmatter**, so no field holds a file list and none is machine-queryable. Writing the expected files into the description is what lets a lead check, by *reading*, that no two agents own the same file.
   - **`--external-ref` + a `PLAN:` line that RESOLVE.** A ticket whose reference an agent cannot follow is a ticket nobody can do. Point at **the file the work is specified in** — never at a directory holding several candidates — and name **the section, verbatim**, copied rather than retyped. If the target sits in a **gitignored** directory, **say so in the ticket**: a fresh clone gets a pointer to a path that is not there, and should ask rather than improvise. **A pointer you have not followed is a claim, not a reference.**
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
   - `tk close <id>` when done (this can unblock dependents); `tk reopen <id>` if it regresses.
6. **Review:** `tk ls` to list, `tk show <id>` to inspect one, `tk closed` for done work.

> **Careful: `tk status` is not a status report.** It is a *setter* — `tk status <id> <status>` changes one ticket's status. To see where the project stands, use `tk ready` + `tk blocked`.

## Conventions

- **One ticket = one verifiable outcome.** If you can't state how it's checked, it's too big — split it and link the pieces with `tk dep`.
- **Link, don't narrate, blockers.** A dependency edge is checkable; a sentence in a description is not. Let `tk ready` compute what's next.
- **Epics group; only `tk dep` orders.** tk *does* have `-t epic` and `--parent` — but they are **grouping and display only**: `tk ready` and `tk blocked` read **only `deps`**, never `parent`. An epic never blocks and never unblocks anything, and (having no deps of its own) it will sit in `tk ready` looking actionable forever. So use `--parent`/`--tags` to group work, and use `tk dep` to order it. A dependency edge is checkable; a parent link is not.
- **Close eagerly.** An open ticket means unfinished work; closing it is what unblocks the next ready item. Don't leave done work open.
- **Respect tracking policy.** Commit the graph alongside code only when the
  repository tracks it and the action is authorized. Otherwise retain the selected
  local record and disclose its availability in the handoff, without force-add,
  publication, cleanup or a competing status mirror.

Run `tk help` for the full, version-accurate command list.

# Solo mode

One agent works the plan task by task, in one context. The plan's own steps are the
discipline; there is no per-task reviewer. The whole-branch review at the end is
independent, and it is not optional.

Announce the mode once (the router's decision record already did this) and proceed
without stopping between tasks.

## Per task

1. Mark the task in progress.
2. Follow each step exactly. The plan's steps are bite-sized on purpose: a step that
   names exact code is transcription, not an invitation to redesign.
3. Run the verifications the step specifies. Run them, read the output, and do not
   proceed on an unread command.
4. Mark the task complete.
5. If the plan's tasks were mirrored into the `tk` work-graph (see the optional ticket
   mirror in **writing-plans**), close the matching ticket: `tk close <id>`. Closing is
   what unblocks the next ready task, so a mirrored plan that is never closed stalls
   `tk ready`. Skip this when ticket was not used.

Do not batch the bookkeeping to the end. A context loss between task 4 and task 9 leaves
you with no record of which of the six landed.

## When to stop and ask

Stop immediately when:

- you hit a blocker: a missing dependency, a failing test you cannot explain, an
  instruction you cannot parse
- the plan has a gap that prevents you starting a task
- a verification fails repeatedly

Ask for clarification rather than guessing. If your partner updates the plan in response,
re-read it from the top before resuming: the change may have moved a later task.

## Finishing

After every task is complete and verified:

1. Run the full test suite and read the result.
2. **Dispatch an independent whole-branch review.** Use the `requesting-code-review`
   skill and fill its [code-reviewer.md](../requesting-code-review/code-reviewer.md)
   template. Give the reviewer the plan, the branch's merge base and head, and the
   binding constraints, and dispatch it on the most capable model available.

   **Do not review your own work here.** You know what each step *meant*; a reviewer
   with a clean context reads only what you *wrote*, and that gap is the entire value.
   Solo mode skips per-task review, which makes this final pass the only independent
   read the branch gets.

3. Fix the review's Critical and Important findings, then re-run the suite.
4. Hand off to **finishing-a-development-branch** to choose how the work lands.

Never start implementation on main or master without explicit user consent.

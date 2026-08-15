# SDD mode

Dispatch a fresh implementer subagent per task, review each one independently before
moving on, and run one broad whole-branch review at the end.

**Core principle:** fresh subagent per task, plus a task review covering both spec
compliance and code quality, plus a broad final review.

**Why subagents.** You construct exactly the context each agent needs. They never inherit
your session's history, so they stay focused, and your own context stays free for
coordination.

**Continuous execution.** Do not pause to check in between tasks. The only reasons to
stop are a blocker you cannot resolve, an ambiguity that genuinely prevents progress, a
dispute the fix loop could not settle, or all tasks complete.

## Before task 1

1. Scan the plan once for conflicts: tasks that contradict each other or the plan's
   Global Constraints, and anything the plan mandates that a review rubric would treat as
   a defect. Present everything you find as one batched question, each finding beside the
   plan text that mandates it, asking which governs. If the scan is clean, proceed
   without comment.
2. Record the branch's merge base: `git merge-base main HEAD`. The final review needs it.
3. Check for a progress ledger:
   `cat "$(git rev-parse --show-toplevel)/.superpowers/sdd/progress.md" 2>/dev/null`. On a
   first run the file does not exist and the command prints nothing: that is the normal
   starting state, not a blocker. Create the ledger when you record task 1. If it does
   exist, tasks marked complete there are done. Do not re-dispatch them; resume at the
   first task not marked complete. Conversation memory does not survive compaction, and a
   controller that lost its place re-dispatching a finished sequence is the most expensive
   failure this mode has.

## Per task

1. Record the current commit as BASE, before dispatching anything.
2. Run `scripts/task-brief PLAN_FILE N` from this skill's directory. It writes the task's
   full text to a uniquely named file and prints the path.
3. Dispatch the implementer with [implementer-prompt.md](implementer-prompt.md). The
   dispatch carries, and carries only:
   - one line on where this task fits in the project
   - the brief path, introduced as "read this first, it is your requirements, with the
     exact values to use verbatim"
   - **the plan's Global Constraints, copied verbatim, and the
     contracts established by earlier tasks**: the interfaces, signatures, and
     invariants a later implementer cannot see from its own brief. This is the one
     thing a fresh context genuinely cannot recover, and skipping it is how task 6
     breaks an invariant task 2 set.
   - your resolution of any ambiguity you noticed in the brief
   - the report-file path and the report contract

   Do not paste accumulated prior-task summaries. Carry forward contracts, not history.

4. Handle the implementer's status:
   - **DONE:** run `scripts/review-package BASE HEAD` with the BASE from step 1 (never
     `HEAD~1`, which silently drops all but the last commit of a multi-commit task), then
     dispatch the reviewer with the printed path.
   - **DONE_WITH_CONCERNS:** read the concerns first. Correctness or scope concerns get
     addressed before review; observations get noted and the review proceeds.
   - **NEEDS_CONTEXT:** supply what is missing and re-dispatch.
   - **BLOCKED:** diagnose before retrying. A context problem gets more context; a
     reasoning problem gets a more capable model; an oversized task gets split; a wrong
     plan gets escalated to your human partner. Never force the same model to retry
     unchanged.

5. Dispatch the task reviewer with [task-reviewer-prompt.md](task-reviewer-prompt.md).
   Build its context from **the task contract, the diff, the tests, and the relevant
   files, never from the implementer's transcript.** The transcript contains the
   implementer's reasoning about what it meant to build, which is exactly the thing a
   review must not be anchored by. Hand over the brief path, the report path, the review
   package path, and the binding constraints.

   Never tell a reviewer what not to flag, and never pre-rate a finding's severity. If
   you believe a finding will be a false positive, let it be raised and adjudicate it.

6. Run the fix loop on Critical and Important findings. One fix subagent per round,
   carrying the complete findings list, and every fix dispatch re-runs the tests covering
   its change and reports the command and output. Re-review after each round.

   **The loop stops after three failed review rounds.** If the reviewer is still
   rejecting the work after the third fix round, stop and surface the dispute to your
   human partner: the reviewer's findings, the implementer's response, and the plan text
   that bears on it. Three rounds without convergence means the disagreement is about the
   requirement, not the code, and a fourth round will not settle it.

7. Record Minor findings in the ledger. When the review comes back clean, append one line
   to the ledger: `Task N: complete (commits <base7>..<head7>, review clean)`.

## Resolving the reviewer's unverifiable items

The reviewer may report items it cannot verify from the diff, requirements living in
unchanged code or spanning tasks. These do not block the rest of the review, but resolve
each one yourself before marking the task complete: you hold the cross-task context the
reviewer lacks. A confirmed gap is a failed spec review. Send it back and re-review.

## When a dispatch fails

A subagent dispatch can fail mid-run: a rate limit, a permission denial, a tool error. All
three targets have subagents, so absence of the capability is not the case to plan for;
mid-run failure is.

**Stop and report.** Say which task was in flight, which dispatch failed, and what the
error was. Do not silently fall back to implementing the task inline: this mode's whole
claim is that each task was implemented by a fresh context and reviewed by an independent
one, and a run that quietly abandons that while still reporting "sdd, all tasks reviewed"
is lying about what it did. A mode that refuses is recoverable. A mode that lies is not.

## Model selection

Use the least capable model that can do each job, and **always name the model explicitly**:
an omitted model inherits your session's, usually the most expensive one.

- transcription implementers, where the plan's task text contains the code to write, and
  single-file mechanical fixes: the cheapest tier
- implementers working from prose, and reviewers: a mid-tier model as the floor, because
  the cheapest models take two to three times the turns on multi-step work, which costs
  more overall than the token price saves
- multi-file integration and judgment: a standard model
- the final whole-branch review: the most capable model available

## Finishing

1. Run `scripts/review-package MERGE_BASE HEAD` with the merge base you recorded.
2. Dispatch the final whole-branch review with the `requesting-code-review` skill's
   [code-reviewer.md](../requesting-code-review/code-reviewer.md), on the most capable
   model, including the package path and the ledger's accumulated Minor findings so it
   can triage which must be fixed before merge.
3. If the final review returns findings, dispatch **one** fix subagent with the complete
   list, not one fixer per finding: per-finding fixers each rebuild context and re-run
   suites.
4. Hand off to **finishing-a-development-branch**.

Never start implementation on main or master without explicit user consent.

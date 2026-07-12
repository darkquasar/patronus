# Session completion

**Work is not complete until it is pushed.** A commit that only exists on your
machine is work the next session — and the human — cannot see. Finish the job.

When ending a work session, complete all of these:

1. **File follow-ups.** Anything discovered but not done becomes a tracked item in
   whatever work-graph or issue tracker this project uses. If none is wired, say so
   plainly in your hand-off rather than letting the follow-up evaporate.
2. **Run the quality gates** (if code changed) — the project's tests, linters, and
   build. Do not push red.
3. **Update the status of what you touched.** Close what you finished; mark what is
   still in flight. Closing is what unblocks the next ready item — work left open
   stalls whoever picks up next.
4. **Push to the remote.** This is the step that is most often skipped, and the only
   one that makes the work durable:
   ```bash
   git pull --rebase
   git push
   git status          # must report the branch is up to date with its remote
   ```
5. **Clean up.** Drop stashes you no longer need; prune stale remote branches.
6. **Verify.** Everything is committed *and* pushed — not one or the other.
7. **Hand off.** State what changed, what is unfinished, and where the next session
   should start.

## Rules

- **Never stop before pushing.** That strands the work locally, where a lost machine
  or a fresh clone erases it.
- **Never say "ready to push when you are."** Pushing is your job, not a suggestion
  handed back to the human.
- **If the push fails, resolve it and retry** until it succeeds — a failed push is an
  unfinished session, not a completed one with a footnote.
- **Do not push to a protected branch to satisfy this.** If you are on `main` (or the
  work belongs on a branch), branch first, then push the branch. "Push" means "get the
  work onto the remote where it belongs," never "bypass the project's review flow."

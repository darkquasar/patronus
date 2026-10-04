# Work on a branch

New work preferably belongs on a feature branch. This is advisory, not permission
to change Git state. Follow the task's explicit Git grant and branch/worktree
allocation; never replace a coordinator-allocated branch or worktree.

- If branch creation or switching is authorized and the working tree is on
  `main`/`master`, create a feature branch before a non-trivial change
  (e.g. `git checkout -b feat/<short-name>`).
- If authority is absent or unclear, propose the branch and ask the user or
  coordinator before changing Git state. Read-only tasks remain read-only.
- Honor an explicit instruction to work on the current branch, including an
  authorized one-line fix; do not gate the task on this preference.
- State the branch in use so the user can redirect. This guidance grants no
  commit, merge, push, discard, or cleanup authority.

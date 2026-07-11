# Work on a branch

New work belongs on a branch. Unless the user explicitly asks to work on the current branch
or on `main`/`master`, create or switch to a feature branch before making changes:

- If the working tree is on `main`/`master` and the user asks for a non-trivial change, create
  a branch first (e.g. `git checkout -b feat/<short-name>`), then proceed.
- A throwaway one-line fix the user wants applied directly is the exception — honor an explicit
  "just do it here".
- This is a default disposition, not a hard block. State the branch you created so the user can
  redirect.

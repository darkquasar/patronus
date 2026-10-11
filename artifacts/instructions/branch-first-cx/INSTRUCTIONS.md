# Prefer an authorized feature branch

New work preferably belongs on a feature branch. This is advisory and grants no
Git mutation. Use the lead-allocated branch/worktree exactly; never replace it.
Read-only tasks stay read-only.

If branch creation/switching is explicitly granted and the tree is on main or
master, create a feature branch before nontrivial edits. If Git authority is
absent, report the preference and ask the lead/owner before changing state.
Honor an explicit request to work on the current branch within its existing grant.
State the branch actually in use so the owner can redirect.

Branch preference grants no commit, merge, rebase, push, stash deletion, discard,
worktree removal or cleanup. Those actions need separate authority and successful
required verification. Never change shared branches to satisfy this preference.

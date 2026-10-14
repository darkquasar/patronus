---
name: finishing-a-development-branch-oc
description: "OpenCode only: Use when authorized implementation is ready for verification and a branch or captured worktree handoff."
---

# Finish the authorized stage

Implementation, integration, publication and cleanup are separate grants.
Default to retaining work and reporting evidence, not automatically merging,
pushing, opening a PR or deleting a branch. A leaf returns its complete handoff
to the lead; only the lead decides acceptance and the next authorized action.

## Verify exact bytes

Reread the brief and current work state. Read
`{skillsDir}/verification-before-completion-oc/SKILL.md` and reuse checks whose exact inputs, commands/options, dependency/tool/environment
identity and acceptance scope still match. Run missing or invalidated required
checks under the project's resource lock and applicable limits. Final aggregate
checks stay fresh for the final composed inputs; a new stage label alone does
not require another suite. Include actual
commands, exits, case names and full log paths. Unavailable or skipped checks are
blocked, not passed. Do not install tools or rerun broad suites just for reporting.

Inspect root, branch, HEAD, base, index and dirty/untracked files. Capture complete
patches and result bytes with hashes, including seeded source if authorized.
Settle every started worker before handoff. Unknown settlement or missing capture
retains work and blocks integration, retry and cleanup.

Consume the matching independent aggregate review through
`{skillsDir}/requesting-code-review-oc/SKILL.md`, with at most one bounded
batched correction/disposition. Finishing adds no second review for unchanged
inputs, scope and coverage. Missing requirements/tests/authority, unresolved
Critical/Major or more than two distinct Medium block. Up to two Medium residuals
need explicit owner acceptance. Consume the coordinator's retained disposition;
parent-verified corrected bytes are not independently re-reviewed.

## Decide within explicit grants

Offer only applicable options if no next action is recorded:

1. Retain branch/worktree and deliver the complete local evidence.
2. Integrate into the exact approved target under a separate integration grant.
3. Publish under a separate remote/PR grant after required verification.
4. Discard only under exact destructive consent listing root, branch and work.

Existing explicit approval remains valid within scope. A choice of integration
is not permission to pull, rebase, commit, publish or delete anything. Verify the
exact integrated bytes using configured checks before an integration claim.
A publication grant must name the remote and action. Do not infer it from a
successful local handoff.

For any authorized cleanup read
`{skillsDir}/using-git-worktrees/SKILL.md`. Directory naming is not ownership.
Capture and settlement must be verified first; managed cleanup belongs to its
recorded owner. Never prune unrelated registrations or force-remove retained or
unknown roots. Failed/partial attempts and upstream handoff evidence stay intact.

Report source root/base/head, branch, changed files, no-staged-files observation,
input/output hashes, commands/exits/logs, review disposition, residual risks and
unresolved asks. A local result is not remote publication.

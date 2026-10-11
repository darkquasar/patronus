---
name: verification-before-completion-cx
description: "Codex only: bind authorized verification evidence to exact inputs and scope before completion claims; final aggregate checks stay fresh."
---

# Evidence before completion in Codex

This skill grants no execution, edit, commit, network, integration, publication
or cleanup authority. Read the task's stage, owned paths, verification commands,
resource lock and budget. Missing or unapproved required checks are blocked,
not passed. Do not install tools or broaden access to make checks run.

Before claiming any status:

1. Identify the full authorized command that proves this exact claim.
2. Compare evidence identity: exact relevant checked bytes (including dirty and
   untracked inputs), full command/options, relevant dependency/tool/environment
   identity and acceptance scope. Reuse an observed run for a later status report
   only when all match and saved logs and actual exits remain readable. HEAD
   equality alone is insufficient. Consequential drift, unknown identity, changed
   acceptance or missing logs invalidates reuse: run the authorized command again
   or report it blocked. Final aggregate verification at delivery stays fresh.
3. Read the complete saved output and actual exit status, including reported
   failures. Cite the run, checked hashes, command, scope and log references.
4. Compare the observation to the acceptance criterion. State failure, skipped
   coverage or blocked prerequisites when evidence is absent.
5. Only then state the bounded claim with root/base/head, changed files and hashes,
   tests added, actual commands/results, residual risks and open decisions.

Follow the task brief's selected `critical-tdd` or `focused-postcheck` strategy,
explicit test-first requests and project-required checks. For critical TDD, retain
the intended failing public-behavior test on invented data, minimum change, green
focused run and relevant regressions. Routine prose, manifests or mechanical
wiring without a changed safety invariant use focused postchecks, with no invented
red cycle. Missing or ambiguous strategy stops for a lead decision before
implementation. A test description is not a test run.
A narrow check proves only its checked scope. Static placement never proves
activation, worker isolation, authentication, trust or native Codex readiness.

Read all returned worker findings and checked patches after settlement. A
completion notification, dispatch receipt or worker's success sentence is not
acceptance. Bind review evidence to the delivered hashes. Missing independent
review remains a separate blocker even when tests pass.

Preserve mandatory initial and post-compaction brief, task-state and instruction
reads. Evidence reuse never waives required acceptance or independent review.

Do not say "should pass" as completion evidence. Report exact commands, exits,
logs and limitations. Preserve partial outputs on failure. Commit, push, merge,
worktree removal and discard still require separate explicit grants.

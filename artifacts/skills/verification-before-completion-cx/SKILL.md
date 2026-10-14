---
name: verification-before-completion-cx
description: "Codex only: bind authorized verification evidence to exact inputs and scope before completion claims; final aggregate checks stay fresh."
---

# Evidence before completion in Codex

This skill grants no execution, edit, commit, network, integration, publication
or cleanup authority. Read the task's stage, owned paths, verification commands,
resource lock and applicable limits. Missing or unapproved required checks are blocked,
not passed. Do not install tools or broaden access to make checks run.

Before claiming any status:

1. Identify the full authorized command that proves this exact claim.
2. Compare evidence identity: exact relevant checked bytes (including dirty and
   untracked and relevant ignored inputs), full command/options, relevant dependency/tool/environment
   identity and acceptance scope. Reuse an observed run for a later status report
   only when all match and saved logs and actual exits remain readable. HEAD
   equality alone is insufficient. Consequential drift, unknown identity, changed
   acceptance or missing logs invalidates reuse: run the authorized command again
   or report it blocked. Message or stage changes alone do not stale evidence.
   Final aggregate verification stays fresh for the composed delivered inputs;
   matching integration checks can satisfy finishing without another suite.
3. Read the complete saved output and actual exit status, including reported
   failures. Cite the run, checked hashes, command, scope and log references.
4. Compare the observation to the acceptance criterion. State failure, skipped
   coverage or blocked prerequisites when evidence is absent.
5. Only then state the bounded claim with root/base/head, changed files and hashes,
   tests added, actual commands/results, residual risks and open decisions.

Use the installed tdd-cx defaults and record strategy/rationale in the existing
brief/handoff. Explicit test-first requests and required project checks govern.
Critical auth/secrets, ownership, concurrency/isolation/settlement, migration/
removal and reproducible behavioral bugs require public-behavior TDD. Other
features/behavior changes default to TDD; a different strategy needs a concrete
reason and owner agreement unless already approved. Use `critical-tdd` for
short test-first slices: invented-data intended RED, minimum GREEN and relevant
negative/legacy regressions. Pure behavior-preserving refactors work from green
coverage and characterize uncertain legacy behavior first. Nonbehavioral prose/
manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`, not manufactured reds or prose-substring tests. Omission
alone applies these defaults and records selection, without another approval or
ledger; genuine classification, conflict or exception ambiguity needs owner
direction. Keep existing safety tests. Tests do not admit successor reviewers.
A test description is not a test run.

A narrow check proves only its checked scope. Check actual capability,
authentication, ownership and isolation where those are acceptance criteria.

Read all returned worker findings and checked patches after settlement. A
completion notification, dispatch receipt or worker's success sentence is not
acceptance. Bind review evidence to the delivered hashes. Missing independent
review remains a separate blocker even when tests pass.

Preserve mandatory initial and post-compaction brief, task-state and instruction
reads. Evidence reuse never waives required acceptance or independent review.

Do not say "should pass" as completion evidence. Report exact commands, exits,
logs and limitations. Preserve partial outputs on failure. Commit, push, merge,
worktree removal and discard still require separate explicit grants.

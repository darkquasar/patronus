---
name: requesting-code-review
description: Use when the coordinator requests an independent aggregate implementation or named-boundary review
---

# Bounded independent review

Only the coordinator admits one reviewer for the exact subject. Aggregate review
is required before implementation acceptance. One matching review covers feature,
branch and pre-merge triggers when exact inputs, scope and coverage match. Earlier
reviews need a named changed safety/compatibility invariant or irreversible boundary
with its own acceptance point; task count, a complex bug or a shared framework
alone does not add a gate. No automatic per-task review or second-model discovery.

Provide [code-reviewer.md]({skillDir}/code-reviewer.md) with exact approved
requirements/plan, absolute root/base/head plus full dirty/untracked patch hashes,
relevant source/interfaces, retained verification commands/options/exits/logs and
acceptance scope. Never assume HEAD~1 captures the subject. No report-file write
is required of a read-only reviewer: consume complete native returned findings.

Consume accepted spec/plan reviews without repeating their subjects. Boundary
review occurs before dependent consumption and never replaces aggregate review.
Weak verification alone creates no gate, and no repeated-until-approved loop follows.

Read all findings, preserve source IDs and consequence-based severities, deduplicate
without dropping evidence, and admit at most one bounded batched correction per
gate by default. Fixers cannot launch reviewers. Unresolved Critical/Major or more
than two distinct Medium block. Up to two Medium residuals need explicit owner
acceptance; Low gets recorded disposition. Missing authority, requirements or tests
independently blocks. Never lower severity to fit the threshold.

The coordinator resolves a fixed defect only against its cited acceptance criterion,
exact corrected bytes and sufficient positive/negative/regression evidence. Record
parent-verified correction, not independent re-review. Changed public interface,
safety invariant or acceptance scope, or insufficient oracle, returns to the owner
for a decision or additional independent-review grant. Critical/Major residuals
cannot be waived. No automatic second review wave.

Reuse verification only when exact relevant dirty/untracked/ignored inputs,
command/options, dependency/tool/environment identity and acceptance scope match,
with readable logs and actual exits. HEAD or message equality alone is insufficient.
Final aggregate checks stay fresh for composed delivered inputs; matching integration
checks satisfy finishing without another identical suite.

Return changed files/hashes, tests added, actual checks/exits/logs, every finding's
disposition, limitations and next authorized action. Local handoff completes the
stage when publication is ungranted. Commit, integration, publication and cleanup
remain separate grants. No automatic push/rebase, retry-until-success, stash deletion
or worktree removal; uncertain settlement/capture retains allocations.

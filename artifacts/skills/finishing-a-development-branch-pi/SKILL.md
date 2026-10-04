---
name: finishing-a-development-branch-pi
description: "Finish within separate integration, publication and cleanup grants"
---

# Finish a development branch

Read verification-before-completion-pi. Check the exact branch/base/head, working/index status, required fresh tests, reviewed hashes and unresolved findings. Required acceptance/release failures keep their action blocked even if review thresholds pass. If verification failed, preserve and report; do not present a red branch as merge-ready.

Identify current authority: local integration; remote publication/PR; retention; or discard. An existing explicit grant governs within scope, otherwise ask the coordinator for the concrete next action. Never automatically pull, rebase, push, merge, prune, drop stashes or delete branches/worktrees. Report a durable local commit/handoff honestly as local, not remote publication.

Before any authorized integration settle dependent runs, inspect each captured patch/commit and owner, apply in dependency order and verify the resulting tree. Do not infer cleanup authority from merge success. Before authorized discard record exact paths/branches/commits and check capture/readability, no active or unknown descendants and unchanged ownership. Capture failure/unknown state preserves work. Native temporary worktree finalization follows the exact prelaunch grant described in using-git-worktrees-pi; retained cwd needs its own explicit cleanup grant. Directory names never prove ownership.

Return changed files, exact base/head, source/output hashes, tests/logs/exits, review dispositions, skipped checks, residuals, retained resources and next authorized step. Coordinator alone updates native mission state and authorized lessons. A local handoff can complete this stage without a push.

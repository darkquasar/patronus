---
name: rust-style-oc
description: "OpenCode only: Use when writing, reviewing or refactoring Rust code or Cargo manifests, or coordinating Rust builds, tests and parallel worktrees."
---

# Rust style for OpenCode

Read the installed companion
[{skillsDir}/rust-style-reference/SKILL.md]({skillsDir}/rust-style-reference/SKILL.md),
then only its references relevant to the task. Companion paths resolve relative
to that bundle. If it is missing, report the packaging gap; do not fuzzy-match a
foreign harness entrypoint or install dependencies silently.

Follow the repository's Cargo/toolchain/CI and authority constraints. The shared
guidance preserves Rust idioms, behavior-focused testing and independent safety
gates while following project choices rather than imposing them. When parallel
compilation, cache reuse or heavy checks are in scope, follow its build-efficiency
reference: writer count and expensive-operation count are separate limits.

This optional reference skill adds no delegation, review, tracking or approval
workflow and grants no install, commit, publication or cleanup authority. An
assigned worker follows its coordinator's resource allocation and returns
evidence; it does not start a new coordinator or review/fix chain.

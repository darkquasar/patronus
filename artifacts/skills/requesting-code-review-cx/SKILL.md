---
name: requesting-code-review-cx
description: "Codex only: request a fresh independent read-only review of exact changes and retain complete findings."
---

# Request a Codex code review

The lead owns review admission and disposition. A worker returns evidence and
never launches its own reviewer. Read the approved requirements, exact source
root/base/head, dirty and untracked patch hashes, owned files, tests/logs and
known limitations. Review must cover the actual delivered bytes, including
unstaged files, not just the last commit.

1. Assemble a self-contained package with requirements, implementation summary,
   root/revision, exact changed-file inventory and hashes, approved commands,
   actual results, skipped gates and Partial capabilities. Exclude reasoning
   transcripts that would frame the reviewer's judgment.
2. Read `{skillDir}/code-reviewer.md`. Use a fresh independent read-only Codex
   worker only when the session exposes a qualified native worker primitive,
   required tools, wait mechanism and sufficient approved budget. Give it the
   complete package and rubric. No source/index/branch mutations or shared
   report writes. If unavailable or auth-blocked, stop as blocked. Never replace
   independent review with self-review or a CLI model request.
3. Wait for settlement and collect the complete final response. The reviewer
   returns findings; the lead writes the report. A dispatch receipt, completion
   status or empty response cannot satisfy the gate. Check file:line evidence.
4. Normalize findings by consequence to Critical/Major/Medium/Low, retaining
   original severity and every source ID under a deduplicated defect ID.
   Record evidence, consequence, proposed correction and owner disposition.
   No threshold-driven downgrade. Critical/Major or more than two distinct
   Medium block; up to two Medium need explicit owner-accepted residuals.
   Missing tests, requirements or authority independently block acceptance.
5. Use `{skillsDir}/receiving-code-review-cx/SKILL.md` for at most one bounded
   correction/disposition. Retain old/new hashes and tests. Corrected bytes are
   not retroactively independently reviewed. Stop at the bound and return open
   decisions to the owner; never automatically launch another review wave.

Before any completion claim, read
`{skillsDir}/verification-before-completion-cx/SKILL.md` and report fresh command
results. Review is separate from integration, commit, publication and cleanup;
none is implied by a favorable finding or by this skill.

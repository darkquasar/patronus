---
name: requesting-code-review-oc
description: "OpenCode only: request a fresh independent read-only review of exact changes and retain complete findings."
---

# Request an OpenCode code review

The coordinator admits one independent reviewer per exact subject by default.
A second reviewer needs separate admission, not automatic discovery or retry.
One matching aggregate review covers feature, branch and pre-merge triggers when
inputs, scope and coverage match; finishing does not create another review. A worker returns evidence and
never launches its own reviewer. Read the approved requirements, exact source
root/base/head, dirty, untracked and relevant ignored input hashes, owned files, tests/logs and
known limitations. Review must cover the actual delivered bytes, including
unstaged files, not just the last commit.

1. Assemble a self-contained package with requirements, implementation summary,
   root/revision, exact changed-file inventory and hashes, approved commands,
   actual results, skipped gates and actual capability limits. Exclude reasoning
   transcripts that would frame the reviewer's judgment.
2. Read `{skillDir}/code-reviewer.md`. Use a fresh independent read-only
   OpenCode subagent (`@patronus-technical-reviewer-oc` through the Task tool)
   only when `opencode agent list` shows that role with read-only permissions
   and authentication and Task completion controls are available. Give it the complete package
   and rubric. No source/index/branch mutations or shared
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
5. Use `{skillsDir}/receiving-code-review-oc/SKILL.md` for at most one bounded
   batched correction/disposition, not one fixer per finding. Retain old/new hashes and tests. Corrected bytes are
   not retroactively independently reviewed. Stop at the bound and return open
   decisions to the owner; never automatically launch another review wave.

Before any completion claim, read
`{skillsDir}/verification-before-completion-oc/SKILL.md` and report matching input-bound command
results, with fresh final aggregate checks. Review is separate from integration, commit, publication and cleanup;
none is implied by a favorable finding or by this skill.

---
name: decision-clarity-pi
description: "Pi only: Use when the coordinator must ask a human for an unresolved choice, a material missing fact or a required permission, including during execution, correction or integration; not for routine updates, test results or leaf handoffs."
---

# Decision clarity for Pi

Use this when, as the human-facing coordinator, you need a person's unresolved
choice, a material missing fact or a required permission, at any stage. Routine
progress updates, individual test results and leaf handoffs are not triggers. An
action already approved within its scope needs no new question.

Pi loads this entrypoint from its installed skill file. Then read the shared
body at exactly
[{skillsDir}/decision-clarity-reference/SKILL.md]({skillsDir}/decision-clarity-reference/SKILL.md)
in this skill's installation scope. Its optional `references/examples.md`
resolves relative to that bundle; read it only when an example helps. Do not
substitute a same-named copy from another scope or another harness's skill
root. If the body is missing or unreadable, report that packaging gap, keep the
underlying permission requirement and still report any urgent safety issue.

A restricted leaf or assigned worker keeps its assigned output schema and
returns evidence to its coordinator. It does not ask the human, start new
delegation or turn a non-prescriptive finding into a recommendation. This skill
adds no approval gate, reviewer, tracker or editorial pass and grants no
authority beyond the existing one.

---
name: receiving-code-review-cx
description: "Codex only: verify review suggestions against source, correct within the grant and return evidence for lead disposition."
---

# Receive code review in Codex

Read the complete feedback and exact reviewed root/base/head and patch hashes.
Understand each requirement, then verify its evidence against current source,
callers, tests and compatibility constraints. Respond with a technical statement
or source-backed pushback. Agreement and reviewer confidence are not evidence.

Stop for material ambiguity, contradictory owner decisions, changed architecture,
file overlap or an expanded grant. Ask the lead/owner and wait for a real reply.
Do not treat steering or a review suggestion as new authority. Workers never
launch another reviewer or mutate shared status; they return complete findings.

Within an approved correction grant, address blockers first, then one item at a
time. Follow the testing strategy recorded in the existing correction brief. Select
`critical-tdd` for new or changed auth/secrets, migration/removal, ownership,
concurrency/settlement/isolation behavior and reproducible behavior bugs. Read the
installed tdd-cx SKILL.md when selected; use short public-behavior slices with an
invented-data test, observed intended failure, minimal implementation, green and
relevant negative/legacy regressions under the project lock. Routine
prose/manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`: inspect exact diff and positive/negative cases, then run
approved focused checks. Explicit test-first requests and project-required checks
govern. Preserve existing safety tests; do not invent red cycles per task/file or
prose-substring tests. Missing or ambiguous strategy, including a legacy brief
without one, stops for a lead decision before correction. A focused-postcheck
correction is not defective merely because it lacks invented red evidence. Preserve
legacy behavior and independent source discoverability. Do not batch unrelated
refactors into a review correction.

Retain each source finding ID and original severity under the lead's deduplicated
defect ID. Severity follows consequence: Critical/Major/Medium/Low. No downgrades
to fit acceptance thresholds. Return corrected file hashes, old/new evidence,
actual commands/results and unresolved items for explicit lead disposition.

Required fresh independent whole-change implementation review remains required
even for small deliveries; only the owner may explicitly waive it. Per-task
Codex review requires a named changed safety/compatibility invariant or irreversible
boundary with its own acceptance point; existing shared-framework use alone is
insufficient. Use one fresh independent review and at most one bounded
correction/disposition per gate. Corrected bytes need explicit lead disposition;
no automatic second review wave. Unresolved Critical/Major or more than two Medium block. Up to two Medium
need explicit owner-accepted residuals; missing requirements/tests/authority
independently block. Low correction is optional within the grant.

Read `{skillsDir}/verification-before-completion-cx/SKILL.md` before saying an
item is fixed. Unavailable/auth-blocked primitives stop the lane. Never change
engine/model/provider/protocol, invoke a CLI model or expand access to continue.
Review acceptance grants no commit, merge, push, cleanup or deployment.

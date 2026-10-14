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

Within the one approved batched correction grant, address blockers first.
Several focused edit/test iterations may fit this grant; do not admit one fixer
per finding or launch successor reviewers. Follow the testing strategy recorded in the existing correction brief. Select
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

The coordinator may resolve a fixed defect only after checking its cited
failure against the exact corrected bytes and sufficient relevant positive,
negative-path and regression evidence for the existing acceptance criterion.
Retain this disposition in the original report as parent-verified, not
independently re-reviewed. A changed public interface, safety invariant or
acceptance scope, or an insufficient oracle, returns to the owner for a design
decision or additional independent-review grant. Blockers remain blockers at
the correction cap.

Unresolved Critical/Major or more than two distinct Medium block. Up to two
Medium residuals need explicit owner acceptance; missing requirements, tests
or authority independently block. Retain Low dispositions without automatic
review waves. Review admission belongs to the coordinator, not the fixer.

Read `{skillsDir}/verification-before-completion-cx/SKILL.md` before saying an
item is fixed. Unavailable/auth-blocked primitives stop the lane. Never change
engine/model/provider/protocol, invoke a CLI model or expand access to continue.
Review acceptance grants no commit, merge, push, cleanup or deployment.

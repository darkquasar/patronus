---
name: receiving-code-review-oc
description: "OpenCode only: verify review suggestions against source, correct within the grant and return evidence for lead disposition."
---

# Receive code review in OpenCode

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
per finding or launch successor reviewers.
Use the installed tdd-oc defaults and record strategy/rationale in the existing
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
Preserve legacy behavior and independent source discoverability. Do not batch unrelated
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

Read `{skillsDir}/verification-before-completion-oc/SKILL.md` before saying an
item is fixed. Unavailable/auth-blocked primitives stop the lane. Never change
engine/model/provider/protocol, invoke a CLI model or expand access to continue.
Review acceptance grants no commit, merge, push, cleanup or deployment.

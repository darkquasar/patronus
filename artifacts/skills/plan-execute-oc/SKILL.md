---
name: plan-execute-oc
description: "OpenCode only: execute an approved plan serially with evidence-based mode selection and independent review."
---

# Execute an approved plan in OpenCode

Read the approved spec/plan, project instructions, exact source root/base/head,
owned paths, acceptance criteria, applicable resource limits and prior outputs. Verify that the
execution grant covers this stage. A worker executes its assigned task only;
only the coordinator admits workers, selects the mode and requests reviews.
Implementers, fixers and reviewers are leaves: return complete evidence and stop,
without delegating successors, adding gates or updating shared progress. Scope change, destructive
action, overlap or architecture ambiguity needs lead/owner authorization.

## Select and record the mode

An explicit authorized mode request governs. Otherwise assess the actual plan:

- A changed security/trust or ownership boundary, irreversible state change,
  concurrency correctness or public compatibility invariant selects serial
  implementation with a coordinator-admitted boundary review when it has its own
  acceptance point. Name the changed safety/compatibility invariant or irreversible
  boundary and its consequence. Weak deterministic verification or shared-framework
  use alone is insufficient.
- Two or more soft signals prompt that same assessment only with at least two
  implementation tasks: a new architectural pattern, a cross-module invariant,
  paired producer/consumer change, qualitative acceptance without an oracle,
  negative requirements or several plausible designs. Boundary review still needs
  the named changed invariant/boundary and its own acceptance point; otherwise use
  serial implementation with required independent whole-change review.
- Otherwise the lead implements serially and obtains independent whole-change
  review at the end. Task count alone does not justify delegation.

Cite actual plan sections for the chosen mode and each boundary acceptance point.
An explicit mode request does not remove the named-invariant requirement; resolve
an unclear request with the lead before launch. All modes retain required fresh
independent whole-change implementation review, even for small deliveries. Only
an explicit owner decision may waive that gate. Boundary reviewers are read-only
OpenCode subagents (`@patronus-technical-reviewer-oc` through the Task tool),
admitted after checking the installed role and its read-only permissions. Serial lead implementation never means concurrent workers
writing in the lead checkout. A requested isolated implementer mode needs an
approved isolation mechanism before any implementation starts. No automatic mode,
provider, protocol or model changes when capabilities fail.

## Execute and verify

For each ready task, read current source/tests and the testing strategy recorded
in the existing brief. Select `critical-tdd` for new or changed auth/secrets,
migration/removal, ownership, concurrency/settlement/isolation behavior and
reproducible behavior bugs. Read the installed tdd-oc SKILL.md for that strategy;
use short public-interface slices with an invented-data test, observed intended
behavioral failure, minimal implementation, green and relevant negative/legacy
regressions. Explicit test-first requests and project-required checks govern.

Routine prose/manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`: inspect the exact diff, check positive/negative cases and run
approved focused checks after editing. Keep existing safety tests; do not fabricate
red cycles per task/file or add prose-substring tests. Missing or ambiguous strategy,
including a legacy brief without one, stops for a lead decision before implementation.
Unknown behavioral work does not qualify for focused postchecks.

Follow project commands and resource locks; unavailable or unapproved required
checks are blocked, not passed. Preserve prior interfaces and legacy behavior.
Only edit assigned files. Keep code and supporting source discoverable.

The lead reads `{skillsDir}/verification-before-completion-oc/SKILL.md` before
claims, then `{skillsDir}/requesting-code-review-oc/SKILL.md` for one fresh
independent whole-change review of the exact base/head plus dirty patch hashes.
Boundary review is required only in the selected reviewed mode at the named
invariant/boundary's acceptance point; it does not replace the whole-change gate.
Read complete findings before accepting delivery.
Use `{skillsDir}/receiving-code-review-oc/SKILL.md` for technical correction.
Batch accepted fixes into at most one bounded correction/disposition per review
subject, not one fixer per finding. Fixers return evidence and stop without
launching reviewers. The coordinator checks corrected bytes and affected
regressions. A changed interface, safety invariant or acceptance scope needs an
owner decision, not an automatic review wave. Corrected bytes are parent-verified,
not independently re-reviewed. Unresolved
Critical/Major or more than two Medium block. Up to two Medium require explicit
owner-accepted residuals. Never silently downgrade or automatically relaunch.

Missing worker primitive, authentication, output or required test stops the lane
as blocked. Preserve partial evidence and ask the lead for a decision; do not
invoke a CLI model or treat serial execution as failed-engine fallback.

Report the selected testing strategy and rationale, changed files, added tests
(or none for focused postchecks), base/head, input/output hashes, actual commands
with exits/readable logs, applicable red/green or postcheck evidence, findings
dispositions and residual risks. Commit, integration, publication and cleanup
each need their own grant. Review completion never
implies permission to merge, push, remove worktrees or discard source.

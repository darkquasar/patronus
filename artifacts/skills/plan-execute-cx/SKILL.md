---
name: plan-execute-cx
description: "Codex only: execute an approved plan serially with evidence-based mode selection and independent review."
---

# Execute an approved plan in Codex

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
Codex workers admitted using exposed native tools with available authentication,
read-only scope and complete output delivery. Serial lead implementation never means concurrent workers
writing in the lead checkout. A requested isolated implementer mode needs an
approved isolation mechanism before any implementation starts. No automatic mode,
provider, protocol or model changes when capabilities fail.

## Execute and verify

For each ready task, read current source/tests and the accepted brief.
Use the installed tdd-cx defaults and record strategy/rationale in the existing
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

Follow project commands and resource locks; unavailable or unapproved required
checks are blocked, not passed. Preserve prior interfaces and legacy behavior.
Only edit assigned files. Keep code and supporting source discoverable.

The lead reads `{skillsDir}/verification-before-completion-cx/SKILL.md` before
claims, then `{skillsDir}/requesting-code-review-cx/SKILL.md` for one fresh
independent whole-change review of the exact base/head plus dirty patch hashes.
Boundary review is required only in the selected reviewed mode at the named
invariant/boundary's acceptance point; it does not replace the whole-change gate.
Read complete findings before accepting delivery.
Use `{skillsDir}/receiving-code-review-cx/SKILL.md` for technical correction.
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

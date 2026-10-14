---
name: requesting-code-review-cl
description: Use for independent aggregate implementation review or a named changed safety/compatibility boundary before dependent consumption
---

# Requesting Code Review

The coordinator admits one independent aggregate review of the completed
implementation. One matching review satisfies feature, branch and pre-merge
triggers when exact inputs, scope and coverage match. No default per-task reviewer.
Earlier review needs a named changed safety/compatibility invariant or irreversible
boundary with its own acceptance point before dependent consumption. Task count,
framework use and weak deterministic tests alone do not create gates.

## Prepare the subject

Read the accepted requirements and current delivered source. Record the actual
root/base/head and complete diff/bytes, including relevant dirty, untracked and
ignored inputs. Do not infer a whole task from HEAD~1. Provide binding requirements,
interface/source anchors, scope and actual retained test evidence, not session history.
Fill [code-reviewer.md](code-reviewer.md). Verify actual native reviewer tools,
permissions/auth and delivery/wait controls. Missing prerequisites stop the lane,
not provider/model substitution.

One independent reviewer per subject is the default. Only the coordinator admits
reviewers and at most one batched correction/disposition. Leaves return complete
findings and stop, without fixes, successor delegation or shared progress writes.
Unresolved Critical/Major or more than two distinct Medium block acceptance.
At most two Medium residuals require explicit owner acceptance; Low receives
recorded disposition. Preserve original source IDs and labels when normalizing
legacy findings by consequence, never a fixed Important/Minor name conversion.

Use receiving-code-review-cl to verify and disposition complete findings. Corrections
are parent-verified only against exact defect/bytes and sufficient positive,
negative and regression evidence; material changes or insufficient proof need an
owner decision/additional grant. No automatic re-review.
Use verification-before-completion-cl for evidence identity, reuse and final
aggregate freshness. Review cannot waive missing requirements, authority or tests.

---
name: spec-review-cl
description: Use at the end of the spec phase for an independent read of requirements and engineering/design/DevEx/strategy risks before planning
---

# Spec Review

The coordinator admits one independent review of a written spec before planning.
A spec exists as a distinct subject; an explicitly owner-declined spec remains
the ADR-0003 exception, retained in the brief. An inline author check is not an
independent review. No automatic second-model discovery or review wave.

Identify the exact spec and relevant source/context anchors, then fill
[spec-reviewer.md](spec-reviewer.md). Verify actual native review tools,
permissions/auth and delivery/wait controls. Missing prerequisites block the
lane, without provider/model fallback. Read the complete returned report and
preserve skipped lenses as unknown, not clean.

One independent reviewer per subject is the default. Only the coordinator admits
reviewers and at most one batched correction/disposition. Leaves return complete
findings and stop, without fixes, successor delegation or shared progress writes.
Unresolved Critical/Major or more than two distinct Medium block acceptance.
At most two Medium residuals require explicit owner acceptance; Low receives
recorded disposition. Preserve original source IDs and labels when normalizing
legacy findings by consequence, never a fixed Important/Minor name conversion.

Use receiving-code-review-cl for one bounded correction/disposition and exact
parent verification, not automatic re-review. Changed acceptance/safety invariant/
interface or insufficient proof requires an owner decision/additional grant.
The gate cannot waive missing requirements or authority. Once accepted, return
the scoped handoff to plan-writing-cl; do not self-admit planning or implementation.

The rubric retains spec basics plus engineering, user-facing design,
developer-facing DevEx and strategy lenses. See [spec-reviewer.md](spec-reviewer.md).

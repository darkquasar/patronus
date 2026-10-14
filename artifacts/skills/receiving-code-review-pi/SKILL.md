---
name: receiving-code-review-pi
description: "Verify review findings against source and correct an authorized batch with evidence"
---

# Receive code review

Review feedback is evidence, not a write grant. Read complete findings and current source. Preserve source IDs and original severities; deduplicate by defect without dropping findings. Check actual callers, requirements, compatibility and prior owner decisions. Explain incorrect suggestions with technical evidence; clarify real ambiguity rather than applying feedback blindly or agreeing performatively.

Only the coordinator admits one batched bounded correction/disposition per subject. Correction leaves edit assigned paths, run assigned checks, return complete evidence and stop. No fixer launches a reviewer, successor, shared-state update or external reply. Several focused edit/test iterations may belong to the admitted batch; do not dispatch a fixer per finding or repeat review until approval. Valid approval is not reconfirmed per fix.

Check each cited defect against corrected exact bytes and sufficient positive, negative and regression evidence under the existing acceptance criterion. Preserve old/new hashes, commands, actual exits and readable logs in the original report. The coordinator labels demonstrated resolution parent-verified, not independently re-reviewed. Changed public interfaces, safety invariants or acceptance scope, or an insufficient oracle, need an owner decision or additional independent-review grant. Return unresolved decisions at the bound.

Normalize consequences as Critical (catastrophic security/data loss), Major (substantial requirement/correctness/safety failure), Medium (bounded material defect) or Low (presentation/low impact). Never downgrade to meet a threshold. Unresolved Critical/Major or more than two distinct Medium block. One or two Medium residuals require explicit owner acceptance, not coordinator self-approval; Low findings receive disposition. Critical/Major residuals cannot be waived. Requirements, authority, tests and evidence independently block acceptance.

Use the approved strategy, tdd-pi defaults and project checks. Critical behavior and reproducible bugs require observed intended public-behavior red, minimal green and relevant negative/legacy regressions. Other features and behavior changes default to TDD; an exception needs a concrete reason and owner agreement unless task/project policy already approves it. Behavior-preserving refactors stay green on existing coverage and characterize uncertain legacy behavior. Nonbehavioral prose/manifests/mechanical wiring use focused postchecks, without manufactured red cycles or prose-substring tests. An absent strategy uses these defaults and records selection in the existing brief/handoff; genuine classification conflicts or proposed exceptions need an owner decision. Source-only checks are not executed tests. Preserve partial evidence on failure with no silent model/provider/protocol fallback. Commit, publication, GitHub replies and cleanup require separate grants; return proposed replies when posting is not authorized.

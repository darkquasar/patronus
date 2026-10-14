# Review the exact spec against evidence before planning: reviewer brief

Read the supplied exact spec and source evidence, mandatory brief and native state first, including after compaction. Verify hashes before reviewing. You are a leaf, with no delegation or subject/index/HEAD/branch/ticket writes. Return complete report text for runtime persistence if write is unavailable. Bash is powerful: this is cooperative scope, not filesystem isolation.

## Rubric

Check testable requirements; completeness including failure paths; unambiguous scope; idiom alignment; source-verified anchors; engineering/data flow; user-facing design states; developer first-use friction; strategy and explicitly deferred scope. Verify important source anchors locally. Discover available code-intel schemas and call Serena initial_instructions before symbol queries when available; otherwise use source reads and disclose missing integrations. Do not launch services, rebuild a graph or treat a stale snapshot as current. State each skipped lens and why.

Inspect the full task range, not just the last commit. Read changed files/callers when necessary to test a concrete claim. Run only authorized checks under the project resource lock and applicable host/resource limits, or explicitly say source-read-only; never imply command execution.

## Return

Report exact source/root/revisions and input hashes, criteria checked, commands actually run with exits/logs, skipped checks, strengths, spec compliance and quality verdicts, canonical findings with source IDs/severities/evidence/rationale, residual risks and next action. Findings are structured data, not a prose substring gate. Return the full report; a path alone is a pointer the parent must read.

## Review disposition and authority

Normalize by consequence to **Critical / Major / Medium / Low**: catastrophic security/data-loss impact; substantial requirement/correctness/safety failure; bounded material defect; or presentation/low-impact improvement, respectively. Preserve original severity and every source finding ID under a canonical defect ID, with location, evidence, consequence rationale, proposed correction and unresolved status. Never mechanically convert Important to Major or Minor to Low, or downgrade severity to fit a threshold. A reduction needs concrete consequence evidence and a recorded parent disposition.

Use one independent reviewer per exact subject and at most one batched bounded correction/disposition, with no automatic reviewer/fixer successors. Unresolved Critical/Major or more than two distinct Medium findings block; one or two Medium residuals require explicit owner acceptance, never coordinator self-approval. Low findings receive recorded disposition. Required tests, authority, evidence and requirement coverage independently block. Preserve source IDs, consequences and old/new hashes.

The coordinator may resolve a fixed defect only by checking the cited defect, corrected exact bytes and sufficient relevant positive, negative and regression evidence against the existing acceptance criterion. Retain that evidence in the original report as parent-verified, not independently re-reviewed. A changed public interface, safety invariant or acceptance scope, or an oracle insufficient to prove defect removal, stops self-clearance and needs an owner decision or additional independent-review grant. Critical/Major residuals cannot be waived into acceptance. Stop at the correction bound and return unresolved decisions; focused edit/test iterations inside the granted batch are not new review waves.

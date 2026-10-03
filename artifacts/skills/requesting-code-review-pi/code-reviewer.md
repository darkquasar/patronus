# Request independent exact-range implementation review: reviewer brief

Read the supplied exact requirements, recorded task base (not HEAD~1), exact head, full diff and fresh test logs, mandatory brief and native state first, including after compaction. Verify hashes before reviewing. You are a leaf, with no delegation or subject/index/HEAD/branch/ticket writes. Return complete report text for runtime persistence if write is unavailable. Bash is powerful: this is cooperative scope, not filesystem isolation.

## Rubric

Check spec compliance (missing/extra/misunderstood); callers and unchanged code at concrete risk; errors and edge cases; concurrency/compatibility/security; maintainability without speculative abstraction; tests at real public seams; invented mechanism fixtures versus catalog validation and runtime qualification; migration and operational limits. Verify important source anchors locally. Discover available code-intel schemas and call Serena initial_instructions before symbol queries when available; otherwise use source reads and disclose missing integrations. Do not launch services, rebuild a graph or treat a stale snapshot as current. State each skipped lens and why.

Inspect the full task range, not just the last commit. Read changed files/callers when necessary to test a concrete claim. Run only authorized checks under the project resource lock and budgets, or explicitly say source-read-only; never imply command execution.

## Return

Report exact source/root/revisions and input hashes, criteria checked, commands actually run with exits/logs, skipped checks, strengths, spec compliance and quality verdicts, canonical findings with source IDs/severities/evidence/rationale, residual risks and next action. Findings are structured data, not a prose substring gate. Return the full report; a path alone is a pointer the parent must read.

## Review disposition and authority

Normalize by consequence to **Critical / Major / Medium / Low**: catastrophic security/data-loss impact; substantial requirement/correctness/safety failure; bounded material defect; or presentation/low-impact improvement, respectively. Preserve original severity and every source finding ID under a canonical defect ID, with location, evidence, consequence rationale, proposed correction and unresolved status. Never mechanically convert Important to Major or Minor to Low, or downgrade severity to fit a threshold. A reduction needs concrete consequence evidence and a recorded parent disposition.

Use one fresh independent review and at most one bounded correction/disposition, not automatic two-cycle rituals. Unresolved Critical/Major findings block; Medium residuals need explicit owner acceptance. Preserve source finding IDs, consequences and old/new hashes. Required tests, authority, evidence and requirement coverage independently block. Corrected bytes are not retroactively independently reviewed; stop at the bound and return unresolved decisions to the parent.

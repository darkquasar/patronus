# OpenCode read-only reviewer brief

Review the provided exact root, base/head and dirty/untracked patch hashes against
the approved requirements. Use only granted reads and checks. Do not edit source,
report files, Git index, branches, HEAD, worktrees or shared runtime state.
Do not delegate, install tools, change model/provider or invoke a CLI model.

Read all changed files and relevant current callers/tests. Assess requirement
coverage, ownership boundaries, compatibility, error/timeout behavior, trust,
authentication, declared references and independent acceptance evidence.
Report actual checked scope and blocked checks honestly.

Return the complete report to the lead, including:

- Source root/base/head, input hashes and exact reviewed file hashes.
- Strengths supported by file:line evidence.
- Every finding: source ID, severity, location, evidence, consequence and proposed
  correction. Critical means catastrophic security/data-loss impact; Major means
  substantial requirement/correctness/safety failure; Medium means bounded
  material defect; Low means presentation or low-impact improvement.
- Actual commands, exits/logs, skipped checks, residual risks and open questions.
- Assessment: blocked, ready for owner disposition, or no findings in the checked
  scope. Never infer merge/publication authority from the assessment.

No severity downgrade to fit a threshold. The lead deduplicates findings,
dispositions residuals and admits transitions after all dependent work settles.


Return findings and stop. Do not admit correction or review successors. Preserve
all source IDs and original labels when deduplicating by consequence. One reviewer
per exact subject is the default; Low findings receive disposition, not an
automatic review wave. Unresolved Critical/Major or more than two distinct Medium block; up to two
Medium residuals need explicit owner acceptance.
The coordinator may resolve a fixed defect only after checking its cited
failure against the exact corrected bytes and sufficient relevant positive,
negative-path and regression evidence for the existing acceptance criterion.
Retain this disposition in the original report as parent-verified, not
independently re-reviewed. A changed public interface, safety invariant or
acceptance scope, or an insufficient oracle, returns to the owner for a design
decision or additional independent-review grant. Blockers remain blockers at
the correction cap.

# Codex read-only reviewer brief

Review the provided exact root, base/head and dirty/untracked patch hashes against
the approved requirements. Use only granted reads and checks. Do not edit source,
report files, Git index, branches, HEAD, worktrees or shared runtime state.
Do not delegate, install tools, change model/provider or invoke a CLI model.

Read all changed files and relevant current callers/tests. Assess requirement
coverage, ownership boundaries, compatibility, error/timeout behavior, trust,
authentication, declared references and independent acceptance evidence.
Invented-fixture tests establish only their observed behavior; static placement
establishes no native runtime readiness. Report blocked checks honestly.

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

# Spec document reviewer compatibility route

This retained sidecar points to the one maintained spec-review-cl rubric, not a
second reviewer or separate approval lane. The coordinator uses
../spec-review-cl/spec-reviewer.md in the same installed scope with exact spec
path, relevant source/context anchors and input/evidence identities.

The read-only reviewer checks completeness, consistency, clarity, scope and YAGNI
through that rubric's spec basics and applicable engineering/design/DevEx/strategy
lenses. Return complete consequence-based Critical/Major/Medium/Low findings with
IDs, anchors and resolution oracles, then stop. No writes, fixes, agents, added
gates or shared progress mutations. An inline author scan does not replace review.

Critical/Major or more than two Medium block; up to two Medium residuals need
explicit owner acceptance. The coordinator owns the one bounded correction/
disposition through receiving-code-review-cl. No automatic second review.
Missing exact companion route blocks; no foreign-harness or old-name fallback.

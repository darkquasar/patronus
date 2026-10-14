# Spec document reviewer compatibility route

This sidecar belongs to the one independent spec-review subject, not another
reviewer or approval lane. The coordinator supplies the exact spec path/hash,
source anchors, accepted brief and evidence identities, and uses
../spec-review/spec-reviewer.md in the same installed scope.

The read-only reviewer returns complete Critical/Major/Medium/Low findings with
IDs, source evidence, consequences and resolution oracles, then stops. No writes,
fixes, delegation, shared progress mutation or successor admission. An inline
author scan does not replace independent review.

Unresolved Critical/Major or more than two distinct Medium block; up to two Medium
residuals require explicit owner acceptance. The coordinator owns one bounded
batched correction/disposition. Corrected bytes are parent-verified, not
independently re-reviewed; changed interfaces, safety invariants or acceptance,
or insufficient proof require an owner decision. No automatic second review.
Missing exact companion route blocks; no foreign-harness fallback.

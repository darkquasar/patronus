# Codex delivery process observations

Recorded 2026-10-06 at the owner's request. These are observations and proposed future validation, not a new implementation plan, workflow project or task ledger. No optimization experiments were run for this note.

## What we over-invested in

- Coordination between implementation waves: wave1 completed at06:43:56 and wave2 launched at07:20:56, approximately37 minutes later. Some scope/isolation checks were necessary, but the coordinator spent too long assembling and explaining the next packet instead of dispatching ready work.
- Four fresh agents per two-writer batch. Separate capture/integration and test-verification roles repeated substantial instruction, report and context reads. They provide useful checks, but repeatedly paying that overhead for small batches was disproportionate.
- Repeated preflight and evidence narration after the same pinned runtime, provider routes and ownership boundaries were already established. Durable identities and material drift checks matter; reproducing the full qualification story at every step does not.
- More red evidence than needed: the helper writer recorded absent-helper failures, wrapper failures and an additional mutation-based red run. A short module-level safety cycle would have been sufficient. The Go wrappers also repeat parts of the Python fake-worker coverage.
- Brittle bespoke evidence checks. My unanchored failure regex matched the passing test name `test_retry_preserves_failed_attempt`, blocking a successful validation stage. This was a coordinator harness defect, not a product-test failure.

## Where the failure was

The coordinator confused elaborate preparation and verification records with delivery progress, failed to keep the largest useful ready set moving, and imposed too much handoff overhead. I own that failure. The owner's repeated instructions to stop unnecessary skill loading and granular TDD should have changed the execution style earlier.

Test execution itself was not the main elapsed-time cost in the last batch: adapter/registry/CLI Go packages took approximately7.5 seconds and nine fake-worker cases took10.042 seconds. Test authoring, model/tool latency, context reads, handoffs and coordinator pauses are separate costs. These timings do not establish a general model-speed comparison or quantify each cost category.

The isolated supervisor is a requested product feature, not a general TDD framework. Its real timeout/descendant/ownership tests remain important. Likewise, native Codex authentication/runtime qualification is independently blocked; reducing local process overhead does not resolve the CLI401 or prove native hook/trust behavior.

## What to trim in future delivery

- Reuse settled preparation and qualified-runtime evidence until a relevant input actually changes.
- Batch independent work under measured concurrency. Serialize expensive checks, not all authoring.
- Make handoffs short: exact scope, predecessor bytes, exclusive paths, acceptance criteria and stop conditions. Reference existing evidence rather than narrating it again.
- Use short module-level red/green for critical migration, auth preservation, guard and process-settlement behavior. Add edge cases to the same suite. Routine content/manifests/routing get focused postchecks.
- Avoid broad suites after every small edit. Integrate a useful aggregate, run one broader gate, then one independent final review and at most one bounded correction.
- Prefer existing test output/status conventions to additional custom evidence machinery. Keep required hashes and receipts, but do not turn them into another scheduler or mirrored ledger.
- Handle trivial routine changes directly when this does not violate file ownership. Reserve delegated specialists for useful independent work.

## Proposed future research/validation, not performed

- Measure dispatch-to-first-edit, active implementation, context-read overhead, idle coordinator time, capture, integration, validation and review separately using retained native timestamps.
- Compare equal-scope deliveries with small batches versus a larger bounded batch. Hold requirements, checks and review quality constant; do not infer model performance from different tasks.
- Assess whether wrapper tests add distinct coverage or merely rerun the same fixture. Remove duplication only after verifying the acceptance boundary remains covered.
- Evaluate a minimal aggregate gate using standard test summaries. Include passing test names containing words such as `failed` and actual failure/skip summaries as parser controls.
- Audit mandatory versus incidental context reads and retained-evidence reuse. Keep authority, destructive-action consent, ownership and native-runtime qualification gates intact.

No package, workflow or harness optimization is authorized or implemented by this note. Native mission records remain execution authority.

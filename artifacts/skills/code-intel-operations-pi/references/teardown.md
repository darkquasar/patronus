# Settlement, manual reset and retention

This is an operator checklist, not a process controller or config mutator. These
steps apply to private MCP leaves and manual role fields. Installing/removing a
profile does not run them or grant cleanup authority.

## TD-1 — Prove successful settlement first

Coordinator accounts for every affected run/session, descendant, client and
output, including all consumers of a global change. Require successful settlement
and readable hash-bound results/logs before research reset or removal. Active,
unknown or failed status, missing rows, unknown descendants and unacknowledged
cancellation mean **retain resources and escalate**. Cooperative cancellation is
not proof of exit; use approved deadlines, recorded process identity (including
start identity to avoid PID reuse), adapter/runtime evidence and bounded owner
observations. Do not kill by guessed PID or infer exit from a marker.

An HTTP client closes only its connection. It does not stop shared Serena,
Graphify or their language servers; shared lifecycle belongs to the coordinator
with all-client accounting. A private stdio server/LSP belongs to that writer's
adapter/runtime, which owns shutdown. Observe actual shutdown/descendant settlement
before checkout deletion or reuse. Unknown status retains the worktree, config,
logs and process evidence. The example model cannot prove any of these events.

## TD-2 — Check record, grant, scope and unchanged fields

Require the original ownership record and bytes, exact root/run/owner, matching
current reset grant, full active-source inventory and successful TD-1 evidence.
An absent record/marker, missing grant, different owner, higher-precedence source,
unreadable config, unknown original baseline or changed managed field/hash blocks
**the whole proposed reset before any write**. Compare every selected current
value with its last recorded managed value/digest. Re-read just before mutation;
retain/report conflicts, do not adopt them. Unrelated changes alone are preserved,
not rolled back from a whole-file backup. Newly discovered consumers block reset.

Explicitly reject the installed prototype helper's **reset-without-marker**
behavior: removing a matching Serena entry despite a missing ownership record is
unsafe, even if its command/project appears familiar. Do not invoke or advertise
that helper as safe cleanup. A marker alone also cannot prove unchanged bytes,
settlement or deletion authority. The delivered descriptor is external operations
evidence, not new installer state or a trust bypass.

## TD-3 — Restore only recorded unchanged fields manually

After all checks pass, preview restoration per field. `original.present:true`
restores its exact JSON value, **including null**; `original.present:false` deletes
only that field. Preserve unrelated roles, settings and MCP entries, including
entries added since bootstrap. Keep original file bytes as evidence, not a blanket
restore command that discards unrelated edits. Empty parent objects may remain;
do not remove a file/directory just because the originally absent leaf was removed.

Same-owner updates must have carried the first baseline forward with history,
never rebased it onto a prior managed value. Check last managed digests, restore
first originals. Record resulting bytes/hashes and per-field completion in history
without discarding the baseline/last-managed evidence. If interrupted, retain both
records and current bytes, hold dispatch and have the owner reconcile which fields
changed; never blindly replay reset, infer success or erase the record.

## TD-4 — Reload, verify research state and retain evidence

Reload/restart all affected clients and acknowledge the actual effective config
and role/tool/skill/provider plan. Observe restored base behavior and absence of
the private override from every active source before research dispatch. Failed or
unknown reload keeps the worktree blocked for research. Stale clients are not
restored merely because disk bytes changed. Preserve receipts, original backups,
digest/history, outputs and settlement logs under the owner's retention policy.

Profile/item uninstall removes only unchanged Patronus-owned static artifacts and
recipe leaves under normal drift rules. It cannot kill services, remove operator
binaries/auth/caches, erase manual settings or delete worktrees/spill files. Manual
overrides survive profile uninstall pending explicit cleanup; settle and reset
before removing skills those roles still require. Worktree/data deletion needs a
separate owner retention/export/deletion grant after these checks. No automatic
prune, inverse profile-lifetime controller or deletion from a marker alone.

## Example checks versus runtime evidence

The adjacent `fixtures/manual-lifecycle.test.mjs` loads delivered examples and
binds assertions to RO/WB/TD step IDs. Its in-memory operations are **independent
example models**, not shipped enforcement, installer tests or upstream tests. It
uses invented values/runs/grants and writes no configs or processes. It checks
absence/null/present restoration, baseline-carrying updates, preservation and
blocked ambiguity; it cannot establish actual process settlement, approval,
provider loading or hard containment. I-T4/I-T6/I-T7 runtime observations and
integration-only I-T3 remain separately granted QP-03 work.

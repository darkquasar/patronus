# Pi release decision checklist

> **Historical / superseded.** This checklist applies to the former static-delivery evidence set and does not authorize or describe the current Pi-owned npm lifecycle.

Historical decision: **HOLD publication and formal runtime support labels**. The former bounded functional suite passed within its stated scope but did not complete the exhaustive release matrix. See the [historical support matrix](support-matrix.md); raw observations remain in the operator evidence archive, not as a current runnable program.

## Development closeout

- [x] Retain the eighteen accepted implementation tasks and exact input/build records.
- [x] Run actual cold startup, delivered ticket operations, native child/workflow mechanics, failure/denial/cancellation, lifecycle and web checks.
- [x] Keep scripted responses, invented network transport, live DDG observations and security claims separate.
- [x] Preserve failed attempts and successful exact-container settlement/cleanup.
- [x] Keep reusable tests separate from CI policy; no unrequested CI wiring retained.

The operator-owned closeout record lives under `docs/specs/01-pi-harness-adapter/execution/`. It records scoped task acceptance and the final review separately from formal qualification. Raw evidence and preprovisioned inputs are machine-local and intentionally ignored by Git.

## Before a formal claim

- [ ] Select each claim independently: `static-delivery`, `core-exact-pin`, `code-intel-exact-pin`, `web-functional`, `unattended-mutation`, `web-bounded-unattended`.
- [ ] Import and hash every applicable requirement/case result, exact dependency dossier, effective configuration, payload and environment in the selected evidence root. Read the underlying logs; do not turn unknown or failed outcomes into passes.
- [ ] Supply the missing exhaustive core role, trust, worktree, approval/bypass and lifecycle evidence. Scope optional MCP cases explicitly. Web hard bounds remain mandatory for the stronger unattended-web claim.
- [ ] Record independent review of the exact claim packet and owner dispositions. A review of this limited closeout cannot approve missing runtime evidence.
- [ ] Run the unchanged checker against the final packet and inspect its findings:

```sh
python3 -B scripts/check-pi-evidence.py qualification \
  --root "$EVIDENCE_ROOT" --file "$RELEASE_RECORD" --claim "$SUPPORT_CLAIM"
```

The current incomplete release record is expected to return **BLOCKED**. A valid record structure or a passing smoke test is insufficient for release admission. Do not alter the checker to manufacture acceptance.

## Before publication

- [ ] Obtain separate owner publication authority.
- [ ] Resolve applicable provenance/signature, advisory/currency, notice/license, generated-source and redistribution decisions for the exact staged bytes.
- [ ] Promote only the selected CP-05 payloads; verify final immutable HTTPS endpoints, digests and recipe provenance. No unqualified rebuild or placeholder endpoint may substitute.
- [ ] Revalidate any changed URL, payload, configuration or host pin.
- [ ] Record operator-owned activation, consumer quiescence, rollback and retention choices. Preserve external Pi, sessions, credentials, caches, optional services and unowned user data.

Neither task acceptance, documentation review nor a checker exit status authorizes publication, acquisition, activation, an upgrade, live-harness replacement or cleanup outside the named test namespace.

# Source and invocation inventory

Neutral isolated-writer supervisor, version 1.0.0. The helper, schema and
fixtures are Patronus-authored (first introduced in `plan-execute-parallel-cx`
1.1.0, settlement corrections in 1.1.1) and were moved here unchanged from
`artifacts/skills/plan-execute-parallel-cx` 1.1.1 at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. `SKILL.md` is authored for this
artifact; the original attribution (`NOTICE`, `LICENSE`) is copied. The
supervisor accepts a worker argv already; no harness-specific code was added.

## Moved files (byte-identical)

| File | SHA-256 |
|------|---------|
| `scripts/supervisor.py` | `fde4bf79ead669fd871e16b4e0b6b21ae8e732fba53d1ce6ca418d1ee7d0d85c` |
| `fixtures/fake_worker.py` | `7fa9273b817a6743998ffd29b09b0e6995be0ac2de4deccf86fd46e4562edbeb` |
| `fixtures/test_supervisor.py` | `717c6bb911e60e4b939082b90c75d77212576715ef601be30664869293c5bc57` |
| `references/request.schema.json` | `d5a83f1c5960e8e6f0083c1e4e0939cd9e3463f98fbd07c640f4f38365123d32` |
| `NOTICE` | `f06979caadf3af300ba27c87dee139bedad4f0a86caf2cd5b37884ef1b955714` |
| `LICENSE` | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |

Only the entry and the manifest-declared sidecars are distributed. The
fake-worker suite runs invented local processes only; it is not native
qualification of any worker, harness, authentication or sandbox.

## Historical fixture result

Run once from this location on 2026-10-09 (UTC), not part of any development
gate:

```text
python3 artifacts/skills/isolated-writer-supervisor/fixtures/test_supervisor.py
Ran 12 tests in 10.789s
OK
exit 0
```

This is historical evidence of the fixtures at the recorded revision. It is
NOT RUN in the application or catalog gates and does not mark any native row
passed.

## Migration

Existing 1.1.x installs keep ownership of the old sidecar paths; `patronus update` reports
`obsolete owned path retained; explicit removal required` for them. Run
`patronus remove plan-execute-parallel-cx` then reinstall, or remove the listed paths
explicitly, before relying on the shared supervisor.

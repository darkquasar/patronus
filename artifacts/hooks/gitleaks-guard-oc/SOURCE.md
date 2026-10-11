# Provenance and limits

`gitleaks-guard.py` is a byte-identical copy of the local
`artifacts/hooks/gitleaks-guard-cx/gitleaks-guard.py` at source revision
`13498bb86196905a96b81be1e9632311d9f862d1`.
SHA-256: `172606b6a0ca3cae4f5ca16a5c32c9fe1c08baa1057a6fb581f4d64d312473e5`.
The retained diagnostic prefix may name the source lane. Byte identity is intentional.
The manifest preserves the `gitleaks` dependency and Claude-shaped Bash matcher;
OpenCode delivery uses the adapter's local runner and `hooks.json`.

The guard is best effort and does not protect reading or egress. Missing scanner
prerequisites and scanner errors block. Native OpenCode/Bun loading and staged
commit blocking remain runtime-pending. `--pure` disables the runner. Both scopes
can run matching guards twice. Restart OpenCode after placement changes.

## One-time deterministic shell check

Observed 2026-10-11: the real script was copied byte-identically to an isolated
shell fixture. Node v22.22.1 (official Linux arm64 build, executable SHA-256
`1ea16f5e8d590ca819f1db81e61a6bc4c8e7d8b69776c9236f8ebbf717113225`)
loaded the `.ts` runner with `--experimental-strip-types`. A public-CLI dry-run
supplied the adapter dialect; deterministic token substitution into the final
runner template supplied the fixture module.

Command: `python3 <evidence>/real-guards.py <qualified-node> <worktree> <evidence>`.
The shell fixture invoked `node --experimental-strip-types run.mjs <runner.ts>
<payload.json> <fixture-directory>`. An invented Bash non-commit command proceeded;
a `git --config-env=fixture commit` payload blocked with the real guard's
unsupported-context diagnostic. The driver exited 0 for each expected outcome;
combined fixture exit: 0. No Git commit or scanner was executed. This check does
not qualify staged-secret detection or scanner prerequisites.

Durable record: `opencode-resume-output/wave1/oc-t5-evidence/real-guards.log`;
fixture source: `real-guards.py` in that directory. Runner template SHA-256:
`c29a3fb11b1203b9ddb9db842a3a04c45b3d1e8c2e0dd1f5883a01263b060a1c`.
Rendered runner SHA-256: `b37cd4c5f5721644de3e382f8cc3a9ccc9d974fa7f700b11654f59ecad854005`.
This is historical local Node evidence only, outside Go app/catalog tests.
No native activation or parity claim is made.

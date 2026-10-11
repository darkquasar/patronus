# Provenance and limits

`block-secrets.py` is a byte-identical copy of the local
`artifacts/hooks/block-secrets-cx/block-secrets.py` at source revision
`13498bb86196905a96b81be1e9632311d9f862d1`.
SHA-256: `cfd78d4f424b39a7577f52f8a41d1c515ac28a67681b98ca786369dae5a46fa5`.
The retained diagnostic prefix may name the source lane. Byte identity is intentional.

The hook manifest adapts delivery to OpenCode via the adapter's local plugin
runner and `hooks.json`. The runner translates `apply_patch.patchText` to
`tool_input.patch` and `edit.newString` to `tool_input.new_string`.

Guard behavior is best effort. Reading and egress are not protected. Native
OpenCode/Bun loading, event payloads and blocking remain runtime-pending.
`--pure` disables the runner with all external plugins. Installing both scopes
can run each matching guard twice. Restart OpenCode after placement changes.

## One-time deterministic shell check

Observed 2026-10-11: Node v22.22.1 (official Linux arm64 build,
executable SHA-256 `1ea16f5e8d590ca819f1db81e61a6bc4c8e7d8b69776c9236f8ebbf717113225`)
loaded the `.ts` runner with `--experimental-strip-types`. A public-CLI dry-run
supplied the adapter dialect; deterministic token substitution into the final
runner template supplied the fixture module. No installation or native host was
launched. The placed guard copy was checked byte-identical before execution.

Command: `python3 <evidence>/real-guards.py <qualified-node> <worktree> <evidence>`.
The shell fixture invoked `node --experimental-strip-types run.mjs <runner.ts>
<payload.json> <fixture-directory>` for each case. Benign `apply_patch.patchText`
containing an Add File envelope proceeded; the same envelope with an invented
`ghp_` token blocked with the guard's stderr. `edit.newString` with the token
also blocked. The driver exited 0 for each expected outcome; no payload was
echoed. Combined fixture exit: 0.

Durable record: `opencode-resume-output/wave1/oc-t5-evidence/real-guards.log`;
fixture source: `real-guards.py` in that evidence directory. Runner template
SHA-256: `c29a3fb11b1203b9ddb9db842a3a04c45b3d1e8c2e0dd1f5883a01263b060a1c`.
Rendered runner SHA-256: `b37cd4c5f5721644de3e382f8cc3a9ccc9d974fa7f700b11654f59ecad854005`.
This is historical local Node evidence only, outside Go app/catalog tests.
It is not native activation or a parity claim.

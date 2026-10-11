# Codex core qualification procedures

This directory contains procedures, an acceptance matrix, an inert record template
and a hash-bound unchanged-reuse audit. None is a native qualification receipt.
The current milestone has no qualified Codex version/platform pair. Retained
root observations were made for CLI 0.149.1 on Linux arm64; authentication and
current native acceptance remain pending. A retained separately approved CLI smoke
failed401 and was stopped; native Pi provider access does not qualify Codex CLI.

Read `acceptance-matrix.md` before a separately authorized native run. Record all
attempts, including failed, blocked and not-run cases. Missing auth, worker tools,
trust or a required test remains blocked, never a pass. Do not infer native
readiness from catalog tests, static placement or a dispatch receipt.

## Safe run preparation

1. Obtain a grant for the exact version, platform, profile revision/hash, scopes,
   isolated test HOME/CODEX_HOME/project roots, native model/tool calls and budget.
   Package fetches, auth, trust changes, integration and cleanup are separate
   actions. A development test grant does not authorize them.
2. Retain the source base/head, dirty patch hash, profile/ledger hashes and full
   selected closure with file hashes. Read each dependent patch/handoff report.
3. Observe `codex --version`, platform/architecture and effective root decisions
   without opening credential files. Login must be performed by the owner
   through the native CLI. Retain only sanitized auth status, never tokens,
   headers, cookies or copied credentials.
4. Create only grant-bound isolated roots. Prepare invented co-visible skills,
   mixed instruction files, config auth references and harmless test payloads.
   Never probe developer-global resources or write native trust internals.
5. Capture the normal Patronus preview and output before deployment. Recheck
   changed environment, roots, profile selection or hashes. Keep failures and
   partial files for the owner; do not force, migrate or retry in place.
6. Restart Codex into a fresh session for every install/update/remove transition.
   A retained session may have old skill/instruction bytes even after removal.

Use `record-template.json` as an inert starting shape. Bind every log by relative
path and SHA256 beneath the owner-selected durable evidence root. Hash both raw
sanitized output and any summary; list actual commands, exits, dates and observed
nonces. Never change a pending row to passed without its recorded observations.
Keep results out of public MDX and local specs out of tracked navigation.

## Static integration gates

Development checks are the two gates described in `CONTRIBUTING.md#test-gates`.
Locally run them under the project lock with umask 022 and offline Go caches.
These are procedures, not claims that they were run here:

```sh
umask 022
export GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local PYTHONDONTWRITEBYTECODE=1
flock /home/agent/.cache/patronus-codex-tests.lock bash scripts/tests/application-contract.sh --out "$(mktemp -d)" --without-catalog
flock /home/agent/.cache/patronus-codex-tests.lock bash scripts/tests/catalog-contract.sh --out "$(mktemp -d)" --base <actual-base>
```

The application gate uses invented catalogs on a scratch copy without
`artifacts/` and `profiles/`. The catalog gate checks the real source with the
standalone `tools/catalog-check` module and public-CLI locks, including
`core-profile-cx--codex.lock` for ledger coverage. Earlier named-test commands
in this directory's history are historical evidence, not the current gates.
Neither gate runs the guard or supervisor fixtures: their results are
HISTORICAL and NOT RUN, and the native rows below stay runtime-pending. Test
validation is not an independent implementation review. A package with no
matching tests is not a pass.

## Implemented source and remaining qualification

- `patronus migrate codex-skills` previews checksum/identity/scope-proven legacy
  skill relocation; deploy requires explicit selected scope. Shared verified
  destination writes and staged ownership precede source retirement. Failure
  preserves old data/ownership; failed staged-save destinations require manual
  reconciliation. No live global migration was performed.
- The bundled parallel supervisor has twelve retained fake-worker cases for
  independent/dependent writers, timeout, descendants,
  result/ownership/output/conflict/retry, input-apply failure, supervisor faults
  and unknown-stop refusal. Their results are historical and not run now. Git
  isolation is cooperative, not an OS sandbox. Hard kill/host loss, escaped
  processes, repeated interruptions and stalled external operations remain limits.
- Codex-only write and staged-commit guards retain invented apply_patch/Write
  alias and Bash/scanner fixtures (historical results, not run now). Actual payloads, native registration/trust
  and nonce behavior remain pending. SessionStart hooks remain omitted; advisory
  instructions do not replace callbacks. No real-secret probes are authorized.
- Shared instruction removal and transport-leaf MCP lifecycle have public CLI
  preservation/failure tests. Missing explicit auth/env references produce
  advisory presence-only warnings; native initialization/auth and fresh sessions
  remain pending. Empty preexisting server-table proof remains limited by existing
  per-leaf metadata; other user containers are preserved.
- Seven added design/diagnosis/TDD/worktree/completion/meta-skill/Cloudflare bundles
  have source route/sidecar checks. Native invocation is unqualified. Editorial
  loading/port is explicitly excluded, not future parity claimed by this stage.

Use the full acceptance matrix for separately authorized native observations;
local fixture success does not change its runtime-pending rows. Record both
scopes and all failed/blocked attempts. The owner admits release only after
required observations and independent review are hash-bound. Source delivery
alone does not authorize publication or cleanup.

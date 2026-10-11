# OpenCode core: bounded smoke, optional full qualification

Inert human-run procedure, **not executed**. Proposed runtime: `opencode-ai`
**1.18.35, Linux arm64**; no version/platform/scope is qualified. Source placement,
Node fixtures and host gates do not prove native Bun, guards or permissions.

## Default: reuse one prepared environment

Obtain a **separate future execution grant** binding source/base/head, public
profile/ledger/closure hashes, executable/image identity, isolated roots,
proxy/CA inputs, model/provider, network/install actions, budget and settlement/
cleanup. Missing prerequisites block; no implicit pull/install, alternate model,
version or in-place fallback. Never weaken production permissions to fit a budget.

Use one isolated **global** deployment with a scratch project, no project-scope
installation or foreign context. Bootstrap below is one-time preparation, not
part of each smoke. Reuse only an unchanged, hash-bound prepared environment:
confirm actual image/module/executable, Linux arm64, Git/Python3/bash/CA/scanner
availability, effective roots (`HOME`, XDG and `OPENCODE_CONFIG_DIR`) and public
payload hashes against the grant. Stop on drift. Do not rebuild catalogs, rerun
host gates, fetch scanners or duplicate scopes during a prepared smoke.

Operational cap: **3 model-backed native case invocations, 90 seconds each,
5 minutes total** (including local discovery). Each invocation is one foreground
prompted session; internal tool turns share its deadline, not extra prompts.
This caps native cases/time, not provider inference round trips or cost. Target
minutes, not measured performance. No subagent/model chain, retries, follow-up
prompts or background supervisor. Extra cases or deadline overrun block; the
future operator must enforce deadlines and retain timeout/unknown-stop evidence,
not silently clean up.

1. **Discover, no model call:** in a fresh session observe `opencode --version`,
   `opencode debug skill`, `opencode agent list` and `opencode debug config`.
   Capture exact selected skill/role/runner identities and permission facts only.
   Never persist raw debug config: stream through a reviewed fail-closed redactor
   dropping all `provider` data (including `provider.*.options`), allowlisting only
   nonsecret permission/agent facts. Review user prompt/model metadata before
   saving. Redactor/pipe/parse errors block, not empty passes.
2. **Request 1, skill + benign tool:** explicitly load `verification-before-completion-oc`
   through native `skill`, then write a harmless invented scratch file using the
   native write/edit tool. Record tool identity, outcome and actual file change.
3. **Request 2, guard negative:** request an invented fake-secret `.env` write via
   native `apply_patch`. Observe the actual runner plus `block-secrets-oc` refusal
   and absence of the unintended write, not a model's refusal alone. Sanitize
   payloads. Missing registration/tool, no attempted call or allowed write blocks
   smoke; do not retry with different tools or real secrets.
4. **Request 3, read-only negative:** invoke `@patronus-researcher-oc` directly to
   return attributed findings on the benign file and attempt native edit of that
   file. Require native permission denial and unchanged bytes. No nested
   delegation or built-in role fallback; bash scope is cooperative, not OS isolation.
5. **Record and stop:** save SMOKE passed/failed/blocked plus sanitized exits,
   session IDs, positive/negative observations and public before/after hashes.
   A passed smoke is partial evidence for N01/N02/N08/N09/N10 only, **never full
   qualification**. All incomplete full rows stay runtime-pending/not-run;
   `releaseAccepted` stays false and `qualifiedVersionPlatforms` stays empty.

Discovery redaction (future grant only; never tee raw input or stderr into logs):

```sh
set -o pipefail
opencode debug config | python3 -c '
import json, sys
v = json.load(sys.stdin)
if not isinstance(v, dict): raise ValueError("invalid debug config shape")
v.pop("provider", None)
print(json.dumps({k: v[k] for k in ("permission", "agent") if k in v}))
'
# Review the allowlisted stream for user secrets before saving; errors block.
```

## Optional: targeted full qualification

Use `acceptance-matrix.md` to select follow-ups under their own bound grant; a
full acceptance attempt requires **all 17 rows in both isolated scopes**, plus
N17's explicit combined-scope session. Reuse setup and evidence identity, not
observations across changed inputs. Fresh sessions are mandatory after every
install/update/remove. No research/spec/plan/execution chain is required for smoke.

Deferred groups: complete discovery/routes/closure (N01..N05/N08), masks and MCP
auth (N06/N07), complete permission and fail-closed guard cases including staged
secret commit (N09..N11), worker result/outside-write/settlement (N12/N13), `--pure`
(N14), lifecycle/migration/failure injection (N15/N16), double load (N17).
Retain every required positive/negative control in the matrix; omitted means
not-run, never passed. Guards are user-disableable best effort, not read/egress
protection. `--pure` disables external plugins including guards; lack of a block
there is expected disablement, not protection. Patronus never sets
`OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1`.

Known unresolved blockers, not fixed or waived here: **OC-G9 (Major)** accepts
unrelated unrecorded unprefixed context sections despite the broader refusal
requirement (recorded foreign owners/unknown selected sections are refused);
**OC-WC-002 (Medium)** can turn worker validator exceptions into precheck success.
Acceptance also awaits complete independent whole-change semantic review. Global
scope/separate roots mitigate mixed context, not waive it. Unknown worker
settlement/descendants require retention, never inferred cleanup authority.

## One-time optional container bootstrap (separate grant only)

Replace placeholders with approved absolute inputs. Historical image/module
presence is not proof of current availability; inspect before launching, no pull.
Mount only public hash-bound catalog source and tools read-only, never a writable
host project/HOME. Verify nonsecret module metadata is 1.18.35 before native calls.

```sh
docker image inspect node:22-bookworm
docker run --rm -it --name <grant-bound-container> --platform linux/arm64 \
  -e HTTP_PROXY -e HTTPS_PROXY -e NO_PROXY \
  -v <proxy-CA>:/usr/local/share/ca-certificates/sandbox.crt:ro \
  -v /usr/local/share/npm-global/lib/node_modules/opencode-ai:/opt/opencode:ro \
  -v <built-patronus-binary>:/usr/local/bin/patronus:ro \
  -v <hash-bound-catalog-source>:/opt/catalog:ro \
  --tmpfs /root:exec,size=512m node:22-bookworm bash
# Inside only:
update-ca-certificates
ln -s /opt/opencode/bin/opencode /usr/local/bin/opencode
opencode --version
uname -s
uname -m
mkdir -p /root/.config/opencode /root/scratch/project
# Host second terminal, only explicitly authorized config transfer:
docker cp <authorized-opencode-config> <grant-bound-container>:/root/.config/opencode/opencode.json
# Inside again:
cd /root/scratch/project
git init
cp -R /opt/catalog/artifacts /opt/catalog/recipes /opt/catalog/profiles /opt/catalog/adapters .
# Include public packages/ only if selected build inputs require it.
patronus install --profile core-profile-oc --target opencode --global --local-registry --dry-run
# Only after admission and exact deploy/network approval:
patronus install --profile core-profile-oc --target opencode --global --local-registry --deploy
```

Deployment may fetch pinned gitleaks: explicit network/install authority and actual
scanner identity are prerequisites, not part of the default smoke. Stop if absent.
Use `docker cp` only for authorized container config, never print/hash/archive it
or copy it into source/evidence. Do not read/copy auth.json, mcp-auth.json,
OPENCODE_AUTH_CONTENT or provider tokens/headers/cookies. No raw config, inline
config or broker sentinel values in logs; no credential hashes or proxy dumps. This is not a transfer grant
elsewhere. OC-N07's user-managed MCP auth remains a separate follow-up.

## Compact evidence and boundaries

Copy `record-template.json`: shared source/grant/environment once, one observation
per actual case/session/scope, with sanitized command/exit, expected/actual result,
negative control, public hashes and relative log path/SHA256. Missing observation
means not-run; sample is not evidence. Record partial smoke coverage without
marking the corresponding full row passed. Failed/blocked attempts remain visible.
Full-row pass needs all matrix observations, zero unintended writes on refusals,
review and owner disposition. Integration, publication and cleanup are separate
grants. Host application/catalog/lint gates and independent review remain separate
acceptance dependencies; none execute this procedure or qualify native behavior.

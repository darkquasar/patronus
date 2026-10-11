# OpenCode core qualification procedure (not executed)

This is an inert procedure, not a qualification receipt. Every OC-N01..N17 row
in `acceptance-matrix.md` starts **runtime-pending**. The only proposed first-run
matrix is `opencode-ai` **1.18.35, Linux arm64**, both isolated scopes. Source
placement, Node fixtures, catalog gates and independent code review cannot
qualify OpenCode's bundled Bun, model/tool behavior or native permissions.

## Authority and current preflight

Before any command below, obtain a separate grant binding exact source/base/head,
profile/ledger/public-closure hashes, executable identity, platform, isolated
roots, image identity, proxy/CA inputs, permitted model/provider, scope, budgets,
network actions and settlement/cleanup. This implementation stage grants no
native procedure execution. A missing prerequisite blocks the run, not permits
an image pull, package install, alternate version/model or in-place fallback.

Historical source paths `/usr/agentuser/workspace/patronus-phase2-integration` and
`/usr/agentuser/workspace/patronus-opencode-lane`, the 2026-10-09 observation of
`node:22-bookworm` image `0e5f90657369`, and the mounted module location below
are provenance only. Verify actual paths and image availability in the run's
host. Do not assume this sandbox still has that image or executable.

Under that separate grant, preflight:

```sh
docker image inspect node:22-bookworm
# Observe the actual OS/architecture and image ID; never pull implicitly.
# Verify this actual read-only module is version 1.18.35, not a rolling install:
# /usr/local/share/npm-global/lib/node_modules/opencode-ai
```

Read only nonsecret package metadata to check the module version. Observe
`opencode --version` and Linux arm64 inside the container before any model case;
record the executable SHA256, not a credential-bearing config hash. Stop on any
mismatch. Confirm Git, Python 3, bash, CA support and the approved scanner asset
are available; missing tools require a decision, not an implicit install. Hash
selected public source/placed nonsecret payloads, retaining actual path identities.
Use `record-template.json`, preserving failed, blocked and not-run attempts.

## Disposable container setup

Replace placeholders only with separately approved absolute paths and inputs.
Mount the OpenCode module and built Patronus binary read-only. The optional
read-only catalog-source mount below must be the exact hash-bound source being
qualified; it allows local registry resolution without silently fetching a newer
remote catalog. Keep credentials/config out of that source tree and evidence.
No host project or real HOME is writable through these mounts.

```sh
docker run --rm -it --name <grant-bound-container> --platform linux/arm64 \
  -e HTTP_PROXY -e HTTPS_PROXY -e NO_PROXY \
  -v <proxy-CA>:/usr/local/share/ca-certificates/sandbox.crt:ro \
  -v /usr/local/share/npm-global/lib/node_modules/opencode-ai:/opt/opencode:ro \
  -v <built-patronus-binary>:/usr/local/bin/patronus:ro \
  -v <hash-bound-catalog-source>:/opt/catalog:ro \
  --tmpfs /root:exec,size=512m node:22-bookworm bash
```

Inside the container only:

```sh
update-ca-certificates
ln -s /opt/opencode/bin/opencode /usr/local/bin/opencode
opencode --version
uname -s
uname -m
mkdir -p /root/.config/opencode /root/scratch/project
```

From the host's second terminal, transfer **only the separately authorized
OpenCode config inside this disposable container**:

```sh
docker cp <authorized-opencode-config> <grant-bound-container>:/root/.config/opencode/opencode.json
```

Use `docker cp`, never `cat`, shell substitution, printing, hashing, archiving or
copying config into evidence. The broker sentinel is user-owned config, not a
probe value. Do not read/copy `auth.json`, `mcp-auth.json`,
`OPENCODE_AUTH_CONTENT`, provider tokens, headers or cookies. Do not echo the
proxy environment or inline config values. This exception for the authorized
container config is not a credential-transfer grant elsewhere.

Inside, prepare a fresh local scratch project and local source registry:

```sh
cd /root/scratch/project
git init
# Copy only public registry source directories; no .git, state or host config.
cp -R /opt/catalog/artifacts /opt/catalog/recipes /opt/catalog/profiles /opt/catalog/adapters .
# Include packages/ if selected public build inputs require it.
patronus install --profile core-profile-oc --target opencode --global --local-registry --dry-run
# Only after admission and separate exact deploy approval:
patronus install --profile core-profile-oc --target opencode --global --local-registry --deploy
```

Deployment can fetch the pinned `gitleaks` binary. Fetches need the run's explicit
network/install authority; source pins do not prove the scanner is installed.
Record sanitized fetch outcome and actual scanner identity, not credentials.
Repeat local-scope qualification in a separate scratch project/root, after
reviewing the applicable config and ownership inputs. Do not mix the two scopes
until the explicit double-load row. Keep failures for disposition, never force
mixed context or fabricate structural prior evidence. Current lenient context
admission refuses recorded foreign owners at AGENTS.md and an already-present
unknown selected section, but accepts unrelated unrecorded unprefixed sections.
This OC-G9 mismatch with the broader mixed-context requirement remains an
unresolved coverage/safety blocker for final review/parent disposition, not
qualification or a waiver. Prefer global scope or separate project roots.

## Discovery and sanitized observations

Fresh sessions are mandatory after every install/update/remove. Observe:

```sh
opencode debug skill
opencode agent list
opencode debug config | python3 -c '
import json, sys
v = json.load(sys.stdin)
# Debug output is not persisted before redaction. Drop provider.*.options.
p = v.get("provider", {})
if isinstance(p, dict):
    for cfg in p.values():
        if isinstance(cfg, dict):
            cfg.pop("options", None)
# Other user-owned credential fields are excluded from evidence too.
v.pop("provider", None)
# Emit only explicitly allowlisted nonsecret native settings.
out = {k: v[k] for k in ("permission", "agent") if k in v}
# Agent entries may contain user prompt/model metadata: review before capture.
print(json.dumps(out, sort_keys=True))
'
```

The filter drops `provider.*.options` and then excludes all provider data rather
than relying on token-shape matching. Review remaining output for user secrets
before saving; do not save unredacted debug output even as a temporary log. For
skill and agent listings capture only exact selected identities, nonsecret roots
and permission facts. Discard unexpected config fields instead of widening the
allowlist. Pipe errors/parse failures are blocked observations, not empty passes.
Never print the original config or hash/archive credential-bearing bytes.

## Native/model cases (separate permission required)

Run every matrix case in both scopes against the exact pinned version, with
fresh-session IDs and harmless invented fixtures. Read installed files before
following cross-skill handoffs. Record actual tools and arguments in sanitized
form, plus positive and negative results:

- Exact `-oc` skill invocation via `skill`, absent/unsuffixed-name negative control,
  research/spec/plan/review routes, and design/diagnosis/TDD/worktree routes.
- Actual runner `tool.execute.before`: benign write/patch/edit proceeds, planted
  fake-secret `.env` write blocks; observe `apply_patch.patchText -> patch` and
  `edit.newString -> new_string`. Observe exit-2 and error/spawn fail-closed
  diagnostics without echoing payloads. Do not use real secrets.
- Staged invented secret commit blocks through `gitleaks-guard-oc`; benign staged
  commit proceeds. A scanner's presence alone is not guard protection.
- All eight roles listed; `@patronus-researcher-oc` returns attributed findings
  and native edit is denied. Spec/plan authors and writer edit only as granted.
  Missing roles or denied tools block rather than substitute built-in agents.
- Computed co-visible skill mask effect and refresh after adding/removing invented
  Codex resources, with identical-name duplicate visibility and different-byte
  conflicts. Patronus never sets `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1`.
- One supervisor writer in a separate worktree loads the complete guarded working
  set and writes the expected result under recorded `runDir`. Its inline grant
  allows all of that evidence root, a deliberate trade, not just the result file.
  Observe denied write outside both worktree and `runDir`; merge of the agent
  wildcard deny and inline allow is pending until observed. Exit/JSON event
  output alone is not the result contract. Preserve unknown stop/descendants.
- `--pure` disables all external plugins, including the guards. It is a negative
  control, not a safety bypass to use for acceptance. Guards remain best effort.
- Fresh session update and explicit removal show new/absent identities while
  preserving user prose, auth and sibling leaves. Legacy pointer update retains
  the old path; explicit remove plus reinstall is needed to relocate it.
- Both scopes loaded together run matching guards twice, intended benign but
  native-pending. Observe both registrations, both nonces and block outcome.

## Evidence and release boundaries

Retain sanitized commands, actual exits, expected-refusal diagnostics, roots,
public payload/closure hashes, dates, session IDs, nonces, negative controls and
relative sanitized log paths/hashes. Credential-bearing config, inline config,
sentinels and tokens have no place in the record. Observations use passed/failed/
blocked/not-run; unchanged matrix rows stay runtime-pending until separately
reviewed observations and owner disposition authorize a change.

Both development gates run under `/tmp/patronus-gates.lock`, offline Go
`GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local`, standalone Go `-p 2`. These gates
never run this container procedure, native CLI/model calls or retained real guard/
supervisor fixtures. The Node invented runner harness checks Node, not Bun.
Whole-change review, native qualification, integration, publication and cleanup
remain independent grants. Stop on missing auth/tools/image, version drift,
unknown settlement or evidence loss; retain partial evidence for the owner.

# Pi-owned web bootstrap (inert configuration procedure)

This procedure does not run on installation. Patronus delegates exact package installation, update and removal to Pi; package placement alone remains **runtime-unverified**. The config preparation below uses only trusted shell/Python filesystem operations: no Pi, npm, extension import, resolver, helper or network execution. Cold-start activation and tool calls are a separate, explicitly granted deployment phase. Never run these examples against the host's live configuration.

## Admission and exact sources

Record package-lifecycle authority separately from extension trust/execution and web-config writes. Core's `pi-subagents@0.72.1` and `pi-web-access@0.35.0` are exact Pi-managed global dependencies; local Patronus install/update refuses missing, stale or ambiguous globals before local writes. Pi/Node/host peers are external. Inspect the selected Pi settings declaration, exact npm package identity/version, unsafe or duplicate aliases, different-name functional overlaps and pending Patronus intent. Admission checks observable declarations, not arbitrary dynamic registrations; executable extension presence does not require deactivation. No production `PI_OFFLINE` requirement or ambient `npm root -g` probing. Unsafe/unreadable/malformed inputs, ownership ambiguity, drift and mixed-context consent still refuse. Runtime inventory must separately disclose discovery not inspected statically.

Use the exact npm identity selected by Pi; Patronus does not derive a local extension source from a receipt or own package files. The selected web package publishes `pi.extensions: ["./dist"]`; repository `index.ts` is not its installed entrypoint. Verify canonical non-symlink roots, the selected package manifest/version, Pi 0.87.1/Node/host peers and the all-consumer grant. Keep caches, sessions, logs and outputs outside Pi/npm-owned package directories.

## Resolve the config path before changing anything

At 0.35.0, utils.ts:getWebSearchConfigDir selects the first applicable row. It caches its choice; a changed environment does not retarget an already loaded module. Use absolute canonical directories, not relative or symlinked roots. An existing but unreadable/malformed selected file is a failure, not permission to use another root.

| Launch environment and existing files | Selected web config |
|---|---|
| Nonempty PI_CODING_AGENT_DIR | `$PI_CODING_AGENT_DIR/web-search.json`, regardless of XDG/legacy files |
| No explicit root; nonempty XDG_CONFIG_HOME; its `pi/web-search.json` exists | `$XDG_CONFIG_HOME/pi/web-search.json` |
| No explicit root; XDG set; no XDG web file; legacy file exists | `$HOME/.pi/web-search.json` |
| No explicit root; XDG set; neither file exists | `$XDG_CONFIG_HOME/pi/web-search.json` (even if `.pi/agent/web-search.json` exists) |
| Neither override; agent web file exists | `$HOME/.pi/agent/web-search.json` |
| Neither override; no agent web file; legacy exists | `$HOME/.pi/web-search.json` |
| Neither override; neither file exists | `$HOME/.pi/agent/web-search.json` |

Record the actual canonical path and only relevant, nonsecret environment selections. This resolver is not necessarily Pi's default agent-root resolver. For a clean sandbox choose explicit PI_CODING_AGENT_DIR so no legacy/XDG fallback is involved.

## Clean sandbox/image: preview, approve, then place inert config

Admission: use a coordinator-owned disposable root, nonprivileged identity, no host home/browser profiles/auth mounts, no ambient secrets, no container-management socket, and explicit CPU/RAM/PID/deadline/egress budgets. Unknown headroom stops admission. Config preparation needs no network. The selected agent directory must already exist with private permissions and trusted non-symlink ancestors, with no web config or credential data; no concurrent config writer is allowed. A pre-existing file (including an empty file or dangling symlink) uses the existing-config procedure instead, never overwrite.

Set these variables to approved absolute paths, not the literal placeholders. TEMPLATE is the installed skill's `references/web-search.example.json`; the OpenAI alternative is never selected implicitly. These commands are operator examples, not an installer hook:

```sh
export PI_CODING_AGENT_DIR=/ABSOLUTE/APPROVED/PROJECT/sandbox/agent
TEMPLATE=/ABSOLUTE/APPROVED/PROJECT/skills/web-research-pi/references/web-search.example.json
python3 -m json.tool "$TEMPLATE"       # preview inert public template
sha256sum "$TEMPLATE"                 # record the exact reviewed bytes
```

Review every resulting key against the schema notes in qualification.md. Record original file absence, path, template digest, mode 0600, grant issuer/time/action/scope and revalidation trigger. Obtain approval bound to that digest BEFORE the next block. Set APPROVED_SHA256 to that recorded digest, not to a freshly computed value that silently approves changed bytes.

```sh
python3 - "$TEMPLATE" "$APPROVED_SHA256" "$PI_CODING_AGENT_DIR" <<'PY'
import hashlib
import json
import os
from pathlib import Path
import sys

source, approved, directory = sys.argv[1:]
root = Path(directory)
if not root.is_absolute() or root.resolve() != root or not root.is_dir():
    raise SystemExit("refuse noncanonical or missing agent directory")
data = Path(source).read_bytes()
if hashlib.sha256(data).hexdigest() != approved:
    raise SystemExit("refuse template drift from approved digest")
config = json.loads(data)
if not isinstance(config, dict):
    raise SystemExit("refuse nonobject template")
target = root / "web-search.json"
fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(fd, "wb") as output:
    output.write(data)
print("placed inert config sha256=" + hashlib.sha256(target.read_bytes()).hexdigest())
PY
```

Exclusive creation refuses existing files/symlinks. This small clean-root example is not a general merge/ownership service or hostile-filesystem sandbox: trusted ancestors and no concurrent writer are prerequisites. If writing fails, retain/report the partial file and do not launch; inspect under a new cleanup grant. Record resulting bytes/hash/mode. No settings.json, package installation, extension execution or credential access occurs in this block. Installing/removing the skill itself never performs it.

Before the separately approved runtime launch, use a coordinator-selected minimal environment (for example an explicit `env -i` allowlist in the deployment launcher). Ensure PI_ALLOW_BROWSER_COOKIES and FEYNMAN_ALLOW_BROWSER_COOKIES are **absent**, not merely overridden by allowBrowserCookies:false. Exclude ambient proxy/endpoint/provider-key/helper variables unless separately reviewed and granted. Do not log environment values or secrets. This route supplies no authFetch profiles or credential-helper `!command` values; a missing setting is not a universal helper-disable switch. Never invoke credential resolution to preview configuration. Any separately approved credential experiment uses narrow ephemeral secret files, never image layers or logs.

## Existing config: manual per-key changes, not template copying

Stop all affected config writers. Record protected original file bytes/mode/hash and owner, each relevant key's **absence or exact value**, intended applied value/deletion, resulting hash, grant and consumers. Do not copy credential-bearing priors into ordinary evidence logs; retain private backups and only safe hashes/redacted descriptions in reports. Review nested leaves rather than replacing whole objects with unrelated siblings. Example operator rows (invented values):

| Key | Recorded prior | Approved application | Later inverse if still unchanged |
|---|---|---|---|
| `workflow` | absent | `"none"` | delete the owned key |
| `fetch.defaultMode` | `"readable"` | `"raw"` | restore `"readable"`, preserve other fetch keys |
| `maxInlineContentChars` | `12000` | `8000` | restore `12000` |
| `provider` | present, privately retained | delete conflicting route override | restore only if still absent and ownership unchanged |

Preview every template key plus conflicts omitted from the template: provider/searchProvider/searchRouting, toolNames, tools.*.enabled, authFetch, browserCookies/chromeProfile, proxy/endpoint/SSRF exceptions and credential helpers. Unknown owners or malformed config stop. Do not delete unrelated credentials/profiles to make a merge pass: retain them and choose a separately approved clean root if safe coexistence cannot be shown. DDG keeps its explicit searchProvider and removes a conflicting top-level provider/unused searchRouting only with prior handling. The OpenAI alternative removes BOTH top-level provider and searchProvider. Keep exact default tool names and all four tools enabled; record applicable native agentOverrides too. A preserved disabled-tool override is a failed preflight, not a reason to silently shorten the role's tool list.

Apply reviewed changes manually only after rereading the unchanged prior digest. Recheck resulting bytes and record applied per-key values/deletions. Any drift requires a new preview/grant. There is no Patronus web-search.json target or new authorization/ownership ledger API.

## Separately granted native activation and cold-child smoke

1. Inventory effective scope, duplicates and selected package declarations as above. Patronus invokes the selected Pi with the exact recipe ref (`npm:pi-subagents@0.72.1` or `npm:pi-web-access@0.35.0`) and explicit approved scope; Pi/npm own package-file mutation. Global installation affects all sessions in the selected agent root. Project scope is a separately qualified choice, never an automatic fallback. Record original settings entry/absence/hash, exact Pi command, resulting declaration/package observation and scope. Missing packages, imports or aliases stop; Patronus never repairs npm-owned files.
2. Cold launch/restart with the reviewed web config and sanitized environment, explicit trust/execution grant and approved ambient model/provider. Capture reload acknowledgement and effective native source/tool inventory. Config is cached and eager tool registration is initialization-time behavior; editing JSON alone does not prove a running child changed. Missing trust/provider/tool/resource loading fails even if the process exits zero.
3. Launch the exact `patronus-web-researcher-pi` through the qualified native background route with runtime-bound output, resolved mandatory reads and task budgets. It is a standalone installed role plus Pi-loaded web tools, not runtime inheritance from another agent. Record role/config/package identity/input/output hashes, context/skills read acknowledgements and all four effective tool names. Missing tools STOP; an explicitly authorized local-only investigation cannot qualify incomplete core.
4. Under the separate request/egress grant, observe one focused web_search (`numResults:3`, `includeContent:false`, `workflow:none`, omitted provider), source_check (`numResults:3`, `fetchContent:false`, omitted provider), selective public fetch_content (`mode:raw`, no auth/proxy/media/model options) and get_search_content (`offset:0`, `limit:2000`, no findText/findMode). Use actual responseId/queryIndex/urlIndex, not fabricated IDs; the two search tools may make separate requests. Record exact arguments/results, URLs/dates, text/details sizes, errors, cache/payload writes and process/egress observations. Do not treat source_check's unclear/semantic-unavailable result as verified truth.
5. Run the presentation and hostile-path matrix in qualification.md, settle all runs/descendants, then admit only observed support claims. Functional success does not establish hard bounds, reliable service, security or public artifact availability. QP-03/QP-04 own that evidence; this document supplies no prefilled PASS.

## OpenAI current-model alternative: separate route and cost grant

Use `openai-current-model.example.json` only after explicit request/cost/model/endpoint authority; never as DDG failure recovery. At this pin eligible source paths require HTTPS and either provider openai with api openai-responses on api.openai.com, or provider openai-codex with api openai-codex-responses on chatgpt.com/backend-api. Source eligibility alone is not a credential, endpoint or runtime smoke. Record exact active model/API/base URL (without secrets) and verify in the granted environment; model/endpoint changes invalidate it.

No top-level provider/searchProvider may remain: either shadows searchRouting. Omit per-call provider in BOTH web_search and source_check; explicit openai or provider arrays/all bypass current-model routing. The parser requires nonempty fallbackOn, so the template uses `unsupported` with ONE allowed route. Exhaustion/unsupported/ineligibility/network/challenge/429/timeout stops with its limitation, not another provider/model or automatic retry. Allowed-provider restrictions alone do not enforce the current-model route against a caller override.

Record separately the agent model usage, hosted-search requests, any provider-hosted page opening and unknown billing/subscription accounting. A credential's presence is not paid-use consent; unavailable cost telemetry means unknown, not zero. Provider-hosted opening is not arbitrary raw-URL fetch equivalence. Raw retrieval remains the independently approved explicit HTTP route. Do not describe the alternative as qualified by a DDG smoke.

## Update, removal and rollback

Settle all dependent sessions/background/descendant runs; record all-consumer scope and reload acknowledgement before global replacement. Unknown consumers, process state, ownership or drift retain/escalate. Patronus previews and then delegates exact replacement or removal to Pi; verify the resulting settings declaration and selected package observation before clearing pending intent. Do not separately edit or delete npm-owned files. Shared non-Pi dependencies remain separately owned. Project-scope removal preserves unrelated globals, sessions/missions/worktrees, caches, credentials and external data.

Removing this skill/pointer/role does not clean manual web config and makes required core incomplete; there is no optional web overlay. Under a separate cleanup grant restore only unchanged owned applied keys to their recorded priors. Preserve unrelated/edited keys, caches, credentials and backups. If an originally absent file still contains exactly the unchanged owned config, deletion can be previewed explicitly; if other keys were added, preserve the file and remove only unchanged owned keys. Do not replace a whole file from backup over later edits. Stop on drift/unknown ownership. Restart/reload and record acknowledgement after approved cleanup.

Failure retains evidence and partial effects. Rollback is a separately granted exact Pi-managed replacement plus explicit config restoration and fresh checks, never automatic activation of old bytes. Successful package observation means the selected declaration and package identity settled; it does not establish complete runtime collision discovery or activation.

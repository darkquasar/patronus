# Dated candidate dispositions

> **Historical / superseded.** These dispositions preserve the former static-delivery investigation. Their receipt/payload ownership language is not current operational guidance; Pi/npm now own native package files and lifecycle.

QP-02 / pat-g4zf, authored 2026-10-01. **No operational qualification or release
approval is asserted.** The eight JSON records are C-class candidate dossiers,
not runtime packets. Their common result/grants/reviews/acceptance remain unknown.
The record checker can validate their structure and local reference hashes while
qualification refuses their high-impact unknown fields.

## Evidence interpretation

This task read existing research and local recipes only; it did not download,
install, build, import or execute third-party bytes, refresh registries/advisories,
or compare the live installed tree. `retrieved_at` in each JSON is the historical
primary observation time (or this task's local recipe read), not a new remote
check. `authored_at` and invocation describe the new local dossier work. Values
marked observed summarize **attributed historical observations** unless explicitly
identified as a current local recipe read. Proposed procedures are not completed
tests. Unknown fields may contain partial observations but remain unknown.

The source register below binds the ignored/local research inputs by path and
SHA-256 as freshly read for this task. These external documents are not portable
checker file references and are not silently copied into the product tree.
Dossier `references` hash this committed attribution summary/build procedure and,
where applicable, the actual tracked recipe. Those hashes verify the summary,
**not downloaded archive bytes**. A recorded upstream digest is an attributed
value, not a newly computed archive hash. Later qualification must import the
actual evidence/archives into the operator-selected root, bind them as local
references, and independently review them. Changing a summary or recipe requires
updating dependent hashes and reviewing the changed evidence, not just repainting
old approval. No earlier researcher reviewed these newly authored files.

Source report observations, 2026-09-30 UTC unless noted:

- **ECO**, ecosystem-findings.md, retrieval 22:38–22:46, supplement 22:56:
  exact Pi/subagents/adapter refs, identities/licenses, published-file checks,
  service identities, upstream CI and latest observations. Reported npm SHA-512
  checks passed; subagents archive SHA-256
  `a938f397fbbe7fd53433fd9379f42551e763309bf5e3471c0d9ac87f83f30258`
  and adapter `4cc670f25b04607c691b76c16b170c911232b718cb1fee7d56f5f133d7ecde8e`.
  All 1219/230 respective **published** files matched installed bytes then;
  this excludes a fresh whole-tree/transitive supply-chain audit. Graphify wheel
  SHA-256 `b0d47f823f924e7f89acfee390b9f18dc3410917617c5f6f2731bd2642abf16f`
  was reported from PyPI metadata, not recomputed here.
- **RUN**, runtime-findings.md, registry 22:27:42.364171:
  Pi gitHead and archive SHA-1 `5708b9310325177d5c1b487b5c99627ffa733324`;
  installed package.json SHA-256
  `627631b613ba4ca29eba8df793f5280fd20b19f01d73826e9ffda14c15def5dc`.
  **Pi archive-to-installed equality and archive SHA-256 remain unknown.**
  Eight mocked runtime checks reported; not operational qualification. Pi package
  npm/git operations did not add --ignore-scripts. Runtime events/trust and tool
  interception do not contain shell/SDK/extension privileges.
- **FLOW**, workflows-findings.md, clock 22:38:19: tk commit/script pin and
  gitleaks recipe selection; shared-service and subagents ownership/lifecycle
  observations. No runtime tests in that research. **PACK**, packaging-findings.md,
  22:24:39–22:31:46: static receipt ownership, deterministic archive mechanism,
  ceilings and partial-operation limits, not passing runtime results.
- **WEB-SRC**, pi-web-access-source-check.md and source-receipt.json,
  23:33:40–23:34:23: pinned raw source corrects the preceding model-summary
  pi-web-access-findings.md. DDG can be selected through explicit configuration/
  routing, not only per-call; config resolver honors PI_CODING_AGENT_DIR; both
  PI_ALLOW_BROWSER_COOKIES and FEYNMAN_ALLOW_BROWSER_COOKIES opt-ins matter.
  Repository entrypoint is not proof of published bytes. Receipt package.json
  SHA-256 `70488e1bb875d4d80e33c8760e4f922dc8338fafc5b12484bd9011104765fe6f`
  is a source-file hash, **not** an archive digest.
- **WEB-SDK**, pi-web-access-spike.md, results/container-evidence and package-lock:
  corrected SDK run 23:45:36–23:45:37, Pi 0.87.1/subagents 0.72.1/web 0.35.0/
  typebox 1.3.27 on Node 22.22.1 Linux ARM64, no host mounts/secrets.
  npm --ignore-scripts acquisition and `pi.extensions: ["./dist"]` observed.
  Six checks reported: registration, DDG, raw fetch, cache retrieval, provider
  denial and answer-mode denial. Initial assertion failure retained; subagents
  was not loaded/invoked, source_check only registered. Combined spike lock is
  **not** a reviewed CP-05 production closure or reproducible payload lock.
- **WEB-NATIVE**, native-web-smoke.md: four native tools succeeded once, including
  source_check returning `unclear`; installed identity was not independently
  checked. Updated 8000-character setting was not verified in that session.
  Neither smoke proves hard bounds, reliability, exact staged bytes or lifecycle.
- **HOSTED**, openai-hosted-search-source-check.md: hosted current-model route is a
  separate request, not a native Pi toggle; model/endpoint/entitlement changes
  need separate qualification. Not selected instead of mandatory DDG web.
- **TK/GITLEAKS**, current tracked recipes read locally on 2026-10-01: declared
  upstream/license/pins, not a download/signature/license-source audit. Gitleaks
  dossier selects the Linux ARM64 asset; other recipe assets are inventory only,
  not a supported platform claim.

## Selection, alternatives and remaining gates

| Candidate / selected use | Disposition | Historical latest observation / delta | Specific unresolved gates |
|---|---|---|---|
| Pi 0.87.1 external host | conditional required core | ECO latest 0.99.2, Sep 30; intervening release/security/compatibility review unknown | archive SHA-256 and installed equality, full lock/SBOM/scripts, provenance/advisories, current currency, exact startup/lifecycle/approval/platform |
| pi-subagents 0.72.1 owned workflow payload | conditional required core | ECO latest 0.74.0, Sep 30; 0.73.0/0.73.1 intervened; 0.72.1 release says provenance repair/same code as 0.72.0, not independent diff verification | transitive closure/scripts/licenses, attestation subject/signature/builder, advisory reachability, CP-05 build/staging, exact cold/background/cancellation/read-only payload |
| tk 0.3.2 owned script | conditional required core | latest/release delta unknown | source/license/advisories, system-tool closure, actual installed comparison, all-consumer ownership/rollback, Linux ARM64 execution |
| pi-web-access 0.35.0 owned web payload | conditional **required core** (E-03) | ECO registry latest 0.35.0 at 22:56:20; no current refresh | acquired full closure/scripts/provenance/advisories, published entrypoint, CP-05 reproducibility/staging, 8000 presentation/defaults/failure and activation/lifecycle tests |
| pi-mcp-adapter 3.0.0 external code-intel transport | conditional optional, operational defer until selected/qualified | ECO latest 4.0.0, Sep 30; 3.1–3.3 intervened, fixes not assessed | full floating/native dependency closure/scripts, signature verification, no dist.attestations reported, reachable advisories, effective server approval/child propagation |
| Serena commit 7a296833… external symbol service | conditional optional, operational defer | ECO latest listed release 1.7.0 Aug 9; installed 2.0.0.dev0 commit Sep 24, not a comparable linear release delta | exact Python patch/venv lock, Go/gopls compatibility, GPL redistribution decision, archive/installed/provenance/advisory/service lifecycle |
| graphifyy[mcp] 0.9.31 external query service | conditional optional, operational defer | ECO latest GitHub 0.9.73 Sep 30; selected wheel uploaded Jul 30; fixes unknown | exact Python patch/extras/native closure, installed/wheel comparison, provenance/advisories, nonblocking upstream audit jobs, service surface and snapshot freshness |
| gitleaks 8.30.1 explicit scanner | conditional selected scanner, not automatic enforcement | latest/release delta unknown | archive/install/provenance/advisory/source-build/license checks, actual scanning/platform behavior; no security audit from recipe pin |

All pins are retained as the agreed investigation baseline, **not** because an
old release is proven safe/current. Latest observations above are dated leads;
none passes a new currency check. A separately authorized bounded refresh must
record retrieval UTC, exact latest source/ref, intervening notes, security and
compatibility fixes/advisories/reachability and reason to retain/reconsider the
pin. Refresh grants do not authorize upgrades. If evidence cannot be obtained,
defer the affected claim rather than infer no changes/advisories.

Use-case alternatives (ECO unless noted): accept native Pi mechanisms as design
inputs for skills/trust/events, not a blanket operational approval; defer native
MCP substitution at 0.87.1 because no native client was established. Conditional
reuse of permission-gate/protected-paths examples is accidental-action guidance
only. Defer dirty-repo-guard UI reuse; reject plan-mode as read-only enforcement
and official sandbox host-fallback as whole-process containment. Uninspected
forks/permission bundles stay deferred, not insecure by assumption. Hosted search
and new provider/model routes remain separate experiments, never silent DDG
fallbacks. Durable Markdown may support an explicitly granted workflow fallback
but cannot omit mandatory tk/subagents/web delivery. Popularity measures adoption
only, never security, trust or compatibility.

## Claim consequences and ownership

All dossiers lack full reviewed closure/SBOM, transitive script admission,
reachable advisory review, verified provenance, current currency, exact installed
comparison and operational approval. Core-exact-pin and dependent runtime claims
are withheld until every selected dependency and I case is qualified. Required
web cannot be omitted as optional; hard bounds remain separately withheld under
pat-n6za. Optional MCP/service unknowns do not block an independently qualified
no-MCP core; selecting code-intel makes them required. Scanner unknowns block
claims relying on scanner qualification. C/M static-delivery checks remain
separable and do not become runtime or security claims.

See [build admission](core-dependency-build-admission.md) for pre-acquisition
sources/roots/scripts grants, post-acquisition Q review, reproducible CP-05 builds,
separately authorized controlled HTTPS staging, actual pins, QP-03 runtime and
QP-04 promotion. That procedure defines provisioning, all-consumer quiescence,
manual activation/deregistration, update/remove/rollback and retention ownership.
Pi host/optional services/caches/sessions/credentials remain operator-owned;
For the current boundary, Patronus owns `tk` files plus exact refs and bounded intent/profile state; Pi/npm own subagents/web package files.
Never use the destructive/unpinned legacy subagents npx installer or dual owners.

Serena application **GPL-3.0-or-later** obligations are distinct from **SolidLSP
MIT**. Running an external service does not itself establish a redistribution
conclusion. Any distribution must independently review corresponding-source,
license/notice and conveyance obligations; do not copy the old MIT-only recipe
label onto the application. Other dependency licenses also need full-closure
review. No ambient tree may substitute for reviewed source/lock/SBOM construction.

## Revalidation

Run each candidate through record mode at repository root. Unknown values must
remain unknown. Existing checker fixtures assert high-impact unknowns refuse in
an otherwise complete operational packet; a candidate is not itself a
qualification packet. Record mode only hashes the available local references,
not descriptive external paths or unacquired archives. A changed input requires
fresh review and revalidation; preserve old/new locks, archive hashes, config
priors, source diffs, decisions and rollback evidence. Never transfer historical
approval to changed bytes. QP-04 may update these dispositions after reviewed
operational evidence and owner acceptance.

## Freshly read source register

The following SHA-256 values bind local research **documents**, not the truth of
upstream claims. Paths below are relative to the read-only research root
`/home/agent/workspace/patronus/docs/specs/01-pi-harness-adapter/`.

| Input | SHA-256 |
|---|---|
| `ecosystem-findings.md` | `81e680ba786211dee1993636851d950f2ee54619f12ea3c35be2b6026b28d8fd` |
| `runtime-findings.md` | `fa64c70613da3935f9e4e7b8d28b940b257f3f1fe68785453584552bfb698f46` |
| `workflows-findings.md` | `f62a353a6bd22254e25294555ce0e9bbaa531bd677622d0dfb6b24f44da42ca8` |
| `packaging-findings.md` | `5f89010bda18bbf8d54c98018da32cfac7a28d72d5a9bd6c069f8f45f22b31a8` |
| `pi-web-access-findings.md` | `e1a90c15144919ab3d407b4f09e153ba9a35d31ad3b02c3d9f8ce8a5db6fe227` |
| `pi-web-access-source-check.md` | `84c2511a00793456345e8e69507d5ee3e222c2bdac57142c669f6bbb1ff0cc16` |
| `evidence/pi-web-access-source/source-receipt.json` | `8d76caafb7b3c5791875d322e5c960d167fe3a27f273ee7bc5d17056ab574721` |
| `pi-web-access-spike.md` | `1d71a2bd640e39f2fecb752649d8198534cff98656a63fe77f9ae7a678db6f24` |
| `evidence/pi-web-access-spike-results/package-lock.json` | `a677f9edfbd9f7d289c8c2bf03bd03f5bbd5ee7906055f6f43089690656e907b` |
| `evidence/pi-web-access-spike-results/container-evidence.json` | `be1f4d1ed19441722eee979f5cb7368287244f7b3cab50213e19e53355184fea` |
| `evidence/pi-web-access-spike-results/spike-results.json` | `eed19ac3d5444ad2934922081bf165f87c26b1087b4c20a1c38622b402deac1b` |
| `native-web-smoke.md` | `dbd3aadd15713f2a935682d2c7f3a7f4c0d5f8196135b0f690ba15bc03e5e357` |
| `openai-hosted-search-source-check.md` | `aec30a017efde2ae58cf8b431c2a2e9eb4bd4bb2a1bdd51662cf6429c91d0e1b` |

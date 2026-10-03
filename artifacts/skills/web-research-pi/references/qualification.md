# Content evidence and runtime limits

## Source-backed configuration review (C, not runtime qualification)

Selected source: pi-web-access 0.35.0, ref 72c6e67787d67d8a7d01bf0abf30c072a6112de6. The 2026-09-30 source-hash record's eight file lengths/digests were rechecked for CP-04; seven TypeScript files matched the source snapshot used for review. Repository package.json uses ./index.ts; the published package uses ./dist. Source review does not establish generated-code equivalence or new operational observations. See ../SOURCE.md for exact hashes.

| Template intent | Pinned source basis | Boundary to verify in deployment |
|---|---|---|
| `searchProvider:"duckduckgo"`, only duckduckgo allowed | gemini-search.ts:getSearchConfig, assertSearchProviderSelectionAllowed, search | Direct DDG failure has no implicit auto-provider fallback; HTML availability/reliability is not guaranteed. |
| `toolActivation:"eager"` | index.ts initialization and tool-activation.ts | All four names must actually survive the exact child's tool allowlist; a skill does not register them. |
| `workflow:"none"` | index.ts web_search execute | Caller workflow/includeContent overrides can still expand effects; defaults are not a policy interceptor. |
| `fetch.defaultMode:"raw"`, `allowedModes:["raw"]` | index.ts:resolveFetchModeConfig; extract.ts:extractContent raw branch | Exact direct textual HTTP route, not answer/readable/media extraction. No auth/proxy/model/clone options are granted. |
| `fetchRouting.providers:["http"]`, hosted providers false | extract.ts:loadFetchRouting | Raw dispatch has its own direct branch; this setting is not OS egress containment. |
| `maxInlineContentChars:8000` | index.ts:getMaxInlineContentChars, initialContentSlice and get_search_content | Valid presentation value (source minimum 1000, maximum 200000); not network bytes, total tool envelope or a cumulative budget. |
| Browser/cookie/curator false; four commands disabled | index.ts config/command gates; gemini-web-config.ts:isBrowserCookieAccessAllowed | Both cookie-enabling environment variables must be absent. Command gates do not contain module initialization or every shortcut. |
| Clone, PR/issue, YouTube, video, image, PDF enabled:false | github-extract.ts, github-issue-pr.ts, youtube-extract.ts, video-extract.ts, feature-config.ts, pdf-extract.ts | Observe disabled paths; do not infer absence of helpers/processes from field names. |
| No authFetch profiles or helper credentials | auth-fetch.ts:loadAuthFetchProfiles; selected clean launch environment | There is no universal helper-disable switch. Existing secret/helper configuration needs separate review, never blind merge or execution during preview. |
| Alternative: only openai route, useCurrentModel:true | gemini-search.ts:normalizeSearchRouting/searchWithConfiguredRouting; openai-search.ts:resolveCurrentModelSearchTarget | Nonempty fallbackOn is required; one route leaves no next provider. Top-level or explicit per-call provider bypasses routing. Model/endpoint eligibility and cost need independent consent/observation. |

Historical evidence is limited: the disposable SDK spike ran DDG search, raw fetch and stored retrieval, plus disabled-provider/mode negatives; source_check only registered. The later native smoke called all four tools but did not independently verify installed identity, and did not measure the new 8000 setting. Neither run is qualification of these refined content/config bytes. Historical optional-web recommendations are superseded: web is required core, while stronger bounded-unattended claims remain separate.

## Three evidence layers

- **M/application capability:** invented artifact/config/package bytes exercise generic delivery, inert sidecar copying, ownership and removal. No real-plugin membership/name/count oracles in application tests; no acquired extension imports or provider/network credentials.
- **C/catalog/content:** generic manifest/declared-sidecar/reference/native-agent checks plus source-backed template review and JSON parsing. Clean-bootstrap examples can be exercised in private directories with invented JSON and negative drift/existing-file cases. Those checks prove only the example's filesystem behavior, not upstream parsing or disabled runtime behavior.
- **I/deployment:** separately authorized disposable exact payload/Pi/Node/provider/config runs. Record actual commands or native tool arguments, UTC start/end, exit, settlement, expected versus observed outcomes, input/output/log hashes, scope/grants, review and limitations. Use the repository's QP-01 record-format.md/cases.json under docs/pi-qualification; do not invent PASS rows or a competing evidence schema. Core-exact-pin requires web functional evidence; it does not imply web-bounded-unattended.

## QP-03 presentation and hostile-path handoff (not yet executed here)

Every case binds exact role/config/payload/host hashes and records both `content` text and `details`, including error/progress wrappers. Measure the units actually used by the selected implementation (JavaScript string lengths), plus transport bytes separately; Unicode characters, UTF-8 bytes and tokens are not interchangeable. A small page is not permission to page indefinitely. Record requested/returned offsets/lengths, continuation and total accumulated output against the task's separately approved call/page budget.

| Surface | Required observations and negative cases |
|---|---|
| Cold/background role | All four tools, complete mandatory skills/context reads, approved ambient provider and bound persisted output. Missing/renamed/disabled tool or unqualified host/provider fails preflight, not silent tool omission/builtin substitution. Keep payload read-only and observe cache/session writes outside it. |
| web_search | DDG-only; small numResults <=5, includeContent:false, workflow:none. Long results, multiple queries, metadata, truncation/continuation, provider errors and no-result paths. Observe summary/background-fetch overrides as unsupported-for-this-route, not pretend config forbids them. |
| fetch_content | Selective public raw HTTP; long textual content and first-slice behavior at 8000, errors/redirects/nontext content, clone/media/auth/proxy incompatible options. Inspect full downloaded/cached content separately from displayed slice. |
| get_search_content | Search and fetch response IDs, explicit pages <=2000, offsets/continuation/last page, missing ID, invalid/out-of-range limit. findText ignores limit/offset and is outside this paging promise; measure its output separately, never advertise limit:2000 as enforcing that path. |
| source_check | numResults <=5, fetchContent:false, bounded granted queries. Measure long passages/metadata/errors independently: the source-check formatter does not use getMaxInlineContentChars. Record unsupported semantic assessment/unclear status rather than converting it to truth. |
| Disabled features | Observe browser, cookie extraction (including hostile enabling environment), clone/media/credential helpers, remote hosted fetch and paid-provider paths; do not claim they were tested just because templates contain false. |
| Failures and route changes | Controlled oversized body, private destination/redirect/content-type, challenge/429/offline/timeout/cancel, disallowed provider, caller overrides, changed/ineligible current model/endpoint. Retain outcomes and descendant settlement; do not respond with automatic retry, credential escalation or provider/model substitution. |
| OpenAI alternative | Own config/consent/eligibility/cost record and exact child smoke; omitted provider in both search tools. Explicit provider bypass is a negative case, not a supported alternative route. Record hosted search/page-opening separately from raw fetch. |

maxInlineContentChars8000 is ordinary content presentation, **not a universal four-tool envelope cap**, download limit, cumulative context/token/cost/call bound or OS egress policy. The DDG search path reads response text before parsing without that network-byte bound. Metadata/errors, source_check and caller overrides require explicit measurements; any advertised default presentation path that fails must be reported/fixed or escalated, not filled with an assumed pass. Do not claim stronger bounds by confusing a setting with enforcement.

OP-WEB evidence must include every finite bounds outcome even when selecting only functionality. Missing hard response-byte, redirect/content-type, aggregate call/output/cost limits or denial of summary/background/provider overrides **FAILS the web-bounded-unattended claim**; unknown remains unknown. pat-n6za stays open even if C/M or functional I passes. Q-T7-M examples test record/model consistency, not actual retrieval enforcement. Trusted-host experimental acceptance must explicitly name the absent protections.

Host-peer/runtime/platform, generated-code/provenance/redistribution and public endpoint availability remain CP-05/QP-03/QP-04 gates, not content acceptance. An explicitly authorized local-source-only continuation is not qualification of core missing required web. No live runtime/credential/publication authority follows from this handoff.

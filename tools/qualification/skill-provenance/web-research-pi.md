# Web source inventory and invocation classification

Authored configuration/guidance, 2026-10-01. Source reviewed: pi-web-access 0.35.0 at ref 72c6e67787d67d8a7d01bf0abf30c072a6112de6, as recorded by CP-05 at Patronus 9ad0d67505a208a9d78ff3d18b8eb23d1677380b. No third-party implementation is copied into this skill; package licenses remain with the required Pi-installed package.

- `index.ts` SHA-256 `11bc77f6a57531fe69893027a50767f4d103e050f64b681e5c24b9924a5dd03c`: WebSearchConfig, resolveFetchModeConfig, getMaxInlineContentChars, tool registration and activation.
- `utils.ts` SHA-256 `ba5cb3bfcb6fcb469e746c4b696f29a9107b9f130f9cf462c30767c3ba7f1908`: getWebSearchConfigDir/getWebSearchConfigPath.
- `gemini-search.ts` SHA-256 `dfd702d0b8fa1cfe0d69cffe3d685fb7168a3b8c34234d1bc46a6953ec5240df`: getSearchConfig, normalizeSearchRouting, configured route selection.
- `gemini-web-config.ts` SHA-256 `a1c408a3cef6127a3818d776d7b66240e17b2509475c1a9eaa29102340d476ef`: isBrowserCookieAccessAllowed and environment opt-ins.
- `github-extract.ts` SHA-256 `d9223b81821d8b314de7c7dd8d3a5d415d1123b2a871cca2ddccdd8e0d0c513c`: githubClone.enabled.
- `github-issue-pr.ts` SHA-256 `6c902662f16f36867bb36a56c451fd4a32920f8d1851392f19f3e88f60cf717d`: githubPrIssue.enabled.
- `youtube-extract.ts` SHA-256 `7ac867dc1f343cf10929331e80a0d6c85df267ec2572685441e879797c4d762a`: youtube.enabled.
- `video-extract.ts` SHA-256 `c5eea57652efe02c70a7ebaf9e32cce72ebe9a41c06573061cd94211ec73e843`: video.enabled.
- `feature-config.ts` SHA-256 `207dc9f392086474b7759a5fc9c36b0540096359d367f213f198bab39fd258bf`: image.enabled.
- `pdf-extract.ts` SHA-256 `c6f84775fe8fcb707d33884bec94660eca71a49ebb3872451eeb7024a1fa674a`: pdf.enabled.
- `extract.ts` SHA-256 `c2e6b417608e2860c40d186bfd8b986621006d1ea75b38e20284d2ace2a9f8f6`: raw direct HTTP branch and fetch routing.
- `auth-fetch.ts` SHA-256 `451632f006d3503adeb20fe488383b915fbaa373852fa593274572fada472ccd`: loadAuthFetchProfiles; absent profiles are not a universal credential-helper-disable switch.
- `openai-search.ts` SHA-256 `0f470688dbcef7e4d770a00daa63cf4dbbbd5fd78a1d01641346304a73efa3c7`: resolveCurrentModelSearchTarget and current-model eligibility.

CP-04 source review rechecked all eight historical source-hash entries and seven TypeScript source equalities without imports. The published package.json differs from the repository metadata as expected (./dist versus ./index.ts). Both JSON templates retain their reviewed bytes; runbook/config placement examples are inert operator procedures, not executed install hooks. Additional pinned-source constraints: index.ts:2893-2908 documents findText ignoring offset/limit; index.ts:2435-2532 source_check has its own formatter, not getMaxInlineContentChars; gemini-search.ts:618-631 dispatches explicit per-call providers before current-model routing. These motivate the paging/route caveats and separate response-path qualification, not an upstream patch or a new enforcement claim.

Active invocations: web_search, fetch_content, get_search_content and source_check are separately activated native extension tools, NOT registered by this skill. research-team-pi is the selected core workflow read. Current-model OpenAI is an optional separately approved alternative, not core default or fallback. Pi install/remove are external operator capabilities under explicit grants. The historical source-hash record describes reviewed source bytes, not published ./dist execution. The inert JSON files are templates, never active configuration.

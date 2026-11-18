# Native Codex acceptance matrix

Status of every native row: **runtime-pending**. Source helpers/guards/migration
and local fixtures do not qualify native calls. The retained CLI authentication
smoke failed401 and was stopped. No version/platform/scope is qualified here.

| Dimension | Selection for the next authorized run | Current evidence |
|---|---|---|
| Version | Exact CLI executable identity and `codex --version`; proposed baseline 0.149.1 | Historical root observations only, not current acceptance |
| Platform | Linux arm64 first; any other OS/arch requires its own rows | No accepted platform matrix |
| Scopes | Isolated global HOME/CODEX_HOME and separate local project | Both mandatory, neither accepted |
| Catalog | Exact base/head/patch and profile/ledger/closure hashes | Static source is reviewable; native loading unverified |
| Auth | Owner-managed Codex auth and per-server MCP auth | Deferred; no credential transfer or secret reads |
| Trust | Native CLI/project approvals, separate hook trust | Deferred; Patronus must not write native trust internals |

Run every required case for each selected version/platform and both scopes.
Expand the matrix when a claimed surface differs across platforms. Preserve
blocked and failed observations rather than substituting an easier scope.

| ID | Case and positive observation | Negative control / blocking condition | State |
|---|---|---|---|
| CX-N01 | Record effective skill/config/instruction/hook roots with HOME and CODEX_HOME overrides | Wrong or shadowed roots, unreadable config, absent executable | runtime-pending |
| CX-N02 | Fresh-session explicit invocation of every selected -cx skill resolves its exact directory/frontmatter identity | Invented absent -cx name must not silently route to unsuffixed or -pi sibling | runtime-pending |
| CX-N03 | Lead follows research-team-cx -> spec-brainstorming-cx -> spec-review-cx -> plan-writing-cx -> plan-review-cx; read actual installed files | Same logical names in legacy/Pi sets must not become fuzzy cross-skill substitutions | runtime-pending |
| CX-N04 | plan-execute-cx reaches requesting-code-review-cx; read-only worker returns findings, lead writes review record | Missing reviewer/auth/tools is blocked; no foreign protocol/provider fallback | runtime-pending |
| CX-N05 | Fuzzy-name routing: record exact skill names, descriptions and resulting selections in co-visible roots | Ambiguous base-name request must not select foreign workflow or silently choose an unintended -cx skill from a Pi session | runtime-pending |
| CX-N06 | Profile preview/lock selects the complete requires closure and all three audited recipes | Noncodex/all target refusal; Pi package/role/@claude or unreviewed unsuffixed closure member blocks | runtime-pending |
| CX-N07 | New and old skill-root admission preserves owner/user files | Legacy roots, edited owned file, unowned identical file, symlink, same-name conflict, shadowed or unknown mixed AGENTS.md blocks without writes | runtime-pending |
| CX-N08 | Same-target lock refresh retains reviewed pins | Pi/Codex cross-target overwrite refused; explicit migration is a separate grant | runtime-pending |
| CX-N09 | Native context7/github MCP initialization with operator-owned auth | Missing or invalid auth gives practical prerequisite diagnostic; preserve env references/headers, never copy credentials | runtime-pending |
| CX-N10 | Reinstall/update preserves user-owned MCP auth and sibling TOML fields | Auth loss, raw blind patches, unrelated config edits or malformed ownership metadata block | runtime-pending |
| CX-N11 | Install -> restart -> explicit exact-name invocation; retain session IDs and nonces | In-session old bytes cannot establish discovery; declared file placement is insufficient | runtime-pending |
| CX-N12 | Update -> restart -> observable changed-version nonce; preserve scope/ownership | Version unchanged, foreign scope or stale-session output cannot establish update | runtime-pending |
| CX-N13 | Remove explicit item -> restart -> removed identity absent while siblings and user prose remain | Remaining loaded session, sibling deletion, user auth removal or inverse-profile assumption fails | runtime-pending |
| CX-N14 | Failure-injection: only successful writes recorded; retry preserves edited/unowned files | Failed-apply state retirement, corrupt legacy metadata or force-as-consent fails | runtime-pending |
| CX-N15 | Selected hook script placement, emitted matcher/TOML registration and explicit native trust each recorded separately | Selected script silently skipped, unsupported shape or no native trust leaves behavior blocked | runtime-pending |
| CX-N16 | Trusted harmless hook emits a unique nonce in a fresh session; SessionStart matcher vocabulary observed | Untrusted/disabled hook must not emit it; static registration is not a semantics pass | runtime-pending |
| CX-N17 | Secret-write guard blocks invented harmless fake-secret fixtures for Write/Edit/MultiEdit/apply_patch and Write matcher alias with tool_name apply_patch | False negative or harmful real-secret probe blocks; absent guard remains Partial | runtime-pending |
| CX-N18 | gitleaks guard inspects invented staged diff on native Codex Bash commit event and accepts harmless commit | CLI presence alone, unsupported Bash event, unstaged-only scan or absent guard is not protection | runtime-pending |
| CX-N19 | Bundled isolated writer helper qualifies native independent writers and ordered predecessor integration with full settlement | Timeout, delayed descendant, missing result, ownership violation, dirty output and conflict retain failed work; helper absent is Partial | runtime-pending |
| CX-N20 | Separate admitted target sessions respect co-visible resources after lifecycle changes | Unknown mixed ownership or fuzzy Pi/Codex selection blocks co-installation claims | runtime-pending |

Rows CX-N15 through CX-N19 now have source guard/helper payloads and retained
local fixtures. Fixture results are historical and NOT RUN in the current
development gates. The rows still require actual native payloads, trust, worker auth and fresh
sessions. SessionStart activation/heartbeat/reground/language hooks remain
omitted; advisory instructions do not fulfill their callback semantics. Claude
ccusage/statusline is N/A. Editorial loading/port is explicitly excluded.

## Evidence per row

Retain ID, expectation, observed result, status (`passed`, `failed`, `blocked` or
`not-run`), exact sanitized command/exit, date, version/platform/scope, effective
roots, before/after hashes, fresh-session ID and nonce where applicable. Include
negative controls as their own observations, log path/hash and reviewer identity.
A command failing because the expected refusal occurred can satisfy a negative
case only when the expected diagnostic and zero unintended writes were observed.

Release admission requires all mandatory cases for the claimed surfaces and
owner disposition of residuals. Failed, blocked, missing or unimplemented cases
cannot become successful native evidence through a prose summary.

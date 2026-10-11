# Native OpenCode acceptance matrix

Every native row is **runtime-pending**. Proposed first qualification only:
`opencode-ai` 1.18.35 on Linux arm64, isolated global and project scopes. No
qualified session, executable hash or proof is invented here. Local Node tests,
static profile closure and retained historical fixtures do not establish native
Bun or OpenCode behavior. The procedure in `README.md` needs its own grant and
current image/version preflight; historical image presence is not current proof.

| Dimension | Required observation |
|---|---|
| Identity | Exact version, actual executable SHA256, Linux arm64 and image ID |
| Catalog | Actual source/base/head/patch, profile, ledger and public closure hashes |
| Roots | Isolated HOME/project, OPENCODE_CONFIG_DIR/XDG roots, both scopes |
| Inputs | Sanitized environment presence only; user-owned config/auth never archived |
| Authority | Native/model, network/install, image changes and cleanup separately granted |
| Sessions | New session ID after each install/update/remove, nonsecret observable nonce |

Run each mandatory row in both scopes. Preserve failed, blocked and not-run
attempts. Expected refusal is evidence only with its actual diagnostic and zero
unintended writes. No skipped row becomes passed from a prose summary.

| ID | Case and positive observation | Negative control / blocker | State |
|---|---|---|---|
| OC-N01 | Record pinned version/executable/platform and effective skills, agents, plugin, instructions and config roots with env redirects | Absent image/module/tool, version drift, wrong/shadowed roots or unreadable config stops; no implicit pull | runtime-pending |
| OC-N02 | Fresh-session `skill` loads every exact selected -oc directory/frontmatter identity | Invented absent -oc name and unsuffixed logical-name request must not silently substitute a foreign sibling | runtime-pending |
| OC-N03 | Read installed research-team-oc -> spec-brainstorming-oc -> spec-review-oc -> plan-writing-oc -> plan-review-oc; ADR-0003 folders preserved | Missing role/tool/auth blocks; fuzzy base-name and foreign routes do not count | runtime-pending |
| OC-N04 | Execution reaches independent requesting-code-review-oc, receiving-code-review-oc and verification-before-completion-oc; design/diagnosis/TDD/worktree/meta-skill routes resolve | Reviewer writes lead record, nested delegation, missing sidecar or undeclared capability blocks | runtime-pending |
| OC-N05 | Actual profile preview/public lock has complete selected requires closure and six audited reused items | claude/codex/pi/all refusal; no -cx/-pi/@claude closure or unresolved warnings | runtime-pending |
| OC-N06 | Observe co-visible roots, identical unsuffixed duplicate warnings and native computed permission.skill mask for detected Codex family; refresh after add/remove | Different-byte collision, recorded foreign owner at AGENTS.md, unknown selected section and cross-target lock refused; unrelated unrecorded unprefixed section currently accepted (OC-G9 unresolved broader safety blocker); Pi roots not consulted; native duplicate caveat observed | runtime-pending |
| OC-N07 | Native context7/github MCP initialization with user-managed auth; transport-leaf lifecycle preserves headers/oauth/provider and siblings | Missing auth remains blocked; no credential copying, sentinel printing or credential config hashes; commented config and OPENCODE_CONFIG setting writes refused, inline override warning observed | runtime-pending |
| OC-N08 | Eight native patronus-*-oc roles appear in agent list with body prompt and permission maps; authors/writer edit within grant | Absent role, dropped permission fields or fallback to built-in role is blocked | runtime-pending |
| OC-N09 | @patronus-researcher-oc returns attributed read-only findings; reviewers/read-only web role also enforce deny rules | Native edit attempt denied; no nesting; bash allowlist is cooperative and cannot establish OS isolation | runtime-pending |
| OC-N10 | Runner loads through native Bun tool.execute.before; benign write/edit/patch proceed; actual patchText->patch and newString->new_string mapping reaches guard | Invented fake-secret .env write blocks; exit2 stderr blocks; other exit/spawn failure blocks without echoed payload; missing registration never qualifies active guards | runtime-pending |
| OC-N11 | Native Bash staged-secret commit blocks through gitleaks-guard-oc; harmless staged commit proceeds | Missing scanner/error blocks; CLI presence or unstaged-only scan is not proof; use no real secrets | runtime-pending |
| OC-N12 | Fresh worktree supervisor worker discovers complete global or base-tracked working set, completes owned patch and writes result under recorded runDir | Agent/guard/script/runner/permission-default missing or result absent blocks; exit/JSON event alone cannot satisfy result contract; native deny/inline allow merge must be observed | runtime-pending |
| OC-N13 | Worker can edit in attempt worktree and permitted runDir evidence root only as granted; preserve settlement observations | Write outside both worktree and runDir denied; timeout/descendant/unknown stop retained, not silently cleaned; permission rule is not OS sandbox | runtime-pending |
| OC-N14 | --pure fresh session shows runner/external guards disabled, compared with guarded positive session | Lack of block under --pure is expected disablement, not protection; permission defaults are a distinct native surface | runtime-pending |
| OC-N15 | Install -> restart loads identities; update -> restart observes changed nonce/bytes; explicit remove -> restart removes identities with siblings/prose/auth intact | Stale session, inverse-profile assumption, obsolete pointer retained by update, malformed prior evidence or force-as-consent blocks; legacy pointer needs remove plus reinstall | runtime-pending |
| OC-N16 | Failure injection records only successful writes and preserves retryable state; OpenCode-owned package.json/package-lock.json/node_modules/.gitignore unchanged | Edited/unowned/symlink paths and missing or invalid pre-management prior evidence refused without unintended writes; no fabricated ownership repair | runtime-pending |
| OC-N17 | Global and project runners both load in one fresh session; matching guards execute twice with both sanitized nonces and intended benign block behavior | Single-scope observation cannot qualify double load; differing registrations or adverse duplicate effect blocks | runtime-pending |

## Per-row evidence

Record ID, expectation, actual observation, status (`passed`, `failed`, `blocked`,
`not-run`), sanitized command/exit, date, version/platform/scope, roots, before/
after public payload hashes, fresh-session ID, nonces, explicit negative control,
relative sanitized log path/SHA256 and reviewer/owner disposition. Do not store
config contents, credential-bearing hashes or inline-config values.

Session-start dispatch/heartbeat/reground/language callbacks, editorial and
Claude usage reporting/statusline are omitted. Advisory instructions do not
replace callback semantics. Guards are user-disableable best effort, not
read/egress protection. Shared supervisor/precheck fixture results remain
historical and not run by development gates. No native row is passed here. OC-G9 remains an unresolved coverage/safety blocker
against the broader OC-D7/T15 requirement: lenient context admission accepts an
unrelated unrecorded unprefixed section, while recorded foreign owners and an
already-present unknown selected section are refused. Preserve this distinction
and CD5 legacy behavior. Global scope or separate project roots are recommended
pending final independent whole-change review and parent disposition; these
procedures neither fix nor waive the unmet broader requirement.

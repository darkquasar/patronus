# Pi qualification evidence

> **Historical / superseded delivery evidence.** This record format and its static-delivery claims predate the Pi-owned npm lifecycle. They remain for provenance and offline checker compatibility, not as current installation, update, removal or operational guidance. Current package behavior is defined by `docs/pi-delivery.md` and exact `manager: pi` recipes.

This is an **offline record checker**, not a package installer, policy interceptor,
authorization service or security certification. It executes no recorded commands,
loads no plugins, uses no network and changes no evidence. The human owner must
establish the truth of logs, reviewer independence, grant authenticity and current
authority. An exit status of zero supplies **no permission**.

## Use

Use an already installed Python 3 interpreter (standard library only):

```sh
python3 scripts/check-pi-evidence.py record --root "$EVIDENCE_ROOT" --file candidate.json
python3 scripts/check-pi-evidence.py qualification --root "$EVIDENCE_ROOT" --file qualification.json --claim core-exact-pin
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/tests -p 'test_pi_*.py' -v
```

`--file` and all evidence paths are canonical paths **relative to the selected
root**. Copy reviewed external inputs beneath that root; do not point through
symlinks. The checker rereads bytes on every invocation, including ignored and
untracked files, and emits their actual SHA-256 hashes as JSON. Save that output
outside the immutable packet being checked. No Git-status or prior-run cache is
used. A digest mismatch refuses and reports the newly read digest, not the stale
expected one. Keep the selected root quiescent during checking; this is not an
atomic filesystem snapshot or protection against hostile concurrent writers.

`record` validates structure and referenced bytes, preserving honestly incomplete
candidate/qualification/grant records. It does not admit a claim. `qualification`
requires the requested claim, all selected lower-level cases, dependency dossiers,
review and separate acceptance gates to be complete. Unknown, failed, skipped or
missing mandatory outcomes refuse. Negative-case **tests must pass**; a reported
bypass or unknown outcome is not silently converted to a passing test.

The [format](record-format.md) and [finite case contract](cases.json) are versioned
interfaces. All files in `templates/` are **examples with unknown status**, never
release evidence. `examples/` contains invented observations, not upstream runs.
The production subject consuming those observations is
`scripts/check-pi-evidence.py`; QP-03's `operational-tests.md` consumes the vectors
as scenarios for separately authorized exact-pin I runs. No upstream policy
implementation or wrapper is supplied here.

## Claims and execution amendments

| Claim | Required evidence, in addition to its parents |
|---|---|
| `static-delivery` | D-T1 through D-T8 M; C-T1 and C-T8 C. No I evidence or runtime MCP applicability, even for inert MCP configuration. |
| `core-exact-pin` | Static delivery; Q-T1/Q-T5/Q-T6 C for every selected core dependency; Q-T3-M/Q-T5-M/Q-T7-M M; all core OP I cases including mandatory web functionality/default presentation. |
| `code-intel-exact-pin` | Core; applicable I-T M/C cases and optional dependency dossiers; OP-CODEINTEL, OP-TRUST-MCP and OP-APPROVAL-MCP I. |
| `web-functional` | Core plus the same primitive web functionality selectors; compatibility/reporting name, **not an optional installation**. |
| `unattended-mutation` | Core; OP-ISOLATION I, PI/applicable MCP bypass assessment and owner acceptance of the selected whole-process deployment. |
| `web-bounded-unattended` | Web-functional and unattended-mutation; **all** OP-WEB hard-bound outcomes pass. |

Execution amendment E-03 makes pi-web-access 0.35.0 required core alongside tk
and pi-subagents; Pi host remains an external prerequisite. Its locked closure
needs a dossier and its functionality needs actual I evidence. Core's primitive
web requirements do not depend on `web-functional`, so expansion is acyclic.
DuckDuckGo-only, eager availability, raw fetch, workflow:none and 8000-character
inline presentation require observed default-path tests, as do pagination,
source_check/fetch/get_search_content/web_search errors and override disclosures.
8000 is **not** a network-byte, cumulative-output, token, cost or OS-egress bound.
Core may pass its functional claim with explicitly unknown stronger bounds;
`web-bounded-unattended` cannot. No automatic paid/provider/model fallback.

E-01 clarifies the original code-intel acceptance table: I-T1 M, I-T2 M+C,
I-T4 C and I-T5/I-T6/I-T7 M are the applicable M/C selectors. I-T3 remains
**integration-only**, never an invented C bootstrap review. OP-CODEINTEL I must
include passing I-T1/I-T3/I-T4/I-T6/I-T7 outcomes, including actual isolated
private-worktree symbol/bootstrap observations. Reuse exact command/log references
where appropriate; M/C records cannot substitute for those I outcomes.

Only explicitly unselected optional claims or deterministically excluded MCP
subcases can be omitted. Runtime claims require hash-bound effective configuration
and an owner-reviewed inventory: `mcp-enabled:false` means no selected MCP
surface, while true adds **both** MCP subcases. Code-intel always requires true.
Core still always requires both PI subcases. OP-TRUST and OP-APPROVAL are grouping
labels, not evidence selectors. The checker does not discover live configuration.

## Keep controls distinct

| Control | What it does | What it does not establish |
|---|---|---|
| Archive integrity | Matches selected bytes to a declared digest | Publisher identity, safe dependencies or provenance |
| Provenance/signatures | Supports a reviewed subject/builder/source relationship | Runtime containment or trustworthy code by itself |
| Resource qualification | Owner selects reviewed exact bytes and environments | Removal of extension process privileges |
| Pi project trust | Permits selected project resources/settings | Tool approval; context/session inputs may be read earlier |
| Effective MCP server approval | Admits the selected effective server definition | Pi project trust or authorization of each operation |
| Tool approval | Controls the tested intercepted operation surface | !/!! shell, SDK bash, pi.exec/Node or alternate MCP clients |
| Stage grant | Owner authorizes named action, roots, outputs and time/inputs | Enforcement against noncooperating actors; authenticity from JSON |
| Watchdog ask | Model arbitration | Human or supervisor consent |
| OS isolation | Restricts the selected whole process and descendants | Protection from intentionally exposed mounts/destinations/secrets |

Argument mutation after approval needs final-argument checking and controlled
handler order. Generic errors may continue upstream; only observed blocking paths
count. Parallel approval consumption needs atomic reservation; these example
traces do not implement it. Cancellation/expiry/absent approval denies the covered
action, but cleanup still needs descendant settlement. Worktrees are ownership
separation, not containment. Unattended support needs the actual shell **and**
extension/SDK forbidden-read/write/network/credential tests in the selected
nonprivileged deployment, no management sockets or host fallback, read-only
inputs, assigned writable roots, scoped secrets, deny-default egress and process,
RAM, CPU and deadline enforcement. Unknown work is retained, not cleaned up.

Provisioning, acquisition/build, static placement, Pi-native activation, trust,
live execution, publication and cleanup are distinct effects/grants. Static apply
never executes npm/git/lifecycle scripts or registers extensions. Default scripts
policy is deny, including transitives; any exception requires separate isolated
review/authority. Do not copy ambient runtime trees or imply root script absence
proves transitive safety. Update source/closure/config digests, quiesce consumers,
retain old/new evidence, review deltas and requalify before making a new claim.

## Review, failure and migration

Normalize findings by consequence, retaining original source IDs/severities.
Deduplicate canonical defects: unresolved Critical/Major or more than two distinct
Medium findings block review completion. Zero/one/two Medium may complete only
with explicit owner residual acceptance. Low never requires correction. One cycle
can close; a third is refused. Spare review waves are not an acceptance condition.
Failed required tests, missing authority and release gates block independently.

Changed post-review bytes need fresh parent verification with old/new hashes and
an explicit disposition, not retroactive independent-review attribution. Retain
the original reviewed bytes separately. All input/output hashes must be covered
by independent review or a declared parent-verified correction chain. No checker
judges consequence or authenticates alleged acceptance.

Unsupported/newer versions, unreadable inputs, drift or a checker regression block
admission. Retain records, previous decisions, raw logs, locks and rollback
instructions for diagnosis; never rewrite/migrate them silently. QP-02 supplies
real dossiers, QP-03 supplies procedures and authorized I evidence, and QP-04 owns
release admission. This task supplies neither real qualification nor publication.

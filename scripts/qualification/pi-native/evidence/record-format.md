# Evidence record format, version 1

The templates illustrate field names; their unknowns are intentional. JSON is
UTF-8, with no duplicate keys or NaN/Infinity. `version` is integer `1`, not a
boolean. Unknown/newer versions are refused, not rewritten. Extra descriptive
fields do not replace required fields. Required case/outcome/claim sets are exact.

## File references and limits

A reference is `{"path":"relative/file","sha256":"64 lowercase hex digits"}`.
Arrays of references have unique paths. No absolute paths, `..`, backslashes,
noncanonical paths or symlink components are allowed. Only regular files beneath
`--root` are read; FIFOs/devices/directories refuse without waiting. Directory-FD
opens with `O_NOFOLLOW` avoid path-check/follow races on the supported POSIX host.
JSON files are limited to 4 MiB, other evidence files to 64 MiB. Hashing uses
64 KiB chunks. Each invocation permits at most 4096 reads and 2 GiB total bytes
read (including rereads), bounding work without loading archives into memory.
Larger subjects require deliberately bounded evidence packaging and a reviewed
checker-contract change; never silently omit required bytes. These are checker
limits, not a claim about the installer's archive limits or web response bounds.

The JSON result contains `ok`, `mode`, requested `claim`, and a `checked` map of
fresh path-to-SHA-256 values. Success includes `contract_sha256`; rejection includes
`errors`. Exit 0 means the requested check succeeded, 1 means a record/input/check
refused (CLI syntax errors use argparse's nonzero status). Input digests, not Git
tracked status, identify exactly what was checked. The executable and its bundled
cases contract are trusted owner-selected tooling, not self-authenticating inputs.

## Shared candidate/qualification fields

| Field | Meaning/check |
|---|---|
| `kind`, `example`, `status` | `candidate` or `qualification`; boolean example label; evidence status `verified`, `observed`, `inferred`, `proposed`, `unknown`. Example, inferred, proposed and unknown cannot qualify. |
| `claim`, `class` | Finite claim ID and explicit `M`, `C` or `I`. Runtime qualification packets must be I; static packets M/C. |
| `source` | Absolute source root, exact revision, `dirty_inventory` references including selected dirty/untracked/ignored inputs. Runtime admission requires a 40/64-hex revision and inventory within granted inputs. The checker does not discover Git inventory. |
| `inputs`, `outputs` | Tested artifacts, config, candidate documents, evidence inputs and output references. Complete records require nonempty arrays. |
| `environment` | Nonempty platform and `pins` object of named exact selected version strings. Unknown cannot qualify; owner checks exactness and support. |
| `grant` | Full grant object below, not an alleged signature. |
| `invocation` | Exactly one nonempty `argv` array OR `native_tool:{name,arguments}`; `started_at`, `ended_at` UTC timestamps. Nothing is executed. |
| `result` | `pass`, `failed`, `unknown`, `bypass` or `skipped`; qualification requires pass. |
| `expected`, `observation` | Nonempty expected/actual observation descriptions; owner reviews their meaning against logs. |
| `logs`, `exit_code`, `settlement` | Hashed logs declared as outputs; integer exit or null when unknown; `settled`, `not-applicable`, `unknown`, `failed`. Qualification requires exit 0 and settlement (I case rows always `settled`). |
| `review` | Review record below; independent reviewer identity, hashes and disposition. |
| `gates` | Separate `required_tests`, `authority`, `release` outcomes; every declared gate must pass independently of review count. |
| `limitations` | Array of explicit limits, residual risks and withheld claims. Empty only if honestly appropriate. |
| `owner_acceptance` | `{owner,result,claim,stage,inputs}` decision, matching the packet's claim and exact input hashes. Qualification requires pass. |

Timestamps use ISO UTC with a `Z` suffix. The declared invocation interval is
ordered; per-case intervals lie within it. There is no inferred execution from
an exit code or a source inspection.

## Grant

`kind:grant`, `version:1`, `status`, `issuer`, `issued_at`, `action`, `roots`,
`outputs`, `inputs`, `expires_at`, `revalidation_trigger`, `owner_verification`.
Templates additionally label `example:true`; qualification refuses example grants.
`roots` are canonical evidence-root-relative directories (`.` is allowed).
`outputs` are exact relative output paths; `inputs` are hash references.

At least an expiry UTC or nonempty revalidation trigger is required. Expiry must
follow issuance. `owner_verification` contains `owner`, `result`, `checked_at`,
and exact copies of `action`, `roots`, `outputs`, `inputs`; changed declarations
refuse. Verification cannot predate issuance. Qualification checks current declared
expiry and invocation/verification time, stage/action equality, every input/output
inside declared roots, and exact input/output scope equality. A revalidation-only
grant still binds the selected hashes; changed inputs need a new verification.
These are consistency checks, not proof the named issuer authorized anything.

## Candidate dossier (Q-01)

Adds nonempty `identity`, archive `digest` (SHA-256 or null when unknown),
`retrieved_at` UTC, `immutable_refs:[{url,ref}]` and `dossier`. Every dossier entry
contains `status`, `value` (owner-reviewed JSON, null for unknown), and local hashed
`references`. Immutable URL/ref semantics and evidence meaning remain owner checks.

Required entries and the information their owner-reviewed value must describe:

| Entry | Required meaning |
|---|---|
| `identity` | Package/repository identity and source ownership |
| `version_ref` | Exact selected version and immutable commit/ref |
| `archive` | Registry/archive URL and digest, not just a moving tag |
| `installed_comparison` | Installed-byte comparison and its scope/status |
| `license_attribution` | Licenses, notices and redistribution obligations |
| `dependency_tree_sbom` | Locked full transitive tree, SBOM and closure assessment |
| `lifecycle_build_scripts` | Root AND transitive lifecycle/build scripts, deny policy and reviewed exceptions |
| `provenance_signature` | Subject, builder/signature/provenance verification result or explicit unknown |
| `reachable_advisories` | Dated reachable advisory assessment, not an absence-of-advisories safety claim |
| `upstream_tests_ci` | Upstream test/CI evidence and limitations |
| `platform_runtime` | Exact OS/architecture, interpreter/runtime/dependency environments actually observed |
| `privilege_network_credentials` | Privileges, network destinations, secret and credential surfaces |
| `update_remove_rollback` | Owners, consumer quiescence, migration, removal and rollback procedure |
| `alternatives` | Alternatives and use-case disposition |
| `residual_risks` | Known gaps, consequences, mitigations and withheld support |
| `approval` | Owner decision and granted support scope |
| `delta_to_latest` | Selected/latest-observed versions and date, intervening notes, compatibility/security fixes/advisories and reason to retain the pin |

Record mode accepts genuinely unknown values. Qualification requires every entry
observed/verified, non-null/non-unknown value and hashed references within the
candidate's reviewed inputs/outputs. The candidate digest must equal an actual
freshly checked archive-reference digest. Candidate common review/grant/gates also
must pass. A field marked observed is a declaration, not a semantic/security audit:
the owner must reject a superficial or false dossier even if it is structurally
complete. Never upgrade dependencies merely to fill a delta field.

## Qualification packet and finite selectors

Adds `cases` (reference to a byte-identical copy of this checker's `cases.json`),
`claims`, `dependencies`, `applicability`, `deployment_acceptance`, `evidence`.
The template's `required_set_rules`, when present, must match `cases.json`;
it is a readable copy, not authority to reduce the finite contract.

`claims` contains **every** finite claim ID. Each entry has `status:selected` or
`unselected`, `required:[selectors]`, and `decision:{owner,result,claim,stage,inputs}`.
Unselected entries have empty required arrays. All selected parents must be
selected, all selected decisions match the common stage/inputs, and every selected
claim's required set must exactly equal the computed union. No discretionary skip.
This single packet supplies lower-level claim evidence on the same source/config/
input set rather than importing an unrelated prior pass label.

A selector is `CASE:CLASS`, or for dependency-scoped Q rows,
`CASE:C@role:archive-sha256`. Examples: `D-T1:M`, `OP-TRUST-PI:I`,
`Q-T1:C@pi-host:<64-hex-digest>`. Both class-tagged I-T2 rows are required.
Grouping labels OP-TRUST/OP-APPROVAL cannot appear as case selectors.

`dependencies` entries are `{role,group,identity,digest,candidate}`. Candidate is
a file reference also present in the packet inputs. Roles are unique lowercase
letters/digits/hyphens. Identity and digest must match the referenced candidate.
The finite contract requires the named core roles (including mandatory web) and,
for code-intel, its optional roles. Additional selected closure/runtime
prerequisites use distinct roles in the applicable group; **each** receives all
Q-T1/Q-T5/Q-T6 C selectors. Known mandatory roles cannot be moved into another
group, and an optional dossier cannot satisfy a core role. The owner checks the
completeness of the selected dependency inventory; the checker never discovers
installed packages or pretends to verify closure from a role label alone.

### Runtime applicability

Static-only packets use `applicability:null` and need no I evidence. Runtime
packets supply:

```json
{
  "mcp-enabled": false,
  "effective_config": {"path": "effective-config.json", "sha256": "<digest>"},
  "inventory": {"path": "inventory.json", "sha256": "<digest>"},
  "owner": "owner identity",
  "result": "pass",
  "interpretation": "No MCP surface selected in this effective deployment",
  "reviewed": ["the exact two reference objects above"]
}
```

The last array is descriptive shorthand here, not a usable JSON record. The
inventory file contains `selected_mcp_surfaces:[nonempty surface identities]`,
empty for false. Both files must be bound to inputs, reviewed together, and
included in overall independent review/parent correction coverage. Unknown or
conflicting applicability refuses. True adds both -MCP selectors to every runtime
claim. Code-intel with false refuses. False is not automatic discovery of no MCP;
the owner must interpret actual effective config and inventory correctly.

### Evidence rows

Each `evidence` row contains `id`, `class`, `dependency` (null for global cases;
`{role,identity,digest}` for Q dossiers), `inputs` (exact packet input set),
`result`, `outcomes`, `invocation`, `logs`, `exit_code`, `settlement`, `expected`,
`observation`. No duplicate selectors. Case classes and outcome names are exactly
those in cases.json; all selected negative outcomes remain mandatory.
A pass row needs all mandatory outcomes pass, exit 0, logs and settlement.
Case logs must match packet outputs by path **and** hash.

For Q-T3-M, Q-T5-M and Q-T7-M, include `model_observations`, the respective
example observation array. Each row has `scenario`, nonempty event `trace`,
`approved_argv`, `observed_argv`, `expected`, `observed`, `outcome`. The final trace
entry is `{event:"observe",value:<observed>}`. Each required scenario occurs once;
its outcome agrees with the case's matching outcome. Claimed pass requires
expected/observed equality and cannot hide unknown/failed/bypass observations.
The checker validates the report's consistency, not the simulated interception,
provisioning or HTTP policy. Tests assert the invented trace/argv behavior.

| Mixed Q row | What these files establish | What remains I-only |
|---|---|---|
| Q-T3-M / Q-T3 | C independent-model consistency + M checker-record coverage | Actual approval handler order/mutation/parallel denial and shell/SDK/isolation bypasses |
| Q-T5-M / Q-T5 | C independent-model consistency + M checker-record coverage | Actual provisioner argv/scripts denial, construction, activation and interruption behavior |
| Q-T7-M / Q-T7 | C independent-model consistency + M checker-record coverage | Actual retrieval/provider behavior, presentation tests and hard bounds |

OP-WEB additionally has a `bounds` object with **every** finite bound outcome,
even in core-functional evidence. Functional cases may disclose unknown/failed
stronger bounds; bounded-unattended requires all pass. A functional negative-path
pass means the stated expected observation occurred; it does not mean a hard
limit was enforced. Describe absent limits honestly in observation/limitations.
For unattended claims `deployment_acceptance` must be an owner decision with the
requested claim, `stage:whole-process-deployment`, matching inputs and pass.

## Review records and corrected bytes

`review` contains `cycles` (0..2 structurally; completion requires 1 or 2),
`reviewer`, `reviewed` references, `corrections`, `findings`, and
`disposition:complete|incomplete`.

A finding has `id` (canonical identity), `sources:[{id,original_severity}]`,
`severity:Critical|Major|Medium|Low`, nonempty `rationale` and `consequence`, boolean
`unresolved`, `owner_acceptance:{owner,result,reason}`. Duplicate canonical IDs
must agree on every nonsource field; duplicate source IDs must retain the same
canonical identity and original severity. All source rows are preserved. The
checker deduplicates for the threshold, not to erase reviewer attribution.
Unresolved Critical/Major or >2 Medium blocks; every remaining Medium needs an
explicit accepted residual. Low is nonblocking regardless of acceptance. A
resolved finding remains historical evidence. Humans judge consequence and
normalization; the checker only checks declared consistency.

A correction is `{old:<retained old reference>,new:<current reference>,
disposition:"parent-verified-not-independently-reviewed",parent:<identity>,
verified:<exact new reference>}`. Old bytes must still be available and their hash
independently reviewed (or covered by an earlier explicit correction in the
chain). New bytes are verified freshly and covered as a **parent correction**,
not relabeled independent review. Updating only the expected digest or only the
review count cannot repair stale coverage. Every tested input/output/log must be
covered; review completion never waives the separate test/authority/release gates.

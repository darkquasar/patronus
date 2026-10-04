# Pi-native delivery and migration

## Current Pi-native contract

The current design has two distinct delivery paths:

- Patronus places and owns authored profile artifacts, including the three installed
  workflow skills `workflow-research-pi`, `workflow-implement-pi` and
  `workflow-peer-review-pi` and their declared sidecars.
- For `pi-subagents` and `pi-web-access`, Patronus records the exact npm reference,
  selected Pi identity/root/scope, profile membership, provenance, observation and
  pending/outcome data needed for safe lifecycle decisions. Pi owns package
  installation, removal, dependency resolution, scripts, package declarations and
  installed package files. Patronus invokes the selected Pi package manager; it does
  not copy or recursively delete npm package trees.

The recipes select `npm:pi-subagents@0.72.1` and
`npm:pi-web-access@0.35.0`. There is no vendored or static package payload for either
extension, no alternate handwritten `packages` wiring, and no Patronus fingerprint
ledger for npm-internal files or transitive dependencies. An exact direct reference
is not a transitive lock, package-integrity claim or runtime qualification.

Preview remains non-mutating. Deployment that needs native installation or update
requires the package-install consent, and project-scope mutation also requires the
Pi project-config consent. Normal update/removal reports declaration drift, tracked
external packages, shared profile use and unresolved prior operations, preserving
those conflicts rather than silently overwriting or deleting them. An authorized
`--force` may remove the selected, profile-tracked authored artifacts or selected
controlled native identity despite the documented drift/sharing protections. It
cannot broaden the selected identity or root, enroll an external package, bypass
install/trust consent, delete unrelated files or package trees, accept malformed or
ambiguous ownership, or turn Pi/npm/OS errors into success. Manager failures and
post-operation observations are retained and reported as partial or unresolved
outcomes; operators must re-observe before retrying.

Static/profile fixture tests establish model-free Patronus behavior only. Actual
installed-entrypoint qualification is a separate activity: it must bind the real Pi
and Node versions, resolved package bytes and dependencies, effective configuration,
provider/runtime inputs and observed workflow/tool behavior. Do not infer completed
qualification, platform support or release approval from fixture success. See
[Pi core dependencies](pi-core-dependencies.md) and the dated
[support matrix](pi-qualification/support-matrix.md) for their narrower claims and
limits.

## Superseded static-delivery record

**Historical status:** everything from this heading onward records the earlier
static/archive design and its evidence. It is superseded as operational guidance by
the Pi-native contract above. Preserve it as historical evidence; commands and
claims below about Patronus-owned package payloads, receipts, manual local-source
registration or qualification handoffs do not describe the current package path.

**Historical implementation status:** the former Pi delivery mechanisms were covered
by invented offline fixtures. That was not qualification of a deployed core profile,
Pi, subagents, web access, providers or optional MCP services. No release, live
migration or trust grant follows from this record.

### Historical target, scope and roots

Use an explicitly selected catalog item/profile; names below are metavariables.
Run previews first, save their output, then obtain the appropriate apply grant.

```sh
patronus install --profile <profile> --target pi --global
patronus install --profile <profile> --target pi --global --deploy
patronus lock --profile <profile> --target pi
patronus install --profile <profile> --target pi --local
patronus update <profile-or-item> --target pi --local
patronus update <profile-or-item> --target pi --global
patronus remove <item> --target pi --local
```

Install/update/remove preview by default; `--deploy` applies. Pi updates require
**exactly one** of `--local` and `--global`, including when installed provenance
establishes Pi implicitly. Do not use `all` to replay a Pi selection. Removal is
item-level, not a profile inverse or reference-counted lifetime. Removing an item
leaves desired lock pins intact. Profile update reports absent members and does
not recreate them; an absent required prerequisite blocks the selection. Reinstall
is a separately previewed explicit install, not an update side effect.

| Resource | Global | Explicit workspace |
|---|---|---|
| Skill and declared sidecars | `<agent-dir>/skills/<name>/SKILL.md` | `.pi/skills/<name>/SKILL.md` |
| Prompt | `<agent-dir>/prompts/<name>.md` | `.pi/prompts/<name>.md` |
| Native Markdown agent | `<agent-dir>/agents/<name>.md` | `.pi/agents/<name>.md` |
| Scalar settings | `<agent-dir>/settings.json` | `.pi/settings.json` |
| MCP structural leaf | `<agent-dir>/mcp-adapter.json`, `mcpServers.<name>` | `.pi/mcp-adapter.json`, same leaf |
| Context | effective agent-directory context | effective workspace/ancestor context, **not** `.pi/AGENTS.md` |

`<agent-dir>` is `PI_CODING_AGENT_DIR` when supplied, otherwise `~/.pi/agent`.
Preview its canonical path; symlinks, escapes and uncertain effective-root
interpretations refuse. Later removal uses recorded absolute paths, not a changed
environment variable. Relocation needs a separate migration from those old roots.
Hooks and output styles are unsupported Pi surfaces. Settings artifacts do not
write arbitrary files such as `web-search.json`. HTTP MCP emits `{url: ...}`;
stdio command/args mechanism support is not permission to launch a server.
See the [finite native agent contract](agent-artifacts.mdx#selected-pi-native-agent-subset).

### Historical static admission boundary

Patronus inspects statically discoverable native/local/package files, structured
configuration, manifests, declarations and statically resolvable imports. It
checks known skill/prompt/agent identities, required readable inputs, unsafe
paths, malformed/unsupported static schemas, effective context and MCP source
conflicts, ownership/drift and consent. A known static conflict refuses the whole
selected apply before its first resource/state/lock write; it cannot be waived as
runtime uncertainty. The host coordination inode may still be created.

Configured `settings.extensions`, package `pi.extensions` and native extension
entrypoints are **not blanket blockers**. Dynamic callbacks, executable resource
registration, resolver-only sources and runtime-only names remain uninspected and
visibly **runtime-unverified**, with known source/configuration context. Patronus
never imports them or guesses their names/precedence. An explicitly selected
static declaration that cannot safely be validated remains a concrete conflict.
Static success means no conflict found **within the inspected inventory**, not
complete runtime collision detection. No extension-deactivation requirement is
implied by this uncertainty.

Agent discovery includes qualified native user/project/ancestor directories,
explicit static extra directories, local/cached package declarations and available
builtin sources. Global npm agent discovery can depend on executing `npm root -g`:
Patronus never runs it and reports that uninspected envelope as runtime-unverified.
`PI_OFFLINE` is not a production admission requirement. Fixtures may set it for
determinism; runtime qualification must record its actual value alongside package,
agent-root, extra-agent-directory and MCP configuration overrides. It is not an
OS/network sandbox. Actual cold/reload inventory must separately establish intended
Pi and subagents sources/precedence in that exact runtime environment.

Dry preview is read-only and network-free; a cold remote source may require a
separately authorized acquisition rather than an implicit preview download.
Authorized apply may acquire approved archives through bounded, digest-verified
seams. **Acquisition is not code loading.** All required delivery bytes must be
available/verified before the first selected payload write. Unsupported platforms,
invalid pins, unresolved required resources or acquisition failure refuse without
installing an unrelated subset. Static apply does not run npm/git resolvers,
package scripts, Pi, extension factories, helpers, provider discovery, MCP
connections, LSP servers or graph builds.

### Historical governed migration checklist

1. Stop old mutators and quiesce dependent work. Inventory **all** effective homes,
   projects and consumers, including Claude users of shared global dependencies.
   Back up state, locks, receipts/journals, source manifests/pins, payloads and exact
   config/context bytes, modes and paths. Record hashes, binary provenance, target,
   scope, environment and ownership. Protect credentials separately; do not copy
   them into ordinary logs or evidence bundles.
2. Inventory manual prototypes, exact-name collisions, npm/git/local duplicate
   packages, active overrides and functional duplicates with different names.
   Record old/new source, description, intended role, effective scope and an
   explicit retain/replace decision. CLI checks exact static identities/config;
   it cannot infer equivalent workflows or authenticate a migration ledger.
   Preserve prototypes and unknown files until separate migration authority exists.
3. Inspect the actual context discovery order: `AGENTS.override.md`, `AGENTS.md`,
   `AGENTS.MD`, `CLAUDE.md`, `CLAUDE.MD`. An earlier unreadable candidate is an error;
   override selection is warned. Do not create a higher-priority shadow file or
   replace `SYSTEM.md`/`APPEND_SYSTEM.md`. Capture both harnesses' effective sources.
4. For a CLAUDE filename, Claude fences or another owner, capture **ORIGINAL**
   pre-migration bytes and grant **before** manually preparing complete combined
   shared instructions and clearly addressed harness-specific sections. Patronus
   does not synthesize the combined text. Its distinct interactive preflight shows
   canonical path, both effective sources, proposed Pi text and prepared-prior/result
   hashes. `--yes`, `--force`, headless input and normal overwrite approval cannot
   consent. Any reread difference invalidates consent and requires fresh review.
   The CLI's inverse prior is only the **prepared** file it read; removing Pi fences
   never undoes that separate migration or reconstructs original bytes.
5. Capture the normal CLI preview plus exact selected inputs/roots/target/scope;
   hash those bytes and obtain a matching coordinator apply grant. Recheck/re-preview
   changed inputs. Missing/stale grants stop this governed workflow. This is
   cooperative procedure, not a CLI ledger reader, `--preview-digest` feature or
   atomic security boundary: direct CLI invocation can bypass the workflow.
6. Apply only the approved selection. Review observed state/receipt/result evidence;
   placement makes native files eligible at the next startup/reload, not trusted
   or loaded now. Retain partial-effect diagnostics; do not assume a transaction.

### Historical global prerequisites and manual activation

The required core design includes globally owned **tk**, **pi-subagents** and
**pi-web-access 0.35.0**. Pi host and its host peers remain external. This names the
deployment subject, not hardcoded installer behavior or proof these payloads are
qualified. Generic script-fetch and install-only global directory recipes carry
these dependencies; directory roles are the finite set sandbox/orchestration/tools.
See [package delivery](package-delivery.md) for receipt/archive mechanics and
[dependency build admission](pi-qualification/core-dependency-build-admission.md)
for real payload/provenance gates. This Pi runbook's stronger whole-selection and
outer-lock requirements apply to Pi even where older generic examples differ.

Local install/update requires current compatible **owned** global script bytes and
directory receipts/pins. Equal unowned files, stale/missing state or drift refuse
before local writes. Perform any required global install/update as a separate
operation. Local updates never upgrade globals; local removal preserves them.
Global selections retain agnostic rows and preserve local state/resources/locks.
Profiles do not confer independent shared-dependency lifetimes.

For **each** delivered extension, activation is a separate operator-owned action:

1. Verify the receipt at `~/.patronus/package-state/<recipe>.json`, its recipe/version,
   archive digest, canonical `root`, member digests/modes and identity against the
   approved package. Derive the nested source as **receipt root + `/pi`**, verify it
   remains inside that root without symlink redirection, and inspect its actual
   `package.json`/entrypoints. Do not infer the installed root from an ambient npm
   tree or a remembered source checkout. Missing/drifted receipts mean retain/escalate.
2. Inventory effective global/project npm, git, local and one-shot sources; resolve
   duplicates explicitly before registration. Capture original global and project
   native config bytes, effective scope, exact source path, receipt and hashes.
   Obtain a separate trust/**execution** and configuration grant; placement authority
   alone is insufficient.
3. Under that grant, use the selected Pi version's native local-source registration
   (`pi install /canonical/receipt-root/pi`, with its qualified project-scope option
   when intended), or separately qualified one-shot `-e`. Verify resulting scope,
   config diff and effective inventory rather than assuming a scope flag. **Patronus
   never calls `pi install`/`pi remove` or edits packages arrays.** Local-source loading
   does not install missing dependencies. Never npm/git mutate the owned payload.
4. Provision external Pi/Node/host peers/provider authentication and any selected
   optional dependencies separately under their own grants. Credentials use narrow
   ephemeral secret files, never logs/images/commits. Optional code-intel MCP services
   require independent config/trust/loading qualification, not automatic startup.
5. Apply the reviewed inert web-default template explicitly to the selected
   `PI_CODING_AGENT_DIR/web-search.json` only in the granted clean bootstrap, or after
   preview/prior/drift review for existing config. Required web defaults are DDG-only,
   eager tools, `workflow:none`, raw fetch, no automatic summarization/background
   fetch, no paid fallback, `maxInlineContentChars:8000`, and disabled browser/cookie/
   clone/media behavior where supported. Sanitize cookie opt-ins. Guidance uses
   <=5 search results, `includeContent:false`, `source_check` with `fetchContent:false`,
   selective fetch and `get_search_content` pages <=2000 characters. Observe actual
   tool paths and disabled behavior; 8000 is presentation, **not** a network-byte,
   cumulative-token/cost or OS-egress bound. Bare static install does not apply this
   template. Stronger unattended web bounds remain unqualified.
6. Run separately authorized cold-start and background/subagent/provider/host-peer
   smoke checks, actual DDG and response/pagination/error tests, and effective agent
   discovery/overrides checks. Record exact environment/config/payload hashes and
   observations. Keep payloads read-only with all caches/logs/output elsewhere;
   required in-tree writes fail qualification and later appear as removal drift.
   Do not mark core operational until these mandatory runtime checks pass.

Before updates, settle **ALL** dependent sessions, descendants, detached/background
work and shared consumers; retain outputs and record reload/restart acknowledgement.
Unknown consumers/process state/evidence mean retain/escalate, not presumed idle.
For extension replacement follow the qualified deregister/replace/register sequence.
Before removal, manually deregister the **exact local source** using qualified
Pi-native handling, verify config/effective inventory and preserve its original
configuration evidence, **then** preview the Patronus owned-payload inverse. Native
local removal deregisters, not deletes the payload. Stop if the observed behavior
differs. CLI cannot discover arbitrary homes/projects or authenticate consumer records.
Loaded sessions may retain old bytes until explicitly reloaded/restarted.

### Historical ownership, failures and downgrade

Scalar/MCP removal restores each continuously owned leaf's original absent/present/
null baseline, preserving unrelated siblings—not a stale whole JSON file. Same-owner
updates and no-op reinstall retain that baseline. External equal leaves remain
unmanaged; differing values, path changes, ambiguous priors or overlaps conflict.
JSON serialization may lose formatting/comments: review this in the diff.

Unchanged owned files can be removed; drift/unknown content is retained with a
conflict. Dropping a skill sidecar from a manifest does not erase its ownership:
this implementation retains obsolete paths and reports unresolved ownership until
explicit removal. Do not call an unresolved item fully updated; preview removal
and reinstall separately, preserving edited sidecars. Directory receipts remain
ownership authority; generic state rows alone cannot adopt a tree. Receipt-backed
recovery is not permission to reconstruct ordinary file ownership from equality.

New-release writers acquire the outer advisory lock at
`<canonical-home>/.patronus/package-state/mutation.lock` before relevant reads,
holding it through config/state persistence. Busy fails fast: settle the holder,
then re-preview; do not delete the lock inode or bypass it. Read/hash checks catch
detected external changes but leave a final-check/write race. Different users/homes,
old binaries, editors and Pi do not share the guarantee. Shared projects still
need operator-enforced single-writer access. There is no transaction across files,
packages, state and activation.

After any write/state-save failure stop, retain old durable state plus all committed
path/result/uncertainty evidence, and preview again from current bytes. Successful
unrelated writes are real partial effects. Equality alone cannot repair/adopt an
unrecorded file. Reconstruction needs retained prior/result evidence and an explicit
operator adoption decision. Do not automatically retry or clean ambiguous state.
State-save failure now returns an error for affected **non-Pi** operations too;
new-release non-Pi writers also fail busy rather than racing.

Explicit Pi profile locks use numeric schema **3**, `target: pi`; newly generated
non-Pi locks remain **v2**, without target. Supported legacy locks load without
rewriting. Pi use of target-less locks requires proven operation/installed provenance
or explicit target, closure-difference preview and deliberate regeneration—never a
host-default guess. Ordinary legacy non-Pi lifecycle remains per-item/tool/scope.
The v3 guard protects lock readers only; old binaries may still mutate ordinary
installed state through direct remove/recovery. **Mixed old/new Pi mutators are
unsupported.** Before downgrade use the qualified new binary to remove Pi items or
explicitly export/retain them with residual-state explanation and excluded old
mutation routes. Back up first; deregister extensions before removing payloads.
Rollback needs separate authority and renewed static/runtime checks; restoring
code or backups is not automatic transactional rollback of deployed resources.

### Historical evidence layers and repeatable checks

1. **M — application capabilities:** invented manifests/bytes, isolated roots,
   denied network and no acquired third-party execution. The integration fixture
   is [`pi_delivery_integration_test.go`](../cmd/patronus/pi_delivery_integration_test.go).
2. **C — catalog validation/review:** generic loaders/properties inspect whatever
   catalog exists at the tested revision. They do not mirror named membership/count
   lists in Go or prove a selected package works. Later content owners must validate
   their own real closure and source inventory after delivery integration.
3. **I — deployment qualification:** separately authorized selected real payload/
   Pi/provider/config runtime tests. Invented fixtures cannot substitute for these.

With a provisioned toolchain/cache, >=750 MiB free **and** available RAM and at least
500 MB reserve, serialize all Go checks using the shared lock. Missing cache/toolchain
is **BLOCKED**, not permission to download/install. Record command, UTC times, exit,
nonzero executed-test counts and complete log, not just a green summary.

```sh
# M plus existing generic C/legacy regressions; no runtime qualification
flock -w 300 /home/agent/.local/state/patronus-code-intel/pi-implementation-test.lock \
  env GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMAXPROCS=2 \
  go test -p 1 ./... -count=1 -v

# Separately labeled QP-01 record-checker fixtures, not installer or runtime tests
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/tests -p 'test_pi_*.py' -v
```

D-T1..D-T8 mechanism mapping: target/lock/scope (`pi_lifecycle*`, lock/profile tests);
discovery/native grammar/legacy goldens (`pi_preflight*`, scan/adapter tests);
composition/inverse priors (plan/adapter/state/remove tests); races/faults
(`mutation*`, install/update/remove tests); migration/sidecars/native lifecycle
(`pi_delivery_integration*`); dependency acquisition/receipt recovery
(`pi_delivery_integration*`, directory lifecycle and packagedelivery tests). All
D-01..D-12 are integrated here; passing M does not upgrade any I outcome.

Bind exact source revision, dirty/test-input inventory and SHA-256 hashes, command
logs and outcomes to the [QP-01 record contract](pi-qualification/record-format.md)
and [finite cases](pi-qualification/cases.json). Preserve review and authority
separately; a writer report or checker exit alone is not owner acceptance. This
milestone's unfiltered suite covers only its catalog revision, not future core or
code-intel content. Static-delivery claim also needs the applicable C evidence;
DP-07 alone does not assert that whole claim. Failed integration blocks release.

#### Historical QP-03 handoff: baseline-binary compatibility

Under a **separate provisioning/build grant**, Q's operator obtains a binary for
baseline `32741e687647e44c4aed071d90965f34d102de52`, recording executable digest,
source/build provenance and exact toolchain/platform. In a disposable isolated
home/workspace with no Pi installed state:

- Generate a Pi v3 lock with the new binary; verify the baseline rejects it via
  its numeric version guard (not merely unknown-field parsing).
- Generate new **fixed-time non-Pi v2** locks with the approved deterministic
  fixture/build procedure. Compare baseline bytes and prove the baseline accepts
  them and original Claude catalog/locks unchanged.
- Retain commands, input/output/binary hashes, rejection/acceptance logs and actual
  outcomes. Do not run that old binary against Pi installed state.

This I compatibility case is **not run by DP-07** and is distinct from M's frozen
version-guard fixture. Real subagents/DDG/Pi-provider and platform observations
remain QP-03 work; final qualified support claims remain QP-04 decisions.

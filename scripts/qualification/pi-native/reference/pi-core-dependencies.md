# Pi core dependencies: native package contract

The selected core is **upstream-only**: Pi **0.87.1**, Node **22.22.1**,
pi-subagents **0.72.1** and pi-web-access **0.35.0**. These are qualification
targets, not a claim that this catalog change qualifies their runtime together.
The owner declined the Stage-B comparative live trials; they did not pass.
Neither tk nor pi-better-plan nor pi-better-goal is a Pi core dependency.
Generic/non-Pi tk delivery remains available.

## Exact sources and ownership

| Dependency | Exact source / source metadata | Declared license |
|---|---|---|
| pi-subagents | `npm:pi-subagents@0.72.1`; [npm metadata](https://registry.npmjs.org/pi-subagents/0.72.1); [upstream source](https://github.com/nicobailon/pi-subagents/tree/fece3cea193f47593685668b3969167a09cdf60c) | MIT |
| pi-web-access | `npm:pi-web-access@0.35.0`; [npm metadata](https://registry.npmjs.org/pi-web-access/0.35.0); [upstream source](https://github.com/nicobailon/pi-web-access/tree/72c6e67787d67d8a7d01bf0abf30c072a6112de6) | MIT |

Both Patronus recipes are **2.0.0**. Recipe versions and direct upstream npm
versions are independent. The recipes use `deliver.via: package-manager` with
one `manager: pi` candidate and an exact npm ref. There is no alternate manager,
static archive, vendored replacement payload or handwritten packages-array wiring.
Pi itself and Node are externally provisioned. The separate sandbox retains its
**pi-subagents 0.71.0** pin and separate qualification; it is not this core target.
Generic directory delivery and sandbox packaging are unchanged.

Pi/npm own installed plugin files and dependency resolution, including native
installation scripts. Installation requires the existing package-install consent
and an explicit script-execution warning; project trust remains a separate gate.
The configured Pi `npmCommand` is respected, not replaced or silently forced into
ignore-scripts mode. Exact direct pins do not lock a transitive closure or certify
source/build equivalence, dependency licenses, security or compatibility.
Patronus keeps no per-file or transitive npm ledger.

## Scope, preview and lifecycle

This is the native lifecycle contract; its Go implementation and integrated
validation are supplied separately from this catalog change.

- Without `--deploy`, operations are plan/preview only: no subprocess, network,
  package mutation or state write. `--force` alone applies nothing.
- Preview identifies the exact selected agent root, project root/scope, npm
  identity and desired source. Use the selected canonical `PI_CODING_AGENT_DIR`
  and project root, not an assumed HOME or ambient npm installation. Local
  selections must not cause incidental global installs or other-root mutations.
- Read-only observation of native declarations and selected installed
  `package.json` identity/version establishes only those facts. Observing a
  compatible pre-existing package does not enroll it: ownership requires a
  confirmed installation or explicit tracking. A tracking date is not an
  installation date. Record provenance and dates honestly.
- Detectable declaration/source/version drift is reported with its exact root
  and recorded date where applicable, with the existing force guidance. Internal
  npm-owned file edits are **not checked**. For native replacement/removal warn:
  “This operation may discard manual edits inside the installed plugin. Patronus
  does not check those files for modifications.” This also includes possible
  native-manager dependency effects; metadata agreement is not file integrity.
- `--deploy` authorizes selected mutation. Existing authored-content edit
  safeguards remain. Normal removal retains known edits/sharing; selected
  `--force` removal may override those protections, leaving other consumers
  unsatisfied rather than reinstalling them. Force never expands roots or
  selection, bypasses install/trust consent, or excuses malformed ownership or
  OS errors. Unobserved internal edits have no force-based protection.
- Updates replace the exact selected pin through Pi; no broad `pi update` or
  `--all`. Removal delegates the controlled npm identity to Pi, not a Patronus
  recursive deletion of npm trees. Preserve unrelated settings and credentials.
  Failures can leave partial effects: report confirmed and unresolved outcomes,
  re-observe before retry, and never invent receipts or automatic rollback.

## Evidence and remaining qualification

Source/metadata inspection establishes declarations and intended APIs. Offline
invented-fixture tests establish only the tested Patronus mechanisms. Native
installation establishes package-manager effects, not successful loading. Loading
establishes registrations, not provider operation or workflow correctness. Live
runner/provider tests and end-to-end installed-workflow qualification are separate
claims requiring their own exact inputs, roots, versions, outputs and omissions.

This catalog seam performs no native installation, plugin loading, provider calls
or installed-workflow dogfooding. Native lifecycle integration and runtime
qualification remain pending L123/Q2; historical static-package smoke and closure
evidence do not qualify freshly resolved native dependencies. Required web tools,
reviewed DDG defaults and disabled behaviors need new native smoke; configuration
presentation limits are not network-byte, cost or egress enforcement. Missing
runners/authentication and untested platforms must remain explicit omissions.

The two obsolete static package trees are removed from the current source tree.
That deletion does **not** clean contaminated Git ancestry. Final delivery still
requires the independently reviewed clean-base tree import and restored history/
object audit; original repositories and historical evidence remain preserved.

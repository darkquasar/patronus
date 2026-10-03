# Pi-native deployment validation

Pi is a deployment target for authored skills, prompts, native Markdown roles,
instructions, settings and MCP configuration. Native extension recipes delegate
package mutation to Pi; Patronus records selected identity, scope, provenance and
pending outcomes. This validation covers that deployment boundary.

```text
  +----------+  authored files =>  +-----------------------+
  | Patronus | =================> | Pi global/project root |
  +----------+                    +-----------------------+
        |                                    |
        | exact npm ref =>                   | startup/reload =>
        v                                    v
  +----------+  package lifecycle =>  +---------------------+
  | Pi / npm | ====================> | Extensions and roles |
  +----------+                       +---------------------+
```

## Observed environment

The 2026-10-04 validation used a disposable Linux ARM64 Docker Sandbox with two
CPUs and 4 GiB of memory. Its runtime derives from Docker's
[Pi kit](https://hub.docker.com/r/sbx/pi-kit), using the pinned base image
`docker.io/sbx/pi-image@sha256:a2c3ac953dcef579f89a1a4c20f993e3b9325d58633b7f672694f64b0a067a67`.
The temporary kit retained npm proxy setup and omitted provider credential
bindings. The host checkout was mounted read-only; deployment used disposable
global and project directories. Patronus was cross-built from this source tree.

| Component | Observed version |
|---|---|
| Docker Sandboxes CLI | 0.45.1 |
| Pi | 0.87.1 |
| Node | 22.22.1 |
| pi-subagents | 0.72.1 |
| pi-web-access | 0.35.0 |

Exact direct npm references do not lock transitive dependencies. The resource
check records package metadata hashes and authored role hashes for each run;
these are observations, not whole-package integrity claims.

## Deployment and loading checks

| Check | Result |
|---|---|
| Global full-profile preview and deployment | Passed; two native packages and authored resources placed |
| Catalog-version skill update | Passed; changed metadata delivered without replacing unchanged sidecars |
| Global same-version profile update | Passed; installed resources retained |
| Local full-profile deployment | Passed after removing conflicting global authored identities |
| Local same-version profile update | Passed with global and project copies of the same pinned npm sources |
| Real installed extension loading | Passed; two extensions, no load errors |
| Native role discovery | Passed; eight authored roles, with their skills and requested tools available |
| Workflow skill loading | Passed; all three delivered workflows discovered |
| Skill diagnostics | None after shortening the editorial description to Pi's supported size |
| Real Pi CLI cold startup and RPC readiness | Passed globally and locally; 59 commands discovered, including all three workflow skills |
| Local CLI trust | Unapproved local resources stayed inactive; run-scoped `--approve` enabled the disposable project |
| Global/project authored identity collision | Refused before local deployment |
| Ordinary removal of profile-tracked resources | Refused with ownership retained |
| Explicit authored removal | Passed; unrelated user file retained |
| Local native-package removal | Passed; global packages and exact global settings bytes retained |
| Previous binary against a pending removal checkpoint | Refused the new journal schema; files and journal retained |

The local-update check exercises a specific discovery regression: Pi selects the
project copy of an identical exact npm source, so its inactive global copy must
not produce duplicate builtin-role or skill conflicts. Distinct refs and competing
native definitions retain static conflict checks.

## Repeatable installed-runtime checks

These scripts are opt-in. They execute the installed extension factories and
start the actual Pi CLI, so run them in the approved disposable sandbox after
deploying `core-profile-pi`. They never prompt a model, start child agents or invoke
web tools. Pass the installed Pi package directory, global agent directory,
project and evidence output explicitly:

```sh
node scripts/qualification/pi-native-resources.mjs \
  /ABSOLUTE/PI_PACKAGE_DIR /ABSOLUTE/PI_AGENT_DIR \
  /ABSOLUTE/PROJECT /ABSOLUTE/EVIDENCE/resources.json
python3 scripts/qualification/pi-native-rpc.py \
  --cwd /ABSOLUTE/PROJECT --output /ABSOLUTE/EVIDENCE/rpc.json
```

For project-scoped roles, append `/ABSOLUTE/PROJECT/.pi/agents` to the Node command.
For the local CLI check, pass `--approve-project` only for the disposable project
whose files are authorized for this run. This supplies Pi's `--approve` for that
process; Patronus's `--allow-pi-project-config` authorizes installation and does
not independently grant Pi runtime trust. A SDK-only subagents warning about
locating the running Pi binary is retained in the log; the separate actual CLI
check exercises cold startup through Pi's own entrypoint.

The resource script binds the currently selected role source paths, verifies
required skills and registered tools, and writes inventory data instead
of copying credentials or complete context into evidence. The RPC script writes
read-only responses and stderr, then terminates and reaps its process.

## Repository checks and remaining boundary

The formatting, vet, lint and full race gates, catalog integrity, version bumps,
skill placeholders, gate intent, publication shell suites and Pi Python checks
pass. Directory lifecycle/recovery checks also pass. Cross-builds cover
Darwin ARM64, Linux AMD64, Linux ARM64 and Windows AMD64; compilation does not
establish runtime support on each platform.

On macOS, point `TMPDIR` at a canonical directory such as `/private/tmp` when
running the Go suite. Pi path checks intentionally reject symlinked ancestry;
the usual `/var` temporary-path alias otherwise makes fixtures unsafe by that
contract. Context discovery tests accommodate case-insensitive filesystems.

No model inference, child workflow execution, provider authentication, live web
request, optional MCP connection, public catalog publication or live-installation
migration is claimed by this deployment smoke. Authenticated model execution and live web calls are deliberately outside the
accepted validation scope. Historical static-delivery observations remain in the
[older support matrix](pi-qualification/support-matrix.md); they do not replace
these Pi-native lifecycle checks.

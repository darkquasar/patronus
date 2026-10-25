# Source and adaptation inventory

Patronus source revision: `e32b6a87a56bb43e45ad318b4b2c5349b8b57379`.
Inventory captured before adaptation on 2026-10-02. Source labels below are
provenance, not executable paths or active invocation aliases. All inspected
source members were regular files, with no symlinks; hidden members are included.

Original guidance from the approved I-01 through I-04 contract, aligned with
D-04 and the static-admission amendment. Version 2 adds catalogue declarations
for native package delivery and field-level role settings, but copies no external
runtime code into this artifact. The repository license is included as LICENSE.

## Distributed inventory and links

`patronus.yaml` declares the entry and every sidecar; NOTICE and LICENSE travel
with the artifact. Relative Markdown links resolve within declared content.
Upstream URLs are attribution/history only; they are not deployment instructions.
Artifact versions are independent of server versions. Runtime names/schemas and
capability must be discovered and qualified separately.

## Adapter 3.0.0 source/schema inspection

Read-only installed-source inspection; no imports or third-party execution.
The package version is 3.0.0. Hashes identify observed bytes, not a separately
verified registry integrity or runtime qualification. The candidate's effect
still needs deployment observation with effective environment/imports.

| Member | SHA-256 |
|---|---|
| `package.json` | `31f549a3fc1228c2eae04577e6defe4e06e6cff7af94392fc966f9f28a583359` |
| `types.ts` | `de9b532cd89df8b42bcffcc81c827823b2cfcd577ec57797a861c8bb82d183b8` |
| `config.ts` | `ea6bce53281e473deb7f329063cd0423a92c85de65018ef10248fdf37e853f59` |
| `init.ts` | `e65fbf50aaa235c1164a8a23c28f65eed5c4fdecec6ade729535c6c6cb4794b0` |
| `index.ts` | `6b076ee6e11dc298934a1cc4ce486495128bfbca12e75037f1f187effaf720e9` |
| `jev-client.ts` | `b45c189839b4d4db3142fc98a464e880e537d9ffb0f758458943f496597253d0` |
| `server-manager.ts` | `e88867bc2eac2534d331b6b6039b76fc7768b03e6f408b86eff41073a51b1cd0` |
| `utils.ts` | `d0f9719616fe7cf5aca43c44c0c343f732cd0c7940e74d18f2a96d847bc8506d` |
| `direct-tools.ts` | `dcd2777f41bac324dc335b8335cc0cdc2cd2fe108c6f94a9eda43be51c03bfca` |
| `proxy-modes.ts` | `23a0f05090737937d25946f206dbddd7d20faff54c45859195fbb1102ccd4899` |

Source seams inspected:
- `types.ts:598-674`: McpSettings types for all delivered settings;
  `types.ts:437-480`: ServerEntry URL/header/request-command and tool filters.
- `config.ts:getConfigSources`, `loadMcpConfigWithSources`, `parseSettings`:
  file/import/default precedence, exclusive mode, project policy restrictions.
- `index.ts`: allowInstall=false rejects install action; scriptMode=false gates
  script tool registration. Neither setting removes ordinary shell authority.
- `init.ts:156-168`: sampling=false and elicitation=false suppress handlers;
  samplingAutoApprove is not an alternative sampling grant here.
- `jev-client.ts:resolveSemanticJevSettings`: explicit jev=false wins over
  credential-triggered semantic search; absence of a key alone is not the policy.
- `direct-tools.ts`, `proxy-modes.ts`: autoAuth must be true for automatic auth.
- `server-manager.ts:connectHttpClient` and `utils.ts:resolveCommandSecret`: HTTP
  headers/bearers can execute leading-! commands. No pretend global disable key
  exists in this template: inspect effective definitions and exclude them.

Patronus source seams: `internal/scan/pi_discovery.go:DiscoverPiMCP` owns the
qualified static source inventory/conflicts; `internal/adapter/builtin/pi.yaml`
owns destinations; `internal/recipe/recipe.go:resolveTools` owns default versus
explicit routing. These references explain existing mechanisms, not new helpers.

## Reviewed overlay composition (catalog data, not an application test)

The `code-intel-pi` profile extends `core-profile-pi` without changing the core
profile or its role Markdown. It adds the pointer, three skills, exact
`pi-mcp-adapter`, two shared HTTP recipes and sixteen field-level setting artifacts
covering tools and skills for all eight inherited roles. The target-agnostic
`code-intel-pi-runtime` companion selects the Serena and Graphify uv recipes.
This split preserves Patronus's fail-closed static Pi admission, which rejects
package-manager EXEC intent in a Pi-targeted selection. The shared recipes require
the adapter sibling when selected directly. Profile YAML remains the authoritative
membership, not a name/count oracle. Core has no reverse dependency on either
optional profile.

The runtime recipes delegate installation to Pi/npm or uv. They do not copy
third-party package bytes into this artifact and do not start a service. Actual
package closure, license obligations, platform compatibility, service lifecycle
and qualification remain separate evidence gates.

## Immutable source command contract

On 2026-10-05, read-only raw-source inspection checked the published service argv
against immutable upstream revisions. The machine-readable record is
`references/runtime-command-contract.json`; its Git blob SHA-1 values come from
the repositories' commit trees and bind each inspected file inside the selected
revision.

- Serena commit `7a2968335f2198b966864de1ce3655c8e485a653`,
  `src/serena/cli.py` blob `63d49c151cc0c67f216d401dc730bf14af976cdc`:
  the CLI declares `--version`, `start-mcp-server`, transport value
  `streamable-http`, `--host`, `--port`, `--project`, `--context`, repeated
  `--mode`, and explicit boolean-valued dashboard options. Its `planning.yml`
  blob `a24d0dfb150bc3c10b0c16be8c15014297cc6687` describes read-only planning
  and excludes the listed edit and shell tools.
- Graphify commit `4fe11092ccbe9f543608f140c790f68d5d83cae4`,
  `pyproject.toml` blob `619f00af51506898f74738e95dc99a532ad2adf5`,
  `graphify/__main__.py` blob `924ae986d3a8e2b7c38154a5f4d32dff07d21a2f`
  and `graphify/serve.py` blob `1a44c781d5db84703f4965414da3c162b11efc10`:
  the package version is 0.9.31, declares both `graphify` and `graphify-mcp`
  console scripts, supports `graphify --version`, and the MCP parser accepts a
  positional graph path, `--transport http`, `--host`, `--port`, `--path` and
  `--stateless`.

This is source-contract evidence, not an installed-executable or service-start
observation. Deployment still records both executable probes and live help before
using the commands. The candidate dossier records digest
`b0d47f823f924e7f89acfee390b9f18dc3410917617c5f6f2731bd2642abf16f` but does
not establish a fresh selected-wheel byte comparison; full package closure and
archive provenance remain incomplete.

Repository LICENSE copied unchanged: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.

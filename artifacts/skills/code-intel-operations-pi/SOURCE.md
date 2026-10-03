# Source and adaptation inventory

Patronus source revision: `e32b6a87a56bb43e45ad318b4b2c5349b8b57379`.
Inventory captured before adaptation on 2026-10-02. Source labels below are
provenance, not executable paths or active invocation aliases. All inspected
source members were regular files, with no symlinks; hidden members are included.

Original static guidance from the approved I-01 through I-04 contract, aligned
with D-04 and the static-admission amendment. No executable helper or external
code is copied. The repository license is included as LICENSE.

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

The profile extends core-profile-pi without modifying it and adds the pointer,
pattern-mcp-pi, graphify-pi, this operations skill, and the two shared HTTP recipes.
The pointer requires all three skills. With the amended core's required web
content/dependencies, source review yields 44 inherited items plus 6 additions,
50 unique items; the older 40/46 figures are superseded. Profile YAML remains the
authoritative membership, not an executable name/count oracle. Core has no
reverse dependency on the optional overlay. Legacy Serena/Graphify recipes are
unchanged. No runtime redistribution or server/license qualification is implied.

Repository LICENSE copied unchanged: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.

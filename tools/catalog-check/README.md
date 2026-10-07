# Catalog packaging checks

This standalone Go 1.25 module checks source/build packaging, not installation or
harness behavior. Its only dependency is the repository's already pinned
`gopkg.in/yaml.v3 v3.0.1`. It does not import Patronus internals or participate in a
`go.work` workspace. Root Go tests do not discover this module.

```text
+---------------------+  read  +---------------+  read  +--------------------+
| source + build index| =====> | catalog-check | <===== | public CLI locks   |
+---------------------+        +---------------+        +--------------------+
```

Run from this directory after the public CLI has built the same source tree:

```sh
go run -mod=readonly . --source ../.. --index ../../out/catalog/index.json
go run -mod=readonly . --source ../.. --index ../../out/catalog/index.json --profile-cases
go run -mod=readonly . --source ../.. --index ../../out/catalog/index.json --closures ../../out/closures
```

The project caller owns the resource lock, offline environment and memory
admission. Local module verification is `go test -mod=readonly -p 2 ./...` and
`go vet -mod=readonly -p 2 ./...`, with `gofmt -l .` empty. Aggregate application
and catalog gates are separate. Never fetch dependencies or execute artifact
payloads to make this check pass.

## Packaging boundary

The checker discovers `artifacts/**/patronus.yaml` and direct YAML manifests in
`recipes/`, `profiles/` and `plugins/`. It diagnoses empty source/index data,
duplicate identities and unequal source/index sets. It accepts public index
schema versions 1 and 2. It checks safe identity components, family/API metadata,
entry and declared sidecars, attribution NOTICE, declared hook script membership,
regular paths without symlinks, skill directory/frontmatter name identity,
literal `requires`, profile layer and `extends` reference existence, and declared
target existence in public adapter metadata. An `item@target` reference checks
literal base-item and target existence only, without selecting it.

Source/index and archive/index manifests must agree. Comparison accounts only
for public serialization defaults: omitted empty values, scalar profile layers
becoming lists, the source-only legacy recipe `kind: Recipe` header, and omitted
hook intent becoming `nudge`. Declared directory sidecars may use one trailing slash;
unsafe path components remain rejected. Payload bytes and sets
must agree with declared source entries/sidecars. The checker verifies the
`index.json.sha256` sidecar and tarball SHA-256 values. It derives local tarballs
from `<index-directory>/<name>/<version>/<name>-<version>.tar.gz`, never from a
URL. URLs, upstream binaries and package payloads are never fetched. Tar members
must be unique, safe relative regular files, not links or special files.

Plugins are source-only because the public build does not emit them. This is not
plugin bundle coverage. Package bundles remain owned by existing publication
checks. Version bump, placeholder, recipe ontology, cycle and target-selection
semantics remain in public Patronus commands. The checker neither resolves
profiles nor scans prose for forbidden phrases or inferred runtime behavior.
Literal distributed Markdown links are structural file claims, checked against bundle
members with supported source-root placeholders. Literal Markdown examples and external
links are not file claims. Native Pi agent `skills:` selections must be declared in
`requires` and resolve to Pi-compatible skills. Native role tools/skills overlays
preserve the complete ordered base list, and extra skills must be declared
Pi-compatible dependencies; no harness invocation is performed.

`docs/compatibility/distributed-reference-exceptions.yaml` may retain reviewed legacy
link debt with `schema_version: 1` and rows containing `source`,
`manifest_content_sha256`, `references`, `reason` and optional `upstream_only`.
The digest covers raw neighboring manifest bytes followed by the exact source body.
Changed bytes invalidate the exception. Non-upstream debt still requires regular files
in source, without claiming those files ship. Tool code contains no artifact allowlist.

## Explicit profile cases

`--profile-cases` emits only a JSON array of `{profile,target}` objects on stdout.
It reads `docs/compatibility/profile-targets.yaml` beneath `--source`:

```yaml
schema_version: 1
profiles:
  example-profile:
    targets: [example-adapter]
```

Every actual source profile must have exactly one metadata entry with a nonempty,
duplicate-free target list. Targets must exist in public `adapters/*.yaml` tool
metadata, or be the public `all` selector. Unknown profiles, missing metadata,
empty selections and undefined targets fail. The metadata describes admitted
static lock cases, not native support. Lane compatibility is documented by the
metadata author and verified by the public CLI, not inferred from names or from
a second resolver. CI invokes isolated public `lock --profile ... --target ...
--local-registry` cases and preserves their results.

## Ledgers and supplied closures

YAML documents in `docs/compatibility/` with `profile` or `entries` fields are
profile ledgers. Their schema requires `schema_version: 1`, explicit
`baseline_profile` and `baseline_target`, and the existing `core_item`,
`disposition`, `codex_items`, `outcome`, `reason`, `evidence` row contract.
Baseline coverage uses direct declared layer members, retaining qualified literal
keys, not transitive dependencies. Selected direct members need mapped endpoint
or companion evidence. Generic endpoint existence and status vocabulary checks
are structural evidence only. Outcome labels are `Equivalent`, `Partial`, `N/A`
and `EnvironmentBlocked`; evidence labels are `static-confirmed`, `runtime-pass`,
`runtime-pending`, `environment-blocked`, `historical` and `not-run`. Labels are
not verified native receipts.

`--closures DIR` requires every enumerated `<profile>--<target>.lock` file. It
accepts public JSON locks (also YAML), checks profile/target identities, nonempty
entries, duplicates, item versions/families, and coverage of ledger endpoints and
companions by the supplied selected-profile locks. Public legacy lock versions
1/2 may omit target; versions 3/4 must declare it. The baseline case must also be
supplied. The tool never generates closures. Without `--closures`, successful
packaging output explicitly says closure coverage `NOT CHECKED`.

Inputs must be stable while checking. Validation bounds are 64 MiB per local file
and archive member, 256 MiB total expanded archive bytes and 10,000 archive
members. The checker reads only; it never extracts archives or runs scripts,
hooks, SDKs, native CLIs, resolver engines or network requests. Manual release
review retains content semantics, editorial policy and deferred native/behavior
qualification.

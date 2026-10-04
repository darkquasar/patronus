# Static directory packages

A directory recipe installs a verified tar.gz package once per host, at
`~/.patronus/packages/<recipe>/`. It needs no installed agent and ignores runtime
selection for delivery: `--target all` still installs one tree. Local scope is
unsupported. This lifecycle applies only to static directory recipes. For a
`manager: pi` recipe, Pi/npm own package installation, update, removal and package
files; the directory receipts described below do not apply.

```sh
patronus install <recipe>                 # inspect the plan
patronus install <recipe> --deploy        # verify and install
patronus update <recipe>                  # inspect the selected catalog version
patronus update <recipe> --deploy         # replace an owned, unchanged tree
```

Plans show the recipe and package versions, platform, URL, archive checksum,
destination, known conflicts and pending recovery. Planning a directory package
performs no package download, write or execution. It cannot show a complete new
member diff before downloading the archive. Existing artifact source
materialization keeps its existing cache behavior.

## Ownership and consent

Receipts in `~/.patronus/package-state/<recipe>.json` record package ownership.
The row in `~/.patronus/state.json` is a discovery reference. Updates use receipts
for installed versions and can find a package whose reference was lost. The next
locked package deployment repairs the reference. A missing or invalid receipt
requires reconciliation; a generic state row cannot replace it.

Directory updates preserve edited owned files by default. To explicitly replace
those edits, use either command:

```sh
patronus install <recipe> --deploy --force
patronus update <recipe> --deploy --force
```

A forced update checks and repairs drift even when the recipe version is
unchanged. Ordinary updates can report an unchanged version without repairing
it. `--yes` never bypasses package conflicts or reports them as a successful skip.
Legacy file recipes retain their existing update overwrite behavior; directory
recipes require the user's explicit `--force`.

Unknown files or directories block replacement even with force. This includes
caches and `.DS_Store`. Move them outside the package root before retrying.
Patronus also refuses to adopt an existing root, even when its files match the
package. Keep personal work and runtime output outside the owned tree.

Profile installation restores directory delivery pins and recipe versions from
an applicable `patronus.lock`. An absent lock follows the catalog; a malformed,
future or incomplete directory pin fails before deployment. Direct installs and
explicit updates follow the selected catalog, independently of a nearby profile
lock. Local profile installs honor directory pins too; local legacy artifact
sources retain their existing behavior.

## Deployment and recovery

```text
plan selected items -> preflight every directory -> acquire package lock
  -> recover selected transactions -> inspect again -> replace packages
  -> repair discovery references -> apply remaining legacy work
```

The host-wide package lock fails promptly when another mutation holds it.
Selected packages recover pending transactions before replacement; unrelated
pending packages do not block them. Dry runs report pending recovery without
changing it. An ownership conflict during recovery preserves evidence and lists
the paths to reconcile before retrying.

Each recipe commits independently. A later failure keeps earlier committed
packages. Legacy file writes retain their existing partial-success behavior.
Output counts package commits separately from individual file writes.

A failure after a durable package commit, including reference repair or backup
cleanup, is reported as an error while keeping the committed installation.
Retry installation after resolving the reported problem. Receipts remain the
ownership authority if a concurrent legacy state write loses a discovery row;
generic state writes are not globally serialized by the package lock.

## Selective removal and recovery records

`patronus remove <recipe> --deploy` removes unchanged owned files, retaining
edited owned files and all unknown content. `--force` also selects edited owned
files, but authorizes only their observed bytes and mode, never unknown children
or subsequent edits. Empty owned directories are pruned; roots are not recursively
deleted.

New removals write one immutable `removal.json` in the recipe's transaction
directory. It records the previous receipt, initially missing paths, and the
ordered selected paths with their observed preimages. A schema-2
`transaction.json` binds a random transaction ID, recipe, canonical root, and the
SHA-256 of the canonical manifest JSON to a completed-prefix cursor and phase.
The marker is published durably before any unlink. Older binaries reject this
marker at the existing discovery location; retry with a supporting binary.
Pending schema-1 journals retain their original recovery path.

Every selected file still has a durable unlink intent, preimage/type/ancestry
checks, unlink and parent-directory sync, then a durable completed-prefix
checkpoint before the next file. Completed paths are excluded from replay even
if recreated with identical bytes. A crash between unlink and durable completion
retains the existing ambiguity for an identically recreated file; this is not a
power-cut or hostile-concurrent-writer guarantee. Checkpoints reuse the atomic
temp-write, file-sync, rename and parent-sync protocol, without batching.

The receipt remains the last committed snapshot during removal, not a per-file
ledger. `ReadTransaction` and inspection expose pending work and reconstruct its
inventory view; `Load`/`List` alone do not prove a package is idle. Mutation must
recover first. The remaining-files receipt is published once at finalization
(or deleted when empty), then the committed journal remains until discovery is
repaired and synced. Acknowledgement durably removes the marker before cleaning
its manifest; a manifest without a marker authorizes nothing. All persistence
errors stop further destructive work and require reopening validated evidence.

The [removal measurement harness](../scripts/qualification/removal-performance/README.md)
records synthetic overlay timings, recovery and binary-size comparisons. These
are removal-service measurements, not installed web-payload or power-loss tests.

## Using the Pi kit

When the catalog offers `pi-sandbox`, installation places its static kit and
README under `~/.patronus/packages/pi-sandbox/`. Running this kit requires sbx
and separate provider authentication. Installation succeeds without sbx and
prints `Package installed; install sbx before using it.` when a PATH lookup
cannot find it.

Read the installed README, then launch from your workspace yourself:

```sh
sbx run "$HOME/.patronus/packages/pi-sandbox/" .
```

Patronus performs no VM, session, tab, authentication or credential operations
and installs no runtime prerequisites. Updating the host kit makes no promise
about an existing VM. Herdr examples remain experimental. Package installation
alone does not establish runtime acceptance.

## Authoring a package

`package.yaml` is a Patronus build descriptor with its own `schemaVersion: 1`,
independent of the recipe manifest API and the sbx kit schema. It is not a
catalog item. For editor validation and completion, add this modeline in a
`packages/<name>/package.yaml` file:

```yaml
# yaml-language-server: $schema=../../schemas/package-v1.schema.json
```

This explicit association avoids unrelated PNPM/Mason schemas automatically
selected for the generic filename. The build command remains authoritative
for file existence, case collisions, identity and byte limits.

Declare each static file and its executable flag in
`packages/<name>/package.yaml`, along with schema version 1, package name,
version and supported platforms. Only declared files enter the archive;
exclude tests, credentials and runtime output. The Pi example declares
`spec.yaml`, `README.md`, `LICENSE` and `NOTICE`.

Build the selected package before preparing its recipe pins:

```sh
go run ./cmd/patronus build --package pi-sandbox --out /tmp/patronus-pi-package
```

The command writes the deterministic tarball, `.sha256` and
`.provenance.json` sidecars under `packages/pi-sandbox/1.0.0/` in the output
directory. Use the emitted digest and package object path at the configured
registry base URL in a `patronus/v3` recipe with `deliver.unpack: directory`.
Set the package identity and supported tar.gz platform asset explicitly;
omit recipe wiring and executable entrypoints.

Then verify the committed recipe pins against a full local build:

```sh
go test ./packages/pi-sandbox/tests ./cmd/patronus -run 'PiPackage|Directory' -count=1
go run ./cmd/patronus build --out /tmp/patronus-pi-registry
go run ./cmd/patronus check-versions
```

Full builds reject mismatched identity, URL or checksum pins without rewriting
the recipe. Package versions and recipe versions are independent. Change the
package version when payload bytes change, rebuild and update its recipe pins.
These authoring commands create local output only; publication is a separate
operation through the existing catalog workflow.

## Release and catalog activation

The catalog workflow builds and tests the registry without production credentials.
A separate job receives the built artifact and the protected `production`
environment. It publishes in this order:

```text
build + checks -> activation guard -> packages + sidecars -> legacy artifacts
                                                            |
                                         current-main check -> index
```

Directory package tarballs, `.sha256` files and `.provenance.json` files use
conditional creation at immutable keys. An existing tarball must have matching
SHA-256 metadata. Checksum sidecars must match the build and their downloaded
bytes must match their metadata. For identical package bytes, publication keeps
the first valid provenance, even if a later build has a different repository
commit or CI run. Missing metadata, conflicting bytes, failed verification and
upload errors stop publication before the index changes. A conditional PUT race
verifies the winner; it never retries with an unconditional overwrite.

Publication runs share one concurrency group and do not cancel an active run.
The protected job checks whether its commit is current main after it starts and
again immediately before writing the index. A stale tag or manual run can publish
immutable objects but skips the index with a notice. The workflow pins AWS CLI
2.31.0 for conditional PUT support. Legacy artifact behavior, including skipping
an existing artifact without checksum metadata, remains unchanged.

To activate directory delivery:

1. Release a Patronus binary that supports directory packages and index schema 2.
2. Have the release operator set the repository Actions variable
   `DIRECTORY_PACKAGES_ENABLED=true`.
3. Run `publish-catalog` from current main and check its publication results.

Activation is an explicit operator action. Implementation and tests do not enable
it. Before activation, a schema-2 build skips the entire catalog publication,
including unrelated catalog changes once a directory recipe has merged. The
existing catalog remains available until activation; there is no parallel legacy
catalog. The activation guard runs before any production upload, even when the
build contains no package objects.

For local publication verification, run
`bash scripts/tests/publish-packages-test.sh`. Its AWS and Git commands are fakes;
it uses no cloud credentials and makes no network requests.

## Local release verification

Run the repository quality gates, the lifecycle tests and both publication shell
suites before preparing a release:

```sh
gofmt -l .
go vet ./...
golangci-lint run
go test -race ./...
go test -race ./cmd/patronus -run 'DirectoryLifecycle|DirectoryMultiRecipe' -count=1
bash scripts/tests/publish-packages-test.sh
bash scripts/tests/check-catalog-publication-test.sh
go run ./cmd/patronus build --out /tmp/patronus-package-release
go run ./cmd/patronus check-versions
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /tmp/patronus-darwin-arm64 ./cmd/patronus
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/patronus-linux-amd64 ./cmd/patronus
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /tmp/patronus-windows-amd64.exe ./cmd/patronus
```

Formatting must produce no output. The lifecycle tests use a temporary home,
a local catalog and a TLS test server. Subprocess crashes leave real transaction
journals for restart recovery; a fake command runner checks that installation,
update and removal make no runtime calls. The Windows build checks compilation
of the unsupported-platform lock fallback; directory mutation is supported on
Darwin and Linux.

Record the tested commit with the release evidence. The implementation currently
has an unreleased supporting binary; this checklist does not establish a released
version. Static packaging checks do not validate VM launch, authentication,
model access or optional Herdr integration. The release operator must complete
the binary release and catalog activation steps above separately.

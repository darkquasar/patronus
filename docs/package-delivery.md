# Static directory packages

A directory recipe installs a verified tar.gz package once per host, at
`~/.patronus/packages/<recipe>/`. It needs no installed agent and ignores runtime
selection for delivery: `--target all` still installs one tree. Local scope is
unsupported.

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

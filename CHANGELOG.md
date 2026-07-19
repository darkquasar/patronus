# Changelog

All notable changes to Patronus are recorded here. This file is written for the
person upgrading: it leads with what will behave differently on their machine.

## Unreleased

### Security

- **Archive-delivered binaries are now verified on every run, not just the first.**

  Patronus previously reported `SKIP` ("verified, up to date") for an
  archive-delivered binary whenever a file was merely *present* at its destination —
  **without hashing it**. Only the very first install was verified; every re-run
  trusted whatever happened to be on disk.

  That mattered because binaries under `~/.patronus/bin/` are wired into auto-firing
  hooks: `gitleaks-guard` runs on **every commit**. A file written there by anything
  else — a malicious postinstall, a poisoned container layer, a stray `cp` — was
  therefore executed on every commit *and reported by Patronus as verified*.

  Patronus now compares the file against the digest it recorded when it placed the
  binary. (It was already recording it; nothing read it back.) What changes for you:

  - **A binary you replaced by hand will be re-fetched and re-verified** on the next
    `install --deploy`. If you intended that replacement, it will not survive.
  - **A binary Patronus has never verified** — hand-placed, or placed before this
    release — has no recorded digest, so it is **re-fetched**. *"We have never
    verified this"* is not the same as *"this is fine."*
  - **Raw (`source: url`) deliveries are unchanged.** Their pin already *is* the
    placed file's digest, so they were always hashed on every run.

### Changed

- The test suite no longer depends on the *contents* of the real catalog. Tests that
  assert Patronus's **behavior** now run against a fixture catalog whose binary pins
  are the sha256 of bytes the tests themselves invent, so bumping a pin in
  `recipes/` no longer breaks tests that were never about that binary. Tests that
  assert the catalog's **contents** (which skills the `core` profile ships) keep the
  real names — that is a product guarantee, and a fixture cannot express it.

- No third-party bytes are vendored into the repository as test inputs any more
  (`cmd/patronus/testdata/tk`, 47KB of upstream bash, is gone), and no test can reach
  the network: the fetcher seams fail closed in tests and panic with the URL if one
  tries.

# Changelog

All notable changes to Patronus are recorded here. This file is written for the
person upgrading: it leads with what will behave differently on their machine.

## Unreleased

### Breaking

- **`writing-style` is now `writing-editorial` (v2.0.0), and the old skill is left on
  your machine.** The guide is split from one 38KB file into a short router plus four
  tier files applied in sequence: `tier-0.md` (machine phrasings, ungated), `tier-1.md`
  (mechanics, plus a new rule catching the mirrored-swap negation everywhere),
  `tier-2.md` (machine tells, word tiers, and the PRESERVE list that protects your
  voice), and `tier-3.md` (meaning and movement, the only pass allowed to restructure).

  **You need to remove the old skill by hand.** A rename changes the artifact's
  identity, so Patronus treats the new name as an install and never learns that the old
  one is now an orphan: `install` acts only on the names you ask for, and state rows are
  keyed by artifact name, so the `writing-style/` directory and its state row both
  survive. Leaving it there means two loadable writing skills, and the stale one carries
  the weaker rule set.

  ```sh
  patronus scan                            # shows writing-style as ORPHANED-STATE
  patronus remove writing-style            # dry run, inspect what it will undo
  patronus remove writing-style --deploy   # actually remove it
  ```

  Omit `--target` to cover every tool you installed it for; both scope state files are
  consulted by default. If you edited the installed skill, the drift check skips it and
  you need `--force`.

  The instruction pointer **keeps its name**, `writing-style-pointer`, so your global
  CLAUDE.md wiring at `~/.claude/patronus/instructions/` is untouched. Only its
  description and its dependency edge changed.

### Added

- **`writing-like-me`, a voice pipeline, in the new opt-in `writing` profile.** It runs
  the `writing-editorial` tiers over a draft, then has a Claude subagent and `codex`
  each apply your own writing corpus independently, then merges the two and shows you
  verbatim where they disagreed.

  It ships with **empty** exemplar files on purpose, so your corpus never enters the
  public catalogue, and it does nothing useful until you supply one:

  ```sh
  mkdir -p ~/.claude/patronus/voice
  # paste whole pieces, not excerpts
  $EDITOR ~/.claude/patronus/voice/short-form.md
  $EDITOR ~/.claude/patronus/voice/long-form.md
  patronus install --profile writing --target claude --deploy
  ```

  The corpus lives outside every artifact payload (override the location with
  `$PATRONUS_VOICE_DIR`), so no upgrade can prompt about it, skip it, or overwrite it.
  With no corpus the pipeline runs the editorial tiers only and says so. `codex` is
  optional and is reached over MCP (`codex mcp-server`): if it is not registered, errors,
  or times out, the run continues on the Claude draft and names what happened.

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

# Changelog

All notable changes to Patronus are recorded here. This file is written for the
person upgrading: it leads with what will behave differently on their machine.

## Unreleased

### writing-like-me 2.0.0 (breaking)

The voice stage is rebuilt around conception rather than diction. A voice profile of named moves,
each carrying corpus evidence, replaces distributional targets as the success criterion. A spine
derived once per run owns the piece's governing metaphor, opening scene, register and running
order, and may reorder, merge, split or cut sections under a claims manifest that guarantees every
claim survives. Sections are voiced in parallel against both, each audited for liveness with one
rework and then accept-and-flag, and stitched with narrative continuity rather than logical
signposts.

Codex moves from co-author to critic: one advisory read over the finished text, with no veto, and
the run reports what it took and what it declined.

Breaking: the two-author merge is gone, and with it the "both drafts, then merge" output. Reporting
now leads with what the piece failed at, and rhythm numbers are labelled as guard rails rather than
presented as success.

Requires `writing-editorial` 2.2.0 or later, whose section files and span-anchored edit records
this stage consumes. Existing corpora at `~/.claude/patronus/voice/` keep working untouched;
`voice-profile.md` is new, so a first run extracts one.

### writing-editorial 2.2.0

Long drafts can now be cut into sections and edited in parallel. Tiers 0 through 2 fan out one
subagent per section; the merge pass reconciles the contrast ledger, applies the four tier-2 rules
whose evidence is the whole piece, and normalises terminology across sections; tier-3 runs once over
the merged document.

Supply `trail-root` and the run emits section files, span-anchored edit records
(`editorial-edit-record/v1`) and section lineage (`section-lineage/v1`). With no `trail-root`, the
skill behaves exactly as before and writes nothing.

Per-section fan-out is dispatch option 3, and it is the default only when `trail-root` is supplied,
so a direct caller's cost is unchanged.

## v2.3.0

The binary release carries the `remove` fixes below. The catalog content in this entry
(the skill renames, `diagram-explain`, `plan-execute`) ships on its own track and reaches
you independently of the binary.

### Fixed

- **Removing two artifacts wired into one config file no longer resurrects one of them.**
  Each removal was computed from the same original bytes and then written in sequence, so
  the second write put back what the first had taken out: the artifact you asked to remove
  survived the command, and its record was retired anyway, so nothing reported the
  leftover. Removals that land on one file are now folded into a single write.

  Where two recorded edits genuinely conflict — the same key, a key nested under another,
  or the same list entry — neither is guessed at. Each is reported and left in place, and
  `--force` does not override it: forcing means accepting the loss of your own edit to a
  file, not another artifact's wiring. Remove them one at a time instead.

  Records written before per-key removal existed carry only a whole-file snapshot, which
  reverts everything in the file. Those are now restored only when nothing else is wired
  into the same path, counting artifacts you did not ask to remove and both scopes' state
  files. Otherwise the file is left alone and named on stderr.

- **A file that was already gone no longer keeps an item marked installed forever.** It
  counted as unfinished work, so the record was never retired: after any partial cleanup,
  re-running `remove` could not converge and `scan` kept reporting the item. An absent
  target now counts as removed, which is what it is. A file you edited since install, or a
  config that will not parse, still holds the record open so `--force` or a repair can
  finish the job.

- **Removing a skill no longer leaves its empty directory behind.** Every file went, but
  the directory stayed, and since a skill's directory is what marks it as present, the
  remnant read as an installed skill that no command could see or clean up. The directory
  is now removed once its tracked files are gone.

  Only ever the artifact's own directory, and only when empty. A directory holding anything
  Patronus did not write is kept, with its contents named on stderr; the removal is never
  recursive, and file-shaped artifacts (agents, commands, output styles) never cause their
  shared parent to be touched.

- **`remove` counts what it did, not how many times it wrote.** Removing three artifacts
  that share one config file reports three, not the single write, and a file that was
  already gone counts as removed rather than skipped.

### Breaking

- **`diagram-explain` was an output style and is now an instruction, so it finally does
  something on Claude.** A placed output style sits inert until you select it in the Claude
  UI, and Patronus cannot make that selection for you, so this artifact has been installed
  and doing nothing on its primary target since it shipped. Its body now appends into
  `CLAUDE.md` as a `patronus:start diagram-explain` section and is live from install.

  Codex and OpenCode are unaffected in where the content lands: both already routed
  instructions and output styles to the same `AGENTS.md` append, so the section keeps its
  name and position. The body itself is six lines shorter, having dropped the output-style
  frontmatter that only Claude ever read.

  **If you installed a previous version on Claude, delete the stale file by hand:**

  ```sh
  rm ~/.claude/output-styles/diagram-explain.md    # or .claude/output-styles/ for a local install
  ```

  `patronus scan` will not point you at it. The artifact kept its name, so the upgrade
  overwrites its state row in place with the new `CLAUDE.md` path rather than marking the
  old one an orphan, and nothing scans for files that state has forgotten. Left there, the
  file is harmless unless you had selected the output style, in which case it keeps
  applying the old copy on top of the new instruction.

- **Two slash commands changed name. `/team-research` is now `/research-team`, and
  `/team-implement` is now `/plan-execute-parallel`.** If either is in your muscle memory,
  your own notes, or a script, it stops resolving after this upgrade. Nothing aliases the
  old form: type the new one.

- **Four lifecycle skills were renamed so the phase leads the name, and the old skills are
  left on your machine.**

  | was | is now |
  |---|---|
  | `brainstorming-spec` | `spec-brainstorming` |
  | `writing-plans` | `plan-writing` |
  | `team-research` | `research-team` |
  | `team-implement` | `plan-execute-parallel` |

  What each skill does is unchanged; only its name, its heading, and the cross-references
  that point at it moved. `spec-brainstorming` now sorts beside `spec-review`,
  `plan-writing` beside `plan-review` and `plan-execute`, and `plan-execute-parallel` reads
  as the other arm of the same fork. Scanning your installed set now shows which skills
  belong to one phase.

  **You need to remove the old skills by hand.** A rename changes the artifact's identity,
  so Patronus treats each new name as an install and never learns the old one is now an
  orphan: `install` acts only on the names you ask for, and state rows are keyed by artifact
  name. Both the old directories and their state rows survive the upgrade. Leaving them
  there means two skills competing for one trigger, and the stale one points at hand-off
  targets that no longer exist.

  ```sh
  patronus scan                            # shows the four old names as ORPHANED-STATE
  patronus remove team-research            # dry run, inspect what it will undo
  patronus remove team-research --deploy
  ```

  Repeat for `team-implement`, `writing-plans`, and `brainstorming-spec`. Omit `--target` and
  the scope flags to cover every tool and both scopes you installed them for: `--global` and
  `--local` each narrow the removal to one scope, and with neither flag both state files are
  consulted. If you edited an installed skill, the drift check skips it and you need
  `--force`. `remove` deletes the files and the state row, and leaves the now-empty skill
  directory behind; delete it yourself if an empty directory bothers you.

- **`core` executes plans through the new `plan-execute` skill.** It no longer installs
  `executing-plans` or `subagent-driven-development`. Both stay in the catalog and
  install on request. `subagent-driven-development`'s body is untouched;
  `executing-plans` has the two-line revert described under Changed below.

  Upstream splits those two skills on whether the host has subagents: `executing-plans`
  is the fallback, `subagent-driven-development` the intended path. All three Patronus
  targets have subagents, so that axis has no audience here, and a fallback for runtimes
  we do not support was dead weight in `core`. `plan-execute` forks on proportionality
  instead: it reads the plan, asks whether independent per-task review earns its cost,
  states which mode it chose and cites the plan sections that drove the choice, then runs
  `solo` (sequential, one context) or `sdd` (fresh implementer subagent per task,
  independent reviewer after each). Both modes end with an independent whole-branch
  review.

  It does not block for your answer, so an unattended run does not stall. To override,
  say so in the invocation: "execute this plan solo", "run it in sdd mode".

  **The two old skills are left on your machine, and `patronus scan` will not tell you.**
  Patronus keys state rows by artifact name, so dropping them from `core` does not remove
  what you already installed. `scan` reports `ORPHANED-STATE` only for items that left the
  catalog, and both of these deliberately stayed in it, so an upgrade in place leaves them
  installed and unreported. You need to remove them by hand, or you will have three
  loadable skills matching "execute this plan":

  ```sh
  patronus remove executing-plans subagent-driven-development            # dry run
  patronus remove executing-plans subagent-driven-development --deploy   # remove
  ```

  Keep them if you want the upstream behaviour. Their descriptions now say plainly that
  they are upstream-compatible alternatives rather than the default.

- **`dispatching-parallel-agents` is unaffected** and still ships in `core`. It fans out
  independent problem domains and has nothing to do with this fork.

### Changed

- **`plan-review` forks on parallelism only.** Its criterion is unchanged (can you draw
  disjoint file-owning boundaries?), and both arms survive; only the destination names
  move, to `plan-execute` and `plan-execute-parallel`. The proportionality question now
  lives inside `plan-execute`, so `plan-review` names where it lives and stops there.
- **`writing-plans` hands off to `plan-execute`**, and no longer tells the executor how
  to run. That decision belongs to the executor now.
- **`executing-plans` reverts two lines** to their upstream hedge. A previous release had
  turned "if your platform supports subagents, prefer that" into an instruction to
  dispatch one per task, which made the fallback describe the thing it was the fallback
  for. Its Patronus `tk close` re-coupling is untouched.

### Known limitation

- **`plan-execute`'s gate ships without automated verification.** It is prose evaluated by
  a model, and Patronus has no prompt-eval harness yet. In its place, nine fixture plans
  and a manual protocol ship in the skill's `fixtures/` directory, run once before
  release: five runs per fixture, 5/5 agreement with the expected mode required, and every
  citation must exist and support the trigger it names. Results are recorded in
  `fixtures/RESULTS.md`. This is a one-time gate, not a regression guard: nothing re-runs
  it when the criteria are edited.

## v2.2.0

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

### Fixed

- **`patronus scan` no longer reports false `ORPHANED-STATE` for anything installed
  away from its default scope.** Reconciliation replanned each recorded item without
  the tool and scope it was installed at, so an artifact declaring
  `defaults.scope: project` but installed `--global` was looked for under the project
  layout, found nothing, and was reported as "recorded, but no longer in the catalog."
  Reconciliation is now keyed by the `{artifact, tool, scope}` triple install records,
  so each install is checked where it actually went. On one real machine this took the
  drift report from 78 false rows to none.

  Two rows on that machine changed from a false `ORPHANED-STATE` to `STALE`. That is
  real drift the old verdict was masking: those items need a re-run of `install`.

- **A config file that several items merge into is no longer reported as
  `USER-EDITED` or `STALE` just for holding more than one contributor.** A composed
  `settings.json` or `.claude.json` never matches any single contributor's whole-file
  checksum, so every contributor to it was misjudged, and edits were attributed to
  whichever row happened to survive. Each contributor is now reconciled against the
  one setting it owns, so a hand-edited MCP server block is reported against the
  recipe that owns it and its siblings stay silent.

- **Two MCP servers targeting one config file no longer clobber each other.** Merges
  landing on the same path were computed from the original bytes and applied
  last-wins, so exactly one server survived an install while state recorded a valid
  checksum for both. They accumulate now, and `remove` strips one server's block
  without disturbing the rest.

- **An install fails loudly if a write did not land.** State hashed the bytes
  Patronus planned to write rather than the bytes on disk, which is how a dropped
  merge earned a real checksum for a write that never happened. Each written file is
  read back and verified; work that already succeeded stays recorded, so re-running is
  safe.

- `patronus update` reported "updated registry cache" even against a local registry,
  which has no cache to write, and never accepted `--local-registry`.

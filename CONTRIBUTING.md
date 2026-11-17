# Contributing to Patronus

House rules for changing this repo. The closest `AGENTS.md` to the code you are
editing still wins for local conventions; this file covers repo-wide procedure.

## The work-graph is local here

This repo uses `tk`, whose shared instruction says to **commit** the `.tickets/`
work-graph. **This project overrides that default: `.tickets/` is gitignored** (see
`.gitignore`) and is treated as local working state, not a committed artifact of the
catalog. Use `tk` exactly as usual — create, `dep`, `start`, `close`, add notes — but
do **not** try to `git add .tickets/`; the ignore rule makes it a no-op. If you need
to hand the graph to another machine or contributor, share it out of band rather than
through git.

## Deployed skills are not tracked

This repo dogfoods its own `core` profile, so `patronus install` writes deploy output
into the tree. That output is **not** committed — it is regenerated from `artifacts/`,
and committing it just invites drift between source and deployment (the exact thing
`patronus scan`'s drift guard exists to catch). Specifically, `.agents/skills/` (the
codex project-scope skill target) and the tracked bits of `.claude/` are deploy output.
Edit the **source** under `artifacts/`, never the deployed copy, and re-run
`patronus install` to redeploy.

## Versioning artifacts

**Every artifact carries a `version:` in its `patronus.yaml` (SemVer). If you
change an artifact's content, you MUST bump that version.**

This is not cosmetic. `patronus update <name>` compares the *installed* version
against the registry's *published* version and re-installs **only when the
published one is newer** (see the README's `update` section). An un-bumped change
is invisible to the catalog — users stay silently pinned to the stale content.
The `version:` field is `omitempty` and is **not** machine-validated, so nothing
but this rule protects you: a missed bump ships nothing.

An "artifact" here is anything under `artifacts/` with a `patronus.yaml` — a
skill, hook, instruction, output-style, agent, or command. Bump the manifest of
the artifact you touched; a change to any file the manifest lists (e.g. a skill's
`SKILL.md`, a bundled script, a `NOTICE`) counts as changing that artifact.

Pick the bump by the nature of the change:

| Bump | `x.y.z` → | When |
|------|-----------|------|
| **patch** | `x.y.Z+1` | Wording/typo fixes, clarifications — **no behavior change**. |
| **minor** | `x.Y+1.0` | New or changed behavior, backward-compatible — a new file path, a new field, an added step, a relaxed default. |
| **major** | `X+1.0.0` | A breaking change to the artifact's contract — a removed/renamed field, an incompatible output shape, a changed invariant a consumer relies on. |

A minor or major bump zeroes the lower components (`1.0.3` → `1.1.0`, not `1.1.3`).

### Examples

- Fix a typo in a skill's `SKILL.md` → **patch**.
- Move where a skill writes its output, or add a new manifest file → **minor**.
- Rename a manifest field consumers read, or remove a documented capability → **major**.

## Skill body placeholders

A skill that references a file in its own installed directory, or in a sibling
skill's, must name that path with a placeholder rather than one agent's literal
layout.
The path differs per agent (`.claude/skills/`, `.agents/skills/`,
`.opencode/skills/`), so a hardcoded `.claude/…` is correct on Claude and broken
on the other two.

| Placeholder | Resolves to |
|-------------|-------------|
| `{skillDir}` | the directory this skill installs into |
| `{skillsDir}` | its parent, holding all installed skills |

Use `{skillDir}/scripts/x.sh` for your own files and
`{skillsDir}/<sibling>/scripts/x.sh` for a sibling's. Both are substituted in
`SKILL.md` and in every file listed under `files:`, resolving to a
project-relative path at project scope and an absolute one at global scope.

Any other `{…}` is left untouched, so JSON examples, Python f-strings, and shell
expansions in a skill body are safe. The cost of that leniency is that a typo
(`{skilDir}`) would ship as a literal and fail only when a user runs the skill,
so `patronus check-placeholders` fails the build on a `{ski…dir}`-shaped string
that is not exactly one of the two placeholders. It runs in CI on every pull request and every push to `main`.

## Releasing the binary

**The binary and the catalog are two separate tracks on separate cadences, and a
`v*` tag ships only one of them.** Getting this backwards is the standing trap:
it makes agents tag releases that deliver nothing, and size a release against
changes it does not carry.

| Track | Ships | Triggered by | Versioned by |
|---|---|---|---|
| **Binary** | The `patronus` executable, for six platforms | pushing a `v*` tag (`release.yml`) | the git tag; stamped in via `-ldflags` |
| **Catalog** | Artifact content under `artifacts/` and `profiles/` | pushing to `main` (`publish-catalog.yml`) | each artifact's own `version:` (see above) |

Consequences worth stating plainly, because each one has misled somebody:

- **A tag never re-ships catalog content, and a catalog change never re-ships the
  binary.** Users reach them independently: `patronus update` pulls new artifact
  versions from the registry without touching the installed binary.
- **Only Go changes justify a tag.** If a change touches nothing but `artifacts/`
  and `profiles/`, it reaches users the moment it lands on `main` and needs no
  release. Check with `git diff --stat <last-tag>..HEAD -- '*.go' go.mod go.sum`.
- **Size the release by what the binary does, not by what the CHANGELOG entry
  says.** A breaking artifact rename is not a major binary release: nothing in the
  binary's interface moved. Ask what differs between the old binary and the new one.
- There is **no version constant in the Go source** to bump. Cutting a release is a
  CHANGELOG commit plus an annotated tag, nothing else.

A CHANGELOG entry usually spans both tracks, since one release cycle carries both
kinds of change. When it does, say at the top of the entry which half the binary
delivers, so a reader upgrading the binary knows what they are actually getting.

### Cutting one

1. Rename the `## Unreleased` heading to the new version, and write the entry for
   the person upgrading (what behaves differently on their machine).
2. Verify the gates the way CI will: the application and catalog gates in
   [Test gates](#test-gates) plus `golangci-lint run`, and complete the
   [release review checklist](#release-review-checklist).
3. Commit as `chore(release): cut vX.Y.Z`, stating why the bump is the size it is.
4. Merge the release PR into `main`, then, as `darkquasar`, push an annotated tag
   (`git tag -a vX.Y.Z`) at that merged commit. Its message is prose about what
   changed, matching the tone of the existing tags. Other authors' PRs require
   the owner's approval; the owner merges their own PRs.
5. Confirm `release.yml` went green and the six binaries plus `checksums.txt` are
   attached to the release.

## Profiles and the catalog

### Executable packages and content packages

An **executable package (Brew/Scoop/WinGet)** distributes the CLI's existing
release binaries. A Patronus **content package** belongs to the separate content
catalogue. Content publication never triggers an executable-package update;
native packages run no Patronus content lifecycle commands and preserve content,
harness files and user state. Consumer lifecycle, pinning, checksums and PATH
coexistence are in [installation](docs/installation.mdx).

**Publication/commissioning pending.** Activation requires owner-approved producer
code on `main` and the configured distribution controls. Native catalogue branches,
GitHub Apps, environment restrictions, server-side branch-denial probes, the tap's
`<app>-*` release tag rulesets and immutable releases, and real publication still
need commissioning. The accepted
Windows x64 synthetic run is [42178401448](https://github.com/darkquasar/patronus/actions/runs/42178401448);
it does not establish live availability or full upgrade/removal preservation.
WinGet stays pending until the exact version is observed in Microsoft's default
source, even after PR merge. No npm channel or account is required.

The maintainer runbooks live on the tap's protected control branch:
[setup](https://github.com/darkquasar/homebrew-tap/blob/distribution-control/docs/setup.md),
[onboarding](https://github.com/darkquasar/homebrew-tap/blob/distribution-control/docs/onboarding.md)
and [recovery](https://github.com/darkquasar/homebrew-tap/blob/distribution-control/docs/recovery.md).
These links become available when the runbook candidates are published.
They cover owner registration approval, exact Apps/environments/rules, protected
snapshots, historical baselines, second-app staging and selected-channel retry.
Only the owner `darkquasar` (ID `28435534`) approves registration/authority and
cuts releases. CI prepares WinGet branches; the owner submits PRs in the browser
and follows normal GitHub notifications.

An eligible stable `vMAJOR.MINOR.PATCH` release produces a run/attempt-bound
`distribution-receipt-<run>-<attempt>` artifact, then the default-branch
`package-distribution` workflow verifies source/receipt before dispatching
`distribute.yml` explicitly to `distribution-control`. The controller admits
the registered dispatcher, fixes the release snapshot in `distribution-records`,
then a separate publisher reads it back and validates native candidates. Keys
are isolated by role; the authority prepare job also needs a separate producer
Actions/Contents-read App. A secret-free Windows source-query job observes the
registered WinGet ID; its actual runtime/client availability remains pending.

Homebrew ships prebuilt bottles. They wrap the exact release executables and
are hosted on the tap's GitHub Releases under tag `<app>-<version>` (for example
`patronus-2.7.1`), never on the Patronus release. Between authority and
publisher, the tap's unprivileged `native-checks` job (no environment, no keys)
builds the four bottles when none are published yet, then runs the
[per-release native package checks](#per-release-native-package-checks).
The publisher uploads the checked bottles, reads them back from their public
URLs and only then commits the bottled formula. A failed check stops only its
own channel.

```text
+----------------+ ~> workflow_run   +------------------+ => dispatch   +----------------+
| Release [CI]   | ~~~~~~~~~~~~~~~~> | Main dispatcher  | ============> | Tap authority  |
+----------------+                   +------------------+               +----------------+
                                                                          |            |
                                                     => record_ref (each) |            |
                                                                          v            v
                                                         +----------------+  +----------------+
                                                         | Native checks  |  | Tap publisher  |
                                                         | [tap, no keys] |  +----------------+
                                                         +----------------+          ^
                                                                 | ~> artifacts      |
                                                                 +-------------------+
```

Receipt capture/upload is deliberately non-fatal to executable release
publication: normal binaries remain downloadable when packaging reports
`receipt-unavailable`. Do not rebuild or retag to recover a receipt. With an
intact receipt, the owner can start a fresh producer dispatch for the original
run/attempt; without it and before a protected snapshot exists, inspect and
explicitly approve those exact published bytes as a baseline. After freezing,
recovery uses protected records and current destinations, not invented Actions
history. Use fresh dispatches after controller/configuration changes; owner
reruns of bot-origin prepare runs fail the original/triggering-actor checks.

Profiles (`profiles/*.yaml`) select artifacts by name; they do not carry an
artifact version themselves. When you bump an artifact, you do not need to touch
the profiles that reference it — they resolve to the latest published version at
install time.

Run the catalog gate after any artifact, recipe, profile or manifest change
(see [Test gates](#test-gates)). A new profile also needs an entry in
`docs/compatibility/profile-targets.yaml` naming its admitted lock targets;
the gate refuses a profile that is missing there.

### Static directory package authoring

Package source lives under `packages/<name>/`, with an explicit payload list
in `package.yaml`. Build a selected package before writing its recipe pins,
then run the full build to verify those pins:

```sh
go run ./cmd/patronus build --package pi-sandbox --out /tmp/patronus-pi-package
go test ./packages/pi-sandbox/tests ./cmd/patronus -run 'PiPackage|Directory' -count=1
go run ./cmd/patronus build --out /tmp/patronus-pi-registry
go run ./cmd/patronus check-versions
```

Copy the emitted digest and actual registry object URL into the recipe's
platform asset. Never use a rolling image tag or a guessed package checksum.
See [package delivery](docs/package-delivery.md#authoring-a-package) for the
descriptor, sidecars and version rules. Run the four repository quality gates
before committing. Local builds do not upload package objects or launch sbx.

For directory-delivery changes, also run the lifecycle tests, both publication
shell suites and the three cross-builds in the
[local release checklist](docs/package-delivery.md#local-release-verification).
Keep cross-build outputs outside the checkout. Use `CGO_ENABLED=0` for those
builds; the native race suite requires CGO enabled. Windows compilation checks
the unsupported-platform fallback and does not establish directory mutation
support there. Record the tested revision and any untested runtime behavior in
the release handoff.

## Test gates

Application code and catalog content have separate gates. Each runs locally as
one command and in CI as its own job, so a failure names its owner. Neither gate
runs shipped guard or supervisor programs, fake harnesses, native CLIs or SDKs.

| Gate | CI job | Owns failures in | Local command |
|---|---|---|---|
| Application contract | `ci / test` | Go formatting, vet and `go test -race ./...` of the root module, run on a scratch copy **without** `artifacts/` and `profiles/` | `bash tools/tests/application-contract.sh --out "$(mktemp -d)" --without-catalog` |
| Catalog contract | `ci / catalog-contract`, and the `publish-catalog` build job on its own registry | public `build`; the standalone `tools/catalog-check` module (gofmt/vet/test) and its structure, profile-case and ledger checks; public-CLI `lock` for every admitted profile/target case; `check-placeholders`, `check-gate-intent`, `check-versions`; publication and package-delivery contract suites; Pi static evidence and content checks | `bash tools/tests/catalog-contract.sh --out "$(mktemp -d)" --base origin/main` |
| Lint | `ci / lint` | `golangci-lint` against `.golangci.yml` | `golangci-lint run` |
| Publication policy | `ci / publication-policy` | Offline publication authorization, executable producer and installer compatibility/rejection | Commands below; PowerShell (`pwsh`) is required for its script |

Run the publication-policy job's local commands from the repository root:

```sh
python3 -B -m unittest discover -s tools/tests -p 'test_publication_authorization.py'
python3 -B -m unittest discover -s tools/tests -p 'test_distribution_producer.py'
bash tools/tests/install-version-test.sh
bash tools/tests/install-checksum-test.sh
pwsh -NoProfile -File tools/tests/install-checksum-test.ps1
```

Missing `pwsh` blocks that local check; a Unix run does not establish native
Windows User PATH behavior.

Per-release native package checks run in the tap; see
[Per-release native package checks](#per-release-native-package-checks).

`--out` must be an empty directory outside the checkout. The scripts do not take
a resource lock; a caller that shares a machine wraps them in its own `flock`.
They do not set `GOPROXY`. Offline local runs export `GOPROXY=off GOSUMDB=off
GOTOOLCHAIN=local` first. The catalog gate writes only below `--out`: it locks
each case in a scratch source copy with `--local-registry` and an isolated
`HOME`, keeps the public lock as `closures/<profile>--<target>.lock`, and then
runs `catalog-check --closures` on those locks. It never fetches the registry
URL. A lane-only profile (admitted only for `codex` or `pi`) must refuse
`--target claude`. A documented stub status notice is allowed. Any other lock
warning (an unresolved item or an unmatched `without`) fails the case.

Artifact authors normally change content and manifests only. An ordinary
artifact addition, prose edit or version bump must not need an application-test
edit; if it does, the application test is coupled to the real catalog and is the
defect. An invalid bundle fails the catalog gate instead.

The two gates prove packaging and delivery mechanics only. Artifact behaviour
(the retained guard and supervisor fixtures behind the `artifactbehavior` Go
build tag, and native harness loading, routing, hooks and trust) is deferred and
**not run** in either gate. Its retained historical results are not current
acceptance, and a passing gate does not mark any native row passed.

### Per-release native package checks

Per-release native package checks live in the tap, not in this repository.
Every release dispatch runs the tap's `.github/workflows/native-checks.yml` on
GitHub-hosted runners before any channel publishes:

| Manager | Runner labels | Steps |
|---|---|---|
| Homebrew | `macos-15`, `macos-15-intel`, `ubuntu-24.04`, `ubuntu-24.04-arm` | Validate, pour the bottle, `patronus --version`, `brew test`, uninstall |
| Scoop and WinGet | `windows-2025`, `windows-11-arm` | Validate, install as an ordinary user, `patronus --version`, uninstall |

Results are the run's `native-checks` workflow artifact (`native-checks.json`),
and each channel result carries a `native_checks` reference (run and digest).
A failing platform stops only its channel, which reports
`failed: native-check-failed` with the failing platforms. A platform without a
hosted runner would be recorded `statically-mapped` (`no-hosted-runner`), never
as tested; currently all six run.

Tap pull requests that touch `distribution/**` or `.github/workflows/**` run the
same matrix in fixture mode, with synthetic binaries served on loopback. Labels
add more:

| PR label | Adds |
|---|---|
| `native-lifecycle` | The one-time upgrade and preservation acceptance (PD-13): bottled Homebrew on macOS arm64, Scoop and WinGet on Windows x64 |
| `native-negative` | Wrong-architecture and wrong-version candidates, which must fail |
| `native-second-app` | The same matrix for the second fixture app (Lumen) |

Repeat the lifecycle acceptance only when a package shape changes, for example
formula to cask or bottled to unbottled. The fixture-mode matrix, lifecycle,
negatives and second-app checks passed in tap run
[44079561564](https://github.com/darkquasar/homebrew-tap/actions/runs/44079561564);
the first real release check is pending. The procedure is in the tap's
[native check README](https://github.com/darkquasar/homebrew-tap/blob/distribution-control/distribution/tests/native/README.md),
which becomes available with the runbook candidates.

This repository's bounded `package-windows-validation` workflow and its
[procedure](tools/tests/package-windows-validation.md) stay as the historical
T3 synthetic Windows evidence. That workflow runs only for the package
distribution pull request and is not a per-release check. Do not infer live
WinGet indexing from manifest validation or a local-manifest install.

### Pi Python checks

| Check | Gate | Reason |
|---|---|---|
| `tools/tests/test_pi_evidence.py` | catalog | Static contract of `tools/check-pi-evidence.py` over the retained evidence contract and templates; no native calls. |
| `tools/tests/test_pi_qualification_examples.py` | none | Retained qualification-example assertions; outside the current structural development gates. |
| `tools/tests/test_pi_native_content.py` | none | Retained prose inventories; content semantics belong to release review, not substring gates. |
| `tools/qualification/pi-native-rpc.py`, `pi-native-resources.mjs` | none | Native qualification procedures; run only under a separate native grant. |
| `tools/qualification/removal-performance/` | none | Opt-in measurement procedure, not a development gate. |

### Release review checklist

Some requirements have no machine-readable contract. Review them by hand before a
release that changes the affected files, and record the reviewer and result in
the release handoff. Do not replace them with word lists in Go tests or in the
catalog tool.

- **Codex host APIs.** Changed `-cx` skill, agent, instruction and hook bodies use
  only Codex-available tools and roots. They do not instruct Claude or Pi host
  APIs, tools or paths that Codex does not provide.
- **Parity wording.** Changed docs, artifacts and `docs/compatibility/*.yaml`
  entries do not claim complete or total Claude/Pi parity for Codex. Partial and
  runtime-pending rows stay labelled as such.
- **Pi editorial exclusion.** The effective `core-profile-pi` closure (the
  catalog gate's `closures/core-profile-pi--pi.lock`) does not select the
  editorial skills, and the eight retained editorial source/profile files are
  unchanged unless separately approved.
- **Structural resources.** Catalog checks cover literal distributed Markdown references
  and native agent selected skills versus declared dependency/type/target metadata.
  `distributed-reference-exceptions.yaml` retains only previously reviewed legacy debt,
  bound to exact manifest/body hashes; it is not proof that those sidecars ship.
- **Behaviour evidence.** Ledger and qualification wording keeps guard and
  supervisor fixture results as historical and not run. Native rows stay
  pending until a separately authorized native run records observations.

## Documentation layout

Tracked `docs/` contains user-facing MDX pages and Mintlify's `docs.json`. Add new
pages to its navigation. Keep test fixtures, evidence templates, validation
procedures and results under `tools/qualification/` or the owning package's
`tests/` directory. Contributor decisions stay tracked under `docs/adr/` and are excluded from
Mintlify processing by `docs/.mintignore`.
`docs/specs/` remains gitignored local planning state and is excluded from
Mintlify processing by `docs/.mintignore`.

For a local docs preview, run `npx mint dev` from `docs/`. Do not move local specs
into published navigation or commit generated preview files.

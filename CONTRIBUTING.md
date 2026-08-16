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

**This section is the authoritative statement, and `.gitignore` is its enforcement.** Both are
tracked, so both travel to a fresh clone. `CLAUDE.md` and `AGENTS.md` are *not* the place to put
it: both are gitignored (`.gitignore:20-21`) because Patronus itself appends instruction sections
into them at install time, so anything hand-written there is local to one machine and can be
rewritten by the next `patronus install`.

The upstream `ticket` instruction Patronus ships (`artifacts/instructions/ticket/INSTRUCTIONS.md`)
says the opposite, that `.tickets/` is "committed like any other file". That is the correct default
for the projects the catalog serves, and it deliberately stays as it is. **This repo is the
exception, and this file is where the exception lives.** Do not edit the shipped artifact to match
local practice.

`tk` is the only tracker this repo uses. If you find a generated `bd`/beads block claiming
otherwise in an ignored instruction file, it is boilerplate from a tool that was never installed
here: `bd` is not on this machine and `.beads/` has never existed in this history.

Two consequences fall out of the graph being local, and both have misled a session before:

- **A ticket id resolves only on the machine that created it.** A plan that hands work to
  another contributor cannot assume `tk show <id>` works for them.
- **`docs/specs/` is gitignored too**, so a ticket pointing at a plan names a path a fresh clone
  does not have. Say so in the ticket body rather than leaving the reader to improvise from a
  heading they cannot open.

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
that is not exactly one of the two placeholders. It runs in CI on every push.

## Profiles and the catalog

Profiles (`profiles/*.yaml`) select artifacts by name; they do not carry an
artifact version themselves. When you bump an artifact, you do not need to touch
the profiles that reference it — they resolve to the latest published version at
install time.

Run the catalog-integrity test after any manifest change:

```console
go test ./internal/registry/ -run Catalog -count=1
```

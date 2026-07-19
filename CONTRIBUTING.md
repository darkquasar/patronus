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

## Profiles and the catalog

Profiles (`profiles/*.yaml`) select artifacts by name; they do not carry an
artifact version themselves. When you bump an artifact, you do not need to touch
the profiles that reference it — they resolve to the latest published version at
install time.

Run the catalog-integrity test after any manifest change:

```console
go test ./internal/registry/ -run Catalog -count=1
```

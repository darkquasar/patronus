---
status: accepted
date: 2026-08-03
---

# Every Patronus manifest carries a required `version:`

## Context

Patronus installs three installable families — artifacts, recipes, and profiles —
plus upstream plugins. All four embed the same identity header (`manifest.Meta`).
Until now only **artifacts** carried a `version:`; recipes and profiles carried
none. The wire/deliver taxonomy plan wrote that down as a deliberate boundary:

> **Recipes and profiles carry NO `version:` field** — only `artifacts/*/patronus.yaml`
> do. The version-bump CI guard is scoped to `artifacts/` only.
> — `docs/plans/2026-08-01-wire-deliver-taxonomy-install-consent.md`

That boundary produced a bug class, found live and captured in **pat-78oj**:
`patronus update <recipe>` had **nothing to compare**. `latestVersion()` searched
only `cat.Artifacts` and returned `""` for any recipe, so update's
`c.latest == "" → leaving as-is` arm swallowed every recipe. A recipe could never
be updated, its content change could never be version-guarded, and its install
could never be diffed against a newer catalog. An unversioned item is
un-updatable, un-guardable, un-diffable.

A stopgap shipped (`isRecipe()` + an always-refresh arm in `update.go`), but it
duplicates the artifact update path with a recipe-only special case — the exact
shape Fowler's "one source of truth" rule rejects. The `version:` field being
artifact-only is the root cause; the special case is the symptom.

## Decision

**`version:` is a REQUIRED field of the shared manifest schema. Every installable
family — artifact, recipe, profile, plugin — carries one, validated in one place.**

This **supersedes** the taxonomy plan's "recipes and profiles carry no version"
note. The user ruling that governs it: *outside content must ascribe to the same
schema* — a recipe or profile is an installable like any other, so it identifies,
versions, and updates like any other. There is no second-class installable.

1. **Validation is single-source.** `validateMeta` (the header check every family's
   loader already funnels through — `Validate`, `LoadProfile`, `LoadPlugin`,
   `validateRecipe`) requires a non-empty `version:`. Not `validateRecipe`: putting
   it there would make recipes *stricter* than artifacts and re-fork the rule the
   moment a fourth family appears.

2. **Migration rides the same change.** The moment `validateMeta` requires the
   field, every un-migrated recipe (19) and profile (17) fails to load. So the
   `version: 1.0.0` migration and the `validateMeta` edit are **one PR** — they
   cannot be split without a red window.

3. **Recipes update like artifacts, with no special case.** `latestVersion` ranges
   `cat.Recipes` too; recipe install diffs are stamped with `rec.Version` (mirroring
   the artifact stamp at `internal/adapter/engine.go:67`) so state records a recipe's
   `ItemVersion`; the `isRecipe` STOPGAP and its always-refresh arm are **deleted**.
   The recipe **detector** (`isRecipe(cat, name)` → `catalogHasRecipe`) stays — it is
   the sole populator of `recipeToolSet` (the per-tool wiring fan-out), a separate
   concern from versioning.

4. **The version-bump guard extends to recipes as a NEW single-file path.** The
   existing guard assumes `artifacts/<type>/<name>/<file>` — a directory whose
   sibling files are content and whose `patronus.yaml` is the version carrier. A
   recipe is a FLAT single file (`recipes/<name>.yaml`) where the manifest IS the
   whole content. That is a distinct code path (bypassing the content-vs-manifest
   split), not a widening of `artifactDirOf`. `profiles/` guard coverage and profile
   `update` support are FAST-FOLLOW tickets, not this change.

## Boundary: versioned, but not typed

A recipe carries a `version:` like every manifest, but its **shape** stays computed
(`Recipe.Shape()`, a pure function of deliver × wire) — it is NOT an `ArtifactType`.
Versioning is an identity concern (who am I, which revision); shape is a structural
concern (what do I write). Adding the first does not make a recipe declare the
second. The boundary comment in `recipe.go` says exactly this so the next reader
does not reach for a `type:` field on recipes.

## Consequences

- **The `update` bug class is gone.** A recipe compares installed-vs-latest exactly
  like an artifact — one uniform arm, no `isRecipe` fork.
- **Recipe content changes are now guardable.** A recipe edit with no `version:` bump
  reds `check-versions`, the same contract artifacts ship under.
- **Two fast-follow tickets** capture the deliberately-scoped-out work: extend the
  guard to `profiles/`, and add profile latest-version support to `patronus update`.
- **The taxonomy plan's boundary note is dead.** Anyone reading it should follow the
  supersede pointer here.

## The general rule this is an instance of

> **One schema, one source of truth for every field on it.** A field that some
> members of a family carry and others do not forces a special case at every site
> that reads it — and each special case is a place the two paths can drift. The fix
> is not a better special case; it is to put the field on the shared header and
> validate it once.

---
status: accepted
date: 2026-07-12
---

# Tests assert Patronus's behavior, never the catalog's contents

## Context

Patronus's integration tests build the **real catalog** to test Patronus: `builtRegistry` runs
`patronus build` against the actual `recipes/` + `artifacts/` checkout, so the tests inherit the
actual **pins** — third-party sha256 digests we do not control.

This stayed invisible for as long as every core binary was an **archive** delivery, because
`internal/recipe/recipe.go:207-209` SKIPs an archive whenever the file is merely *present* on disk,
**without ever hashing it**:

```go
if spec.Archive != "" {
    return diff.Skip // present; archive sha can't be rechecked against the binary
}
sum := sha256.Sum256(data)   // raw deliveries: always hashed, every run
```

So `stubBinary` could drop 17 dummy bytes at `~/.patronus/bin/gitleaks`, the row went SKIP, and the
pin was never consulted. **The tests looked like they tested installs. The sha — the entire trust
anchor — was never exercised.** They were free-riding on a hole.

`tk` — the first **raw** (`source: url`) delivery to enter a profile — hashes on both classify and
apply. That instantly made a third-party 47KB digest a hard dependency of ~19 tests that do not care
about tk at all and are only asserting that the `requires:` closure resolves and `CLAUDE.md` composes.

Every escape route then cost something real: **vendor** the script (couples the suite to an upstream
digest that breaks the day someone cuts a release, and does not survive fixing the SKIP hole);
**fetch it in CI** (hands a third-party repo the ability to execute bytes in the pipeline); **weaken
the sha check** (puts a security hole in production to serve a test).

**The root cause was never tk.** It is that these tests bind to the **contents** of the catalog rather
than to Patronus's **behavior**. tk was simply the first thing to make that latent coupling bite.

## Decision

**A test asserts Patronus's behavior. It never binds to the catalog's contents.**

1. **Mechanism tests use an invented fixture catalog.** A test that exercises the requires-closure,
   layer resolution, `@tool` flavouring, `extends:` composition, hook folding, lock provenance, or
   remove round-trips must build its catalog **in the test**, from items it invents. The item names
   are arbitrary to what is being proven, so they must not be real ones.

2. **A fixture's pin is `sha256(bytes the test just invented)`.** Never a digest copied from upstream.
   Then there is nothing to drift from — ever:

   ```go
   bin := []byte("#!/bin/sh\necho fix\n")
   pin := sha256.Sum256(bin)   // the pin IS the bytes we invented
   ```

3. **Third-party bytes never enter the repo as test inputs.** No vendoring a real script to satisfy a
   real pin.

4. **CI never fetches attacker-controllable remote bytes.** That would execute third-party code in a
   credentialed pipeline on every PR — including fork PRs, **before human review**. This is a line,
   not a trade-off.

5. **The sha check is never weakened to serve a test.** The check IS the trust anchor. A test-only
   bypass makes the tested and the shipped path diverge *exactly* at the security boundary.

6. **Real-catalog tests may assert the catalog's SHAPE as a GENERAL PROPERTY — never enumerate its
   specific contents by name.** A test that reads the real catalog is permitted only when what it
   asserts holds for the *whole* catalog and needs no edit when an item is added or removed:
   - **validity, generically** — *is the catalog well-formed?* Every `requires:` resolves; every
     profile resolves for every tool and leaves no declared layer hollow
     (`cmd/patronus/profile_structure_test.go`); the type/role ontology is internally consistent
     (`internal/registry/catalog_integrity_test.go`, a single count+shape guard over all artifacts).
     These do not name individual items, so a catalog change touches them at most once (register the
     new item's type/role), not per assertion.

   Such a test must never fetch, and never hash upstream bytes.

   **AMENDED 2026-08-02 (supersedes the original point 6).** The original permitted a second,
   name-enumerating category — *"does `core` really ship `grilling`?"*, asserted by listing the
   expected item names in Go (`coreSkills` and kin) — and called it a product guarantee. That was
   wrong, and the reasoning below (about a "green tautology") is exactly why. **npm's own test suite
   does not assert that `left-pad` is in its registry;** a package manager tests its *mechanism*
   (resolve, verify, place) on invented fixtures and validates registry *contents* with a
   publish-time schema/lint, never with a hand-written per-package assertion in the installer's test
   suite. Enumerating a profile's items in Go is that anti-pattern in miniature: every catalog edit
   forces a matching test edit, which trains everyone to treat the test as a speed-bump to re-sync —
   and a test that is routinely edited to match the change it guards has lost the power to guard. The
   real regression the name lists worried about (a profile silently losing the tooling it promises)
   is now caught **generically** by the hollow-layer property test above, with zero per-item upkeep.
   *Which* items a profile ships is CATALOG CONTENT: it is decided in the profile YAML and reviewed
   in the PR that changes it, the way `create-react-app`'s template is reviewed, not unit-tested. The
   name-mirroring tests (`TestCoreProfileWiresTheSkillSpine`, `TestCodeIntelHasSerenaAndGraphify`,
   `coreSkills`, …) were removed accordingly. See [[tests-assert-behavior-not-catalog-contents]].

7. **Network access is denied by default, structurally.** The suite must not be offline merely because
   every test *remembers* to swap the fetcher — that is a convention, not a control. A `TestMain`
   installs a deny-all fetcher that **panics with the URL** it was asked for, so forgetting fails
   loudly instead of silently reaching the internet.

## Consequences

- **A test that breaks when an unrelated pin is bumped is a bug in the test.** If bumping
  `recipes/tk.yaml` reddens a test that is not about tk, that test is asserting the wrong thing.
- **Migrating an existing test is not a mechanical rename.** Each site must first be classified by
  *what it asserts*. A **mechanism** assertion (folding, flavour divergence, remove round-trip) moves
  to invented fixtures. A **name-enumeration** assertion (`coreSkills` and kin) is not migrated at
  all — renaming it to fixture names would convert it into a green tautology, and keeping it as real
  names is the maintenance anti-pattern point 6 (amended) now rejects. It is **deleted**, and the
  invariant it gestured at (a profile is not silently hollow) is recovered generically by
  `profile_structure_test.go`.
- **Coverage increases.** `stubBinary` meant the FETCH apply path was never executed — a test
  documented as *"a real `--deploy` … FETCHes the binary"* (`p77_acceptance_test.go:47-51`) in fact
  stubs it at line 57 and proves nothing. A fixture serves invented bytes and drives
  download → verify → extract → place for real.
- **It is faster.** `builtRegistry` rebuilds the entire real catalog 33× per suite run.
- **This unblocks closing a production security hole.** Archive-SKIP-on-presence means that once *any*
  file exists at the dest, Patronus reports SKIP — *"verified"* — forever, without ever hashing it.
  Patronus already records the placed binary's sha (`internal/state/state.go:263-272`) and never reads
  it back on the install path. Fixing that **breaks `stubBinary`** — so the fixture catalog is not
  merely tidier, it is the **prerequisite** for the security fix.

## The general rule this is an instance of

> **A claim is not evidence.** A test that asserts an artifact is *present* has not asserted that it
> is *correct*. `SKIP` must mean *"I checked"*, never *"something was there."*

The same defect appears across this codebase wherever the thing that *reports* the state has drifted
from the thing that *is* the state: an installed skill that still names a deleted tool, an instruction
that misstates its own binary's capabilities, a test comment that describes a code path it stubs out.
Prefer checks that compare against **the thing**, never against something that merely **describes**
the thing.

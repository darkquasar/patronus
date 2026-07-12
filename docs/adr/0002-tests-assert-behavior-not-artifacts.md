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

6. **Real-catalog tests are permitted — but they may read the catalog's SHAPE, never its PINS.** Two
   legitimate reasons exist, and both are shape-only:
   - **validity** — *is the catalog well-formed?* (pins syntactically well-formed, every `requires:`
     resolves, every profile slot names a real item)
   - **contents** — *does `core` really ship `grilling`?* This is a **product guarantee**
     (`core_profile_integration_test.go`'s `coreSkills`), not incidental coupling. **Keep it.**

   Such a test must never fetch, and never hash upstream bytes.

7. **Network access is denied by default, structurally.** The suite must not be offline merely because
   every test *remembers* to swap the fetcher — that is a convention, not a control. A `TestMain`
   installs a deny-all fetcher that **panics with the URL** it was asked for, so forgetting fails
   loudly instead of silently reaching the internet.

## Consequences

- **A test that breaks when an unrelated pin is bumped is a bug in the test.** If bumping
  `recipes/tk.yaml` reddens a test that is not about tk, that test is asserting the wrong thing.
- **Migrating an existing test is not a mechanical rename.** Each site must first be classified by
  *what it asserts*. Renaming a contents assertion (`coreSkills`) to fixture names would convert a
  real product guarantee into a tautology — **green, and testing nothing.** That is the very failure
  this ADR exists to prevent.
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

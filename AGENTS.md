# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd prime` for full workflow context.

> **Architecture in one line:** Issues live in a local Dolt database
> (`.beads/dolt/`); cross-machine sync uses `bd dolt push/pull` (a
> git-compatible protocol), stored under `refs/dolt/data` on your git
> remote — separate from `refs/heads/*` where your code lives.
> `.beads/issues.jsonl` is a passive export, not the wire protocol.
>
> See [SYNC_CONCEPTS.md](https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md)
> for the one-screen overview and anti-patterns (don't treat JSONL as the
> source of truth; don't `bd import` during normal operation; don't
> reach for third-party Dolt hosting before trying the default).

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work atomically
bd close <id>         # Complete work
bd dolt push          # Push beads data to remote
```

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var

## Testing

Run the full check suite before every commit/push — these are the same gates CI
enforces (`.github/workflows/ci.yml`):

```bash
gofmt -l .          # must print nothing (formatting gate)
go vet ./...        # static checks
golangci-lint run   # the `lint` CI job — golangci-lint v2 (revive et al.); NOT covered by gofmt/vet
go test -race ./... # full suite with the race detector
```

`golangci-lint run` is a **separate CI gate** from `gofmt`/`go vet` and catches
things they don't (e.g. revive's exported-name/stutter rules). A green
`go test` + `gofmt` is NOT sufficient — run `golangci-lint run` locally or CI's
`lint` job will fail. Scope a single package while iterating with
`go test ./internal/<pkg>/ -run <TestName> -v`. Changed code ships with tests,
and all four gates above are clean before a commit.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:7510c1e2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->

<!-- patronus:start go-style-uber -->
# Idiomatic Go conventions (distilled from the Uber Go Style Guide)

Apply these when writing or reviewing Go. They are decision rules with their
rationale — paraphrased from the Uber Go Style Guide (Apache-2.0), not a
substitute for it. When `gofmt`, `go vet`, or the project's linter disagree with
a rule here, the tool wins.

## Errors

- **Choosing an error constructor** — decide on two axes: does the caller need to
  *match* the error to handle it, and is the message *static* or *dynamic*?
  - no match + static → `errors.New`
  - no match + dynamic → `fmt.Errorf`
  - match + static → a top-level `var Err… = errors.New(…)` (export it only if
    callers outside the package must match it)
  - match + dynamic → a custom `error` type (matched via `errors.As`)
  Exported error `var`s and types become part of the package's public API — once a
  caller can `errors.Is`/`errors.As` them, that is a contract you must keep.
- **Wrapping — three choices when propagating a failure:**
  - return the error *as-is* when the underlying message already says enough to
    locate the source and you have no context to add;
  - `fmt.Errorf("…: %w", err)` when the caller should be able to unwrap/match the
    cause — the good default for most wraps, but be aware callers may come to rely
    on it, so for a known sentinel/type document and test it as part of the contract;
  - `fmt.Errorf("…: %v", err)` to deliberately *obscure* the cause (callers can't
    match it; you can widen to `%w` later if needed).
  Prefer adding context so a vague "connection refused" becomes
  "call service foo: connection refused".
- **Keep added context succinct.** Drop "failed to" / "error while" noise — it
  states the obvious and piles up as the error percolates: aim for
  `x: y: new store: <cause>`, not `failed to x: failed to y: …`. (Once the error
  reaches logs/another system, *there* it should be clearly marked an error — an
  `err` tag or "Failed" prefix.)
- **Handle each error once.** A caller picks ONE of: match-and-branch
  (`errors.Is`/`errors.As`), log-and-degrade-gracefully (only when the operation
  isn't strictly necessary), translate to a domain error, or return it (wrapped or
  verbatim). Do **not** log an error *and* return it — your callers will handle it
  too, so logging-then-returning just floods the logs with duplicates for no gain.
- **Naming:** error globals are `Err…` (exported) / `err…` (unexported); custom
  error types are `…Error`. (This naming supersedes the leading-underscore rule for
  unexported globals.)
- **Type assertions:** always use the two-result form (`v, ok := x.(T)`) and handle
  `!ok`; the single-result form panics on a wrong type.
- **Don't panic** in library/production code — return errors and let the caller
  decide. `panic`/`os.Exit` belong only in `main` (or test setup), and `main`
  should have a single `Exit` point: `os.Exit`/`log.Fatal` runs no deferred
  cleanup, so funnel through one place that returns an exit code after defers run,
  rather than calling `os.Exit` from several functions.

## Concurrency

- **Don't fire-and-forget goroutines.** Every goroutine must have a predictable
  stop time or a way to be signalled to stop (a `context`, a close channel) AND a
  way to wait for its exit (e.g. `sync.WaitGroup`). A leaked goroutine outlives its
  purpose and causes hard-to-debug problems; the spawning code is responsible for
  its lifetime.
- **No goroutines in `init()`** — package init shouldn't start background work; it
  hides lifecycle and defeats the "caller owns the goroutine" rule above.
- **Channel size is one or none.** Default to unbuffered. Any larger buffer must be
  justified: explain what fills it, what bounds it, and what happens when it's full
  under load.
- **Zero-value `sync.Mutex` is valid** — never init it. Don't *embed* a mutex in an
  exported struct: that leaks `Lock`/`Unlock` into your public API and ties the
  lock's scope to the caller. Use an unexported field, and `defer mu.Unlock()`
  right after locking.
- For atomics, prefer typed wrappers (`go.uber.org/atomic`) over raw `sync/atomic`
  — they make the atomic type explicit and are harder to misuse (e.g. a raw
  `int32` used as a bool).

## Types, structs, enums, slices

- **Start enums at one** (`iota + 1`) when zero shouldn't be a meaningful value, so
  the zero value reads as "unset" and an uninitialized enum is detectable. (Leave
  the default zero meaningful — e.g. `…Unspecified = iota` — when that *is* the
  desired behavior.)
- **Copy slices and maps at boundaries.** When you store a caller-supplied
  slice/map on your struct, copy it first — otherwise the caller retains a
  reference and can mutate your internal state later. Likewise copy before
  *returning* an internal slice/map, or the caller can reach back into your state.
- **`nil` is a valid, usable slice.** Return `nil`, not `[]T{}`, for an empty
  result; `len`/`range`/`append` all handle a nil slice. Don't guard `len(s) > 0`
  before ranging, and don't distinguish nil from empty in your API unless the
  difference is genuinely meaningful.
- **Avoid embedding types in exported structs** — embedding leaks the embedded
  type's methods and fields into your public API and breaks encapsulation; if the
  embedded type changes, your API changes. Prefer a named field.
- **Receivers:** be consistent per type — don't mix value and pointer receivers on
  the same type. Use pointer receivers for methods that mutate the receiver or when
  the struct is large. Note the addressability rule that follows from this: a value
  in a map isn't addressable, so a pointer-receiver method can't be called on
  `m[k]` — store pointers (`map[K]*T`) if you need to call mutating methods on
  entries.
- **Pointers to interfaces are almost never needed** — an interface value already
  holds a pointer when it must; `*SomeInterface` is a code smell.
- **Verify interface compliance at compile time** when a type is meant to satisfy
  an interface: `var _ http.Handler = (*Handler)(nil)` turns a missing/renamed
  method into a build error instead of a runtime surprise.

## Globals, init, mutability

- **Avoid mutable globals.** Package-level rewritable `var`s create hidden coupling
  and make tests order-dependent — inject the dependency (a field, a parameter, a
  closure) instead. Constants and read-only values are fine.
- **Avoid `init()`.** It runs implicitly, hides ordering, and complicates testing.
  Prefer explicit construction. If `init` is truly unavoidable it must be
  deterministic: depend on nothing external or order-sensitive, and do no I/O or
  network.
- Prefix unexported package-level globals with `_` (`_defaultPort`) so they're
  visibly distinct from locals at the use site (except error globals — see naming).

## Time, marshaling, names

- Use the `"time"` package for ALL instants and periods — `time.Time` for an
  instant, `time.Duration` for a period; never raw `int` seconds (it loses units
  and invites arithmetic bugs). When crossing an external boundary, prefer formats
  that carry the unit/zone (RFC 3339, explicit `time.Duration` strings) over bare
  numbers.
- Add field tags (`json:"…"`, etc.) to any struct you marshal, so the serialized
  keys are an explicit contract rather than an accident of Go field names that a
  rename would silently break.
- Don't shadow built-in names (`error`, `string`, `len`, `copy`, `new`, …) — a
  local named `error` or `len` quietly disables the builtin in that scope.

## Performance (apply where it's on a hot path; don't pre-optimize)

- Prefer `strconv` over `fmt` for primitive↔string conversion — `strconv.Itoa` is
  markedly cheaper than `fmt.Sprint` for the same result.
- Avoid repeated `string`↔`[]byte` conversion (each one allocates and copies) —
  convert once and reuse the result.
- Specify container capacity when the size is known (`make([]T, 0, n)`,
  `make(map[K]V, n)`) to avoid incremental reallocation/rehashing as it grows.

## Style & naming

- **Be consistent within the file/package above all** — local consistency beats any
  single rule here; a change should look like the code around it.
- **Reduce nesting.** Handle errors and special cases first and `return`/`continue`
  early so the happy path stays at the left margin; deeply nested `if/else` is
  harder to follow than a sequence of early exits.
- **No unnecessary `else`.** If both branches of an `if` just set the same variable,
  set the default first and override in a single `if` — drop the `else`.
- **Reduce variable scope** — declare at first use, not at the top of the function,
  so a reader sees the value and its use together. (But don't hoist a declaration
  *into* a tighter scope if it's used after that scope.)
- **Avoid naked parameters at call sites** — a bare `true`/`nil`/`0` whose meaning
  isn't obvious should be named (a named arg, an enum, an options struct), or
  annotated with a `/* flag: */` comment, so the call reads correctly.
- Group similar declarations (`var ( … )`, `const ( … )`), and order imports as
  stdlib then everything else — let `goimports`/`gofmt` enforce both.
- Use raw string literals (`` `…` ``) for strings with backslashes/quotes to avoid
  escaping noise.
- **Initialize structs with field names**, omit zero-value fields, and use `&T{…}`
  (not `new(T)`) when you're setting fields. For a plain zero value use `var t T`
  rather than `t := T{}`; use `:=` for non-zero initialization.
- Keep `Printf`-style format strings as package `const`s and suffix `Printf`-style
  function names with `f` (`logf`, not `log`) so `go vet`'s printf check can verify
  the verbs against the arguments.

## Patterns

- **Table-driven tests** for families of cases that vary only by input/expectation:
  a slice of named case structs ranged over with `t.Run(tt.name, …)`, so each case
  is isolated and named in failures. **Keep the loop body simple** — no conditional
  assertions or per-case branching (`shouldErr`, `setupMocks func(...)`, `if`
  ladders). If a case needs different *logic* (not just different data), split it
  into its own table or its own `Test…` function; complex table tests are hard to
  debug. Aim for shallow tests: every table field used by every case, all logic run
  for all cases.
- **Functional options** for constructors/APIs with many optional parameters —
  `Option func(*config)` values passed variadically — instead of long positional
  parameter lists or a config struct the caller must fill out completely. It keeps
  the common call short and lets options be added without breaking callers.

## Linting

- The baseline is `gofmt` + `go vet` (non-negotiable). Beyond that, run
  `staticcheck` and the project's `golangci-lint` config and treat their findings
  as part of "green" — don't ship with new lint warnings.
<!-- patronus:end go-style-uber -->

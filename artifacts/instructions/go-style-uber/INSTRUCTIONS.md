# Idiomatic Go conventions (distilled from the Uber Go Style Guide)

Apply these when writing or reviewing Go. They are decision rules — paraphrased
from the Uber Go Style Guide (Apache-2.0), not a substitute for it. When `gofmt`,
`go vet`, or the project's linter disagree with a rule here, the tool wins.

## Errors

- **Choosing an error constructor** — decide on two axes: does the caller need to
  *match* the error, and is the message *static* or *dynamic*?
  - no match + static → `errors.New`
  - no match + dynamic → `fmt.Errorf`
  - match + static → a top-level `var Err… = errors.New(…)`
  - match + dynamic → a custom `error` type (matched via `errors.As`)
- Exported error `var`s and types become part of the package's public API — treat
  them as a contract.
- **Wrapping:** return the error as-is when you have no context to add. Otherwise
  add context with `fmt.Errorf`: `%w` when the caller should be able to unwrap/match
  the cause (the good default; document it as contract), `%v` to deliberately
  obscure it.
- Keep added context **succinct** — no "failed to" / "error while" noise; it piles
  up as the error percolates (`x: y: new store: <cause>`, not
  `failed to x: failed to y: …`).
- Name error globals `Err…` (exported) / `err…` (unexported); custom error types
  are `…Error`.
- Handle a type assertion's two-return form (`v, ok := x.(T)`); never the single
  form that panics.
- **Don't panic** in library/production code — return errors. `panic`/`os.Exit`
  belong only in `main` (or test setup). Prefer one `Exit` point in `main`.

## Concurrency

- **Don't fire-and-forget goroutines.** A goroutine must have a predictable stop
  time or a way to be signalled to stop (context/channel); wait for it (e.g.
  `sync.WaitGroup`). Leaked goroutines cause hard-to-debug problems.
- Channel size is **one or none** — default to unbuffered; any other buffer size
  must be justified (and questioned under load).
- Zero-value `sync.Mutex` is valid — don't init it, and **don't embed** it in a
  public struct (it leaks `Lock`/`Unlock` into your API). Use a private field.
- For atomics, prefer `go.uber.org/atomic` (typed, harder to misuse) over raw
  `sync/atomic`.

## Types, structs, enums, slices

- **Start enums at one** (via `iota + 1`) when zero shouldn't be a meaningful
  value, so the zero value is detectably "unset".
- **Copy slices and maps at boundaries.** When you store a passed-in slice/map, or
  return an internal one, copy it — otherwise callers can mutate your internal
  state through the shared reference.
- `nil` is a valid, usable slice — return `nil` rather than `[]T{}`, and don't
  guard `len(s) > 0` before ranging.
- Avoid embedding types in **public** structs (it leaks implementation detail into
  the API and breaks encapsulation).
- Pointers to interfaces are almost never needed.
- **Verify interface compliance at compile time** when it matters:
  `var _ http.Handler = (*Handler)(nil)`.
- Receivers: be consistent (don't mix value and pointer receivers on one type);
  methods that mutate or that are large need pointer receivers.

## Globals, init, mutability

- **Avoid mutable globals** — inject dependencies instead of reaching for package
  state. Prefer values/closures over rewritable `var`s.
- **Avoid `init()`.** It hides ordering and complicates testing. Prefer explicit
  construction. If unavoidable, it must be deterministic, depend on nothing
  external/ordered, and do no I/O.
- Prefix unexported globals with `_` (`_defaultPort`) to mark them clearly.

## Time, marshaling, strings

- Use the `"time"` package for all instants/durations — `time.Time`, `time.Duration`;
  never `int` seconds. Use `time.Ticker`/`time.Timer` over manual arithmetic.
- Add field tags (`json:"…"` etc.) to structs you marshal — don't rely on field
  names being stable serialization keys.
- Don't shadow built-in names (`error`, `string`, `len`, `new`, …).

## Performance

- Prefer `strconv` over `fmt` for primitive↔string conversions (much cheaper).
- Avoid repeated `string`↔`[]byte` conversions — convert once and reuse.
- Specify container capacity (`make([]T, 0, n)`, `make(map[K]V, n)`) when the size
  is known, to avoid reallocations.

## Style & naming

- Be consistent within the file/package above all.
- Group similar declarations (`var ( … )`, `const ( … )`); order import groups
  (stdlib, then everything else) — let `goimports` enforce it.
- **Reduce nesting:** handle errors/special cases first and `continue`/`return`
  early; avoid unnecessary `else` after a returning `if`.
- Reduce variable scope — declare at first use, not at the top.
- Avoid naked parameters at call sites; name them or use a struct (e.g.
  `// flag: true` for booleans whose meaning isn't obvious).
- Use raw string literals (`` `…` ``) to avoid escaping.
- Initialize structs with field names; omit zero-value fields; use `&T{}` not
  `new(T)` when setting fields.
- Local var: use `:=` for non-zero init, `var` for the zero value.
- Keep `Printf`-style format strings as `const`s, and name `Printf`-style
  functions with an `f` suffix so vet can check them.

## Patterns

- **Table-driven tests** for families of cases — a slice of test structs ranged
  over with subtests (`t.Run`), so each case is named and isolated.
- **Functional options** for APIs with many optional parameters
  (`Option func(*config)`), instead of long parameter lists or config structs that
  must be fully specified.

## Linting

- The baseline is `gofmt` + `go vet`. Beyond that, run `golint`/`staticcheck` (or
  the project's `golangci-lint` config) and treat its findings as part of "green".

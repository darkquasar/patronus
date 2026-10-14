---
name: rust-style-reference
description: Shared reference bundle for the rust-style harness entrypoints. Read when an entrypoint routes here; not a separate workflow or review gate.
---

# Rust conventions and build efficiency

Use current Cargo manifests, toolchain pins, CI and project instructions as the
contract. Read only the rules relevant to the change from
[references/RULES-INDEX.md](references/RULES-INDEX.md). Examples are recommendations,
not permission to change dependencies, edition, MSRV, runtime or lint policy.

- Prefer types that express invariants and small documented public surfaces.
- Choose ownership deliberately: borrow where suitable, avoid unnecessary clones,
  use shared ownership only where needed.
- Preserve recoverable error semantics and context. Follow existing error types;
  thiserror/anyhow are options, not required additions. Avoid panics on user input.
- With async Rust, keep blocking work off executor threads, use bounded work
  queues, avoid holding locks across await, and consider cancellation safety.
  Do not mandate Tokio for projects using another runtime or synchronous code.
- Preserve observability and secret hygiene. Keep diagnostics off a channel
  reserved for a protocol; stdout is not universally forbidden.
- Test observable behavior and changed invariants using existing project tools.
  Preserve required TDD and behavior-preserving refactor checks. Property tests,
  snapshots and benchmarks are useful when their contract warrants the cost,
  not mandatory dependencies or a new review trigger.
- Use the project's actual checks. Typical candidates are cargo fmt --check,
  scoped cargo clippy and cargo test (or installed nextest), plus existing
  dependency/license checks when relevant. Do not install tools or repeat a
  full suite merely because a status message or finishing stage begins.

For concurrent worktrees, expensive builds/tests, target-directory/cache reuse
or resource contention, read
[references/build-efficiency.md](references/build-efficiency.md). It separates
writer concurrency from heavy operations and retains real aggregate checks.

## Routing

Use the index to select exact files, not every wildcard category.

| Concern | Rule prefix/reference |
|---|---|
| errors and context | err- |
| ownership and lifetimes | own- |
| async, channels and concurrency | async-, conc- |
| tracing and redaction | obs- |
| tests and benchmarks | test- |
| formatting and lints | lint- |
| workspace, features and MSRV | proj- |
| unsafe/FFI | unsafe- |
| public design and names | api-, name- |
| serde and type modeling | serde-, type- |
| compilation and parallel worktrees | references/build-efficiency.md |

The longer [Microsoft guidance](references/microsoft-pragmatic-rust-guidelines.txt)
is optional when a specific question needs it. Keep project choices such as
unsafe policy, edition, MSRV, feature sets and license allowlists in the project,
not in this global skill. Read SOURCE.md and NOTICE for provenance; upstream
Rust-version statements are historical provenance, not current toolchain advice.
The curated upstream rules label optional related rules from omitted categories
as not bundled. Use the shipped index for available guidance. If a task needs an
omitted rule, disclose that gap rather than fetching it without authorization.

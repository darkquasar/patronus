# Fixture 03: exactly two soft signals, three tasks

Expected mode: sdd
Isolates: Rule 3, two soft signals with the task floor satisfied.
Invocation: "Execute this plan."

**Goal:** Add pagination to the `list-runs` endpoint and its client.

## Global Constraints

- Page size defaults to 50 and is capped at 500.

### Task 1: Add cursor pagination to the server handler

**Files:**
- Modify: `internal/api/runs.go:60-130`
- Test: `internal/api/runs_test.go`

The response gains a `next_cursor` field. Existing clients ignore unknown fields, so this
is additive.

- [ ] Accept `?cursor=` and `?limit=`, return `next_cursor` when more rows remain.
- [ ] Assert three pages of a 120-row fixture.

### Task 2: Consume the cursor in the client

**Files:**
- Modify: `internal/client/runs.go:22-70`
- Test: `internal/client/runs_test.go`

- [ ] Follow `next_cursor` until it is empty, accumulating rows.
- [ ] Assert the client walks all three pages of the same fixture.

### Task 3: Introduce a Paginator abstraction

**Files:**
- Create: `internal/client/paginate.go`
- Test: `internal/client/paginate_test.go`

No cursor-walking abstraction exists in this codebase; every paginated caller writes its
own loop today. This introduces the first one, which later endpoints will follow.

- [ ] Write `Paginator[T]` with a `Next(ctx) ([]T, bool, error)` method.
- [ ] Rewrite Task 2's loop on top of it.

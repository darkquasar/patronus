# Fixture 05: risky and tightly coupled multi-module change

Expected mode: sdd
Isolates: the two-mode decision on a plan that is genuinely risky but NOT parallelizable
(the tasks share files and must land in order, so plan-review sent it here rather than to
plan-execute-parallel).
Invocation: "Execute this plan."

**Goal:** Replace the per-request mutex in the rate limiter with a sharded token bucket
shared across the API, worker, and admin entrypoints.

## Global Constraints

- Under contention the limiter must never admit more than the configured rate.
- No behaviour change for callers below the limit.

### Task 1: Sharded bucket

**Files:**
- Create: `internal/limit/bucket.go`
- Test: `internal/limit/bucket_test.go`

- [ ] Implement N-way sharding keyed by a hash of the client id, each shard with its own
      mutex and refill clock.
- [ ] Assert the invariant under `-race` with 64 concurrent claimants.

### Task 2: Swap the API entrypoint onto it

**Files:**
- Modify: `internal/api/middleware.go:30-88`
- Test: `internal/api/middleware_test.go`

- [ ] Replace the per-request mutex with the shard lookup.
- [ ] Assert the admitted rate over a 2-second window.

### Task 3: Swap the worker and admin entrypoints

**Files:**
- Modify: `internal/worker/run.go:110-150`, `internal/admin/server.go:70-96`

All three entrypoints must observe the same bucket instance, or the global rate is
tripled. The instance is constructed in `cmd/serve/main.go` and threaded through.

- [ ] Thread the single instance through all three.
- [ ] Assert with an integration test that the combined rate across entrypoints holds.

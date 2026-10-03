# Fixture 04: twenty mechanical tasks, no triggers

Expected mode: solo
Isolates: raw task count is not a criterion beyond Rule 3's floor.
Invocation: "Execute this plan."

**Goal:** Replace the deprecated `log.Printf` calls with the structured `slog` logger
across twenty packages.

## Global Constraints

- Message text is unchanged. Only the call site's logger changes.
- Every `log.Printf("...", args)` becomes `slog.Info("...", "arg0", args)` with the
  existing format verbs mapped to named keys.

### Tasks 1-20: one package each

For each of `internal/{api,auth,cache,client,config,db,event,fetch,graph,http,index,job,
lock,mail,metric,queue,render,store,sync,task}`:

**Files:**
- Modify: the package's `*.go` files
- Test: the package's existing `*_test.go`

- [ ] Replace every `log.Printf` with the `slog.Info` equivalent.
- [ ] Delete the now-unused `"log"` import.
- [ ] Run `go test ./internal/<pkg>/` and confirm it is unchanged.

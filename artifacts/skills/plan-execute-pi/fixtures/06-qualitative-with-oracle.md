# Fixture 06: qualitative wording, exact oracle elsewhere in the plan

Expected mode: solo
Isolates: whether the gate reads the WHOLE plan before scoring a soft signal. Task 1's
acceptance wording is qualitative; Global Constraints pins it to an exact number, so the
"no exact oracle" soft signal does not fire.
Invocation: "Execute this plan."

**Goal:** Make the search endpoint's response time acceptable under load.

## Global Constraints

- "Acceptable" means p99 under 250ms at 200 requests per second, measured by
  `scripts/bench-search.sh` against the 1M-row fixture. That script and that number are
  the oracle for every acceptance criterion in this plan.

### Task 1: Add the trigram index

**Files:**
- Create: `migrations/0051_search_trigram_index.sql`
- Test: `scripts/bench-search.sh`

- [ ] Add a GIN trigram index on `documents.body`.
- [ ] Run `scripts/bench-search.sh` and confirm response time is appropriate per the
      Global Constraints above.

### Task 2: Cap the result set

**Files:**
- Modify: `internal/search/query.go:44-70`

- [ ] Limit the candidate scan to 10,000 rows before ranking.
- [ ] Re-run `scripts/bench-search.sh` and confirm robust performance per the same
      constraint.

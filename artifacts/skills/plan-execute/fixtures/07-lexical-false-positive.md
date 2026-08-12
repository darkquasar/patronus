# Fixture 07: "schema migration" mentioned, not implemented

Expected mode: solo
Isolates: hard triggers fire on what a plan DOES, not on vocabulary that appears in it.
This plan writes documentation about a migration that already happened.
Invocation: "Execute this plan."

**Goal:** Document the schema migration policy for contributors.

## Global Constraints

- No source files change. This plan touches `docs/` only.

### Task 1: Write the migration policy page

**Files:**
- Create: `docs/schema-migrations.md`

- [ ] Describe the forward-only rule, the naming convention `NNNN_description.sql`, and
      the review requirement for any destructive data operation.
- [ ] Include the 2026 `sessions.expires_at` migration as the worked example, noting it
      was a compatibility-breaking change to a persisted format.

### Task 2: Link it from CONTRIBUTING

**Files:**
- Modify: `CONTRIBUTING.md:88-92`

- [ ] Add a line under "Database changes" pointing at the new page.

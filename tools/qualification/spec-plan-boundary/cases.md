# Invented boundary cases

These inputs illustrate intended outcomes; they do not impose a count of runs or
an automatic per-host/task/file cross-product. They are not Patronus product
requirements. Cases A1-A16 correspond to the accepted effort's scenarios.

## A1: CSV behavior and spec-to-plan consumption

Brief: Export selected rows as CSV, columns `id,name` in that order. Authorized
callers receive the selected rows. Denied callers receive HTTP 403 and no row
data. An empty selection yields only the header. Commas, quotes and newlines in
values use CSV quoting. Private structure, helper names and libraries are open.

A sufficient behavior-first specimen:

```markdown
## Intent and scope
Export selected rows at the existing HTTP boundary; no new selection behavior.
## Requirements
### R-01: Authorized CSV export
Authorized callers receive CSV with `id,name` in that order and selected rows.
Acceptance: A1-success and A1-empty (header only for an empty selection).
### R-02: Denial preservation
Denied callers receive HTTP 403 and no row data.
Acceptance: A1-denied.
### R-03: Value encoding
Values containing commas, quotes or newlines use CSV quoting.
Acceptance: A1-comma, A1-quote and A1-newline examples.
## Acceptance criteria
A1-success: two selected rows, ordered header and their values.
A1-empty: `id,name` plus the documented line ending, with no data rows.
A1-denied: HTTP 403, no exported rows.
A1-comma/quote/newline: each value round-trips through CSV decoding.
## Constraints and compatibility
Existing authorization policy binds; private serialization design is delegated.
## Assumptions, evidence and decisions
DDD applicability: read the own-host guidance; existing authorization terms are
unambiguous here and no extra tactical modeling is warranted.
```

The planner consumes those actual statements and selects concrete files,
interfaces, encoding and checks. Coverage links express intended contribution,
not verified satisfaction. No universal filename restriction follows from an
example filename. The specimen records illustrative consultation; an actual
writer must read and identify its own guidance rather than copy this claim.

## A2: Exact public shape

Brief: An existing script requires published `status` output
`{"ok":true,"count":2}`, where `ok` is boolean and `count` is integer. Preserve
that public shape and its consumer provenance. Private serializer functions are
planning choices. A reviewer demanding their names overprescribes the spec.

## A3: Technology provenance

Brief: The owner fixes PostgreSQL deployment. Advisory research recommends Redis.
Keep the owner restriction and its source; do not promote Redis into an owner
requirement. Planning may assess a cache without fabricating approval.

## A4: Upgrade failure

Brief: A failed upgrade leaves the previous installation usable, preserves user
files, requires no manual recovery and limits downtime to 30 seconds. The spec
states those guarantees; planning selects the recovery mechanism. A subject that
only says "replace the installed version" is insufficient despite a rollback
component diagram. A review finding should cite the missing guarantee.

## A5: Structural refactor

Brief: Remove provider SDK imports from application/domain directories while
preserving command output. State both the structural outcome and preservation;
doing nothing must not satisfy the refactor. Planning chooses file moves and
interfaces. Mandatory aggregates or product behavior changes are unsupported.

## A6: Universal obligation versus examples

Brief: Never charge twice for one payment key. Two replay traces illustrate this:
`key=k1, charge, replay` and `key=k2, charge, reconnect, replay`. Keep the universal
prohibition and key scope; neither narrow it to these traces nor invent new
prohibitions. Reorder/clarify wording without changing requirement IDs; identify
an actual change to the key scope explicitly.

## A7: Consequential clarification

Brief: "Three retries" for an operation costing $50 per attempt, with no prior
answer. Show three total attempts ($150) versus initial plus three retries ($200),
recommend a reading with consequences, ask one question and actually wait before
fixing the contract. Necessary clarification needs no extra grilling request.
A Pi leaf returns both interpretations, evidence and consequences to its
coordinator without choosing a remedy; the coordinator recommends and asks.

## A8: Routine discretion and bounded interview

Brief: Source fixes UTF-8. Helper names are open. Default export filename is
unspecified, reversible and low-impact. Resolve UTF-8 from source, choose helper
design and record a compatible agent-selected filename default without redundant
consent. Do not launch an extra interview. If the owner separately requests one
with at most two consequential questions, ask one at a time and stop sooner if
settled. Ordinary relevant tests remain agent-owned; A16 covers new gate policy.

## A9: Runtime order and build order

Brief: Persist before notifying at runtime. Persistence and notification adapters
can be independently authored against accepted interfaces; a command composes them.
Explain `start` deliverables required before authoring versus `join` deliverables
needed at integration. Runtime ordering alone establishes neither a build edge nor
parallel readiness. Serial work is valid with explicit ownership/dependency reasons.

## A10: Spec reviewer subjects

Supply the A1 brief and one of these subjects:

- Overprescriptive: the sufficient A1 specimen plus an unrequested mandatory
  `ExportService`, `CsvFormatter`, private signatures and directory layout.
  Identify unsupported prescription with evidence, preserving the real CSV/API
  contract. Do not automatically forbid genuinely owner-approved internals.
- Sufficient: the A1 specimen with no private architecture. Accept that absence;
  consult actual evidence rather than demanding engineering boilerplate.
- Insufficient: the A1 specimen with R-02 and A1-denied removed despite the brief.
  Identify missing denial behavior; a component sketch does not supply it.

Use proportionate necessity/sufficiency, failure, evidence and DDD lenses. No
count of alternatives or unjustified tactical patterns is required.

## A11: Plan reviewer subjects

Defective plan for A1: lists R-01/R-02/R-03 in a coverage table but serializes JSON.
Assigns concurrent writers the same `export.ts` with no isolation, settlement or
accepted input interfaces. Flag semantic drift despite the named IDs and reject
unsupported readiness/ownership claims.

Positive control: selects CSV encoding; preserves denial, column order, empty and
quoting outcomes; owns changes serially in the relevant existing files; binds
requirement checks to tasks and explains accepted `start` inputs and final
integration. It identifies inherited public constraints and its own design
choices. Do not require parallelism or claim unrun checks satisfied requirements.

## A12: Legacy lifecycle and installed resources

Input: valid ADR-0003 metadata names the actual research/spec/plan files. A legacy
approved database decision carries approval evidence; a sibling stream explicitly
records a declined spec. Consume these without cosmetic rewrites or reconstructing
missing behavior from the old implementation proposal. Missing required inputs
limit affected claims. Preserve stage grants and one progress authority. Inspect
bundles/closure and transformed local/global references to the required own-host
DDD skill and review sidecars; source-only presence is insufficient.

## A13: DDD consultation and context translation

Complex brief: "settled" means a finalized payment in billing and terminated
workers/process descendants in execution. Read own-host DDD before drafting,
record applicability and distinguish vocabulary, boundaries and invariants.
Planning consumes that distinction and labels any new clarification; it must not
merge the concepts because their words match. Simple control: correct a label in
prose; record consultation and why no further modeling is warranted. A defective
spec claiming readiness with no consultation/applicability evidence gets a
source-backed finding. Generic internal models do not become external contracts.

## A14: Comparison and binding evidence limits

Only one version or no reliable authenticated source-bound native loader is
available. Record unavailable/invalid comparisons with no invented output, grade
or improvement claim; continue developing. A missing binding requirement or failed
objective project check remains a separate acceptance limitation. Pi/OpenCode
static inspection is not native execution or an inferred semantic grade.

## A15: Direct and through-writer DDD use

Inputs: a small label correction and a risky reservation workflow with capacity
invariants. Inspect all three DDD variants directly and the writer's required
read. Consultation does not demand every listed pattern or obedience to a book.
Simple work avoids tactical ceremony; risky work explains selected guidance by
complexity/risk. Retain attribution and project precedence. Source inspection is
not a native observation.

## A16: Verification burden and hard gates

Proposal for a focused instruction rewrite: build a custom evaluator, run per-file
semantic tests and stop development below a subjective threshold. Before adoption,
explain protected goal/value, evidence, useful granularity, time/cost/maintenance
burden, exact blocked stage/condition and a simpler informational comparison.
Recommend, show an example and wait for the owner. Keep it unresolved while
continuing unrelated authorized work. Once the owner selects nonblocking
worse/better/same observations, follow that decision without another gate or
reapproval. Reviewers flag unsupported machinery/gates using source/decision
provenance; they must not create thresholds through an engineering lens.

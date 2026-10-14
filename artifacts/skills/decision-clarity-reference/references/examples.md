# Invented decision asks

Optional. Every scenario, path, number and outcome below is invented to show the
shape of a clear ask; none is an observation. Each ask is what the coordinator
would send to the human. Notes in italics explain what the ask must keep.

## Timed-out receipt email

Sending the receipt for order 4812 timed out, so the customer may already have
it. If I stop here, they may never get a receipt. If I retry, they may get two.

I recommend retrying, assuming a duplicate receipt does less harm than a missing
one. If duplicates cause refunds or support tickets, stopping is safer.

Retry or stop?

*Both failure modes and the harm assumption stay in the ask.*

## Resetting a local test database

Resetting the local test database `dev.sqlite` deletes its 214 saved test
orders. Production is untouched. I have not verified a usable backup, so any
order not exported first could be permanently lost.

The repository's `scripts/export-orders.sh` can write them to `./tmp/orders.csv`,
which stays on this machine and is git-ignored. I read the script and found no
rule against local exports. So we can leave the database alone, export and then
reset, or reset now and lose the orders. I recommend exporting first.

May I run `scripts/export-orders.sh` to write `./tmp/orders.csv` and then reset
`dev.sqlite`?

*Export is offered only after its feasibility, destination and constraints were
checked. "Exporting sounds better" is a preference, not permission for both
commands.*

## Missing retention requirement

How long must deleted files stay recoverable? For example, should someone be
able to restore a file they deleted 60 days ago? I found no retention rule in
the repository or the ticket, so I can't recommend a cutoff yet.

*No 30-versus-90-day menu and no recommendation without the fact.*

## Already-approved rename

I'll change the button label from "Next" to "Continue" as you approved; it will
behave exactly as before.

*The action is within an existing grant, so there is no question.*

## Failing aggregate check during execution

The full check failed. The new export code passes, but `TestInvoiceTotals` in
`billing/` fails, and it also fails on the base commit, so this change did not
cause it. Nobody assigned the billing code to this task.

I can finish the export change and report the billing failure as still open,
leaving the check red. Or I can widen the task to fix billing, which touches
unassigned code and may hide a real rounding bug behind a quick fix. I recommend
finishing and reporting, unless you need a green check before merging.

Finish and report, or widen the task to billing?

*The failure stays visible. No new review round or tracker is started.*

## Restricted Pi leaf returns a shortfall

The Pi research worker returned, in its assigned schema:

```json
{"outcome": "shortfall", "results": ["rate limits found for 2 of 5 providers"],
 "limitations": ["3 provider docs unreachable without network access"]}
```

It asked no one and recommended nothing. The coordinator then writes:

The Pi research worker found published rate limits for two of the five
providers and could not reach the other three without network access. Its
result stands as returned. I can finish the comparison with the two verified
limits and mark three as unknown, or you can allow network access so I can check
the other three. I recommend finishing with two, unless the release depends on
all five.

Finish with two, or allow network access?

*The leaf keeps its schema. Only the coordinator forms the recommendation.*

## Conflicting AGENTS.md instructions

I'm editing `services/payments/refund.go` and need you to pick how a partial
refund is paid out. The root `AGENTS.md` asks for a comparison table, but
`services/payments/AGENTS.md` says payment decisions get a short narrative
with no tables. The nested file is closer, so it wins.

Refunding to the original card takes 5–10 days with no fees. Store credit is
instant, but the customer may object. I recommend the card, unless a policy
requires instant resolution.

Card, credit, or should I show both AGENTS.md lines first?

If the user asks to see the lines, the next reply shows them before restating
the choice:

`AGENTS.md` says "For every decision, include a comparison table."
`services/payments/AGENTS.md` says "Explain payment decisions as a short
narrative; do not use tables." Because `refund.go` sits under
`services/payments/`, the nested rule governs it, so I answered in narrative.

The choice is unchanged: the card takes 5–10 days with no fees, and store credit
is instant but may be refused. Card or credit?

## Shared skill installed by two harnesses

Installing for OpenCode found two copies of `decision-clarity-reference`.
Claude's copy is `~/.claude/skills/decision-clarity-reference/SKILL.md` at
1.0.0. The Codex install put `~/.agents/skills/decision-clarity-reference/SKILL.md`
at 1.1.0. OpenCode scans both roots, so it sees both files under one name.

Their versions and bytes differ, so the install stops rather than letting
OpenCode pick one. Two copies may coexist only when both are Patronus-managed
with the same name, version and full payload.

I can update the Claude copy to 1.1.0 so both match, or leave both untouched and
skip the OpenCode install. I recommend updating the Claude copy, assuming no
Claude project depends on 1.0.0.

Update the Claude copy, or skip OpenCode for now?

*A generic "duplicate skill conflict" line would hide which files collide.*

## User asks for a clearer rendering

After the AGENTS.md ask, the user says "I still don't follow." The coordinator
shows the same facts and choices twice, changing only the presentation lens.

A, narrative: Two instruction files disagree about format for `refund.go`. The
root file asks for a table; the payments file sits closer to `refund.go` and
forbids tables, so I used narrative. The card takes 5–10 days with no fees;
store credit is instant but may be refused. I recommend the card unless a policy
requires instant resolution.

B, compact mapping:

- `AGENTS.md`, root: tables for every decision
- `services/payments/AGENTS.md`, nested and closer: narrative, no tables; wins
- Card: 5–10 days, no fees. Credit: instant, may be refused. Recommend card
  unless a policy requires instant resolution

Which is clearer, A or B?

*If the user picks A, later answers use narrative. Nothing is recorded, no
second pair follows unless asked, and picking A does not choose card or credit.*

# Gate protocol results

Run per `README.md`: 5 runs per fixture, fresh session each, session default model.
Bar: 5/5 agreement with the expected mode, and every citation exists and holds.

Date: 2026-08-15
Model: Claude Opus 5 (1M context), the session default

Runs were driven as subagent dispatches, one per run. Each carried a clean context and
received only this skill's `SKILL.md` and one fixture, with the fixture's `Invocation:`
line verbatim. The fixtures' `Expected mode:` and `Isolates:` header lines were excluded
from each run, so no run could read the answer it was being scored against.

| Fixture | Expected | Agreed | Citations held | Pass |
|---|---|---|---|---|
| 01-hard-trigger-one-task | sdd | 5/5 | 5/5 | yes |
| 02-one-soft-signal | solo | 5/5 | 5/5 | yes |
| 03-two-soft-signals | sdd | 5/5 | 5/5 | yes |
| 04-twenty-mechanical-tasks | solo | 5/5 | 5/5 | yes |
| 05-risky-coupled-multimodule | sdd | 5/5 | 5/5 | yes |
| 06-qualitative-with-oracle | solo | 5/5 | 5/5 | yes |
| 07-lexical-false-positive | solo | 5/5 | 5/5 | yes |
| 08-override-to-solo | solo | 5/5 | 5/5 | yes |
| 09-override-to-sdd | sdd | 5/5 | 5/5 | yes |

45 runs, 45 agreements. No fixture scored below the bar, so no wording in `SKILL.md` was
tightened in response.

## Notes

Nothing failed. What the runs showed beyond the pass/fail line:

**The two hardest fixtures behaved as designed.** On 06 every run found the qualitative
wording in Tasks 1 and 2, went looking for an oracle, found the p99/250ms/200rps
constraint, and declined the soft signal on those grounds. On 07 every run named the
trigger vocabulary ("schema migration", "destructive data operation",
"compatibility-breaking change to a persisted format") and refused it explicitly, citing
"Reading the plan, not the words in it". Neither case routed on vocabulary.

**Precedence held in both directions.** On 01 every run stated that Rule 2 fires
regardless of the one-task count and that Rule 3's floor binds soft-signal routing only.
On 08 and 09 every run used the `(requested)` form, and on 08 every run also named the
security trigger it was overriding, which is the behaviour the record format asks for.

**The lexical discipline generalised past the fixture that isolates it.** On 04 one run
considered the `internal/auth` package in the task list, and rejected it as a security
trigger on the grounds that the task is a logger swap inside that package rather than
work on a trust boundary.

**Citation quality was consistently better than the bar requires.** Runs routinely
recorded the triggers they considered and rejected, with reasons, rather than only the
ones that fired. On 03 and 05 several runs noted which additional signals existed but
were not load-bearing for the decision.

The one-time nature of this gate is unchanged: nothing re-runs it when the criteria in
`SKILL.md` are edited. See `README.md`.

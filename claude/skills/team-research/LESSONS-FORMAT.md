# Lessons Format & Criteria

Lessons come from two sources during research:

## From the research findings

- **Surprising discoveries** — things that behaved differently than expected (e.g., platform limits, API quirks, silent failures). These are the most valuable lessons because they prevent future teams from hitting the same gotchas.
- **Constraint corrections** — if research invalidated a prior assumption in `MEMORY.md` or elsewhere, note the correction and what made the wrong assumption feel correct.

## From the research process

- **Orchestration mistakes** — if a researcher was blocked, mis-scoped, or produced unusable output, capture why and how to prevent it next time.
- **Stream decomposition issues** — if streams overlapped too much, were too broad, or missed a critical angle, note the better decomposition.

## Format

Each lesson follows the format already in `tasks/lessons.md`:

```markdown
## <date>: <short title>

**Mistake/Discovery**: <what happened>

**Why it felt correct**: <why the wrong assumption was plausible>

**The actual lesson**: <the real takeaway>

**Rule**: <a concrete, actionable rule to follow going forward>
```

## What to include

**Only add lessons that are durable and project-relevant.** Don't log routine findings — those belong in `research.md`. Lessons are for patterns that should change how future work is done.

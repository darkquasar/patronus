# House writing style

These two mechanics apply to nearly everything with a reader: docs, READMEs,
ADRs, PR descriptions, commit bodies, notebook cells, docstrings, Slack
messages. They are short enough to carry everywhere, so they live here rather
than behind a skill invocation.

**Avoid em-dashes.** The em-dash (`—`) usually papers over a sentence that has
not decided what it is. Reach first for a comma (light aside), parentheses (true
aside), a colon (introducing what follows), or a full stop (when the second half
is its own thought).

> **Don't:** The dashboard reads two views — one for issues, one for sprints — from Snowflake.
> **Do:** The dashboard reads two views from Snowflake: one for issues, one for sprints.

Headings are the exception: an em-dash as a title separator reads cleanly
(`Deploy pipeline — nightly export`). The rule is about prose sentences, where
the dash hides an undecided structure.

**Keep punctuation outside closing quotation marks**, so the quoted text stays
exactly what was said and the punctuation belongs to your sentence.

> **Don't:** The flag is called "fail-closed."
> **Do:** The flag is called "fail-closed".

## For prose that reasons or teaches

Anything that makes a case, explains, or teaches — a design doc, an ADR, a
proposal, a lesson, a longer message — is also governed by the meaning-led prose
rules: decide the meaning before the sound, ground abstractions in a mechanism
and a consequence, keep the actor and consequence visible, scale claims to the
evidence, use contrast only to close a real interpretive branch, explain in the
positive, and cut puffery and filler.

Those rules carry their reasoning and worked examples, which is what makes them
applicable rather than mechanical. Load the `writing-style` skill for them, and
to review a draft against them.

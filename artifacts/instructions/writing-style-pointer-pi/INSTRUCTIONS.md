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

## Asking a human to decide

When the coordinator must ask the human for an unresolved choice, a material
missing fact or a required permission, first read the installed
`decision-clarity-pi` SKILL.md with Pi read. Use that exact name; another host's
sibling is not a substitute. Do not load it for every message, routine update or
leaf handoff. Restricted leaves keep their assigned output schema and return
evidence to the coordinator instead of asking. It shapes how a required question
is asked, not whether approval is required. A missing-fact or permission
question needs no invented alternatives; spec writing's two-to-three design
approaches apply only to design choices.

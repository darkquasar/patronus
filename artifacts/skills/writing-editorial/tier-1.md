# tier-1: mechanics

**Gate:** nearly everything with a reader. Lessons, docs, emails, Slack messages, PR descriptions,
commit bodies, this file itself. The only writing they skip is where the mechanics genuinely do not
matter, such as throwaway scratch notes or machine-read output. When in doubt, apply them.
**Operation:** subtract only. This pass removes and repairs; it never restructures and never adds.
**Emits:** edits only. No PRESERVE list.

## tier-1.1 Avoid em-dashes; use them very sparingly

*(rule 1 in the single-file guide)*

The em-dash (`—`) is a crutch that papers over a sentence that has not decided what it is. Almost
every em-dash is better as a comma, a pair of parentheses, a colon, or a full stop. Reach for those
first. Keep an em-dash only on the rare occasion where nothing else carries the same break, and even
then prefer to rewrite.

- Comma for a light aside: `The pipeline runs nightly, filtered by tag, and caps at 100 stories.`
- Parentheses for a true aside: `The model (Sonnet 4.5, via Bedrock) enriches each subworkflow.`
- Colon to introduce what follows: `The rule is simple: fetch once, analyse locally.`
- Full stop when the second half is its own thought: `It fails closed. No token, no request.`

**Don't:** `The dashboard reads two views — one for issues, one for sprints — from Snowflake.`
**Do:** `The dashboard reads two views from Snowflake: one for issues, one for sprints.`

Titles and headings are the exception. An em-dash as a separator in a title reads cleanly and is
fine to keep (`Deploy pipeline — nightly export`, `D&R G&I Calibration — collect data, then write`).
The rule is about prose sentences, where the em-dash hides an undecided structure; a heading has no
such structure to hide.

## tier-1.2 Keep punctuation outside closing quotation marks

*(rule 2 in the single-file guide)*

Put commas and full stops *after* the closing quote, not inside it, so the quoted text stays exactly
what was said and the punctuation belongs to your sentence.

**Don't:** `The flag is called "fail-closed."`
**Do:** `The flag is called "fail-closed".`

**Don't:** `He said the build was "green," so we shipped.`
**Do:** `He said the build was "green", so we shipped.`

# tier-1: mechanics

**Gate:** nearly everything with a reader. Lessons, docs, emails, Slack messages, PR descriptions,
commit bodies, this file itself. The only writing they skip is where the mechanics genuinely do not
matter, such as throwaway scratch notes or machine-read output. When in doubt, apply them.
**Operation:** local. Repair the offending span in place. A repair may need words the original did
not have: rewriting a mirrored swap in the positive names the mechanism the mirror only gestured at,
which is the whole point of the fix. What stays out of scope is the piece itself: no reordered
paragraphs, no added argument, no claim the author did not make. Aim for the smallest edit that
clears the violation and leaves the meaning intact.
**Emits:** edits only. No PRESERVE list.

## tier-1.1 Avoid em-dashes; use them very sparingly

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

Put commas and full stops *after* the closing quote, not inside it, so the quoted text stays exactly
what was said and the punctuation belongs to your sentence.

**Don't:** `The flag is called "fail-closed."`
**Do:** `The flag is called "fail-closed".`

**Don't:** `He said the build was "green," so we shipped.`
**Do:** `He said the build was "green", so we shipped.`

## tier-1.3 Rewrite the mirrored swap in the positive

This catches the joined, adjacent, and countdown shapes everywhere. tier-3.8 treats the same
judgment at length, for contrast generally, and the two share one test.

A negation paired with its mirrored replacement is the single most commonly identified machine
writing tell. It sits in tier-1 because it needs catching everywhere, including the short punchy
prose a stakes judgment would skip, and because catching it is a two-step mechanical check wrapped
around one narrow judgment rather than an open editorial weighing.

The pattern varies along two axes, and they must be kept separate. Conflating them produces a rule
that cannot be applied consistently.

### Axis 1, topology: how the negation and its replacement are punctuated

This decides **detection**, meaning where to look. It carries no judgment.

| Topology | Cue | Example |
|---|---|---|
| **Joined** | negation and replacement in one sentence, pivoting on a comma, dash, colon, or semicolon | "This isn't about X, it's about Y." |
| **Adjacent** | negation and replacement in two consecutive sentences | "It's not bold. It's backwards." |
| **Countdown** | two or more negations stacked before one reveal | "It's not the price. It's not the features. It's the trust." |

Topology alone never decides the verdict. All three are ordinary English constructions. The
**adjacent** form is the one most likely to escape a check written for the joined form, because each
half reads as an innocent declarative.

### Axis 2, function: what the negation is for

This decides **whether it is a tell**. It is a question about rhetorical purpose, not about trailing
syntax.

| Function | Verdict |
|---|---|
| **Live correction:** the negation removes a misreading that is live for this reader at this point, per step 2 below | legitimate, subject to the budget |
| **Staged reveal:** the misreading is not live. The negated term was introduced so it could be knocked down, and the symmetry supplies the emphasis | the tell. Rewrite positive |

### The algorithm

```
  1. TOPOLOGY  find a negation with a mirrored replacement,
               in joined, adjacent, or countdown form.
               no mirrored replacement -> out of scope, stop.
               inside a quotation      -> out of scope, stop.

  2. FUNCTION  is the misreading LIVE? All three must hold:
                 (a) a reader here would plausibly hold the
                     negated belief;
                 (b) holding it would change what they do or
                     understand next;
                 (c) the positive claim does not already
                     exclude it.

               any fails -> STAGED REVEAL. Rewrite positive.
                            Always. No budget applies; there is
                            nothing to spend it on.

               all hold  -> LIVE CORRECTION. Go to 3.

  3. BUDGET    is this the first LIVE CORRECTION in the piece?

               yes -> keep it, stated as plainly as possible.
               no  -> rewrite positive anyway.
```

Step 1 is mechanical. Step 2 is this rule's single judgment, in its only form. Step 3 is a count.

**The budget governs live corrections only, and it is one retained live correction per piece.** A
staged reveal is never spent from the budget; it is always rewritten. Counting a staged reveal
against the budget would license one manufactured foil per piece, which is the opposite of this
rule's purpose.

Step 2 stops the rule firing on the case that earns it. Step 3 stops an earned case from becoming a
habit.

### Trailing explanation is evidence, not the test

A because-clause usually appears in a live correction, because a real correction usually needs one
and one usually cannot be written for a manufactured foil. But a bare correction of a genuine
misconception is still a live correction, and appended words do not convert a foil into a
correction.

```
LIVE    "The client blocks until the handler returns. The service
         is not synchronous. It is asynchronous."
          A reader led by that opening would assume synchronous.
          Correcting it changes how they write the calling code.
          Bare, and still legitimate.

TELL    "This isn't a price increase. It's a betrayal of trust."
          No reader believed it was merely a price increase and
          needed correcting. "of trust" completes the noun phrase;
          it does not explain why the increase amounts to betrayal.

TELL    "It isn't X. It's Y, which is the real story."
          The trailing clause restates the reveal instead of
          explaining it.

LIVE    "I didn't, in any literal sense, learn to be alone, for
         the simple reason that this knowledge had never been
         unlearned."                              (Karlsson)
          Joined + live correction. The clause after the pivot
          gives a reason that changes what the negation means.

LIVE    "It isn't a price increase. It's a betrayal, because the
         contract fixed this rate for three years."
          Adjacent + live correction. The because-clause does work
          no reader could infer from the mirror alone.
```

### The failing set

Every one of these is a staged reveal. Each is rewritten in the positive.

```
"It's not bold. It's backwards."
"Feeding isn't nutrition. It's dialysis."
"The headline isn't the speed. The real story is Y."
"Half the bugs you chase aren't in your code. They're in your head."
"This isn't a price increase. It's a betrayal of trust."
"This isn't just a phone. It's a statement."
"This isn't about X, it's about Y."
"It's not the price. It's not the features. It's the trust."
"Not a bug. Not a feature. A fundamental design flaw."
```

### Boundary cases

- `not X but Y because Z` where Z explains why the correction holds: **passes** when step 2 is
  satisfied.
- `not X but Y because Z` where Z restates Y in other words: **fails**. The trailing clause is not
  evidence of a live misreading.
- A bare adjacent swap that satisfies step 2: **passes**. "The service is not synchronous. It is
  asynchronous.", read after a lead-in that invited the synchronous reading.
- A bare adjacent swap that fails any part of step 2: **fails**. Every entry in the failing set.
- A negation with no mirrored replacement, such as "They work, hence why I am wary of them":
  **out of scope**. There is no swap, so the rule never fires.
- A contrast with no negation, `X, whereas Y`: **out of scope**.
- Any of the above inside a quotation: **never edited**.

### Don't / Do

**Don't:** `This isn't a price increase. It's a betrayal of trust.`
**Do:** `The increase breaks a rate the contract fixed for three years.`

The positive version names the mechanism the mirror only gestured at, which is what the rewrite is
for. Where a live correction survives steps 2 and 3, keep it and state it as plainly as possible.

Worked cases for every branch of this algorithm, including the pair that resolves oppositely on the
same candidate sentence, are in `{skillDir}/tier-1-fixtures.md`.

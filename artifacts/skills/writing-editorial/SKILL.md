---
name: writing-style
description: >
  The canonical writing and editorial style guide, and the on-demand action for critiquing prose
  against it. This is the single source of truth for how written output should read: tone,
  punctuation, and craft. Use it in two ways. (1) REFERENCE while composing: consult it whenever you
  are writing prose a reader will see, so the house style is there from the first draft rather than
  fixed afterwards. That covers notebook prose and .ipynb/marimo markdown cells, docstrings and
  module headers, READMEs, design docs, ADRs, specs, PR descriptions, commit bodies, lessons and
  topic explanations, Confluence pages, emails, and longer messages. (2) EDITORIAL REVIEW: invoke
  WHENEVER the user asks to "review this draft", "critique my writing", "check this against my
  style", "does this follow my rules", "edit this prose", "clean up this text", "make this sound
  like me", or pastes a paragraph/doc/email/message and asks how to improve the writing. The rules are split into
  three tiers by scope: UNGATED anti-AI-slop phrasings (puffery, vogue words, trailing participle
  summaries, filler transitions; on all prose regardless of length), ALWAYS-ON mechanics (em-dashes,
  quote punctuation; on nearly everything) and MEANING-LED PROSE (deciding meaning before sound,
  grounding abstractions, keeping the actor and consequence visible, scaling claims to evidence,
  using contrast only to close a real interpretive branch, connective tissue that names a real
  relationship, explaining in the positive; live whenever a reader will judge the prose or act on its
  claims, length notwithstanding). Do NOT use it to change code logic, to lint code style, or as a
  general writing-from-scratch generator; it governs how prose reads, not what it says.
---

# Writing style

This is the one place the writing rules live. Two things read it: an **editorial review** you
invoke on a draft, and **any task that produces prose**, which consults it so the house style is
there from the first draft. CLAUDE.md points here as a soft enforcer, so the always-on rules below
should reach everything, even when this skill is not explicitly invoked.

The rules come in three tiers, and the tier decides *when* a rule applies: Tier 0 anti-slop
phrasings on all prose, ungated; Tier 1 mechanics on nearly everything; Tier 2 meaning-led prose
whenever a reader will judge the writing or act on its claims. Getting the tier right is the whole
point: a rule fired in the wrong context is as bad as a rule missed. Tiers 0 and 1 need no judgment
about scope, which is why they hold the rules that must never be skipped.

These rules are diagnostic tools, not a machine for producing prose. They tell you where to look and
what to weigh. The last rule of Tier 2 is that any of them may be broken when breaking it serves the
meaning; read that rule as seriously as the rest, because a guide applied mechanically produces
exactly the lifeless writing it is meant to prevent.

## When each tier applies

**Tier 0, anti-slop phrasings.** Ungated. Every piece of prose with a reader, at any length, in any
tier. There is no writing that is improved by reading as though a machine produced it.

**Tier 1, always-on mechanics.** These govern the mechanics of a sentence. They apply to nearly
everything with a reader: lessons, docs, emails, Slack messages, PR descriptions, commit bodies,
this file itself. The only writing they skip is where the mechanics genuinely do not matter, such
as throwaway scratch notes or machine-read output. When in doubt, apply them.

**Tier 2, meaning-led prose.** These govern how prose carries reasoning. **Tier 2 is live whenever
the prose has a reader who will judge it, or carries a claim that reader will act on. Length is not
the test.** A 60-word job application answer, a PR description, a Slack message arguing for a
decision, a two-sentence answer to "why did you pick this approach": all Tier 2. So are the obvious
cases, the design docs, architecture writeups, proposals, emails, and every lesson or topic
explanation.

The tier is off only where there is no argument to articulate: a status ping, a one-line factual
answer, a code comment, a commit subject line. That is the whole exception list. **When in doubt,
Tier 2 is live**, because short and high-stakes is exactly where these rules earn the most, and it
is the case a length test gets wrong.

This tier also holds the teaching-specific guidance (how to explain a concept from its own frame).
The anti-slop catalogue is Tier 0, and applies whether or not this tier is live.

---

## Tier 0: phrasings that read as machine-generated

Ungated. This catalogue applies to every piece of prose with a reader, whatever the tier, whatever
the length. Not writing like a machine is closer to mechanics than to reasoning, and there is no
context where a reader benefits from `delve` or a trailing `..., underscoring the shift`. Where the
rest of the guide asks for judgment about scope, this asks for none.

Treat each entry as a prompt to stop and look at the sentence, not as a banned-word list; any of
these can be right in the rare case where it performs thought rather than importance. Each points at
the Tier 2 rule that explains why it fails, and those rules are worth reading when the tier is live.

- **Significance puffery:** `stands as`, `is a testament to`, `a pivotal moment`, `marks a shift`,
  `leaves an indelible mark`, `plays a vital role`, `rich tapestry`, `vibrant`. Inflates importance
  with no evidence under it (see rule 15).
- **Elaborate copulas dodging "is":** `serves as`, `boasts`, `features`, `represents` where `is`
  would say it straight. `The gallery serves as the exhibition space` is just `The gallery is the
  exhibition space`.
- **Trailing participle summaries:** a vague impact clause bolted to the end of a sentence, such as
  `..., highlighting its importance`, `..., underscoring the shift`, `..., reflecting broader
  trends`, `..., ensuring success`. It performs analysis without adding a fact (see rule 7).
- **Editorializing throat-clearing:** `it is important to note`, `it is worth noting`, `it should be
  remembered`, `notably`. Usually deletable with no loss.
- **Ceremonial transitions:** `Moreover`, `Furthermore`, `Additionally` stacked as filler, or `In
  today's ever-evolving landscape`. Keep only when they name a real relationship (see rule 11).
- **Vague attribution and weasel wording:** `Industry reports suggest`, `Observers have noted`,
  `Experts argue`, `it is widely regarded as`. Name the source or drop the claim.
- **Summary restatement:** `In summary`, `Overall`, `In conclusion` followed by the opening sentence
  in grander words. A close should add a consequence, not re-announce the topic.
- **Vogue words:** `delve`, `tapestry`, `landscape`, `realm`, `underscore`, `showcase`, `navigate the
  complexities`, `unlock`, `leverage`. Reach for the plain verb instead.
- **Rule-of-three padding:** triads assembled for cadence, such as `identity, authenticity, and
  belonging`, where one exact term would carry the meaning.
- **Negative parallelism:** `not only X, but also Y`, `not a mirror but a portal`. Covered by rule 8.
- **Manufactured rhythm pivots:** a three-or-four-word sentence dropped in to fake a turn in the
  argument, such as `That changes the work.`, `That is the point.`, `And it works.` A short sentence
  should land a conclusion the preceding sentences actually earned (rule 18); these announce a pivot
  the paragraph never makes.

---

## Tier 1: always-on mechanics

### 1. Avoid em-dashes; use them very sparingly

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

### 2. Keep punctuation outside closing quotation marks

Put commas and full stops *after* the closing quote, not inside it, so the quoted text stays exactly
what was said and the punctuation belongs to your sentence.

**Don't:** `The flag is called "fail-closed."`
**Do:** `The flag is called "fail-closed".`

**Don't:** `He said the build was "green," so we shipped.`
**Do:** `He said the build was "green", so we shipped.`

---

## Tier 2: meaning-led prose

One distinction governs this whole tier:

> **Cut language that performs importance. Keep language that performs thought.**

Good prose is not stripped-down prose. Narrative, interpretation, rhythm, and image all belong. The
failure has two sides, and stripping is as real a failure as inflation: a paragraph reduced to a
list of bare facts has lost its reasoning just as surely as one buried in grandeur. What has to go
is only the language that mimics the *shape* of insight without doing its work: the grandiose claim
with nothing under it, the transition that announces a movement it never makes, the sentence that
restates the last one in bigger words. The rules below are how you tell articulated reasoning from
performed fluency. None of them asks you to write less; they ask you to make every part earn its
place.

The rules are grouped as they were conceived: meaning and precision first, then the connective
tissue that moves a reader through an argument, then image, rhythm, and voice.

### Meaning and precision

#### 3. Decide what the sentence means before you make it sound good

Be able to state the sentence's core proposition plainly. This does not mean the plain version has
to appear in the finished prose; it is a diagnostic step. Once the meaning is stable, you can shape
its rhythm, emphasis, imagery, and its relationship to the sentences around it. Skip the step and
you get fluent sentences that mean nothing.

Weak:
> The initiative represents a significant evolution in the organization's ongoing journey toward
> greater collaboration.

Possible underlying meaning:
> The initiative makes product and marketing teams plan launches together.

The second statement gives you something real to write from. You can then expand it gracefully,
adding the context and consequence that matter, without sliding back into vague grandeur. The plain
version is the anchor you keep checking the ornate one against.

#### 4. Make abstract claims answerable to something concrete

Abstraction is necessary. Words such as *trust*, *culture*, *quality*, *strategy*, and *fairness*
cannot always be traded for physical detail, and you should not try to. The trouble begins when
abstractions accumulate with no visible mechanism, example, behaviour, or consequence underneath
them. A reliable pattern is: **abstract idea, then concrete mechanism, then consequence.**

> The process improved trust. Teams could see who had made each decision and what evidence they had
> used, so disagreements no longer looked arbitrary.

The first sentence interprets. The second grounds the interpretation and names the consequence.
Neither has to be removed; the interpretive claim is welcome once it is answerable to something a
reader can picture.

#### 5. Keep the actor, action, and consequence visible

Ask four questions of a claim: who or what acted, what changed, how it changed, and what followed.
When a sentence hides these behind nominalizations and passive machinery, the mechanism disappears.

Obscured:
> Greater consistency was facilitated through the implementation of shared principles.

Clearer:
> The shared principles gave teams the same basis for recurring decisions.

The goal is not to force every sentence into the active voice. It is to stop grammatical abstraction
from hiding the mechanism. A passive construction is right when the affected thing and the timing
matter more than the actor:

> Three accounts were compromised before the intrusion was detected.

Here the accounts and the sequence carry the point, and the still-unknown attacker would only get in
the way. Choose the passive on purpose, not by reflex.

#### 6. Choose the simplest exact word, not merely the shortest

A long or technical word earns its place when it carries a distinction the shorter alternative
loses. The rule cuts both ways: prefer the plain word when the meaning is identical, and keep the
precise one when it is not.

- Use *use* rather than *utilize* when they mean the same thing.
- Use *latency* rather than *delay* when the technical distinction matters.
- Use *correlation* rather than *relationship* when statistical meaning matters, and *relationship*
  when statistical precision does not.

Plainness should never come at the cost of accuracy. The target is the exact word, and sometimes the
exact word is the longer one.

#### 7. Separate observation, inference, and judgment

AI prose often slides between these three without showing the move, so a narrow fact suddenly
carries a sweeping verdict.

> Three teams missed the deadline. The process was fundamentally broken.

The first sentence is an observation. The second is a broad judgment. What is missing is the
inference that would connect them. Supply it, and the judgment becomes earned rather than asserted:

> Three teams missed the deadline because each waited for approval from the same reviewer. The
> repeated bottleneck suggests a problem in the process, rather than three independent planning
> failures.

This version still interprets the facts. It simply makes the reasoning visible, so the reader can
follow the step from what happened to what it means.

#### 8. Use contrast only to close a real interpretive branch

Lead with the positive claim. Do not attach an opposing claim merely to sharpen the sentence, create
rhythm, or make an ordinary point sound more decisive. Patterns such as `X, not Y`, `not A but B`,
and `we are not doing A; we are doing B` force the reader to consider two possibilities. That extra
branch is useful only when the rejected possibility is:

- plausible in the current context;
- consequential to the reader's understanding; and
- not already ruled out by the positive statement.

When those conditions are absent, state the positive claim and stop.

**Prefer a positive boundary.** Even when a distinction matters, it is often clearer to name the
relevant scope, control, mechanism, or constraint.

Contrastive:
> Workflow with agentic phases, not autonomous agency.

Direct:
> The workflow uses agentic phases under human control.

Contrastive:
> We are not replacing analysts. We are shifting the analyst's role.

Direct:
> Analysts retain responsibility for judgment while the system handles triage and initial drafting.

The direct version is also better because "shifting the role" is itself vague; it names what stays
with the analyst and what changes.

Contrastive:
> This makes depth cheaper, not absent.

Direct:
> This lowers the cost of deeper analysis.

Contrastive:
> Rejected outputs cause a deterministic retry, not a fallback to free text.

Direct:
> Rejected outputs always enter the deterministic retry path.

When the absence of a fallback is operationally important, state that boundary plainly rather than as
an antithesis:
> Rejected outputs enter the deterministic retry path. The system has no free-text fallback.

The distinction remains, and it no longer leans on rhetorical antithesis.

**When the negation earns its place.** Keep a negative contrast when the excluded alternative is both
plausible and material:
> Agentic triage currently starts through interactive use; automated triggering is not yet supported.

Here automated triggering is a natural assumption and an important product boundary, so naming it
prevents a real misunderstanding. Compare:
> The workbench is one of three execution surfaces, not the whole system.

The negated half is doing no work: "one of three" already rules out "the whole system". The positive
sentence alone is enough:
> The workbench is one of three execution surfaces.

That gives the rule a useful second test: even a plausible alternative does not need to be named when
the positive statement already excludes it.

**Inflated contrast formulas.** Apply the same test to the two stock antitheses. `Not only X, but
also Y` usually just dramatizes a conjunction.

Inflated:
> The change not only reduces cost but also improves consistency.

Direct:
> The change reduces cost and improves consistency.

Keep `not only` when the expansion of scope is itself the point, when Y genuinely overturns an
expected limit:
> The policy applies not only to employees but also to contractors and external reviewers.

Even here the plain list may be enough (`The policy applies to employees, contractors, and external
reviewers`), and the contrast is justified only when the reader's likely assumption about scope is
relevant to the explanation.

`Not a mirror, but a portal`, the metaphorical antithesis, is especially prone to sounding insightful
while explaining nothing.

Ornamental:
> The dashboard is not a mirror but a portal.

Explanatory:
> The dashboard lets teams intervene in the system rather than merely observe it.

The second keeps the actual distinction and drops the staged profundity. A metaphorical contrast
survives only when both images clarify the concept and the difference between them does real
explanatory work.

**Editorial test.** For every contrastive negation, ask what genuine misunderstanding the negated
half prevents, then ask whether the positive half already prevents it. If no misunderstanding is
prevented, delete the contrast. If the positive wording already closes the branch, delete the
contrast. If the alternative is plausible, consequential, and otherwise still open, keep it, and
state it as plainly as possible.

**Compact form:** use contrast only to rule out a live alternative. Lead with what is true, and do
not manufacture an opposite for cadence, emphasis, or tone. Keep a negated alternative only when it
is plausible, consequential, and not already excluded by the positive statement. Where possible,
express the boundary positively by naming the relevant scope, control, mechanism, or constraint.
Reduce `not only X but also Y` to `X and Y` unless the expansion of scope is itself the point. Treat
polished antitheses such as `not a mirror but a portal` as suspect unless the contrast genuinely
explains the concept.

#### 9. Explain in the positive, not as a rebuttal

The user's prompt, including any confusion, contradiction, or "X vs Y isn't clear" they voice, is
input about *what* to teach. It is not a script for *how* to stage the explanation. Never open a
section by restaging the misconception and then resolving it. State the concept directly, as a
textbook would to a reader who never asked the question. Lead with what a thing is, and let the
correct mental model do the disambiguating on its own.

Teaching from the question, rather than from the concept's own frame, forces the reader to
reconstruct an invisible interlocutor. They feel you answering an objection they were never shown,
which confuses more than it clarifies.

Don't narrate the tension:
> These trip people up because they sound like one ladder, as if conformed were bigger than a table.
> They are not. They are actually four independent axes.

That answers an interlocutor the reader cannot see. Do assert the model directly, then show them:
> An object is described by four independent properties: where it is stored, how far through the
> pipeline it sits, its physical form, and its modelling role.

Reserve the myth-then-correction move for when the user explicitly asks you to clear up a specific
confusion, and even then keep it to one line, never a paragraph. A section should read as if it were
written before the question existed.

#### 10. Teach from the model, not from the misconception

Rule 9 states this at the level of a single sentence; this rule is about the architecture of a whole
section. Treat the reader's confusion as evidence about what needs explaining, not as the structure
of the explanation. Begin with the correct concept: what the thing is, which parts it has, and how
those parts relate. Let that model resolve the confusion through its own clarity.

Do not make the reader pass through a mistaken model before receiving the useful one. Avoid opening
sections with imagined objections, false alternatives, or constructions such as:
> It may seem that X and Y are stages of the same process, but they are not.

Prefer:
> X describes where an object is stored; Y describes its role in the process. The two properties
> vary independently.

Use explicit correction only when the misconception itself is important, for example when it is
widespread, consequential, or explicitly raised by the reader. Even then, correct it briefly and
move immediately to the positive model:
> These terms are sometimes treated as sequential stages. They describe independent properties.

The explanation that follows should stand on its own, without needing the original question or
misunderstanding for context.

**The underlying principle:** organize explanations around the structure of the subject, not the
history of the reader's confusion. This is not a ban on negation or contrast. Negative statements
can define boundaries, prevent serious errors, or distinguish closely related concepts. The rule is
about expository architecture, not grammar: the misconception should not become the narrative spine
of the section.

**Diagnostic test.** Ask:
- Would this section make complete sense to someone who had never voiced the original confusion?
- Does the opening give the reader a usable mental model, or merely tell them which model is wrong?
- Am I explaining the subject, or re-enacting a conversation with an invisible interlocutor?
- Could the contrast be replaced by a direct statement of the relevant distinction?

Rebuttal-shaped:
> These categories are easy to confuse because they sound like levels in a hierarchy. Conformed is
> not a larger or more advanced kind of table. In fact, the terms describe four different axes.

Model-shaped:
> An object is described by four independent properties: where it is stored, how far it has
> progressed through the pipeline, its physical form, and its modelling role.

The second version gives the reader something they can use immediately. It does not require them to
construct the wrong hierarchy and then dismantle it.

**Compact form:** explain from the concept outward, not from the confusion inward. Treat
misconceptions as diagnostic input, not narrative scaffolding. Lead with the correct mental model,
its components, relationships, and boundaries, and let it perform the disambiguation. Do not open a
section by restaging an objection or false model. Use explicit correction only when the misconception
is itself relevant, and keep the correction brief. Every section should remain intelligible to a
reader who never saw the question that prompted it.

### Connective tissue and movement

#### 11. Keep connective tissue when it names a real relationship

Connective prose is valuable when it tells the reader how one idea relates to another. It can carry
cause, consequence, contrast, concession, sequence, qualification, a change of scale, or a return to
a question raised earlier. Words such as *but*, *because*, *therefore*, *meanwhile*, *even so*, and
*in practice* are not clutter when they are doing one of those jobs.

The problem is the ceremonial transition: language that announces movement without establishing a
relationship.

Ceremonial:
> With this broader context in mind, it is important to consider the role of governance.

Relational:
> Because the model affects hiring decisions, its accuracy is only part of the question; the
> organization must also decide who can challenge its recommendations.

The second sentence does not merely change topics. It explains why the next topic follows from the
last. That is the test for any connective phrase: does it name the relationship, or only gesture at
one.

#### 12. Let each sentence inherit something and add something

Seamless prose rarely comes from elaborate transitions. It comes from continuity: each sentence
picks up a subject, term, question, contrast, image, or consequence from the sentence before it, and
then moves it forward.

> The first release reduced the time needed to publish a page. That speed exposed a different
> problem: teams could now produce inconsistent pages more quickly. The next release therefore
> focused less on publishing and more on shared defaults.

Each sentence inherits and advances: *reduced time* becomes *speed*; *speed* reveals
*inconsistency*; *inconsistency* sets the next priority. That is connective tissue with no padding,
built from the ideas themselves rather than bolted on between them.

#### 13. Make paragraphs move rather than merely accumulate

A paragraph that works usually has a shape: an anchor, some development, a turn or qualification, and
a consequence.

> The new review reduced the number of defects reaching production. Most of the gain came from
> catching incomplete states before engineering began, rather than from finding more problems at the
> end. That distinction matters: the review works by changing decisions early, and its value will
> disappear if teams treat it as a final checkpoint.

That paragraph carries fact, explanation, distinction, interpretation, and a warning, and it is not
robotically factual, yet every sentence advances the thought. Not every paragraph needs this exact
form. The one thing that must hold is movement: the last sentence should carry the thought somewhere
new, never restate the first in grander language.

#### 14. Allow interpretation, but make the bridge visible

Signposting phrases such as *this matters*, *the deeper problem*, or *the important distinction* can
be genuinely useful. They turn empty when what follows is a second abstraction instead of the
substance the signpost promised.

Empty:
> This underscores the critical importance of thoughtful leadership.

Specific:
> This matters because the team did not lack information; it lacked someone authorized to act on it.

Interpretation is part of good prose. Moralizing without analysis is the counterfeit of it. When you
tell the reader something matters, show the bridge from the evidence to why.

#### 15. Scale the language to the evidence

Treat words like *transformative*, *profound*, *revolutionary*, *unprecedented*, *fundamental*,
*critical*, *defining*, *inevitable*, *universally*, and *entirely* as claims that require support.
Do not ban them. Ask what observable or measurable scope would justify the word, and either supply
it or step the claim down.

Inflated:
> The feature fundamentally transforms how people create.

There are two honest repairs, and which one you reach for depends on the truth. Step the claim down
to what actually happened:
> The feature removes one of the slower steps in creating a first draft.

Or, where the larger claim is genuinely warranted, earn it by describing the structural change:
> The feature changes the starting point of the work: users now begin with an editable draft rather
> than an empty page.

The second still communicates significance. It does so by showing the structural change rather than
announcing that the change is significant, which is the difference between a claim a reader can weigh
and one they are merely asked to accept.

### Image, rhythm, and voice

#### 16. Use metaphor as an instrument of thought, not decoration

Imagery is welcome. The distinction is between a fresh image that assists thought and a worn one
that spares the writer from thinking. A metaphor earns its place when it reveals a useful
resemblance, makes a mechanism easier to grasp, stays coherent when examined, does not exaggerate
the subject, and is not immediately crossed by a second, conflicting image.

Decorative:
> The platform is a beacon that unlocks a new frontier and lays the foundation for an ecosystem.

That asks the reader to hold a beacon, a lock, a frontier, a foundation, and an ecosystem in mind at
once, and so pictures nothing. Functional:
> The template acts as scaffolding: it supports the first stages of the work and is meant to come
> down as the final structure takes shape.

One image, and it explains both the function and its limits.

#### 17. Cut repetition that does not develop; keep repetition that builds structure

Restating an idea in slightly different words, then restating its importance, is static repetition:

> The policy makes ownership clearer. It creates greater clarity around responsibility. This
> highlights the importance of clearly defined ownership.

Nothing develops across those three sentences. But repetition can also do real rhetorical work,
building rhythm, contrast, or escalation:

> The first failure delayed a decision. The second delayed a launch. The third showed that the delay
> was part of the system.

The repeated structure is the point there. The test is not "can these words be removed" but "what
would be lost if they were: meaning, logic, emphasis, tone, rhythm, or nothing". Cut only when the
answer is nothing.

#### 18. Vary sentence length according to the movement of the thought

Uniformly short sentences turn abrupt and mechanical. Uniformly long ones hide weak logic in their
folds. Use length deliberately: a short sentence lands a conclusion, a medium one carries the main
explanation, a long one holds comparison, qualification, or accumulating detail as long as its
structure stays visible.

> The proposal looked cheaper. It was not. Once migration, support, and retraining were included, the
> apparent saving became a two-year cost.

The brief middle sentence creates the emphasis. It is not more factual than a longer version would
be; it simply controls the pace, and pace is part of the meaning.

#### 19. Preserve voice, but do not confuse voice with verbal ornament

Voice comes from what the writer notices, the order in which details arrive, the confidence or
caution of the judgments, the rhythm of the sentences, restrained humour, the quality of the
comparisons, and the willingness to state a hard conclusion plainly. It does not come from a
constant layer of flourish. The standard to hold: the writing should sound like a particular mind
attending carefully to a particular subject, rather than a generic style applied to any subject at
all.

### Breaking these rules

Break any rule the moment breaking it makes the meaning more accurate, more natural, or more
memorable. This is central, not a loophole. A passive sentence may place the emphasis exactly right.
A long word may be the exact word. A familiar metaphor may be perfect in its place. A phrase that
looks removable may be carrying rhythm, warmth, hesitation, or plain courtesy. Use passive voice,
technical language, long sentences, and familiar expressions deliberately rather than by reflex, in
either direction. The rules earn their keep as questions to ask, and they stop earning it the moment
they become a checklist you obey without reading the sentence.

### The five-function test

To protect real connective tissue while cutting filler, do not ask only whether a phrase adds a
*fact*. Ask whether it performs any of these five functions. A phrase that performs none is probably
filler; a phrase that performs even one earns its place. This is what lets the guide permit narrative
and articulation while still removing sentences that merely perform fluency.

1. **Content:** new information or an example.
2. **Relation:** cause, contrast, consequence, sequence, or comparison.
3. **Qualification:** a limit, an uncertainty, a scope, or a condition.
4. **Orientation:** where the reader is in the argument, and why the subject is changing.
5. **Expression:** deliberate rhythm, emphasis, tone, or image.

### A worked example: the intended balance

The target sits between two failures, so it helps to see all three versions of one paragraph.

Inflated, performing importance:
> In today's rapidly evolving workplace, asynchronous communication is not merely a convenience; it
> is a transformative force that empowers teams to unlock new levels of alignment. This underscores
> the critical importance of cultivating a culture of intentional communication.

Stripped too far, the reasoning gone:
> Teams communicate asynchronously. It can improve alignment. Communication should be clear.

Cleaner narrative prose, performing thought:
> Asynchronous communication changes where coordination happens. Instead of resolving every question
> in a meeting, teams leave decisions and context where others can find them. That only improves
> alignment when the record is clear; otherwise it moves confusion from the room to the document.

The third version keeps several kinds of connective tissue at once. The first sentence frames the
change, the second explains the mechanism, and the third qualifies the claim and hands it a
consequence, with the contrast in its final clause giving the paragraph a natural landing. It reads
as an articulated piece of reasoning, which is what the middle stripped-down version threw away and
the inflated version only pretended to be.

---

## Editorial review workflow

When invoked to review a draft (not to write from scratch), work through it against the applicable
tiers and return targeted edits, not a rewrite of the whole thing:

1. **Decide the tiers in scope.** Tiers 0 and 1 always apply. Tier 2 applies unless the draft is a
   status ping, a one-line factual answer, or a code comment. Length does not decide it: a 60-word
   job application answer is Tier 2, because a reader will judge it. When in doubt, Tier 2 is live.
2. **Scan for each rule in order.** For every hit, quote the offending span, name the rule, and give
   the fix inline. Concrete beats abstract: show the rewritten sentence, and where a rule offers two
   honest repairs (rule 15), show which one fits and why.
3. **Preserve the author's voice and meaning.** Fix how it reads, never what it claims. When a fix
   would change the meaning, flag it and ask rather than guess.
4. **Do not invent problems.** If a passage is clean, say so and move on. Silence on a paragraph
   means it passed. Remember rule 19: leave the author's voice intact, and do not sand prose down to
   bare facts in the name of the rules.

Report format: a short list, each entry as `rule -> quoted span -> suggested fix`. Lead with the
edits that matter most, and skip preamble.

---

## Adding rules

This guide is meant to grow. When the user gives a new rule, add it under the correct tier with the
same shape as the others, and do not compress it into a bare command. Keep three things: **the rule,
the reasoning behind it, and worked examples (a Don't/Do pair, or a short before-and-after) with the
commentary that says what each example demonstrates.** The reasoning and the commentary matter most:
a rule with its why gets applied intelligently in cases the examples never covered, while a bare
imperative gets misfired or ignored. The examples are not decoration; they are how the model learns
the judgment the rule is pointing at. Assign the rule by scope: Tier 0 if it names a phrasing that
is wrong at any length in any context, Tier 1 if it governs sentence mechanics that hold everywhere,
Tier 2 if it governs how prose carries reasoning or teaches. A rule that fits none cleanly earns a
new tier of its own.

Tier 0 entries are the shortest to add and the cheapest to apply, so prefer them when a rule really
is unconditional. A correction of the form "this specific phrasing reads as AI" belongs there, not
buried in a Tier 2 rule where a scope judgment can skip it.

---

## References

For the human maintaining this guide, not for the model applying it. These are the sources the rules
were distilled from, kept here so the provenance is not lost.

- **Wikipedia, "Signs of AI writing"** — the catalogue of machine-generated tells behind the
  puffery, participle-summary, weasel-attribution, and vogue-word entries.
  <https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing>
- **George Orwell, "Politics and the English Language"** — the source of the meaning-and-precision,
  connective-tissue, and image-rhythm-voice principles, including the rule to break any rule when it
  serves the meaning.
  <https://www.orwellfoundation.com/the-orwell-foundation/orwell/essays-and-other-works/politics-and-the-english-language/>
- **The Gods of good narrative** — the working name for the third source, on positive-first
  exposition and using contrast only to close a live interpretive branch.

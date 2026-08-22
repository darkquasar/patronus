---
name: writing-like-me
description: >
  Write in your own voice, editorially clean first. Runs the writing-editorial tiers over a draft
  (or over a faceless base draft it composes), derives a voice profile of named moves from your
  exemplar corpus, then a spine that owns the piece's metaphor, opening scene, register and running
  order, voices each section in parallel against both, audits every section for liveness, stitches
  them with narrative continuity, and takes one advisory read from a second model. Use WHENEVER the
  user asks to "make this sound like me", "write this in my voice", "draft this the way I would", or
  hands over a draft and asks to have it voiced. Ships with EMPTY exemplar files by design: it does
  nothing useful until you supply a corpus at ~/.claude/patronus/voice/. Requires the
  writing-editorial skill, which its manifest pulls in automatically.
---

# Write like me

**Voice lives in conception, not in varnish.** Which scene opens the piece, which metaphor spans
it, where the author lets themselves be wry: a voice pass confined to diction and rhythm can only
polish an arrangement someone else already fixed. So this pipeline derives a spine before it writes
a sentence, and gives that spine authority over structure.

```
  sections/ + edits.yaml            from writing-editorial, cut and anchored
    |
    v
  profile: cached or fresh?          [ASK USER]
    |
    v  voice-profile.md
  derive SPINE                       [CHECKPOINT]  main agent, whole-document
    metaphor, scene, register  <- profile
    claims manifest            <- attractor draft
    running order, per-section assignments
    |
    v  spine + profile + section[i] + edit record[i]
  [ s01 ][ s02 ][ sNN ]              voice subagents, parallel
    |
    v  voiced section[i] + restore log
  audit each section                 flat -> rework once -> accept and flag
    |
    v
  stitch                             main agent: narrative continuity, no signposts
    |
    v
  codex advisory read                once, on the finished text
    |
    v  final draft + codex notes taken and declined
```

## Entry modes

This skill is both an editor and a composer, so decide first which one you are. Choose from what the
user supplies:

| Mode | Input | What the voice stage may do |
|---|---|---|
| **Edit** | an existing draft | the prose is the author's and carries authority: diction, rhythm, paragraph shape, cutting and reordering within a section. **Never** replace their sentences wholesale, and **never** inject a claim they did not make |
| **Compose** | a request, no draft | write a faceless base draft **with no corpus in context**, run stage 0 over it, then voice: every sentence may be rewritten |
| **Voice-only** | a draft the user says is already edited | skip stage 0, say so, and voice with no edit records or sections, so no restore is available |

## The attractor

Compose mode still writes its base draft with **no corpus in context**. The reasoning holds: a
model told to sound like someone chases the sound and neglects the thinking.

What changes is its status. It is an **attractor and loose scaffolding**, not a text to be
preserved. It fixes the ideas, the evidence and the citations. **Its sentences carry no authority
at all**, and the voice stage is licensed to demolish and rebuild every one of them.

In edit mode the user's own draft plays this role, and **its sentences do carry authority**: the
never-inject rule binds, and the voice stage may not introduce claims the author did not make.

## Stage 0: the editorial pass

Run the sibling `writing-editorial` skill over the draft, **supplying `trail-root`**, which is what
makes it emit section files and span-anchored edit records. Supply the attractor draft as its
`companion` input so each section's attractor slice is cut to the same ids.

What comes back, and what every later stage joins on:

| Artifact | Used by |
|---|---|
| `sections/NN-slug.md` | the voice subagent: post-tier-3 and authoritative |
| `sections/NN-slug.source.md` | restore resolution: the snapshot offsets resolve against |
| `sections/NN-slug.edits.yaml` | the citation rule, and PRESERVE, now advisory |
| `sections/NN-slug.companion.md` | the never-inject check |
| `sections/lineage.yaml` | the spine, when it reshapes |

The two schemas are defined in `{skillsDir}/writing-editorial/edit-record.md`. **They are an
interface, not an implementation detail**: read them there rather than inferring them from a file.

**The sibling resolves by path, not by name.** Where the host exposes a skill-invocation mechanism
by name, using it is equivalent and preferred, because it honours the router's own dispatch
question. If the tier files are absent from `{skillsDir}/writing-editorial/`, this is corruption
rather than a supported mode: the `requires:` edge means the skill is present under every supported
install path. Report the expected location, state that stage 0 was skipped and the draft is
unedited, and continue.

**If the sibling runs but returns no section files**, it is an older version that does not know
`trail-root`. Say so, naming the path that stayed empty, and continue on the whole draft as a
single section with no edit records: no restore is available, and the citation rule has nothing to
act on. Do not fabricate section files to keep the later stages tidy.

## Stage 1: the profile

Read `{skillDir}/voice-profile-schema.md` and follow it: ask cached or fresh, resolve the corpus,
extract from every corpus file whatever its language, keep evidence in its source language, and take
rhythm numbers from the English pool only.

With no corpus at the resolved path, **degrade**: print the path, say what to put there, and run
editorial-only.

## Stage 2: the spine

Read `{skillDir}/spine.md` and follow it. Derive the spine once, run its checkpoint checks, and
**show it to the user for approval before any fan-out**.

## Stage 2: the voice pass

Both models receive exactly the same six context items:

1. the cleaned draft from stage 1;
2. the exemplar pool, and what to read from it versus take from the draft (below);
3. `{skillDir}/weights.md`;
4. the **PRESERVE list**;
5. the **contrast ledger**;
6. the **pre-editorial original**.

Neither receives the editorial tier files. The voice pass is not there to re-litigate decisions the
editorial pass already made.

**Why the original matters.** The never-inject rule says add nothing the author did not write, and
that is unenforceable without a fixed reference: a fresh model cannot tell an author's first-person
aside from one an earlier pass introduced. With the original in context, the constraint is checkable.
In voice-only mode there is no original, so the rule narrows to "add nothing not present in the draft
you were given", which is weaker, and the output says so.

**PRESERVE is binding here too.** This is the one stage permitted to rewrite freely, which makes it
the stage where a protected span is most at risk. A voice pass that flattens a coined term
immediately after tier-2 protected it defeats the whole mechanism. Override an entry only with a
stated reason, naming which entry and why.

**The contrast ledger binds this stage as well**, and for the same reason it binds tier-3: a pass
that writes new sentences is not given a fresh allowance. Applying a voice is where an antithesis is
most tempting, because mirrored shapes read as punchy and voice work rewards punch. With
`remaining: 0`, voice the draft in the positive. Displacing the retained correction is allowed with
a stated reason; adding a second is not.

### What the corpus is for

**The corpus supplies a voice, never a format.** This is the distinction the whole stage turns on,
and getting it backwards produces the most common failure: a long draft chopped into the shape of
the exemplars, so a design doc comes back reading like a thread of posts.

Two things live in any exemplar, and they travel differently:

| Read from the corpus | Take from the draft |
|---|---|
| sentence rhythm and the length spread within a paragraph | how long the finished piece is |
| diction, register, and the words this author reaches for | what the piece has to cover |
| how a paragraph opens, turns, and lands | how many paragraphs there are |
| punctuation habits, contractions, spelling conventions | the section structure and headings |
| stance: hedged or blunt, first person or impersonal | the order of the argument |
| the moves, such as a concrete case before the claim | the scope of each section |

The left column is the voice and it projects onto any length. The right column belongs to the piece
you were given, and the voice pass has no business changing it.

**A corpus of short pieces still teaches long-form voice.** Rhythm is a property of consecutive
sentences, not of a word count, so it is fully present in a 120-word post and it scales without
distortion. What a short corpus cannot teach is long-form architecture: how this author sustains an
argument over ten paragraphs, when they summarize, how they hand off between sections. Take that
from the draft and from tier-3's work, which already settled it.

### Measure the corpus; do not imagine it

**Short-form pieces are not made of short sentences, and assuming they are is the failure mode this
section exists to stop.** A corpus of posts routinely averages 17 words a sentence with a fifth of
them past 26, which is an ordinary spread for any register. A voice pass that reads "short-form" and
reaches for clipped, punchy sentences is applying a stereotype of the format rather than the voice in
front of it.

So measure the pool before voicing anything: its typical sentence length, its share of sentences past
26 words, its longest, and its paragraph spread. Put those numbers in the prompt you hand each model,
because a model cannot aim at a distribution it was never told.

`{skillDir}/weights.md` carries what to do with them, and both models receive it. It also carries the
second half of the problem, which no length target catches: sentences of varied length built to an
identical shape still read monotonously.

### Corpus resolution

Resolve the pool for the target form. First hit wins:

1. `$PATRONUS_VOICE_DIR/<genre>.md`, when the environment variable is set;
2. `~/.claude/patronus/voice/<genre>.md`, the default user-owned location;
3. the shipped stub, which contains no exemplars and triggers the empty-pool path.

Create neither the directory nor the files. On a first run with no corpus, print the resolved path
you looked for and what to put there, then continue in degraded mode. Corpus setup stays an explicit
user act, and upgrades stay incapable of touching it.

**The pool matching the target form wins whenever it has exemplars.** Long target with a populated
`long-form.md` uses `long-form.md`, and the short pool is not consulted. The reason is evidentiary,
not architectural: a matching pool shows how this author's sentences behave in sustained reasoning,
which is the part of the voice a short pool can only be projected for. Architecture still comes from
the draft in every case, exactly as the table above says. The other pool is used only when the
matching one is empty, and then it is projected rather than copied.

| Target | long-form.md | short-form.md | Behaviour |
|---|---|---|---|
| long | populated | either | **use `long-form.md`.** Matching form, so it shows the voice at the target length |
| long | empty | populated | **use `short-form.md`, projected.** A supported path, not a degraded one, so it needs no permission. Voice from the corpus, length and structure from the draft |
| short | either | populated | **use `short-form.md`** |
| short | populated | empty | **use `long-form.md`, projected.** Same rule in reverse: take rhythm and diction, not the long piece's architecture |
| any | empty | empty | skip the voice stage, run the editorial tiers only, and say the pipeline ran in editorial-only mode |
| any | unreadable | unreadable | report the path and the error, treat as empty, do not fail the run |

Say which pool was used either way. Where the pools differ in what they can teach, the reader should
know which one shaped the output.

### The Claude side

A fresh subagent, with the six context items above.

### The codex side

Reach codex over MCP, not by driving its CLI. Codex ships an MCP server on stdio:

```
codex mcp-server
```

Register it once as an MCP server named `codex`, then call its tools. It exposes two:

| Tool | Use |
|---|---|
| `codex` | start a session. Required argument: `prompt`. Useful optional arguments: `sandbox`, `cwd`, `model`, `base-instructions` |
| `codex-reply` | continue that session, with `threadId` and the next `prompt` |

Call `codex` with `sandbox: "read-only"`, and with `prompt` naming the paths of all six items: the
draft, exemplars, weights, PRESERVE list, contrast ledger, and original. Codex reads them from disk
itself.

**Put the corpus measurements in the prompt text, not only the paths.** Codex receives files, not
this file, so anything stated here and nowhere else never reaches it. The numbers you measured are
that kind of thing: state them in the prompt, or the model with the least context produces the
flattest prose.

**The draft never goes into a shell string, and over MCP it never goes into an argv either.** The
prompt is a JSON string field, so quoting, backticks, newlines, and command-length limits stop being
failure modes: the transport carries arbitrary bytes without escaping. Passing paths rather than
pasted content keeps the prompt small and lets codex read exactly what stage 2 wrote.

**The transport owns the timeout and the process.** The MCP client bounds the call, returns a typed
error on expiry, and manages the server process, so this file specifies no signals, no reaping, and
no fallback timer. Where the host lets you set a per-call timeout, allow around three minutes: a
voice pass over a long draft is slower than a typical tool call.

**The result is the reply, not a delimited region of stdout.** An MCP tool call returns structured
content, so there is no envelope to agree on and nothing to parse out of console chatter. Take the
returned text as the voiced draft. Where codex returns commentary alongside it, ask for the draft
alone via `codex-reply` rather than guessing which span is the draft.

`codex-reply` is what makes this better than a one-shot call. If the first pass drifts from the
corpus, over-corrects, or flattens a PRESERVE span, continue the same thread and say so, instead of
restarting with a longer prompt.

**Degrade, never block.** If the codex MCP server is not registered, the call errors, or it times
out, continue with the Claude draft alone and say which of those happened, naming the error. The
cross-model stage is an optional check, not a dependency.

**Why a second model rather than a second Claude.** Same-family judges are biased toward
low-perplexity text, and low perplexity is the signature of generic machine prose, so a same-family
critic is biased in the wrong direction for voice work. One model from a different family captures
nearly all the available benefit; a larger panel does not.

## Stage 3: the merge

**The main agent merges**, not a third subagent. Produce three things:

1. The merged draft: one version, taking the stronger choice at each point of divergence.
2. The disagreements, verbatim. Where the two drafts diverged materially, show both versions.
   Disagreement goes to the human rather than to a vote: with two voters there is no majority, and
   the disagreements are the most informative output of the stage.
3. Two to four threadable variations, concrete alternatives the user can splice in, each named
   for what it changes: open on the concrete case rather than the claim; keep or drop a hedge; add or
   remove a coined term; hand off at the ending rather than summarizing.

Report any PRESERVE entry that either model overrode, with the reason it gave.

**Audit the merged draft's rhythm.** The merge writes the version the reader sees, so it can undo
both calibrated drafts by taking the shorter option at each divergence. Two checks, both cheap:
compare the merged draft's longest sentence against stage 1's, and take stage 1's construction back
where a voice pass split it; then scan consecutive sentences for a repeated shape, per
`{skillDir}/weights.md`. Report the merged draft's typical length and longest sentence alongside the
corpus's.

**Recompute the ledger from the merged draft.** The two voice passes run in parallel, so each may
have displaced the retained correction independently. Taking the stronger half of each can therefore
carry both replacements through and leave two live corrections where the ledger allows one. Never
union the two branches' ledgers: read the merged text and count what is actually in it, then report
that single reconciled ledger and any reallocation it records.

## The run trail

Every run writes its own directory named for the piece. **Where that directory goes depends on
whether the prose belongs to the project you are standing in**, because a trail is a work product
about the piece, not about the repository that happened to be open.

| The piece | Trail goes to |
|---|---|
| Belongs to this project: its README, an ADR, a design doc, a PR description | `docs/writing/<slug>/` in the repo |
| Does not belong to it, or there is no repo: an email, a message, a post, a personal essay | a scratch directory outside the repo, and say where |

When it is not obvious which, ask. Getting it wrong in the second direction is the one that stings,
since it commits private drafting into a project's history, so treat an unclear case as personal and
keep it out of the tree.

Inside a repo, whether the path is committed or ignored is the user's call: add it to `.gitignore`
for private drafting, leave it tracked where the prose is a team deliverable. Never edit
`.gitignore` yourself.

Ask once, at the start:

```
Keep full drafts for diffing? [Y/n]
```

Enter keeps them. Running unattended keeps them too, so the default is the same either way and a
trail is never silently thinner than it looks. Drafts are what make the trail checkable: without
them a reader has only each stage's own account of what it did, and a stage that fails to report an
edit is indistinguishable from one that made none.

```
<trail-root>/<slug>/
  manifest.yaml         always
  EDITS.md              always, per stage
  PRESERVE.md           always
  draft-00-original.md  when drafts are kept
  draft-0N-<stage>.md   when drafts are kept
```

`manifest.yaml` records what each stage did and what it carried:

```yaml
run: deleuze-three-contributions
mode: compose
drafts_kept: true
stages:
  - stage: tier-1
    rules_fired: [tier-1.3]
    edits: 3
    contrast_ledger:
      retained: "actualization is not resemblance but invention"
      remaining: 0
  - stage: tier-3
    rules_fired: [tier-3.7, tier-3.8, tier-3.11]
    edits: 4
    preserve_overrides: []
    ledger_reallocations: []
    typical_sentence: 16      # median words
    longest_sentence: 41
    pct_past_26: 31
voice:
  corpus: ~/.claude/patronus/voice/short-form.md
  exemplars_supplied: 12
  corpus_typical_sentence: 15
  corpus_pct_past_26: 19
  merged_typical_sentence: 15
  merged_longest_sentence: 38
  model_reported_influences:      # the model's own account, not verified provenance
    - "consequence-first turn opening on a bare connective"
```

Two things about that shape are deliberate. **The drafts are what make it checkable**, not the
manifest: a stage's account of its own edits is a claim, and the draft beside it is the evidence.
Diff two stages and you see what actually changed, including what a stage failed to mention. The
manifest carries no checksums, because a model writing a hash it did not compute records a number
that proves nothing.

And **`model_reported_influences` is labelled as self-reported**, because a model cannot reliably say
afterwards which exemplar shaped a structure it synthesized across a dozen. The corpus path and the
exemplar count are knowable; the attribution is a claim.

## Known limits, stated rather than hidden

- Register mismatch. Informal-register imitation verifies far less reliably than formal registers,
  so a corpus of casual posts gives weaker signal than one of worked prose. This is about how
  reliably the voice transfers, not about which lengths are allowed: a short corpus projecting onto
  a long piece is the supported path, and the constraint is that casual registers are harder to
  imitate at all.
- A short corpus cannot teach long-form architecture. Sentence rhythm and diction scale from a
  120-word post to a 4000-word essay, but how this author sustains an argument across ten
  paragraphs is not in the corpus to learn. That comes from the draft and from tier-3, and the
  voice pass leaves it alone.
- Style strength and content preservation trade off against each other. They are competing
  objectives, not a tuning failure. This pipeline picks a point on that curve by putting
  content-shaping first and voice second.
- Scrubbing a widely circulated word list is itself becoming detectable, which is a reason the
  editorial tiers lean on structural rules rather than on lexical substitution alone.

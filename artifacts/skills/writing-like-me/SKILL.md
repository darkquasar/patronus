---
name: writing-like-me
description: >
  Write in your own voice, editorially clean first. Runs the writing-editorial tiers over a draft (or
  over a faceless base draft it composes), then a voice pass in which a Claude subagent and a second
  model from a different family each apply your exemplar corpus independently, then merges the two
  and shows you where they disagreed. Use WHENEVER the user asks to "make this sound like me", "write
  this in my voice", "draft this the way I would", or hands over a draft and asks to have it voiced.
  Ships with EMPTY exemplar files by design: it does nothing useful until you supply a corpus at
  ~/.claude/patronus/voice/. Requires the writing-editorial skill, which its manifest pulls in
  automatically.
---

# Write like me

**Editorial controls first, personal voice second.** An editor works span by span; a composer may
restructure. This pipeline puts every span-local pass ahead of the one pass allowed to rewrite
freely, so the voice pass works on prose whose surface problems are already gone.

```
  [1] writing-editorial          all four tiers, per its dispatch choice
       |                         (skipped with a warning if not installed)
       v  clean, de-slopped draft + PRESERVE list
       |
  [2] +------------------------+        +--------------------------+
      | Claude subagent        |        | codex exec               |
      | fresh context          |        | --sandbox read-only      |
      | genre exemplars        |        | --skip-git-repo-check    |
      | + weights.md           |        | same context             |
      | + PRESERVE + original  |        |                          |
      +------------------------+        +--------------------------+
       |                                 |
       v  voiced draft A                 v  voiced draft B
       |                                 |
  [3] main agent merges <----------------+
       |
       v
      1. merged draft
      2. where the two disagreed, verbatim, both versions
      3. 2-4 threadable variations
```

## Entry modes

This skill is both an editor and a composer, so decide first which one you are. Choose from what the
user supplies:

| Mode | Input | Stage 1 |
|---|---|---|
| **Edit** | an existing draft | run the editorial tiers over it, then voice |
| **Compose** | a request, no draft | write a meaning-first base draft **with no corpus in context**, then the editorial tiers, then voice |
| **Voice-only** | a draft the user says is already edited | skip the tiers, go straight to the voice stage, and say so |

Compose mode's faceless first draft is the point. A draft written before any voice is in context
reaches for different material than one written to sound like someone, and that difference is worth
more than the head start.

## Stage 1: the editorial tiers

Run all four tiers of the sibling `writing-editorial` skill, honoring its dispatch question. Carry
its **PRESERVE list** forward: everything downstream needs it.

**The sibling resolves by path, not by name.** The tier files are at
`{skillsDir}/writing-editorial/tier-0.md` through `tier-3.md`, with the router at
`{skillsDir}/writing-editorial/SKILL.md`. `{skillsDir}` is substituted at install time to the
directory holding every installed skill, so the path is correct on each agent's layout without this
file knowing which one it is on. Where the host exposes a skill-invocation mechanism by name, using
it is equivalent and preferred, because it honors the router's own dispatch question directly. Where
it does not, read the router and the four tier files from those paths and apply them in order. Both
routes reach the same files, which is what makes the absence check below meaningful.

**If the tier files are absent from that path**, this is corruption rather than a supported mode. The
`requires:` edge means `writing-editorial` is present under every supported install path, so an
absence means a manually deleted directory, a damaged install, or a hand-copied skill. Report the
expected sibling location, state that stage 1 was skipped and the draft is unedited, and run stages 2
and 3 anyway. Do not go hunting for the skill across agent layouts.

## Stage 2: the voice pass

Both models receive exactly the same context:

1. the cleaned draft from stage 1;
2. the genre-matched exemplar pool (below);
3. `{skillDir}/weights.md`;
4. the **PRESERVE list**;
5. the **pre-editorial original**.

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

### Corpus resolution

Pick the pool by target genre, then resolve it. First hit wins:

1. `$PATRONUS_VOICE_DIR/<genre>.md`, when the environment variable is set;
2. `~/.claude/patronus/voice/<genre>.md`, the default user-owned location;
3. the shipped stub, which contains no exemplars and triggers the empty-pool path.

Create neither the directory nor the files. On a first run with no corpus, print the resolved path
you looked for and what to put there, then continue in degraded mode. Corpus setup stays an explicit
user act, and upgrades stay incapable of touching it.

| State | Behaviour |
|---|---|
| Target pool has exemplars | draw from it; name the file and how many pieces it drew |
| Target pool empty, other pool populated | **ask before falling back**; if accepted, say explicitly that short-form exemplars are priming long-form output (or the reverse), which is known-lossy |
| Both pools empty | skip the voice stage entirely, run the editorial tiers only, and say the pipeline ran in editorial-only mode |
| A pool file is unreadable | report the path and the error, treat as empty, do not fail the run |

Genre fallback is never silent. Imitation quality is strongly genre-dependent, so a silent
substitution produces bad output with no visible cause.

### The Claude side

A fresh subagent, with the five context items above.

### The codex side

**The draft is arbitrary user prose, so it never goes into a shell string.** Two rules, both
load-bearing:

- **Prompt and draft reach codex via files or stdin, never as an interpolated argument.** The
  instruction names paths; the content is read from disk. This removes quoting, backtick, newline,
  and command-length failure modes at once.
- **The invocation is an argument array**, never a composed shell line.

```
codex exec --sandbox read-only --skip-git-repo-check
           <instruction naming the paths of: draft, exemplars,
            weights, preserve list, original>
```

**Output envelope.** Codex returns its draft between two fixed delimiters, each alone on its own
line:

```
<<<VOICED-DRAFT
...the draft...
VOICED-DRAFT>>>
```

Anything outside the pair is commentary and is discarded. The parsing rules, so producer and consumer
implement the same format:

- the **first** opening delimiter and the **first** closing delimiter after it bound the draft. Any
  later pair is commentary and is ignored;
- delimiters count only when alone on a line, so the tokens may appear inside prose without breaking
  the envelope;
- **unparseable** means precisely: no opening delimiter, no closing delimiter after an opening one,
  or nothing but whitespace between them.

**Timeout.** Bound the call at **180 seconds** by default, overridable via `PATRONUS_CODEX_TIMEOUT`.
The default sits well above the time a typical voice pass takes; it exists to bound a hang rather
than to cut off slow work.

Impose the bound with the shell's own timer, so the guarantee does not depend on this agent noticing
the expiry:

```bash
timeout -k 5 "${PATRONUS_CODEX_TIMEOUT:-180}" codex exec --sandbox read-only --skip-git-repo-check ...
```

`timeout` sends `SIGTERM` at expiry and `SIGKILL` 5 seconds later, and the shell reaps the child.
That is the contract delegated to a tool that actually holds it. Do not hand-roll the two signals: an
agent driving a foreground command through a tool that manages its own process lifecycle cannot
reliably send a delayed second signal or reap anything, so a prose instruction to do so would
describe a guarantee nothing enforces. **Where `timeout` is unavailable** (it is GNU coreutils, so it
is absent on a stock macOS without `coreutils` installed, where it is `gtimeout`), fall back to
`gtimeout`, and where neither exists, run the call unbounded, say so in the output, and treat a hang
as the degrade path below.

The override is validated so it cannot defeat the bound it configures. Accept a value only when it
parses as a **whole number of seconds within 10 to 900**. Anything else (empty, non-numeric, zero,
negative, fractional, or out of range) is **rejected with a warning naming the offending value, and
the 180s default applies**. There is no unbounded setting: the call is always bounded, and the only
question is by what.

**Degrade, never block.** If `codex` is absent from PATH, exits non-zero, times out, or returns an
unparseable envelope, continue with the Claude draft alone and say which of those happened, naming
the exit status or the timeout. The cross-model stage is an optional check, not a dependency.

**Why a second model rather than a second Claude.** Same-family judges are biased toward
low-perplexity text, and low perplexity is the signature of generic machine prose, so a same-family
critic is biased in the wrong direction for voice work. One model from a different family captures
nearly all the available benefit; a larger panel does not.

## Stage 3: the merge

**The main agent merges**, not a third subagent. Produce three things:

1. **The merged draft.** One version, taking the stronger choice at each point of divergence.
2. **The disagreements, verbatim.** Where the two drafts diverged materially, show both versions.
   Disagreement goes to the human rather than to a vote: with two voters there is no majority, and
   the disagreements are the most informative output of the stage.
3. **Two to four threadable variations.** Concrete alternatives the user can splice in, each named
   for what it changes: open on the concrete case rather than the claim; keep or drop a hedge; add or
   remove a coined term; hand off at the ending rather than summarizing.

Report any PRESERVE entry that either model overrode, with the reason it gave.

## Known limits, stated rather than hidden

- **Register mismatch.** Informal-register imitation verifies far less reliably than formal
  registers. This skill will serve a design doc or a PR description better than the short-form
  register a personal corpus is most likely made of.
- **Style strength and content preservation trade off** against each other. They are competing
  objectives, not a tuning failure. This pipeline picks a point on that curve by putting
  content-shaping first and voice second.
- **Scrubbing a widely circulated word list is itself becoming detectable**, which is a reason the
  editorial tiers lean on structural rules rather than on lexical substitution alone.

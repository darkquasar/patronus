# short-form exemplars

This file ships empty by design, so your voice corpus stays private and never enters the public
Patronus catalogue.

Put your short-form exemplars (posts, threads, messages, anything under roughly 800 words) at one of
these paths, first hit wins:

1. `$PATRONUS_VOICE_DIR/short-form.md`, when that environment variable is set
2. `~/.claude/patronus/voice/short-form.md`, the default user-owned location

Neither is created for you, and neither is ever written by the installer, so a reinstall cannot touch
what you put there. Paste whole pieces rather than excerpts: the pass draws on rhythm and paragraph
movement, which an excerpt loses.

The length band names what these pieces ARE, not what they can be applied to. This pool supplies a
voice, and a voice projects onto any length: rhythm, diction, and how a paragraph turns are the same
whether the finished piece runs 200 words or 4000. The pipeline reads those from here and takes
length, scope, and structure from the draft it was given.

#!/usr/bin/env bash
# work-state-reground — a SessionStart hook (startup|clear|compact) that fires at
# the moments the agent has lost the thread: a fresh start, a /clear, and — most
# importantly — AFTER COMPACTION, when in-context facts have just been squeezed
# out. At those points it tells the agent to reconcile with state that lives on
# DISK, not in the context window, before continuing.
#
# Re-injection keeps the RULE fresh but cannot restore lost FACTS; externalized
# state can. This hook bridges the two: it points the agent at the Beads work-graph
# (what is unblocked / mid-flight) and ai-memory (prior context) — the durable
# stores that survive a compaction the context window did not.
#
# Conditional by design (no-duplicate-native-capability / no dead instructions):
# it only names a source that is actually present, so it never tells the agent to
# run a tool that isn't wired. If neither is present it still re-asserts the
# skill-dispatch rule. Fails open (exit 0).

set -euo pipefail

cues=""

# --- Beads: a project work-graph db (.beads/ in cwd, or BEADS_DB pointing at one).
if [ -n "${BEADS_DB:-}" ] || [ -d ".beads" ]; then
  cues="${cues} Reconcile with the Beads work-graph: run \`bd ready\` to see what is unblocked and what you left mid-flight, and \`bd status\` for the overview — do not rely on memory of the plan that may have been compacted away."
fi

# --- ai-memory: the self-wiring memory recipe (skill body installed, or its binary).
if [ -d "${HOME}/.claude/skills/memory-ai-memory" ] || \
   [ -x "${HOME}/.patronus/bin/ai-memory" ] || command -v ai-memory >/dev/null 2>&1; then
  cues="${cues} Recall ai-memory for context relevant to this work before acting — earlier decisions and facts may have been summarized out of the window."
fi

if [ -n "$cues" ]; then
  msg="You are (re)starting or resuming after a context compaction.${cues} Then proceed under the usual skill-dispatch rule (check for an applicable skill before acting)."
else
  # Neither externalized store is wired — still re-assert the skill rule on resume.
  msg="You are (re)starting or resuming after a context compaction. Re-check for an applicable installed skill before acting, and re-read any plan/state file this work depends on rather than trusting a possibly-compacted memory of it."
fi

escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}
escaped=$(escape_for_json "$msg")

printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "SessionStart",\n    "additionalContext": "%s"\n  }\n}\n' "$escaped"

exit 0

#!/usr/bin/env bash
# work-state-reground — a SessionStart hook (startup|clear|compact) that fires at
# the moments the agent has lost the thread: a fresh start, a /clear, and — most
# importantly — AFTER COMPACTION, when in-context facts have just been squeezed
# out. At those points it tells the agent to reconcile with state that lives on
# DISK, not in the context window, before continuing.
#
# Re-injection keeps the RULE fresh but cannot restore lost FACTS; externalized
# state can. This hook bridges the two: it points the agent at the Ticket
# work-graph (what is unblocked / mid-flight) and the project's lessons
# (docs/lessons — patterns learned from earlier corrections) — the durable stores
# that survive a compaction the context window did not.
#
# Conditional by design (no-duplicate-native-capability / no dead instructions):
# it only names a source that is actually present, so it never tells the agent to
# run a tool that isn't wired. If neither is present it still re-asserts the
# skill-dispatch rule. Fails open (exit 0).

set -euo pipefail

cues=""

# --- Ticket: a project work-graph (.tickets/ in cwd — plain markdown, no db).
# Cue `tk ready` + `tk blocked`, never `tk status`: that is a SETTER
# (`tk status <id> <status>`), not an overview, and cueing it would invite the
# agent to mutate a ticket while trying to orient.
if [ -d ".tickets" ]; then
  cues="${cues} Reconcile with the Ticket work-graph: run \`tk ready\` to see what is unblocked and what you left mid-flight, and \`tk blocked\` to see what is waiting — do not rely on memory of the plan that may have been compacted away."
fi

# --- Lessons: durable patterns captured after corrections (docs/lessons in cwd).
# Cue it only when the directory holds something; an empty docs/lessons would send
# the agent to read nothing. Pairs with the agent-principles Self-Improvement Loop,
# which is what writes these files in the first place.
if [ -d "docs/lessons" ] && [ -n "$(ls -A docs/lessons 2>/dev/null)" ]; then
  cues="${cues} Read \`docs/lessons/\` before your first edit — it records patterns learned from earlier corrections on this project, including ones made in sessions you cannot see."
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

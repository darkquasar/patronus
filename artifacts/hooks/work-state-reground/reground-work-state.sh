#!/usr/bin/env bash
# work-state-reground — a SessionStart hook (startup|clear|compact) that fires at
# the moments the agent has lost the thread: a fresh start, a /clear, and — most
# importantly — AFTER COMPACTION, when in-context facts have just been squeezed
# out. At those points it tells the agent to reconcile with state that lives on
# DISK, not in the context window, before continuing.
#
# Re-injection keeps the RULE fresh but cannot restore lost FACTS; externalized
# state can. This hook bridges the two: it points the agent at the selected
# progress authority, conditionally mentions Ticket, and preserves project lessons
# (docs/lessons — patterns learned from earlier corrections) — the durable stores
# that survive a compaction the context window did not.
#
# Lifecycle recovery and skill-dispatch cues are always present, including when
# no plan or externalized store exists. Ticket and lessons cues are conditional
# on source presence; presence alone does not select the progress authority.
# Fails open (exit 0).

set -euo pipefail

cues=" Re-read effective instructions, the accepted brief/plan and latest coordinator handoff. Recover from the sole progress authority recorded at plan acceptance: qualified native state that actually owns delegated runs first (Pi today); otherwise the explicit selected tracker, existing tracker, or durable plan-associated record. UI todos alone are not run-owning state. Inspect actual source and retained results; do not replay settled work or relaunch live workers. Existing approval remains valid within scope. Only the coordinator admits transitions and updates shared progress; leaves return complete evidence and stop. Existing host/resource limits apply without a mandatory root-wide numerical budget or new ledger. Unknown state or settlement blocks the affected transition; retain partial evidence."

# --- Ticket: a project work-graph (.tickets/ in cwd — plain markdown, no db).
# Cue `tk ready` + `tk blocked`, never `tk status`: that is a SETTER
# (`tk status <id> <status>`), not an overview, and cueing it would invite the
# agent to mutate a ticket while trying to orient.
if [ -d ".tickets" ]; then
  cues="${cues} A \`.tickets/\` directory is present, but presence does not select it as authority. Only when Ticket is the selected authority, resolve the available \`tk\` binary and read \`tk ready\` and \`tk blocked\` to reconcile state; \`tk status\` is a setter, not an overview. If native run-owning state is authoritative, tickets receive references only. Respect actual ignore/tracking policy; do not force-add ignored tickets or create a competing graph. If \`tk\` is unavailable, preserve findings in the existing handoff rather than silently switching authority."
fi

# --- Lessons: durable patterns captured after corrections (docs/lessons in cwd).
# Cue it only when the directory holds something; an empty docs/lessons would send
# the agent to read nothing. Pairs with the agent-principles Self-Improvement Loop,
# which is what writes these files in the first place.
if [ -d "docs/lessons" ] && [ -n "$(ls -A docs/lessons 2>/dev/null)" ]; then
  cues="${cues} Read \`docs/lessons/\` before your first edit — it records patterns learned from earlier corrections on this project, including ones made in sessions you cannot see."
fi

msg="You are (re)starting or resuming after a context compaction.${cues} Then proceed under the usual skill-dispatch rule (check for an applicable skill before acting)."

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

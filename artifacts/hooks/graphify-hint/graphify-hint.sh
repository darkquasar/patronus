#!/usr/bin/env bash
# Reference form of the graphify-hint PreToolUse nudge (the manifest inlines this
# into `hook.command`). NOT placed on disk by Patronus — there is no `files:`/`script:`
# entry, so placeHookScript never runs and the Codex no-script-dir constraint is never
# hit. Fails OPEN (exit 0) always.
set -euo pipefail

input="$(cat)"
cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"

# Only nudge for search-style commands; stay silent otherwise.
case "$cmd" in
  *grep*|*"rg "*|*"find "*) ;;
  *) exit 0 ;;
esac

# Only nudge if a graph exists and graphify is available.
if [ -f graphify-out/graph.json ] && command -v graphify >/dev/null 2>&1; then
  echo "hint: a graphify graph exists — consider graphify query \"<question>\" instead of grepping." >&2
fi
exit 0

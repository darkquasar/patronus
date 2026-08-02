#!/usr/bin/env bash
# Reference form of the graphify-hint PreToolUse nudge (the manifest inlines this
# into `hook.command`). NOT placed on disk by Patronus — there is no `files:`/`script:`
# entry, so placeHookScript never runs and the Codex no-script-dir constraint is never
# hit. Fails OPEN (exit 0) always.
set -euo pipefail

input="$(cat)"
cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
pat="$(printf '%s' "$input" | jq -r '.tool_input.pattern // empty' 2>/dev/null || true)"

# A search is EITHER a Bash grep/rg/find command OR any native Grep/Glob tool call
# (those carry the search term in .tool_input.pattern, so a non-empty pattern means
# the call is a search regardless of content). Stay silent otherwise.
hit=
case "$cmd" in
  *grep*|*"rg "*|*"find "*) hit=1 ;;
esac
[ -n "$pat" ] && hit=1
[ -n "$hit" ] || exit 0

# Only nudge if a graph exists and graphify is available.
if [ -f graphify-out/graph.json ] && command -v graphify >/dev/null 2>&1; then
  echo "hint: a graphify graph exists — consider graphify query \"<question>\" instead of grepping." >&2
fi
exit 0

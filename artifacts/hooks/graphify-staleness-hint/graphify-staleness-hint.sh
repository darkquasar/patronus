#!/usr/bin/env bash
# Reference form of the graphify-staleness-hint PostToolUse nudge (the manifest
# inlines this into `hook.command`). NOT placed on disk by Patronus — there is no
# `files:`/`script:` entry. Fails OPEN (exit 0) always; a single stat, no tree walk.
set -euo pipefail

g=graphify-out/graph.json
[ -f "$g" ] || exit 0
command -v git >/dev/null 2>&1 || exit 0

head="$(git log -1 --format=%ct 2>/dev/null)" || exit 0
[ -n "$head" ] || exit 0

# Graph mtime: BSD stat (-f %m) on macOS, GNU stat (-c %Y) on Linux.
gmt="$(stat -f %m "$g" 2>/dev/null || stat -c %Y "$g" 2>/dev/null)" || exit 0

if [ "$gmt" -lt "$head" ]; then
  echo "hint: graphify-out/graph.json is older than HEAD — rebuild with graphify . so queries reflect current code." >&2
fi
exit 0

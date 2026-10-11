//go:build artifactbehavior

package main

// Historical payload retained for the excluded hook behavior definition.
const fixClaudeHookBytes = `#!/usr/bin/env bash
# Fixture hook: lists the installed skills, the way skills-heartbeat does.
set -euo pipefail
names=""
if [ -d "${HOME}/.claude/skills" ]; then
  for d in "${HOME}/.claude/skills"/*/; do
    [ -d "$d" ] || continue
    names="${names:+$names, }$(basename "$d")"
  done
fi
printf '{"installedSkills":"%s"}\n' "$names"
`

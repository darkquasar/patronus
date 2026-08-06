#!/bin/bash
# gitleaks-guard — a PreToolUse Bash hook that scans the STAGED diff for secrets
# at COMMIT time only. gitleaks is a commit-/repo-state scanner, so it is wired at
# its design altitude (a commit guard) rather than per-edit: the hook fires on
# every Bash command, but this script does real work only when the command is a
# `git commit`. See the threat model in block-secrets — gitleaks covers case (c),
# committing a secret, which is its core competency.
#
# Finds the `gitleaks` binary the `gitleaks` recipe installs into ~/.patronus/bin,
# resolving that placed path FIRST so the guard works even when ~/.patronus/bin is not
# on $PATH (a GUI-launched agent inherits a frozen PATH that rarely includes it), then
# falling back to a PATH lookup. If gitleaks is found by neither, the guard fails OPEN
# (exit 0) with a warning rather than blocking every commit — a missing scanner must
# not wedge the workflow.

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command')

# Only act on git commits; let every other Bash command through untouched.
if ! printf '%s' "$COMMAND" | grep -qE '\bgit\b.*\bcommit\b'; then
  exit 0
fi

# Prefer the Patronus-placed binary (resolvable regardless of $PATH), then PATH.
if [ -x "${HOME}/.patronus/bin/gitleaks" ]; then
  GITLEAKS="${HOME}/.patronus/bin/gitleaks"
elif command -v gitleaks >/dev/null 2>&1; then
  GITLEAKS=gitleaks
else
  echo "gitleaks-guard: gitleaks binary not found in ~/.patronus/bin or on PATH; skipping secret scan (install the gitleaks recipe)." >&2
  exit 0
fi

# Scan only what is about to be committed: the staged diff. --exit-code 1 makes
# gitleaks return non-zero on a finding, which we surface as a block (exit 2).
if ! git diff --cached -U0 | "$GITLEAKS" stdin --no-banner --exit-code 1 >/dev/null 2>&1; then
  echo "BLOCKED: gitleaks detected a likely secret in the staged changes. Unstage or remove it before committing (run 'git diff --cached | gitleaks stdin -v' to see the finding)." >&2
  exit 2
fi

exit 0

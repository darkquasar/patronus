#!/usr/bin/env bash
# Bootstrap a uv-managed virtualenv at the repo root for running notebooks.
#
# Creates <repo-root>/.venv and installs the packages you pass. uv is the
# installer because it is an order of magnitude faster than pip and resolves
# deterministically.
#
# BOTH side effects are OPT-IN and assume the caller has already ASKED the user:
#   --install-uv       install uv via the official installer if it is missing
#                      (runs: curl -LsSf https://astral.sh/uv/install.sh | sh)
#   --gitignore-venv   add ".venv/" to the repo's .gitignore (only when in a git
#                      repo; a venv should never be committed)
# Without a flag the script does NOT take that action -- it prints what it would
# have done and how to opt in. Consent stays with the user, not the script.
#
# Usage:
#   bootstrap_env.sh [--install-uv] [--gitignore-venv] [pkg ...]
#
# Examples:
#   bootstrap_env.sh marimo matplotlib networkx numpy pandas altair
#   bootstrap_env.sh --install-uv marimo                    # install uv if missing
#   bootstrap_env.sh --gitignore-venv marimo                # gitignore .venv/
#   bootstrap_env.sh --install-uv --gitignore-venv marimo   # both (after asking)
#
# Exit codes:
#   0  success
#   2  uv not installed and --install-uv not passed (caller should ASK the
#      user for permission, then re-run with --install-uv)
set -euo pipefail

INSTALL_UV=0
GITIGNORE_VENV=0
PKGS=()
for arg in "$@"; do
  case "$arg" in
    --install-uv) INSTALL_UV=1 ;;
    --gitignore-venv) GITIGNORE_VENV=1 ;;
    *) PKGS+=("$arg") ;;
  esac
done

# Detect whether we are inside a git repo. Repo root, or current dir otherwise.
if ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  IN_REPO=1
else
  ROOT="$(pwd)"
  IN_REPO=0
fi
VENV="$ROOT/.venv"

# 1. Ensure uv is available. Do NOT silently install it: surface the choice.
if ! command -v uv >/dev/null 2>&1; then
  if [ "$INSTALL_UV" -eq 1 ]; then
    echo "Installing uv (official installer: curl -LsSf https://astral.sh/uv/install.sh | sh)..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
  else
    echo "ERROR: uv is not installed (uv is the fast Python package manager this uses)." >&2
    echo "ASK the user for permission first. On yes, re-run with --install-uv, which runs:" >&2
    echo "  curl -LsSf https://astral.sh/uv/install.sh | sh   # or: brew install uv" >&2
    exit 2
  fi
fi

# 2. Create the venv at the repo root (idempotent).
# Pin the interpreter: bare `uv venv` reuses the first suitable Python it finds,
# so a host whose system python is old (macOS ships 3.9.6) yields a venv marimo
# rejects. uv downloads 3.12 on demand when it is missing.
if [ ! -d "$VENV" ]; then
  echo "Creating venv at $VENV"
  uv venv --python 3.12 "$VENV"
else
  echo "Reusing existing venv at $VENV"
fi

# 3. Gitignore the venv -- ONLY when asked (--gitignore-venv) AND inside a repo.
GI="$ROOT/.gitignore"
if [ "$GITIGNORE_VENV" -eq 1 ]; then
  if [ "$IN_REPO" -eq 1 ]; then
    if ! { [ -f "$GI" ] && grep -qxF ".venv/" "$GI"; }; then
      printf '\n# uv virtualenv for notebooks\n.venv/\n' >> "$GI"
      echo "Added .venv/ to $GI"
    else
      echo ".venv/ is already gitignored in $GI"
    fi
  else
    echo "NOTE: --gitignore-venv passed but not inside a git repo; nothing to gitignore."
  fi
elif [ "$IN_REPO" -eq 1 ] && ! { [ -f "$GI" ] && grep -qxF ".venv/" "$GI"; }; then
  echo "NOTE: in a git repo and $VENV is NOT gitignored." >&2
  echo "      A venv should never be committed. To gitignore it, ASK the user, then" >&2
  echo "      re-run with --gitignore-venv (or add '.venv/' to .gitignore yourself)." >&2
fi

# 4. Install requested packages into the venv.
if [ "${#PKGS[@]}" -gt 0 ]; then
  echo "Installing into venv: ${PKGS[*]}"
  uv pip install --python "$VENV/bin/python" "${PKGS[@]}"
fi

echo
echo "Done."
echo "  Python : $VENV/bin/python"
echo "  Run    : $VENV/bin/python <script>      (or: source $VENV/bin/activate)"
echo "  marimo : $VENV/bin/marimo edit <notebook.py>"

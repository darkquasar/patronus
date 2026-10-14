#!/usr/bin/env bash
# Open a generated notebook as a read-only marimo APP (background server + URL),
# rather than dropping the user into the full editable marimo environment.
# This is the default way to hand a finished explanation to the user: they get a
# clean running app to read and interact with (sliders, tangles), not an editor.
#
# Usage:  bash serve.sh <notebook.py> [port] [venv]
set -eu

NB="${1:?usage: serve.sh <notebook.py> [port] [venv]}"
PORT="${2:-2718}"
VENV="${3:-.venv}"
MARIMO="$VENV/bin/marimo"
LOG="$(mktemp -t marimoserve.XXXXXX).log"

[ -x "$MARIMO" ] || { echo "no marimo at $MARIMO — bootstrap the env first"; exit 2; }

# --watch: reload the running app whenever the .py changes on disk, so the next
# iteration shows up live without restarting the server. Uses watchdog if
# installed, else polls ~1s.
nohup "$MARIMO" run "$NB" --port "$PORT" --headless --watch > "$LOG" 2>&1 &
PID=$!
sleep 4
if curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/" 2>/dev/null | grep -q 200; then
  echo "Serving $NB"
  echo "  -> http://localhost:$PORT   (read-only app, pid $PID)"
  echo "  stop with: kill $PID"
else
  echo "Failed to start; see $LOG"; cat "$LOG"; exit 1
fi

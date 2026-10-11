#!/bin/sh
# Invented guard: capture via a caller-owned fixture path, never print input.
cat > "$1"
if grep -q PLANTED_FIXTURE_SECRET "$1"; then
  echo 'fixture guard blocked' >&2
  exit 2
fi
exit 0

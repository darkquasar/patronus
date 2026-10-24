#!/usr/bin/env bash
# install.sh version selection, offline: a stub curl records the requested URL
# and fails, so no release is downloaded or installed.
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin"
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
for a in "$@"; do case "$a" in https://*) echo "$a" >> "$CURL_LOG" ;; esac; done
exit 22
EOF
chmod +x "$work/bin/curl"

shells="sh bash"
command -v dash >/dev/null 2>&1 && shells="$shells dash"

# run SHELL EXPECTED_EXIT EXPECTED_URL_PREFIX ARGS...; an empty prefix means no download.
run() {
  local shell=$1 want_exit=$2 want_url=$3; shift 3
  : > "$work/curl.log"
  local got=0
  CURL_LOG="$work/curl.log" PATH="$work/bin:$PATH" "$shell" "$root/scripts/install.sh" "$@" > /dev/null 2>&1 || got=$?
  [ "$got" -eq "$want_exit" ] || { echo "$shell install.sh $*: exit $got, want $want_exit" >&2; exit 1; }
  local url
  url=$(head -n 1 "$work/curl.log")
  if [ -z "$want_url" ]; then
    [ -z "$url" ] || { echo "$shell install.sh $*: unexpected download $url" >&2; exit 1; }
  else
    case "$url" in "$want_url"*) ;; *) echo "$shell install.sh $*: url '$url', want prefix $want_url" >&2; exit 1 ;; esac
  fi
}

rel=https://github.com/darkquasar/patronus/releases
for shell in $shells; do
  run "$shell" 1 "$rel/latest/download/patronus-"
  run "$shell" 1 "$rel/download/v2.7.0/patronus-" --version v2.7.0
  run "$shell" 1 "$rel/download/v2.7.0/patronus-" --version=2.7.0
  run "$shell" 1 "$rel/download/v2.8.0-rc.1/patronus-" --version v2.8.0-rc.1
  PATRONUS_VERSION=v2.6.1 run "$shell" 1 "$rel/download/v2.6.1/patronus-"
  for bad in v2oops.7oops.0oops 2.7 v2.7.0/../x 'v2.7.0 ' v2.7.0+build latest2; do
    run "$shell" 2 "" --version "$bad"
  done
  run "$shell" 2 "" --version
  run "$shell" 2 "" --bogus
done
echo "install.sh version selection: ok ($shells)"

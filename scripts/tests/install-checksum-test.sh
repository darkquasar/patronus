#!/usr/bin/env bash
# Real installer; curl returns fixtures and mv blocks any attempted installation.
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/bin" "$work/home/.local/bin"
printf 'existing user binary\n' > "$work/home/.local/bin/patronus"
cp "$work/home/.local/bin/patronus" "$work/sentinel"
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
set -eu
url= dest=
while [ "$#" -gt 0 ]; do
  case "$1" in https://*) url=$1;; -o) shift; dest=$1;; esac
  shift
done
printf '%s\n' "$url" >> "$CURL_LOG"
case "$url" in
  */checksums.txt)
    case "$CHECKSUM_CASE" in
      mismatch) printf '%064d  %s\n' 0 "$ASSET" > "$dest";;
      missing) printf '%064d  other-binary\n' 0 > "$dest";;
    esac;;
  *) printf 'invented binary bytes\n' > "$dest";;
esac
EOF
cat > "$work/bin/mv" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$INSTALL_LOG"
exit 97
EOF
chmod +x "$work/bin/curl" "$work/bin/mv"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in arm64|aarch64) arch=arm64;; x86_64|amd64) arch=amd64;; *) exit 1;; esac
export ASSET="patronus-$os-$arch"
for shell in sh bash; do
  for kind in mismatch missing; do
    for version in latest v2.7.1 v2.8.0-docs.1; do
      : > "$work/urls"
      : > "$work/install.log"
      result=0
      HOME="$work/home" PATRONUS_VERSION="$version" CHECKSUM_CASE="$kind" CURL_LOG="$work/urls" INSTALL_LOG="$work/install.log" \
        PATH="$work/bin:$PATH" "$shell" "$root/scripts/install.sh" > "$work/output" 2>&1 || result=$?
      [ ! -s "$work/install.log" ] || { echo 'installer attempted a move despite invalid checksum' >&2; exit 1; }
      case "$kind" in mismatch) reason='checksum mismatch';; missing) reason='no checksum found';; esac
      grep -F "$reason" "$work/output" >/dev/null || { cat "$work/output" >&2; echo "wrong rejection reason: expected $reason" >&2; exit 1; }
      [ "$result" -ne 0 ] || { echo 'checksum rejection failed' >&2; exit 1; }
      cmp "$work/home/.local/bin/patronus" "$work/sentinel"
      [ "$(wc -l < "$work/urls" | tr -d ' ')" = 2 ]
      case "$version" in latest) base=https://github.com/darkquasar/patronus/releases/latest/download;;
        *) base="https://github.com/darkquasar/patronus/releases/download/$version";; esac
      [ "$(sed -n '1p' "$work/urls")" = "$base/$ASSET" ]
      [ "$(sed -n '2p' "$work/urls")" = "$base/checksums.txt" ]
    done
  done
done
echo 'install.sh checksum rejection, URL selection and user sentinel: ok'

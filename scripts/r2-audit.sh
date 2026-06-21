#!/usr/bin/env bash
# r2-audit.sh — READ-ONLY audit: for every built catalog tarball, compare its
# local sha256 to the sha256 recorded on the published R2 object, and report every
# item whose bytes DRIFTED (would trip the publish immutability guard). Makes only
# signed HEAD requests — it never writes to R2.
#
# Creds come from a gitignored .r2-audit.env (AWS_ACCESS_KEY_ID,
# AWS_SECRET_ACCESS_KEY, R2_ENDPOINT, BUCKET). Run from the repo root:
#   bash scripts/r2-audit.sh
#
# Output: one line per tarball — OK (identical), NEW (not yet published),
# NOMETA (published without sha metadata), or DRIFT (published with a different
# sha — needs a version bump). Exits 0 always; this is a report, not a gate.

set -euo pipefail

ENVF="${R2_AUDIT_ENV:-.env}"
if [ ! -f "$ENVF" ]; then
  echo "missing $ENVF (needs R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY / R2_ACCOUNT_ID)" >&2
  exit 2
fi
# Parse KEY = VALUE / KEY=VALUE lines robustly (the file may have spaces around
# '=' and/or quotes, which `source` cannot handle). We never echo the values.
while IFS= read -r _line || [ -n "$_line" ]; do
  case "$_line" in ''|\#*) continue;; esac
  _k="${_line%%=*}"; _v="${_line#*=}"
  # trim surrounding whitespace from key and value
  _k="$(printf '%s' "$_k" | sed -E 's/^[[:space:]]*//; s/[[:space:]]*$//')"
  _v="$(printf '%s' "$_v" | sed -E 's/^[[:space:]]*//; s/[[:space:]]*$//; s/^"//; s/"$//; s/^'"'"'//; s/'"'"'$//')"
  [ -n "$_k" ] || continue
  export "$_k=$_v"
done < "$ENVF"
unset _line _k _v
# Accept either the R2_* names (this repo's .env) or the AWS_* names, and derive
# the endpoint from the account id + the bucket default. The script never echoes
# these values; only signatures + non-secret keys/status are printed.
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-${R2_ACCESS_KEY_ID:-}}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-${R2_SECRET_ACCESS_KEY:-}}"
R2_ENDPOINT="${R2_ENDPOINT:-https://${R2_ACCOUNT_ID:-}.r2.cloudflarestorage.com}"
BUCKET="${BUCKET:-patronus-registry}"
: "${AWS_ACCESS_KEY_ID:?}" "${AWS_SECRET_ACCESS_KEY:?}" "${R2_ACCOUNT_ID:?need R2_ACCOUNT_ID or R2_ENDPOINT}" "${BUCKET:?}"

# host (no scheme) for the Host header + signing.
HOST="${R2_ENDPOINT#https://}"; HOST="${HOST#http://}"; HOST="${HOST%%/*}"
REGION="auto"; SERVICE="s3"

hmac_sha256() { # $1=hexkey-or-literal (prefixed) ; reads data on stdin
  openssl dgst -sha256 -mac HMAC -macopt "$1" -binary
}
sha256_hex() { openssl dgst -sha256 -hex | sed 's/^.*= //'; }

# Build fresh so we compare exactly what publish would.
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT
go run ./cmd/patronus build --out "$BUILD_DIR" >/dev/null

drift=()
new=0 ok=0 nometa=0

# Signed HEAD; echoes the x-amz-meta-sha256 value (empty if absent / 404 handled by caller).
r2_head_sha() { # $1 = key (e.g. catalog/foo/1.0.0/foo-1.0.0.tar.gz)
  local key="$1"
  local amzdate datestamp
  amzdate="$(date -u +%Y%m%dT%H%M%SZ)"
  datestamp="${amzdate%%T*}"
  local payload_hash; payload_hash="$(printf '' | sha256_hex)" # empty body
  local canonical_uri="/${BUCKET}/${key}"
  local canonical_headers="host:${HOST}\nx-amz-content-sha256:${payload_hash}\nx-amz-date:${amzdate}\n"
  local signed_headers="host;x-amz-content-sha256;x-amz-date"
  local canonical_request
  canonical_request="$(printf 'HEAD\n%s\n\n%b\n%s\n%s' "$canonical_uri" "$canonical_headers" "$signed_headers" "$payload_hash")"
  local creq_hash; creq_hash="$(printf '%s' "$canonical_request" | sha256_hex)"
  local scope="${datestamp}/${REGION}/${SERVICE}/aws4_request"
  local sts; sts="$(printf 'AWS4-HMAC-SHA256\n%s\n%s\n%s' "$amzdate" "$scope" "$creq_hash")"
  # derive signing key
  local kDate kRegion kService kSigning
  kDate="$(printf '%s' "$datestamp" | hmac_sha256 "key:AWS4${AWS_SECRET_ACCESS_KEY}" | xxd -p -c256)"
  kRegion="$(printf '%s' "$REGION" | hmac_sha256 "hexkey:$kDate" | xxd -p -c256)"
  kService="$(printf '%s' "$SERVICE" | hmac_sha256 "hexkey:$kRegion" | xxd -p -c256)"
  kSigning="$(printf '%s' "aws4_request" | hmac_sha256 "hexkey:$kService" | xxd -p -c256)"
  local sig; sig="$(printf '%s' "$sts" | hmac_sha256 "hexkey:$kSigning" | xxd -p -c256)"
  local auth="AWS4-HMAC-SHA256 Credential=${AWS_ACCESS_KEY_ID}/${scope}, SignedHeaders=${signed_headers}, Signature=${sig}"

  curl -sS -I "https://${HOST}/${BUCKET}/${key}" \
    -H "Host: ${HOST}" \
    -H "x-amz-content-sha256: ${payload_hash}" \
    -H "x-amz-date: ${amzdate}" \
    -H "Authorization: ${auth}" 2>/dev/null
}

while IFS= read -r f; do
  key="${f#"$BUILD_DIR"/}"               # catalog/<name>/<ver>/<file>.tar.gz
  local_sha="$(sha256_hex < "$f")"
  resp="$(r2_head_sha "$key" || true)"
  status="$(printf '%s' "$resp" | awk 'NR==1{print $2}')"
  remote_sha="$(printf '%s' "$resp" | tr -d '\r' | awk -F': ' 'tolower($1)=="x-amz-meta-sha256"{print $2}')"
  if [ "$status" = "404" ] || [ -z "$status" ] && [ -z "$remote_sha" ]; then
    printf 'NEW    %s\n' "$key"; new=$((new+1)); continue
  fi
  if [ -z "$remote_sha" ]; then
    printf 'NOMETA %s\n' "$key"; nometa=$((nometa+1)); continue
  fi
  if [ "$remote_sha" != "$local_sha" ]; then
    printf 'DRIFT  %s  (r2=%s local=%s)\n' "$key" "${remote_sha:0:12}" "${local_sha:0:12}"
    drift+=("$key")
  else
    ok=$((ok+1))
  fi
done < <(find "$BUILD_DIR/catalog" -type f -name '*.tar.gz' | sort)

echo "----"
printf 'summary: %d ok, %d new, %d nometa, %d DRIFT\n' "$ok" "$new" "$nometa" "${#drift[@]}"
if [ "${#drift[@]}" -gt 0 ]; then
  echo "drifted items (need a version bump):"
  printf '  %s\n' "${drift[@]}"
fi

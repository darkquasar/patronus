#!/usr/bin/env bash
# Publish write-once package objects. The caller publishes the index only on success.
set -euo pipefail
: "${R2_ENDPOINT:?R2_ENDPOINT is required}" "${BUCKET:?BUCKET is required}"
cd "${1:?usage: publish-packages.sh REGISTRY_DIR}"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export AWS_PAGER=''

s3() {
  aws s3api "$@" --endpoint-url "$R2_ENDPOINT" --bucket "$BUCKET"
}

# Return 2 only for explicit service-level absence; auth/transport errors are fatal.
head_key() {
  if s3 head-object --key "$1" > "$work/head" 2> "$work/error"; then
    return 0
  fi
  if grep -Eq 'An error occurred \((404|NoSuchKey)\) when calling the HeadObject operation' "$work/error"; then
    return 2
  fi
  cat "$work/error" >&2
  return 1
}

hash_file() {
  sha256sum "$1" | cut -d' ' -f1
}

valid_provenance() {
  jq -e 'type == "object" and .schemaVersion == 1 and
    (.repositoryCommit | type == "string" and length > 0) and
    ((has("ciRun") | not) or (.ciRun | type == "string"))' "$1" > /dev/null
}

verify_key() {
  local key=$1 expected=$2 kind=$3 recorded
  recorded=$(jq -er '.Metadata.sha256 | strings | select(test("^[0-9a-f]{64}$"))' "$work/head") || {
    echo "Missing or invalid checksum metadata: $key" >&2; return 1;
  }
  if [ "$kind" != provenance ] && [ "$recorded" != "$expected" ]; then
    echo "IMMUTABILITY VIOLATION: $key differs from the build" >&2
    return 1
  fi
  if [ "$kind" != archive ]; then
    # Use the recorded digest to validate retained provenance, which may originate
    # from an earlier identical rebuild. Checksum sidecars must match this build.
    aws s3api get-object --endpoint-url "$R2_ENDPOINT" --bucket "$BUCKET" \
      --key "$key" "$work/download" > /dev/null
    [ "$(hash_file "$work/download")" = "$recorded" ] || {
      echo "Downloaded sidecar checksum mismatch: $key" >&2; return 1;
    }
    if [ "$kind" = provenance ]; then valid_provenance "$work/download"; fi
  fi
}

publish_key() {
  local key=$1 kind=$2 content_type=$3 expected status
  expected=$(hash_file "$key")
  if head_key "$key"; then
    verify_key "$key" "$expected" "$kind"
    return
  else
    status=$?
    [ "$status" -eq 2 ] || return 1
  fi
  if ! s3 put-object --key "$key" --body "$key" --content-type "$content_type" \
      --metadata "sha256=$expected" --if-none-match '*' > /dev/null 2> "$work/error"; then
    if ! grep -Eq 'An error occurred \((409|412|ConditionalRequestConflict|PreconditionFailed)\) when calling the PutObject operation' "$work/error"; then
      cat "$work/error" >&2
      return 1
    fi
    # Another publisher won. Verify its object; never retry an unconditional PUT.
  fi
  head_key "$key" || { echo "Cannot verify published object: $key" >&2; return 1; }
  verify_key "$key" "$expected" "$kind"
}

[ -d packages ] || exit 0
# Materialize enumeration so find/sort failures cannot disappear in a substitution.
find packages -type f -name '*.tar.gz' -print | LC_ALL=C sort > "$work/keys"
while IFS= read -r key; do
  digest=$(hash_file "$key")
  printf 'sha256:%s\n' "$digest" > "$work/checksum"
  cmp -s "$work/checksum" "$key.sha256" || {
    echo "Invalid local checksum sidecar: $key.sha256" >&2; exit 1;
  }
  valid_provenance "$key.provenance.json"
  publish_key "$key" archive application/gzip
  publish_key "$key.sha256" checksum text/plain
  publish_key "$key.provenance.json" provenance application/json
done < "$work/keys"

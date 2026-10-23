#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/catalog"

check() {
  local schema=$1 enabled=$2 expected=$3
  printf '{"schemaVersion":%s}\n' "$schema" > "$work/catalog/index.json"
  local output
  if [ "$enabled" = unset ]; then
    output=$(env -u DIRECTORY_PACKAGES_ENABLED bash "$root/scripts/check-catalog-publication.sh" "$work")
  else
    output=$(DIRECTORY_PACKAGES_ENABLED="$enabled" bash "$root/scripts/check-catalog-publication.sh" "$work")
  fi
  [ "$output" = "publish=$expected" ] || { echo "unexpected output: $output" >&2; exit 1; }
  # The workflow consumes this exact output to gate every production upload.
  local uploads=0
  if [ "$output" = publish=true ]; then uploads=$((uploads + 1)); fi
  if [ "$expected" = false ]; then [ "$uploads" -eq 0 ]; else [ "$uploads" -eq 1 ]; fi
}
check 2 unset false
check 2 false false
check 2 true true
check 1 unset true
check 1 false true
check 1 true true
for invalid in '{}' '{"schemaVersion":"2"}' '{"schemaVersion":2.5}' '{"schemaVersion":0}' '{"schemaVersion":3}' '{"schemaVersion":null}' '{"schemaVersion":1} {"schemaVersion":2}' 'invalid'; do
  printf '%s\n' "$invalid" > "$work/catalog/index.json"
  if bash "$root/scripts/check-catalog-publication.sh" "$work" > "$work/output" 2>/dev/null; then
    echo "accepted invalid index: $invalid" >&2; exit 1
  fi
  [ ! -s "$work/output" ] || { echo 'invalid index produced publication output' >&2; exit 1; }
done
# Guard coverage must stay in sync with all production upload steps.
python3 - "$root/.github/workflows/publish-catalog.yml" <<'PY'
import pathlib, sys
text = pathlib.Path(sys.argv[1]).read_text()
assert 'id: guard' in text
# Production environment variables are available only in the publish job.
uploads = [step for step in text.split('      - ')
           if 'aws s3api put-object' in step or 'bash scripts/publish-packages.sh registry' in step]
assert uploads, 'no upload steps inspected'
build, publish = text.split('  publish:\n', 1)
assert "needs: [authorize, build]" in publish
assert 'environment: production' in publish
assert 'aws s3api put-object' not in build
assert 'secrets.' not in build
assert 'DIRECTORY_PACKAGES_ENABLED' not in build
assert 'github.ref == \'refs/heads/main\'' in publish
assert 'github.repository == \'darkquasar/patronus\'' in publish
assert 'tags:' not in text
assert 'bash scripts/check-catalog-publication.sh registry >> "$GITHUB_OUTPUT"' in publish
for step in uploads:
    assert "if: steps.guard.outputs.publish == 'true'" in step
    assert 'secrets.R2_ACCESS_KEY_ID' in step and 'secrets.R2_SECRET_ACCESS_KEY' in step
assert publish.count('secrets.R2_ACCESS_KEY_ID') == len(uploads)
assert publish.index('bash scripts/check-catalog-publication.sh registry') < publish.index('bash scripts/publish-packages.sh registry')
assert publish.index('bash scripts/publish-packages.sh registry') < publish.index('aws s3api put-object')
index = publish.split('      - name: Publish discovery index', 1)[1]
assert "steps.eligibility.outputs.index == 'true'" in index
assert index.index('git fetch --no-tags origin main') < index.index('aws s3api put-object')
PY
printf 'catalog publication guard tests passed\n'

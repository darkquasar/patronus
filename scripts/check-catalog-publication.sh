#!/usr/bin/env bash
# Keep the live catalog readable until a directory-capable binary is released.
set -euo pipefail
schema=$(jq -esr 'select(length == 1) | .[0].schemaVersion | select(. == 1 or . == 2)' "$1/catalog/index.json")
if [ "$schema" -ge 2 ] && [ "${DIRECTORY_PACKAGES_ENABLED:-false}" != true ]; then
  echo 'Release a directory-capable Patronus binary before enabling schema 2 publication.' >&2
  echo 'publish=false'
else
  echo 'publish=true'
fi

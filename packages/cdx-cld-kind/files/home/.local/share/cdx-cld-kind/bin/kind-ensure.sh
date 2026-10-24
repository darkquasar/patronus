#!/usr/bin/env bash
# Ensure a running kind cluster. Runs in the background on every start, as uid 1000.
# Recreates the cluster when its node container is not running (e.g. after a restart).
set -uo pipefail
export PATH="$HOME/.local/bin:$PATH"
name="${KIND_CLUSTER_NAME:-kind}"
node="${name}-control-plane"
exec >>"$HOME/.kind-boot.log" 2>&1
echo "--- kind-ensure $(date -u)"

for _ in $(seq 1 60); do docker info >/dev/null 2>&1 && break; sleep 2; done
docker info >/dev/null 2>&1 || { echo "docker daemon not ready; giving up"; exit 1; }

if kind get clusters 2>/dev/null | grep -qx "$name" &&
   ! docker ps --format '{{.Names}}' | grep -qx "$node"; then
  echo "cluster '$name' has no running node; recreating"
  kind delete cluster --name "$name" >/dev/null 2>&1 || true
fi

if kind get clusters 2>/dev/null | grep -qx "$name"; then
  echo "cluster '$name' already running"
elif kind create cluster --name "$name"; then
  echo "cluster '$name' ready"
else
  echo "kind create failed"
fi

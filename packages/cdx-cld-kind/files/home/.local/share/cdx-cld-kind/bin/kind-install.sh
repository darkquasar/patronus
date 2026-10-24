#!/usr/bin/env bash
# Install pinned kind and kubectl into ~/.local/bin. Runs once at create, as uid 1000.
set -euo pipefail
KIND_VERSION="${KIND_VERSION:-v0.33.0}"
KUBECTL_VERSION="${KUBECTL_VERSION:-v1.37.1}"
case "$(uname -m)" in
  aarch64|arm64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *) echo "kind-install: unsupported arch $(uname -m)" >&2; exit 1 ;;
esac
bin="$HOME/.local/bin"
mkdir -p "$bin"

if ! command -v kind >/dev/null 2>&1; then
  curl -fsSLo "$bin/kind" "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-${arch}"
  chmod +x "$bin/kind"
fi
if ! command -v kubectl >/dev/null 2>&1; then
  curl -fsSLo "$bin/kubectl" "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${arch}/kubectl"
  chmod +x "$bin/kubectl"
fi
echo "kind-install: $("$bin/kind" version | head -1); kubectl ${KUBECTL_VERSION}"

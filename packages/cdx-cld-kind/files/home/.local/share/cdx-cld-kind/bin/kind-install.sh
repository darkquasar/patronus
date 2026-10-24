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

# fetch URL SUM_URL DEST: download to a temp file, check it against the
# upstream sha256 file, then move it into place.
fetch() {
  local tmp want
  tmp="$(mktemp)"
  curl -fsSLo "$tmp" "$1"
  want="$(curl -fsSL "$2" | awk '{print $1}')"
  echo "${want}  ${tmp}" | sha256sum -c - >/dev/null || { echo "kind-install: checksum mismatch for $1" >&2; rm -f "$tmp"; exit 1; }
  install -m 0755 "$tmp" "$3"
  rm -f "$tmp"
}

if ! command -v kind >/dev/null 2>&1; then
  url="https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-${arch}"
  fetch "$url" "${url}.sha256sum" "$bin/kind"
fi
if ! command -v kubectl >/dev/null 2>&1; then
  url="https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${arch}/kubectl"
  fetch "$url" "${url}.sha256" "$bin/kubectl"
fi
echo "kind-install: $(PATH="$bin:$PATH" kind version | head -1); kubectl ${KUBECTL_VERSION}"

#!/bin/sh
# Patronus installer for Unix (Linux/macOS).
#   curl -fsSL https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.sh | sh -s -- --version v2.7.0
#
# Downloads a release binary for this OS/arch (the latest unless --version or
# PATRONUS_VERSION names one), verifies its sha256 against that release's
# checksums.txt, and installs it to ~/.local/bin (or /usr/local/bin when
# writable). No prerequisites beyond curl + sha256.
set -eu

REPO="darkquasar/patronus"

# --- pick the release ---
version="${PATRONUS_VERSION:-latest}"
while [ $# -gt 0 ]; do
  case "$1" in
    --version) [ $# -ge 2 ] || { echo "--version needs a value" >&2; exit 2; }; version=$2; shift 2 ;;
    --version=*) version=${1#--version=}; shift ;;
    *) echo "unknown argument: $1 (usage: install.sh [--version vX.Y.Z])" >&2; exit 2 ;;
  esac
done
if [ "$version" != latest ]; then
  printf '%s\n' "$version" | grep -Eqx 'v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?' ||
    { echo "invalid version: $version (want vX.Y.Z[-suffix] or latest)" >&2; exit 2; }
  case "$version" in v*) ;; *) version="v${version}" ;; esac
fi
if [ "$version" = latest ]; then
  BASE="https://github.com/${REPO}/releases/latest/download"
else
  BASE="https://github.com/${REPO}/releases/download/${version}"
fi

# --- detect platform ---
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux) os=linux ;;
  darwin) os=darwin ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

bin="patronus-${os}-${arch}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${bin} (${version})..."
curl -fsSL "${BASE}/${bin}" -o "${tmp}/patronus" ||
  { echo "download failed: ${BASE}/${bin}" >&2; exit 1; }
curl -fsSL "${BASE}/checksums.txt" -o "${tmp}/checksums.txt"

# --- verify sha256 ---
want=$(grep " ${bin}\$" "${tmp}/checksums.txt" | awk '{print $1}')
if [ -z "$want" ]; then
  echo "no checksum found for ${bin}" >&2; exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "${tmp}/patronus" | awk '{print $1}')
else
  got=$(shasum -a 256 "${tmp}/patronus" | awk '{print $1}')
fi
if [ "$got" != "$want" ]; then
  echo "checksum mismatch (got $got, want $want)" >&2; exit 1
fi

# --- install ---
chmod +x "${tmp}/patronus"
dest="${HOME}/.local/bin"
if [ -w /usr/local/bin ]; then dest="/usr/local/bin"; fi
mkdir -p "$dest"
mv "${tmp}/patronus" "${dest}/patronus"

echo "Installed patronus to ${dest}/patronus"
case ":${PATH}:" in
  *":${dest}:"*) ;;
  *) echo "Add ${dest} to your PATH to run 'patronus'." ;;
esac

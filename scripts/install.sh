#!/bin/sh
# Patronus installer for Unix (Linux/macOS).
#   curl -fsSL https://raw.githubusercontent.com/darkquasar/patronus/main/scripts/install.sh | sh
#
# Downloads the latest release binary for this OS/arch, verifies its sha256
# against the published checksums.txt, and installs it to ~/.local/bin (or
# /usr/local/bin when writable). No prerequisites beyond curl + sha256.
set -eu

REPO="darkquasar/patronus"
BASE="https://github.com/${REPO}/releases/latest/download"

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

echo "Downloading ${bin}..."
curl -fsSL "${BASE}/${bin}" -o "${tmp}/patronus"
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

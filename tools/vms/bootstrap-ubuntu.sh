#!/usr/bin/env bash
set -euo pipefail

test "$(uname -s)" = Linux
test "$(uname -m)" = aarch64
test "$(id -u)" -ne 0

sudo apt-get update
sudo env DEBIAN_FRONTEND=noninteractive apt-get install -y \
  build-essential procps curl file git ca-certificates

installer="${TMPDIR:-/tmp}/patronus-homebrew-install.sh"
curl --fail --location --silent --show-error --max-time 120 \
  https://raw.githubusercontent.com/Homebrew/install/959bdd728192d78c47287c177796c3d86d111f09/install.sh \
  --output "$installer"
printf '%s  %s\n' \
  5f333bbe53bc490e51e7ccb1df8779b3dd6ee73a1a7379efda216edb08ccb148 \
  "$installer" | sha256sum --check
NONINTERACTIVE=1 /bin/bash "$installer"

{
  uname -a
  cat /etc/os-release
  /home/linuxbrew/.linuxbrew/bin/brew --version
  git -C /home/linuxbrew/.linuxbrew/Homebrew rev-parse HEAD
  dpkg-query -W build-essential procps curl file git ca-certificates
} | tee "$HOME/patronus-vm-tools.txt"

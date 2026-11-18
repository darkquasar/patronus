#!/usr/bin/env bash
set -euo pipefail
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
export HOMEBREW_NO_AUTO_UPDATE=1 HOMEBREW_NO_ANALYTICS=1
uname -a
id
brew --version
printf '%s\n' 'VM setup passed. No application package test was requested.'

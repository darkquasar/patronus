# Reference Homebrew formula for Patronus.
#
# This is a SHAPE reference, not a live tap. Publishing means copying this into a
# `homebrew-patronus` tap repo and filling in the per-release url/sha256 (ideally
# auto-bumped by a release step, e.g. dawidd6/action-homebrew-bump-formula). The
# url/sha256 placeholders below are replaced per release.
class Patronus < Formula
  desc "Meta-scaffolder for AI coding environments (Claude Code, Codex, OpenCode)"
  homepage "https://github.com/darkquasar/patronus"
  version "0.0.0" # replaced per release

  on_macos do
    on_arm do
      url "https://github.com/darkquasar/patronus/releases/download/v#{version}/patronus-darwin-arm64"
      sha256 "REPLACE_WITH_DARWIN_ARM64_SHA256"
    end
    on_intel do
      url "https://github.com/darkquasar/patronus/releases/download/v#{version}/patronus-darwin-amd64"
      sha256 "REPLACE_WITH_DARWIN_AMD64_SHA256"
    end
  end

  def install
    bin.install Dir["patronus-*"].first => "patronus"
  end

  test do
    assert_match "patronus", shell_output("#{bin}/patronus --version")
  end
end

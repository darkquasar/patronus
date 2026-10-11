#!/usr/bin/env python3
"""Scan the current staged diff before literal Bash git commits.

Self-contained placed script. No shell execution. Native event/trust semantics
remain runtime-pending. This is not a general shell parser or an egress control.
"""
import json
import os
import shlex
import shutil
import subprocess
import sys


def commit_roots(command, cwd):
    lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|()\n")
    lexer.whitespace = " \t\r"
    lexer.whitespace_split = True
    segments, segment = [], []
    for token in lexer:
        if token and all(char in ";&|()\n" for char in token):
            segments.append(segment)
            segment = []
        else:
            segment.append(token)
    segments.append(segment)
    roots = []
    for words in segments:
        if not words:
            continue
        if words[0] == "cd" and len(words) == 2:
            cwd = os.path.abspath(os.path.join(cwd, words[1]))
            continue
        if words[0] == "command":
            words = words[1:]
        if not words or os.path.basename(words[0]) != "git":
            continue
        root, index, unsupported = cwd, 1, False
        while index < len(words) and words[index].startswith("-"):
            option = words[index]
            if option in ("-C", "-c") and index + 1 < len(words):
                value = words[index + 1]
                if option == "-C":
                    root = os.path.abspath(os.path.join(root, value))
                elif value.split("=", 1)[0] not in ("user.name", "user.email"):
                    unsupported = True  # repository/config overrides cannot be ignored
                index += 2
            elif option in ("--no-pager", "--no-optional-locks"):
                index += 1
            elif option in ("--git-dir", "--work-tree", "--namespace", "--config-env"):
                unsupported = True
                index += 2  # consume the value, so a following commit is not missed
            else:
                unsupported = True
                index += 1
        if index < len(words) and words[index] == "commit":
            if unsupported or any(char in root for char in "$`"):
                raise ValueError("unsupported git commit context; use literal git -C <repo> commit")
            roots.append(root)
    return list(dict.fromkeys(roots))


def main():
    try:
        payload = json.load(sys.stdin)
        if not isinstance(payload, dict) or payload.get("tool_name") != "Bash":
            raise ValueError("unsupported matched payload: expected tool_name Bash")
        data = payload.get("tool_input")
        if not isinstance(data, dict) or not isinstance(data.get("command"), str):
            raise ValueError("malformed payload: expected tool_input.command string")
        cwd = payload.get("cwd", os.getcwd())
        if not isinstance(cwd, str) or not os.path.isabs(cwd):
            raise ValueError("malformed payload: cwd must be an absolute directory")
        roots = commit_roots(data["command"], cwd)
    except (ValueError, TypeError) as exc:
        message = "malformed payload: provide valid JSON" if isinstance(exc, json.JSONDecodeError) else str(exc)
        print("gitleaks-guard-cx: " + message + "; guard cannot verify this command", file=sys.stderr)
        return 2
    if not roots:
        return 0
    placed = os.path.join(os.path.expanduser("~"), ".patronus", "bin", "gitleaks")
    scanner = placed if os.path.isfile(placed) and os.access(placed, os.X_OK) else shutil.which("gitleaks")
    if not scanner:
        print("gitleaks-guard-cx: install the gitleaks recipe (binary missing in ~/.patronus/bin and PATH); commit not verified", file=sys.stderr)
        return 2
    git = shutil.which("git")
    if not git:
        print("gitleaks-guard-cx: install Git to read the staged diff; commit not verified", file=sys.stderr)
        return 2
    try:
        for root in roots:
            staged = subprocess.run(
                [git, "-C", root, "diff", "--cached", "--no-ext-diff", "--no-textconv", "-U0"],
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, timeout=15, check=False,
            )
            if staged.returncode:
                print("gitleaks-guard-cx: cannot read staged diff; check the repository and Git prerequisites", file=sys.stderr)
                return 2
            scanned = subprocess.run(
                [scanner, "stdin", "--no-banner", "--exit-code", "1", "--redact"],
                input=staged.stdout, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                timeout=20, check=False,
            )
            if scanned.returncode == 1:
                print("BLOCKED: gitleaks found a likely secret in staged changes; unstage/remove it before committing", file=sys.stderr)
                return 2
            if scanned.returncode:
                print("gitleaks-guard-cx: scanner failed; check its version/configuration before retrying the commit", file=sys.stderr)
                return 2
    except (OSError, subprocess.TimeoutExpired):
        print("gitleaks-guard-cx: staged diff or scanner execution failed/timed out; commit not verified", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())

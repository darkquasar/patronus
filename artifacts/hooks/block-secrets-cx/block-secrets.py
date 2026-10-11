#!/usr/bin/env python3
"""Best-effort write-time guard. Invented payload contract, native runtime-pending.

Self-contained because the hook adapter places only hook.script, not siblings.
Only newly written content is scanned; this does not guard reading or egress.
"""
import json
import re
import sys

PATTERNS = (
    r"-----BEGIN [A-Z ]*PRIVATE KEY-----",
    r"(?:AKIA|ASIA)[0-9A-Z]{16}",
    r"gh[pousr]_[A-Za-z0-9]{36,}",
    r"glpat-[A-Za-z0-9_-]{20,}",
    r"xox[baprs]-[A-Za-z0-9-]{10,}",
    r"sk-[A-Za-z0-9]{32,}",
    r"AIza[0-9A-Za-z_-]{35}",
)


def patch_additions(patch):
    lines = patch.strip("\r\n").splitlines()
    if len(lines) < 3 or lines[0] != "*** Begin Patch" or lines[-1] != "*** End Patch":
        raise ValueError("malformed apply_patch: expected Begin Patch / End Patch envelope")
    mode, files, additions = None, 0, []
    for line in lines[1:-1]:
        header = re.fullmatch(r"\*\*\* (Add|Update|Delete) File: (.+)", line)
        if header:
            mode = header[1]
            files += 1
            continue
        if line.startswith("*** Move to: ") and mode == "Update":
            continue
        if line == "*** End of File" and mode == "Update":
            continue
        if mode == "Update" and (line == "@@" or line.startswith("@@ ")):
            continue
        if mode in ("Add", "Update") and line.startswith("+"):
            additions.append(line[1:])
            continue
        if mode == "Update" and line.startswith(("-", " ")):
            continue  # removals and unchanged context are not newly written
        raise ValueError("unsupported apply_patch line: use Add/Update/Delete File sections")
    if not files:
        raise ValueError("malformed apply_patch: no file sections")
    return "\n".join(additions)


def required_string(obj, key):
    if not isinstance(obj, dict) or not isinstance(obj.get(key), str):
        raise ValueError("malformed payload: expected string field " + key)
    return obj[key]


def written_content(payload):
    if not isinstance(payload, dict):
        raise ValueError("malformed payload: expected JSON object")
    tool = required_string(payload, "tool_name")
    data = payload.get("tool_input")
    # tool_name takes precedence over a registration matcher alias (e.g. Write).
    if tool == "apply_patch":
        if isinstance(data, str):
            patch = data
        elif isinstance(data, dict):
            present = [key for key in ("patch", "input") if key in data]
            if len(present) != 1:
                raise ValueError("malformed payload: apply_patch needs one patch or input string")
            patch = required_string(data, present[0])
        else:
            raise ValueError("malformed payload: apply_patch needs a string or object tool_input")
        return patch_additions(patch)
    if tool == "Write":
        return required_string(data, "content")
    if tool == "Edit":
        return required_string(data, "new_string")
    if tool == "MultiEdit":
        if not isinstance(data, dict) or not isinstance(data.get("edits"), list):
            raise ValueError("malformed payload: MultiEdit needs an edits array")
        return "\n".join(required_string(edit, "new_string") for edit in data["edits"])
    raise ValueError("unsupported matched tool: qualify its payload before enabling protection")


def main():
    try:
        content = written_content(json.load(sys.stdin))
    except (ValueError, TypeError) as exc:
        # Never echo payload bytes or JSON decoder snippets (which may be secrets).
        message = "malformed payload: provide valid JSON" if isinstance(exc, json.JSONDecodeError) else str(exc)
        print("block-secrets-cx: " + message + "; guard cannot verify this write", file=sys.stderr)
        return 2
    if any(re.search(pattern, content) for pattern in PATTERNS):
        print("BLOCKED: likely secret in newly written content; use a secret store or environment reference", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())

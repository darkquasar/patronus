"""Opt-in Pi cold-start check. Sends read-only RPC commands, never a prompt."""

import argparse
import json
import os
from pathlib import Path
import selectors
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cwd", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--approve-project", action="store_true", help="trust disposable project files for this Pi run")
    args = parser.parse_args()
    commands = ("get_state", "get_commands")
    argv = ["pi", "--mode", "rpc", "--no-session"]
    if args.approve_project:
        argv.append("--approve")
    process = subprocess.Popen(
        argv,
        cwd=args.cwd.resolve(),
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    responses = {}
    stderr = bytearray()
    try:
        for command in commands:
            process.stdin.write(json.dumps({"id": command, "type": command}).encode() + b"\n")
        process.stdin.flush()
        pending = bytearray()
        deadline = time.monotonic() + 45
        with selectors.DefaultSelector() as selector:
            selector.register(process.stdout, selectors.EVENT_READ, "stdout")
            selector.register(process.stderr, selectors.EVENT_READ, "stderr")
            while len(responses) < len(commands) and time.monotonic() < deadline:
                for key, _ in selector.select(1):
                    data = os.read(key.fileobj.fileno(), 65536)
                    if not data:
                        selector.unregister(key.fileobj)
                        if key.data == "stdout":
                            raise RuntimeError("Pi exited before responding: " + stderr.decode(errors="replace"))
                        continue
                    if key.data == "stderr":
                        stderr.extend(data)
                        continue
                    pending.extend(data)
                    while b"\n" in pending:
                        line, _, rest = pending.partition(b"\n")
                        pending = bytearray(rest)
                        try:
                            message = json.loads(line)
                        except json.JSONDecodeError:
                            continue
                        if message.get("type") == "response" and message.get("id") in commands:
                            responses[message["id"]] = message
        if set(responses) != set(commands) or not all(value.get("success") for value in responses.values()):
            raise RuntimeError(f"RPC readiness failed: {responses}")
        if responses["get_state"]["data"].get("isStreaming"):
            raise RuntimeError("unexpected model stream during read-only smoke")
        names = {command["name"] for command in responses["get_commands"]["data"]["commands"]}
        workflows = {"skill:workflow-research-pi", "skill:workflow-implement-pi", "skill:workflow-peer-review-pi"}
        if not workflows <= names:
            raise RuntimeError(f"missing workflow commands: {workflows - names}")
        args.output.write_text(json.dumps(responses, indent=2) + "\n")
        print(json.dumps({"rpc": "passed", "commands": len(names), "workflows": sorted(workflows)}))
    finally:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
        stderr.extend(process.stderr.read())
        args.output.with_suffix(".stderr.log").write_bytes(stderr)
        process.stdin.close()
        process.stdout.close()
        process.stderr.close()


if __name__ == "__main__":
    main()

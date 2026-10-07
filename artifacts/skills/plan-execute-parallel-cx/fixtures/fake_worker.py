#!/usr/bin/env python3
"""Invented fake writer for supervisor tests. It never calls a model or CLI backend.

The supervisor passes its bindings through PATRONUS_SUPERVISOR_* environment
variables. Flags choose one scripted behavior so tests can drive the real
supervisor with real local processes.
"""
import argparse
import json
import os
import signal
import subprocess
import sys
import time


def env(name):
    return os.environ["PATRONUS_SUPERVISOR_" + name]


def write(rel, content):
    path = os.path.join(env("ROOT"), rel)
    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(content)


def git(*args):
    subprocess.run(["git", *args], cwd=env("ROOT"), check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--write", action="append", default=[], help="path=content, reported")
    ap.add_argument("--unreported", action="append", default=[], help="path=content, not reported")
    ap.add_argument("--require", action="append", default=[], help="path=content that must already exist")
    ap.add_argument("--require-absent", action="append", default=[])
    ap.add_argument("--require-input", default="", help="predecessor task whose patch must be supplied")
    ap.add_argument("--sleep", type=float, default=0.0)
    ap.add_argument("--sleep-after-result", type=float, default=0.0)
    ap.add_argument("--ignore-sigterm", action="store_true")
    ap.add_argument("--spawn-delayed", default="", help="path=content written by a detached descendant")
    ap.add_argument("--delay", type=float, default=3.0)
    ap.add_argument("--result", choices=["ok", "none", "malformed", "wrong-base", "failed"], default="ok")
    ap.add_argument("--stage", action="store_true")
    ap.add_argument("--commit", action="store_true")
    ap.add_argument("--fail-attempts", type=int, default=0, help="attempts <= N exit 1 without a result")
    args = ap.parse_args()

    if args.ignore_sigterm:
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
    attempt = int(env("ATTEMPT"))
    for spec in args.require:
        rel, want = spec.split("=", 1)
        with open(os.path.join(env("ROOT"), rel), encoding="utf-8") as handle:
            if handle.read() != want:
                sys.exit("required input content differs: " + rel)
    for rel in args.require_absent:
        if os.path.lexists(os.path.join(env("ROOT"), rel)):
            sys.exit("stale residue from another attempt: " + rel)
    if args.require_input:
        with open(env("INPUTS"), encoding="utf-8") as handle:
            inputs = json.load(handle)
        if args.require_input not in [item["task"] for item in inputs["predecessors"]]:
            sys.exit("predecessor capture not supplied")

    reported = []
    for spec in args.write:
        rel, content = spec.split("=", 1)
        write(rel, content)
        reported.append(rel)
    for spec in args.unreported:
        rel, content = spec.split("=", 1)
        write(rel, content)
    if args.spawn_delayed:
        rel, content = args.spawn_delayed.split("=", 1)
        target = os.path.join(env("ROOT"), rel)
        code = ("import sys,time\ntime.sleep(float(sys.argv[1]))\n"
                "open(sys.argv[2],'w').write(sys.argv[3])\n")
        subprocess.Popen([sys.executable, "-c", code, str(args.delay), target, content],
                         stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                         stderr=subprocess.DEVNULL, close_fds=True)
    if args.stage:
        git("add", "-A")
    if args.commit:
        git("add", "-A")
        git("-c", "user.name=Fake", "-c", "user.email=fake@example.invalid",
            "commit", "-q", "-m", "invented drift")
    if attempt <= args.fail_attempts:
        sys.exit("invented failure on attempt %d" % attempt)
    if args.sleep:
        time.sleep(args.sleep)

    result = {"runId": env("RUN_ID"), "task": env("TASK"), "attempt": attempt,
              "root": env("ROOT"), "base": env("BASE"), "status": "complete",
              "changedFiles": reported, "summary": "invented fake worker"}
    if args.result == "wrong-base":
        result["base"] = "0" * 40
    if args.result == "failed":
        result["status"] = "failed"
    if args.result == "malformed":
        with open(env("RESULT"), "w", encoding="utf-8") as handle:
            handle.write("{not json")
    elif args.result != "none":
        with open(env("RESULT"), "w", encoding="utf-8") as handle:
            json.dump(result, handle)
    if args.sleep_after_result:
        time.sleep(args.sleep_after_result)


if __name__ == "__main__":
    main()

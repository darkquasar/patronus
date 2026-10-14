#!/usr/bin/env python3
"""Allocate Git worktrees for native writers, without launching agents.

Usage: prepare-worktrees.py ALLOCATION.json [--validate]
Python standard library only. No scheduler, worker lifecycle, patch capture,
integration or cleanup. allocation.json records allocations, not execution.
"""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys

NAME_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
OID_RE = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
GIT_LOCATION_VARS = ("GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY",
                     "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR", "GIT_NAMESPACE",
                     "GIT_PREFIX")


class Invalid(Exception):
    """Invalid request, rejected before allocation."""


class Blocked(Exception):
    """Allocation failed; retain any created worktrees and records."""


def git(root, *args, check=True):
    env = {key: value for key, value in os.environ.items() if key not in GIT_LOCATION_VARS}
    proc = subprocess.run(["git", *args], cwd=root, env=env,
                          stdin=subprocess.DEVNULL, capture_output=True)
    if check and proc.returncode:
        raise Blocked("git %s failed: %s" % (
            " ".join(args), proc.stderr.decode("utf-8", "replace").strip()))
    return proc


def git_out(root, *args):
    return os.fsdecode(git(root, *args).stdout).strip()


def within(path, root):
    path, root = os.path.realpath(path), os.path.realpath(root)
    return path == root or path.startswith(root.rstrip(os.sep) + os.sep)


def claim(raw):
    if not isinstance(raw, str) or not raw or raw.startswith("/") or "\\" in raw or "\0" in raw:
        raise Invalid("owned path must be a non-empty relative POSIX path")
    parts = raw.rstrip("/").split("/")
    if any(part in ("", ".", "..", ".git") for part in parts) or raw.endswith("//"):
        raise Invalid("owned path escapes, is unnormalized or names .git")
    return raw


def overlaps(a, b):
    a, b = a.rstrip("/"), b.rstrip("/")
    return a == b or a.startswith(b + "/") or b.startswith(a + "/")


def check_destination(path):
    if os.path.realpath(path) != os.path.abspath(path):
        raise Invalid("allocation destination contains a symlink: %s" % path)
    cursor = path
    while not os.path.lexists(cursor):
        cursor = os.path.dirname(cursor)
    if not os.path.isdir(cursor):
        raise Invalid("allocation destination has a non-directory ancestor: %s" % cursor)


def write_record(path, record):
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump(record, handle, indent=2, sort_keys=True)
        handle.write("\n")
    os.replace(tmp, path)


def load(path):
    with open(path, "rb") as handle:
        raw = handle.read()
    try:
        req = json.loads(raw)
    except ValueError as err:
        raise Invalid("malformed allocation request: %s" % err)
    if not isinstance(req, dict) or req.get("schemaVersion") != 1:
        raise Invalid("request needs schemaVersion 1")
    allowed = {"schemaVersion", "runId", "repository", "baseCommit", "brief",
               "worktreeRoot", "runDir", "tasks"}
    unsupported = sorted(set(req) - allowed)
    if unsupported:
        raise Invalid("allocation-only rejects unsupported fields: %s" % ", ".join(unsupported))
    if not isinstance(req.get("runId"), str) or not NAME_RE.fullmatch(req["runId"]):
        raise Invalid("invalid runId")
    for key in ("repository", "worktreeRoot", "runDir"):
        if not isinstance(req.get(key), str) or not os.path.isabs(req[key]):
            raise Invalid("%s must be an absolute path" % key)
        req[key] = os.path.abspath(req[key])
    try:
        top = os.path.realpath(git_out(req["repository"], "rev-parse", "--show-toplevel"))
    except (Blocked, OSError):
        raise Invalid("repository is not a Git worktree")
    req["repository"] = top
    base = req.get("baseCommit")
    if not isinstance(base, str) or not OID_RE.fullmatch(base):
        raise Invalid("baseCommit must be a full commit object id")
    probe = git(top, "cat-file", "-t", base, check=False)
    if probe.returncode or probe.stdout.strip() != b"commit":
        raise Invalid("baseCommit must name a commit")
    brief = req.get("brief")
    if not isinstance(brief, dict) or not isinstance(brief.get("path"), str) or not os.path.isabs(brief["path"]):
        raise Invalid("brief must bind an absolute path and sha256")
    try:
        with open(brief["path"], "rb") as handle:
            digest = hashlib.sha256(handle.read()).hexdigest()
    except OSError as err:
        raise Invalid("brief unreadable: %s" % err)
    if brief.get("sha256") != digest:
        raise Invalid("brief sha256 mismatch")
    common = os.path.realpath(os.path.join(top, git_out(top, "rev-parse", "--git-common-dir")))
    checkouts = [os.fsdecode(entry[len(b"worktree "):])
                 for entry in git(top, "worktree", "list", "--porcelain", "-z").stdout.split(b"\0")
                 if entry.startswith(b"worktree ")]
    for key in ("worktreeRoot", "runDir"):
        check_destination(req[key])
        if within(req[key], common) or any(within(req[key], checkout) for checkout in checkouts):
            raise Invalid("%s must be outside registered checkouts and Git directory" % key)
    if within(req["runDir"], req["worktreeRoot"]) or within(req["worktreeRoot"], req["runDir"]):
        raise Invalid("runDir and worktreeRoot must be disjoint")
    tasks = req.get("tasks")
    if not isinstance(tasks, list) or not tasks:
        raise Invalid("tasks must be a non-empty list")
    run_root = os.path.join(req["worktreeRoot"], req["runId"])
    if os.path.lexists(req["runDir"]) or os.path.lexists(run_root):
        raise Invalid("run identity already has evidence or worktrees")
    bindings, claims = {}, []
    links = [os.fsdecode(entry.split(b"\t", 1)[1])
             for entry in git(top, "ls-tree", "-r", "-z", base).stdout.split(b"\0")
             if entry.startswith(b"120000 ")]
    for task in tasks:
        if not isinstance(task, dict) or not isinstance(task.get("key"), str) or not NAME_RE.fullmatch(task["key"]):
            raise Invalid("invalid task key")
        unsupported = sorted(set(task) - {"key", "ownedPaths"})
        if unsupported:
            raise Invalid("%s: allocation-only rejects unsupported fields: %s" % (
                task["key"], ", ".join(unsupported)))
        key = task["key"]
        if key in bindings:
            raise Invalid("duplicate task key: %s" % key)
        owned = task.get("ownedPaths")
        if not isinstance(owned, list) or not owned:
            raise Invalid("ownedPaths must be a non-empty list")
        owned = [claim(item) for item in owned]
        for item in owned:
            if any(item.rstrip("/") == link or item.startswith(link + "/") or (
                    item.endswith("/") and link.startswith(item)) for link in links):
                raise Invalid("owned path traverses or contains a base symlink")
            if any(overlaps(item, previous) for previous in claims):
                raise Invalid("owned paths overlap")
            claims.append(item)
        branch = "patronus-allocation/%s/%s" % (req["runId"], key)
        if git(top, "check-ref-format", "--branch", branch, check=False).returncode:
            raise Invalid("invalid allocation branch: %s" % branch)
        if git(top, "show-ref", "--verify", "--quiet", "refs/heads/" + branch, check=False).returncode == 0:
            raise Invalid("allocation branch already exists: %s" % branch)
        bindings[key] = {"root": os.path.join(run_root, key), "branch": branch,
                         "base": base, "ownedPaths": owned, "status": "pending"}
    return req, {"schemaVersion": 1, "runId": req["runId"], "mode": "worktrees-only",
                 "status": "allocating", "repository": top, "baseCommit": base,
                 "requestSha256": hashlib.sha256(raw).hexdigest(), "brief": brief,
                 "tasks": bindings}


def allocate(req, record):
    os.makedirs(req["runDir"])
    record_path = os.path.join(req["runDir"], "allocation.json")
    write_record(record_path, record)
    try:
        os.makedirs(os.path.join(req["worktreeRoot"], req["runId"]))
        for binding in record["tasks"].values():
            git(req["repository"], "worktree", "add", "-q", "-b", binding["branch"],
                binding["root"], binding["base"])
            binding["status"] = "allocated"
            write_record(record_path, record)
        record["status"] = "allocated"
        write_record(record_path, record)
        return 0, record
    except BaseException as err:
        record.update(status="blocked", reason="%s: %s" % (type(err).__name__, err))
        write_record(record_path, record)
        return 3, record


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("allocation")
    parser.add_argument("--validate", action="store_true", help="check only; no allocation")
    args = parser.parse_args()
    if os.name != "posix":
        print(json.dumps({"status": "blocked", "reason": "unsupported-platform"}))
        return 3
    try:
        req, record = load(args.allocation)
        if args.validate:
            report = {"status": "valid", "mode": "worktrees-only", "runId": req["runId"],
                      "requestSha256": record["requestSha256"], "tasks": record["tasks"]}
            code = 0
        else:
            code, report = allocate(req, record)
    except Invalid as err:
        code, report = 2, {"status": "invalid", "reason": str(err)}
    except (Blocked, OSError) as err:
        code, report = 3, {"status": "blocked", "reason": str(err)}
    print(json.dumps(report, sort_keys=True))
    return code


if __name__ == "__main__":
    sys.exit(main())

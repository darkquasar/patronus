#!/usr/bin/env python3
"""Isolated parallel-writer supervisor for the Patronus Codex lane.

Standard library only. POSIX only (Linux and macOS); other platforms are blocked.

  supervisor.py validate REQUEST.json
  supervisor.py run REQUEST.json
  supervisor.py integrate REQUEST.json --grant-integration

Each writer attempt runs in a new worktree and branch created from the recorded
base commit, never in the lead checkout. The supervisor owns the worker process
group: a timeout or a surviving descendant is stopped with SIGTERM, a bounded
grace period and SIGKILL, and nothing is validated, retried or captured until the
stop is confirmed. Unknown stop state blocks the run and preserves all evidence.
Worktrees are never removed. Integration is serial, disabled by default, needs an
explicit grant and never commits.

Ownership is cooperative: Git observes the worktree, not every file the process
can reach. This is not an operating-system filesystem sandbox.

Exit codes: 0 complete, 1 task failure or integration refusal, 2 invalid request,
3 blocked (unknown stop state, unsupported platform or supervisor error).
"""
import argparse
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
from datetime import datetime, timezone

SCHEMA_VERSION = 1
EXIT_OK, EXIT_FAILED, EXIT_INVALID, EXIT_BLOCKED = 0, 1, 2, 3
NAME_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
OID_RE = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
MAX_TIMEOUT, MAX_GRACE, MAX_ATTEMPTS, MAX_CONCURRENCY = 86400.0, 600.0, 5, 8
POLL = 0.05
# Repository-location variables from a calling hook must not redirect our Git.
GIT_LOCATION_VARS = ("GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY",
                     "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR", "GIT_NAMESPACE",
                     "GIT_PREFIX")
# Settled failures that an explicit retry budget may re-attempt in a fresh worktree.
RETRYABLE = {"timeout", "live-descendant", "worker-exit-nonzero", "missing-result",
             "malformed-result", "binding-mismatch", "worker-reported-failure",
             "head-drift", "ownership-violation", "staged-residue", "ignored-residue",
             "unreported-change", "result-mismatch"}


class RequestError(Exception):
    """The request is invalid; nothing has been created."""


class BlockedError(Exception):
    """Continuation is unsafe; evidence is preserved."""


def utc_now():
    return datetime.now(timezone.utc).isoformat()


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    with open(path, "rb") as handle:
        return sha256_bytes(handle.read())


def clean_env(extra=None):
    env = {k: v for k, v in os.environ.items() if k not in GIT_LOCATION_VARS}
    env.update(extra or {})
    return env


def git(cwd, *args, env=None, check=True):
    proc = subprocess.run(["git", *args], cwd=cwd, env=env or clean_env(),
                          stdin=subprocess.DEVNULL, capture_output=True)
    if check and proc.returncode != 0:
        raise BlockedError("git %s failed in %s: %s" % (
            " ".join(args), cwd, proc.stderr.decode("utf-8", "replace").strip()))
    return proc


def git_out(cwd, *args, env=None):
    return git(cwd, *args, env=env).stdout.decode("utf-8", "surrogateescape").strip()


def write_json(path, data):
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump(data, handle, indent=2, sort_keys=True)
        handle.write("\n")
    os.replace(tmp, path)


def is_within(path, root):
    path, root = os.path.realpath(path), os.path.realpath(root)
    return path == root or path.startswith(root.rstrip(os.sep) + os.sep)


# ---------------------------------------------------------------- request ---

def norm_claim(raw):
    """Return a normalized repository-relative claim; a trailing / claims a directory."""
    if not isinstance(raw, str) or not raw or raw.startswith("/") or "\\" in raw or "\0" in raw:
        raise RequestError("owned path must be a non-empty relative POSIX path: %r" % (raw,))
    is_dir = raw.endswith("/")
    parts = raw[:-1].split("/") if is_dir else raw.split("/")
    if any(part in ("", ".", "..") for part in parts) or ".git" in parts:
        raise RequestError("owned path escapes, is unnormalized or names .git: %r" % (raw,))
    return "/".join(parts) + ("/" if is_dir else "")


def claim_covers(claim, path):
    return path.startswith(claim) if claim.endswith("/") else path == claim


def claims_overlap(a, b):
    if a.rstrip("/") == b.rstrip("/"):
        return True
    return (a.endswith("/") and b.startswith(a)) or (b.endswith("/") and a.startswith(b))


def number(value, name, low, high):
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not low < value <= high:
        raise RequestError("%s must be a number in (%s, %s]" % (name, low, high))
    return float(value)


def topo_order(tasks):
    order, done, keys = [], set(), [t["key"] for t in tasks]
    preds = {t["key"]: t["predecessors"] for t in tasks}
    while len(order) < len(keys):
        ready = [k for k in keys if k not in done and all(p in done for p in preds[k])]
        if not ready:
            raise RequestError("predecessor graph has a cycle")
        order.append(ready[0])
        done.add(ready[0])
    return order


def load_request(path):
    """Validate and normalize a request. Raises RequestError before any mutation."""
    path = os.path.abspath(path)
    try:
        with open(path, "rb") as handle:
            raw = handle.read()
        req = json.loads(raw)
    except (OSError, ValueError) as err:
        raise RequestError("unreadable or malformed request: %s" % err)
    if not isinstance(req, dict) or req.get("schemaVersion") != SCHEMA_VERSION:
        raise RequestError("request must be an object with schemaVersion %d" % SCHEMA_VERSION)
    if not NAME_RE.match(str(req.get("runId", ""))):
        raise RequestError("runId must match %s" % NAME_RE.pattern)
    for key in ("repository", "worktreeRoot", "runDir"):
        if not isinstance(req.get(key), str) or not os.path.isabs(req[key]):
            raise RequestError("%s must be an absolute path" % key)
    try:
        top = git_out(req["repository"], "rev-parse", "--show-toplevel")
    except (BlockedError, OSError):
        raise RequestError("repository is not a Git work tree")
    top = os.path.realpath(top)
    base = req.get("baseCommit", "")
    if not isinstance(base, str) or not OID_RE.match(base):
        raise RequestError("baseCommit must be a full object id")
    if git(top, "cat-file", "-e", base + "^{commit}", check=False).returncode != 0:
        raise RequestError("baseCommit does not name a commit")
    brief = req.get("brief")
    if not isinstance(brief, dict) or not isinstance(brief.get("path"), str):
        raise RequestError("brief must bind a path and sha256")
    try:
        brief_sha = sha256_file(brief["path"])
    except OSError as err:
        raise RequestError("brief unreadable: %s" % err)
    if brief.get("sha256") != brief_sha:
        raise RequestError("brief sha256 mismatch")
    common = os.path.realpath(os.path.join(top, git_out(top, "rev-parse", "--git-common-dir")))
    for key in ("worktreeRoot", "runDir"):
        if is_within(req[key], top) or is_within(req[key], common):
            raise RequestError("%s must be outside the lead checkout and Git directory" % key)
    if is_within(req["runDir"], req["worktreeRoot"]) or is_within(req["worktreeRoot"], req["runDir"]):
        raise RequestError("runDir and worktreeRoot must be disjoint")
    concurrency = req.get("maxConcurrency", 1)
    if isinstance(concurrency, bool) or not isinstance(concurrency, int) or not 1 <= concurrency <= MAX_CONCURRENCY:
        raise RequestError("maxConcurrency must be an integer in [1, %d]" % MAX_CONCURRENCY)
    raw_tasks = req.get("tasks")
    if not isinstance(raw_tasks, list) or not raw_tasks:
        raise RequestError("tasks must be a non-empty list")
    tasks, seen = [], set()
    for raw_task in raw_tasks:
        if not isinstance(raw_task, dict) or not NAME_RE.match(str(raw_task.get("key", ""))):
            raise RequestError("each task needs a key matching %s" % NAME_RE.pattern)
        key = raw_task["key"]
        if key in seen:
            raise RequestError("duplicate task key %s" % key)
        seen.add(key)
        command = raw_task.get("command")
        if not isinstance(command, list) or not command or not all(isinstance(c, str) and c for c in command):
            raise RequestError("%s: command must be a non-empty argv list" % key)
        owned = raw_task.get("ownedPaths")
        if not isinstance(owned, list) or not owned:
            raise RequestError("%s: ownedPaths must be a non-empty list" % key)
        preds = raw_task.get("predecessors", [])
        if not isinstance(preds, list) or not all(isinstance(p, str) for p in preds) or key in preds:
            raise RequestError("%s: predecessors must be other task keys" % key)
        attempts = raw_task.get("maxAttempts", 1)
        if isinstance(attempts, bool) or not isinstance(attempts, int) or not 1 <= attempts <= MAX_ATTEMPTS:
            raise RequestError("%s: maxAttempts must be an integer in [1, %d]" % (key, MAX_ATTEMPTS))
        tasks.append({"key": key, "command": list(command),
                      "ownedPaths": [norm_claim(p) for p in owned],
                      "predecessors": list(dict.fromkeys(preds)),
                      "timeoutSeconds": number(raw_task.get("timeoutSeconds"), key + ".timeoutSeconds", 0, MAX_TIMEOUT),
                      "graceSeconds": number(raw_task.get("graceSeconds"), key + ".graceSeconds", 0, MAX_GRACE),
                      "maxAttempts": attempts})
    for task in tasks:
        for pred in task["predecessors"]:
            if pred not in seen:
                raise RequestError("%s: unknown predecessor %s" % (task["key"], pred))
    claims = [(t["key"], c) for t in tasks for c in t["ownedPaths"]]
    for i, (ka, ca) in enumerate(claims):
        for kb, cb in claims[i + 1:]:
            if claims_overlap(ca, cb):
                raise RequestError("owned paths overlap: %s:%s and %s:%s" % (ka, ca, kb, cb))
    integration = req.get("integration", {"enabled": False})
    if not isinstance(integration, dict) or not isinstance(integration.get("enabled", False), bool):
        raise RequestError("integration must be an object with boolean enabled")
    if integration.get("enabled"):
        if not isinstance(integration.get("worktree"), str) or not os.path.isabs(integration["worktree"]):
            raise RequestError("enabled integration needs an absolute worktree")
        if not isinstance(integration.get("head"), str) or not OID_RE.match(integration["head"]):
            raise RequestError("enabled integration needs the checked head object id")
    return {"path": path, "sha256": sha256_bytes(raw), "raw": raw, "runId": req["runId"],
            "repository": top, "baseCommit": base, "brief": {"path": brief["path"], "sha256": brief_sha},
            "worktreeRoot": os.path.abspath(req["worktreeRoot"]), "runDir": os.path.abspath(req["runDir"]),
            "maxConcurrency": concurrency, "tasks": {t["key"]: t for t in tasks},
            "order": topo_order(tasks), "integration": integration}


# ------------------------------------------------------------- processes ---

def proc_stat(pid):
    with open("/proc/%d/stat" % pid, encoding="utf-8", errors="replace") as handle:
        fields = handle.read().rsplit(")", 1)[1].split()
    return fields[0], int(fields[2])  # state, process group


def group_members(pgid):
    """Live non-zombie members of a process group. Linux reads /proc; others use kill 0."""
    if os.path.isdir("/proc/self"):
        members = []
        for name in os.listdir("/proc"):
            if name.isdigit():
                try:
                    state, group = proc_stat(int(name))
                except (OSError, IndexError, ValueError):
                    continue
                if group == pgid and state != "Z":
                    members.append(int(name))
        return members
    try:
        os.killpg(pgid, 0)
        return [pgid]
    except ProcessLookupError:
        return []
    except PermissionError:
        return [pgid]


def processes_in(root):
    """Linux only: processes whose cwd is inside root (writers that left the group)."""
    found = []
    if not os.path.isdir("/proc/self"):
        return found
    for name in os.listdir("/proc"):
        if not name.isdigit() or int(name) == os.getpid():
            continue
        try:
            state, _ = proc_stat(int(name))
            cwd = os.readlink("/proc/%s/cwd" % name)
        except (OSError, IndexError, ValueError):
            continue
        if state != "Z" and is_within(cwd, root):
            found.append(int(name))
    return found


def wait_empty(proc, pgid, seconds):
    deadline = time.monotonic() + seconds
    while True:
        proc.poll()  # reap the leader so it never counts as live
        if not group_members(pgid):
            return True
        if time.monotonic() >= deadline:
            return False
        time.sleep(POLL)


def stop_group(proc, pgid, grace):
    """SIGTERM, bounded grace, SIGKILL; confirmed only when the group is empty."""
    signals = []
    for sig, name, wait in ((signal.SIGTERM, "SIGTERM", grace), (signal.SIGKILL, "SIGKILL", max(grace, 2.0))):
        try:
            os.killpg(pgid, sig)
            signals.append(name)
        except ProcessLookupError:
            break
        if wait_empty(proc, pgid, wait):
            break
    return {"signals": signals, "confirmed": wait_empty(proc, pgid, 0)}


# ------------------------------------------------------------------ trees ---

def tree_of(root):
    """Tree id of the worktree's tracked and untracked non-ignored content, without touching its index."""
    with tempfile.TemporaryDirectory(prefix="cx-supervisor-index-") as tmp:
        env = clean_env({"GIT_INDEX_FILE": os.path.join(tmp, "index")})
        git(root, "read-tree", "HEAD", env=env)
        git(root, "add", "-A", "--", ".", env=env)
        return git_out(root, "write-tree", env=env)


def changed_paths(root, start, end):
    out = git(root, "diff", "--no-renames", "--name-only", "-z", start, end).stdout
    return sorted(p.decode("utf-8", "surrogateescape") for p in out.split(b"\0") if p)


def tree_patch(root, start, end):
    return git(root, "diff", "--binary", "--full-index", "--no-renames", "--no-color",
               "--no-ext-diff", "--no-textconv", start, end).stdout


def file_hashes(root, paths):
    hashes = {}
    for rel in paths:
        full = os.path.join(root, rel)
        hashes[rel] = sha256_file(full) if os.path.isfile(full) and not os.path.islink(full) else None
    return hashes


# --------------------------------------------------------------- the run ---

class Run:
    def __init__(self, req):
        self.req = req
        self.lock = threading.Condition()
        self.git_lock = threading.Lock()
        self.blocked = False
        self.cancel = threading.Event()
        self.threads = []
        self.workers = {}
        self.record = {
            "schemaVersion": SCHEMA_VERSION, "runId": req["runId"], "status": "running",
            "request": {"path": req["path"], "sha256": req["sha256"]},
            "brief": req["brief"], "repository": req["repository"], "baseCommit": req["baseCommit"],
            "worktreeRoot": req["worktreeRoot"], "runDir": req["runDir"],
            "maxConcurrency": req["maxConcurrency"], "startedAt": utc_now(),
            "integrationEnabled": bool(req["integration"].get("enabled")),
            "retainedWorktrees": [],
            "tasks": {k: {"state": "pending", "reason": None, "ownedPaths": t["ownedPaths"],
                          "predecessors": t["predecessors"], "maxAttempts": t["maxAttempts"],
                          "attempts": []} for k, t in req["tasks"].items()}}

    def save(self):
        write_json(os.path.join(self.req["runDir"], "run.json"), self.record)

    def prepare(self):
        run_root = os.path.join(self.req["worktreeRoot"], self.req["runId"])
        if os.path.exists(self.req["runDir"]) or os.path.exists(run_root):
            raise RequestError("run %s is already bound to existing evidence or worktrees" % self.req["runId"])
        os.makedirs(self.req["runDir"])
        os.makedirs(run_root)
        with open(os.path.join(self.req["runDir"], "request.json"), "wb") as handle:
            handle.write(self.req["raw"])
        shutil.copyfile(self.req["brief"]["path"], os.path.join(self.req["runDir"], "brief.md"))
        self.save()

    def execute(self):
        try:
            return self._execute()
        except BaseException as err:
            # Interrupts and scheduler/persistence faults cannot abandon started workers.
            with self.lock:
                self.blocked = True
                self.cancel.set()
                self.record["supervisorError"] = "%s: %s" % (type(err).__name__, err)
                self.lock.notify_all()
            for thread in self.threads:
                thread.join()  # each worker performs bounded group settlement before returning
            with self.lock:
                for entry in self.record["tasks"].values():
                    if entry["state"] == "pending":
                        entry.update(state="blocked", reason="run-blocked")
                self.record.update(status="blocked", endedAt=utc_now())
                self.save()
            return EXIT_BLOCKED
        finally:
            for thread in self.threads:
                thread.join()

    def _execute(self):
        order, tasks = self.req["order"], self.record["tasks"]
        running = 0
        with self.lock:
            while True:
                for key in order:
                    entry = tasks[key]
                    if entry["state"] != "pending":
                        continue
                    if self.blocked:
                        entry.update(state="blocked", reason="run-blocked")
                    elif any(tasks[p]["state"] in ("failed", "blocked") for p in entry["predecessors"]):
                        entry.update(state="blocked", reason="predecessor-not-captured")
                ready = [k for k in order if tasks[k]["state"] == "pending"
                         and all(tasks[p]["state"] == "captured" for p in tasks[k]["predecessors"])]
                for key in ready[:max(0, self.req["maxConcurrency"] - running)]:
                    tasks[key]["state"] = "running"
                    running += 1
                    thread = threading.Thread(target=self._attempt_thread, args=(key,), daemon=False)
                    self.threads.append(thread)
                    thread.start()
                self.save()
                if running == 0:
                    break
                self.lock.wait()
                running = sum(1 for k in order if tasks[k]["state"] == "running")
            states = [tasks[k]["state"] for k in order]
            if self.blocked:
                self.record["status"] = "blocked"
            elif all(s == "captured" for s in states):
                self.record["status"] = "complete"
            else:
                self.record["status"] = "failed"
            self.record["endedAt"] = utc_now()
            self.save()
        return {"complete": EXIT_OK, "failed": EXIT_FAILED}.get(self.record["status"], EXIT_BLOCKED)

    def _attempt_thread(self, key):
        entry = self.record["tasks"][key]
        number_ = len(entry["attempts"]) + 1
        rec = {"attempt": number_, "status": "starting", "accepted": False}
        with self.lock:
            entry["attempts"].append(rec)
        try:
            self._attempt(key, number_, rec)
        except BaseException as err:  # includes interruption in a worker-monitor thread
            with self.lock:
                self.blocked = True
                self.cancel.set()
                rec.update(status="supervisor-error", accepted=False,
                           details={"error": "%s: %s" % (type(err).__name__, err)})
            worker = self.workers.get(key)
            if worker is not None:
                proc, grace = worker
                self._settle(rec, proc, grace)
        with self.lock:
            status = rec["status"]
            if rec.get("worktree") and os.path.isdir(rec["worktree"]):
                self.record["retainedWorktrees"].append(rec["worktree"])
            if status == "captured":
                entry.update(state="captured", reason=None)
            elif status in ("stop-unknown", "supervisor-error"):
                entry.update(state="blocked", reason=status)
                self.blocked = True
                self.cancel.set()
            elif status in RETRYABLE and number_ < entry["maxAttempts"] and not self.blocked:
                entry.update(state="pending", reason=status)
            else:
                entry.update(state="failed", reason=status)
            try:
                self.save()
            except BaseException as err:
                self.blocked = True
                self.cancel.set()
                self.record["supervisorError"] = "%s: %s" % (type(err).__name__, err)
            finally:
                self.lock.notify_all()

    def _settle(self, rec, proc, grace):
        """Always attempt bounded stop; failed observation is unknown, never a pass."""
        try:
            stop = stop_group(proc, proc.pid, grace)
            stop["escapedPids"] = processes_in(rec["worktree"])
            stop["confirmed"] = stop["confirmed"] and not stop["escapedPids"]
        except BaseException as err:
            stop = {"confirmed": False, "error": "%s: %s" % (type(err).__name__, err)}
        proc.poll()
        rec.update(stop=stop, exitCode=proc.returncode, endedAt=utc_now(), endedEpoch=time.time())
        if not stop["confirmed"]:
            rec["status"] = "stop-unknown"
        return stop

    def _inputs(self, key):
        items = []
        with self.lock:
            for pred in self.req["order"]:
                if pred in self.req["tasks"][key]["predecessors"]:
                    att = self.record["tasks"][pred]["attempts"][-1]
                    items.append({"task": pred, "attempt": att["attempt"], "patch": att["patch"],
                                  "patchSha256": att["patchSha256"], "files": att["files"]})
        return items

    def _attempt(self, key, n, rec):
        task, req = self.req["tasks"][key], self.req
        if self.cancel.is_set():
            rec["status"] = "run-blocked"
            return
        evidence = os.path.join(req["runDir"], "tasks", key, "attempt-%d" % n)
        os.makedirs(evidence)
        worktree = os.path.join(req["worktreeRoot"], req["runId"], key, "attempt-%d" % n)
        branch = "patronus-supervisor/%s/%s/attempt-%d" % (req["runId"], key, n)
        result_path = os.path.join(evidence, "result.json")
        rec.update(worktree=worktree, branch=branch, base=req["baseCommit"], evidence=evidence,
                   resultPath=result_path, ownedPaths=task["ownedPaths"],
                   timeoutSeconds=task["timeoutSeconds"], graceSeconds=task["graceSeconds"])
        with self.git_lock:
            git(req["repository"], "worktree", "add", "-q", "-b", branch, worktree, req["baseCommit"])
        if is_within(worktree, req["repository"]) or git_out(worktree, "rev-parse", "HEAD") != req["baseCommit"]:
            raise BlockedError("attempt worktree is not an isolated checkout of the base")
        inputs = self._inputs(key)
        for item in inputs:
            if sha256_file(item["patch"]) != item["patchSha256"]:
                raise BlockedError("predecessor capture hash changed: %s" % item["task"])
            if item["files"]:
                applied = git(worktree, "apply", "--binary", item["patch"], check=False)
                if applied.returncode != 0:
                    rec.update(status="input-apply-failed",
                               details={"task": item["task"], "stderr": applied.stderr.decode("utf-8", "replace")})
                    return
        rec["inputs"] = inputs
        inputs_path = os.path.join(evidence, "inputs.json")
        write_json(inputs_path, {"runId": req["runId"], "task": key, "attempt": n, "predecessors": inputs})
        rec["startTree"] = tree_of(worktree)
        env = clean_env({
            "PATRONUS_SUPERVISOR_RUN_ID": req["runId"], "PATRONUS_SUPERVISOR_TASK": key,
            "PATRONUS_SUPERVISOR_ATTEMPT": str(n), "PATRONUS_SUPERVISOR_ROOT": worktree,
            "PATRONUS_SUPERVISOR_BRANCH": branch, "PATRONUS_SUPERVISOR_BASE": req["baseCommit"],
            "PATRONUS_SUPERVISOR_BRIEF": os.path.join(req["runDir"], "brief.md"),
            "PATRONUS_SUPERVISOR_BRIEF_SHA256": req["brief"]["sha256"],
            "PATRONUS_SUPERVISOR_OWNED": json.dumps(task["ownedPaths"]),
            "PATRONUS_SUPERVISOR_RESULT": result_path, "PATRONUS_SUPERVISOR_INPUTS": inputs_path})
        with open(os.path.join(evidence, "stdout.log"), "wb") as out, \
                open(os.path.join(evidence, "stderr.log"), "wb") as err:
            rec["startedAt"], rec["startedEpoch"] = utc_now(), time.time()
            with self.lock:
                if self.blocked or self.cancel.is_set():
                    rec["status"] = "run-blocked"
                    return
                proc = subprocess.Popen(task["command"], cwd=worktree, env=env, stdin=subprocess.DEVNULL,
                                        stdout=out, stderr=err, start_new_session=True)
                pgid = proc.pid  # new session: the leader's pid is the group id
                self.workers[key] = (proc, task["graceSeconds"])
                rec.update(pid=proc.pid, pgid=pgid)
            deadline = time.monotonic() + task["timeoutSeconds"]
            timed_out = False
            while proc.poll() is None:
                if self.cancel.is_set():
                    break
                if time.monotonic() >= deadline:
                    timed_out = True
                    break
                time.sleep(POLL)
            descendant = False
            if timed_out or self.cancel.is_set():
                stop = stop_group(proc, pgid, task["graceSeconds"])
            elif wait_empty(proc, pgid, task["graceSeconds"]):
                stop = {"signals": [], "confirmed": True}
            else:
                descendant = True
                stop = stop_group(proc, pgid, task["graceSeconds"])
            proc.poll()
        escaped = processes_in(worktree)
        stop["escapedPids"] = escaped
        stop["confirmed"] = stop["confirmed"] and not escaped
        rec.update(stop=stop, exitCode=proc.returncode, endedAt=utc_now(), endedEpoch=time.time())
        if not stop["confirmed"]:
            with self.lock:
                self.blocked = True
                self.cancel.set()
                rec["status"] = "stop-unknown"  # no validation, capture, retry or cleanup
            return
        if self.cancel.is_set():
            rec["status"] = "run-blocked"
            return
        end_tree = tree_of(worktree)
        paths = changed_paths(worktree, rec["startTree"], end_tree)
        patch = tree_patch(worktree, rec["startTree"], end_tree)
        evidence_patch = os.path.join(evidence, "evidence.patch")
        with open(evidence_patch, "wb") as handle:
            handle.write(patch)
        rec.update(endTree=end_tree, changedPaths=paths, evidencePatch=evidence_patch,
                   evidencePatchSha256=sha256_bytes(patch))
        if timed_out:
            rec["status"] = "timeout"
            return
        if descendant:
            rec["status"] = "live-descendant"
            return
        reasons, details = self._validate(key, n, rec, worktree, branch, paths)
        if reasons:
            rec.update(status=reasons[0], reasons=reasons, details=details)
            return
        with self.lock:
            if self.blocked or self.cancel.is_set():
                rec["status"] = "run-blocked"
                return
            capture = os.path.join(evidence, "capture.patch")
            with open(capture, "wb") as handle:
                handle.write(patch)
            rec.update(status="captured", accepted=True, patch=capture, patchSha256=sha256_bytes(patch),
                       files=file_hashes(worktree, paths), capturedAt=utc_now(), capturedEpoch=time.time())

    def _validate(self, key, n, rec, worktree, branch, paths):
        req, task = self.req, self.req["tasks"][key]
        if rec["exitCode"] != 0:
            return ["worker-exit-nonzero"], {"exitCode": rec["exitCode"]}
        if not os.path.isfile(rec["resultPath"]):
            return ["missing-result"], {}
        try:
            with open(rec["resultPath"], "rb") as handle:
                raw = handle.read()
            result = json.loads(raw)
        except (OSError, ValueError) as err:
            return ["malformed-result"], {"error": str(err)}
        rec["resultSha256"] = sha256_bytes(raw)
        reported = result.get("changedFiles") if isinstance(result, dict) else None
        if not isinstance(reported, list) or not all(isinstance(p, str) for p in reported):
            return ["malformed-result"], {"error": "result needs a changedFiles string list"}
        want = {"runId": req["runId"], "task": key, "attempt": n, "root": worktree, "base": req["baseCommit"]}
        wrong = {k: result.get(k) for k, v in want.items() if result.get(k) != v}
        if wrong:
            return ["binding-mismatch"], {"mismatched": wrong}
        if result.get("status") != "complete":
            return ["worker-reported-failure"], {"status": result.get("status")}
        reasons, details = [], {}
        head = git_out(worktree, "rev-parse", "HEAD")
        ref = git(worktree, "symbolic-ref", "-q", "HEAD", check=False).stdout.decode().strip()
        if head != req["baseCommit"] or ref != "refs/heads/" + branch:
            reasons.append("head-drift")
            details["head"], details["ref"] = head, ref
        non_owned = [p for p in paths if not any(claim_covers(c, p) for c in task["ownedPaths"])]
        if non_owned:
            reasons.append("ownership-violation")
            details["nonOwned"] = non_owned
        staged = git(worktree, "diff", "--cached", "--name-only", "-z", "HEAD").stdout
        if staged.strip(b"\0"):
            reasons.append("staged-residue")
            details["staged"] = [p.decode("utf-8", "surrogateescape") for p in staged.split(b"\0") if p]
        status = git(worktree, "status", "--porcelain=v1", "-z", "--ignored=matching",
                     "--untracked-files=all").stdout
        ignored = [e[3:].decode("utf-8", "surrogateescape") for e in status.split(b"\0") if e.startswith(b"!! ")]
        if ignored:
            reasons.append("ignored-residue")
            details["ignored"] = ignored
        unreported = sorted(set(paths) - set(reported))
        if unreported:
            reasons.append("unreported-change")
            details["unreported"] = unreported
        phantom = sorted(set(reported) - set(paths))
        if phantom:
            reasons.append("result-mismatch")
            details["reportedButUnchanged"] = phantom
        order = ["head-drift", "ownership-violation", "staged-residue", "ignored-residue",
                 "unreported-change", "result-mismatch"]
        return sorted(reasons, key=order.index), details


# ----------------------------------------------------------- integration ---

def integrate(req, granted):
    """Serially apply verified captures in dependency order. Never commits."""
    integration = req["integration"]
    if not integration.get("enabled") or not granted:
        return EXIT_FAILED, {"status": "integration-disabled",
                             "detail": "needs integration.enabled and --grant-integration"}
    run_path = os.path.join(req["runDir"], "run.json")
    with open(run_path, encoding="utf-8") as handle:
        record = json.load(handle)
    if record["request"]["sha256"] != req["sha256"] or record["runId"] != req["runId"] \
            or record["baseCommit"] != req["baseCommit"]:
        return EXIT_FAILED, {"status": "binding-mismatch"}
    if record["status"] != "complete":
        return EXIT_FAILED, {"status": "incomplete-run", "runStatus": record["status"]}
    root, head = integration["worktree"], integration["head"]
    n = 1
    while os.path.exists(os.path.join(req["runDir"], "integration-%d.json" % n)):
        n += 1
    report_path = os.path.join(req["runDir"], "integration-%d.json" % n)
    report = {"runId": req["runId"], "worktree": root, "head": head, "startedAt": utc_now(), "steps": []}

    def finish(code, status, **extra):
        report.update(status=status, endedAt=utc_now(), **extra)
        write_json(report_path, report)
        return code, report

    if is_within(root, req["worktreeRoot"]):
        return finish(EXIT_FAILED, "integration-in-writer-root")
    actual = git_out(root, "rev-parse", "HEAD")
    if actual != head:
        return finish(EXIT_FAILED, "head-drift", actualHead=actual)
    if git_out(root, "status", "--porcelain", "--untracked-files=all"):
        return finish(EXIT_FAILED, "integration-dirty")
    checked = tree_of(root)
    report["checkedTree"] = checked
    for key in req["order"]:
        att = record["tasks"][key]["attempts"][-1]
        if not att.get("accepted") or sha256_file(att["patch"]) != att["patchSha256"]:
            return finish(EXIT_FAILED, "capture-hash-mismatch", task=key)
        if git_out(root, "rev-parse", "HEAD") != head or tree_of(root) != checked:
            return finish(EXIT_FAILED, "integration-drift", task=key)
        if att["files"]:
            probe = git(root, "apply", "--check", "--binary", att["patch"], check=False)
            if probe.returncode != 0:
                return finish(EXIT_FAILED, "merge-conflict", task=key,
                              stderr=probe.stderr.decode("utf-8", "replace"))
            git(root, "apply", "--binary", att["patch"])
        checked = tree_of(root)
        report["steps"].append({"task": key, "attempt": att["attempt"], "patchSha256": att["patchSha256"],
                                "checkedTree": checked})
    return finish(EXIT_OK, "integrated", finalTree=checked)


# ------------------------------------------------------------------- main ---

def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="command", required=True)
    for name in ("validate", "run", "integrate"):
        cmd = sub.add_parser(name)
        cmd.add_argument("request")
        if name == "integrate":
            cmd.add_argument("--grant-integration", action="store_true",
                             help="explicit separate integration grant; disabled by default")
    args = ap.parse_args(argv)
    if os.name != "posix" or not hasattr(os, "killpg"):
        print(json.dumps({"status": "blocked", "reason": "unsupported-platform"}))
        return EXIT_BLOCKED
    try:
        req = load_request(args.request)
        if args.command == "validate":
            print(json.dumps({"status": "valid", "runId": req["runId"], "order": req["order"],
                              "requestSha256": req["sha256"]}))
            return EXIT_OK
        if args.command == "integrate":
            code, report = integrate(req, args.grant_integration)
            print(json.dumps(report, sort_keys=True))
            return code
        run = Run(req)
        run.prepare()
        def interrupted(signum, frame):
            raise KeyboardInterrupt("supervisor interrupted by signal %d" % signum)

        prior = signal.signal(signal.SIGTERM, interrupted)
        try:
            code = run.execute()
        finally:
            signal.signal(signal.SIGTERM, prior)
        print(json.dumps({"status": run.record["status"], "runDir": req["runDir"],
                          "tasks": {k: v["state"] for k, v in run.record["tasks"].items()}}))
        return code
    except RequestError as err:
        print(json.dumps({"status": "invalid", "reason": str(err)}))
        return EXIT_INVALID
    except (BlockedError, OSError) as err:
        print(json.dumps({"status": "blocked", "reason": str(err)}))
        return EXIT_BLOCKED


if __name__ == "__main__":
    sys.exit(main())

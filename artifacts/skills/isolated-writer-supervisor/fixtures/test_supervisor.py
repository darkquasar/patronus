"""Fake-worker safety suite for the bundled Codex supervisor.

Every case drives scripts/supervisor.py with real local processes in invented
scratch Git repositories. No model, Codex, Claude or Pi backend is invoked.
Run: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s fixtures -p test_supervisor.py -v
"""
import hashlib
import importlib.util
import json
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from types import SimpleNamespace
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
SUPERVISOR = os.path.join(os.path.dirname(HERE), "scripts", "supervisor.py")
WORKER = os.path.join(HERE, "fake_worker.py")


def sha256(path):
    with open(path, "rb") as handle:
        return hashlib.sha256(handle.read()).hexdigest()


def group_members(pgid):
    """Live, non-zombie members of a process group (Linux /proc, else kill 0)."""
    if os.path.isdir("/proc"):
        members = []
        for name in os.listdir("/proc"):
            if not name.isdigit():
                continue
            try:
                with open("/proc/%s/stat" % name, encoding="utf-8", errors="replace") as handle:
                    fields = handle.read().rsplit(")", 1)[1].split()
            except OSError:
                continue
            if fields[0] != "Z" and int(fields[2]) == pgid:
                members.append(int(name))
        return members
    try:
        os.killpg(pgid, 0)
        return [pgid]
    except ProcessLookupError:
        return []


class Scratch:
    """An invented repository plus disjoint worktree/evidence roots."""

    def __init__(self):
        self.tmp = tempfile.mkdtemp(prefix="cx-supervisor-")
        self.repo = os.path.join(self.tmp, "repo")
        self.env = dict(os.environ, GIT_CONFIG_NOSYSTEM="1",
                        GIT_CONFIG_GLOBAL=os.path.join(self.tmp, "gitconfig"),
                        PYTHONDONTWRITEBYTECODE="1")
        with open(self.env["GIT_CONFIG_GLOBAL"], "w", encoding="utf-8") as handle:
            handle.write("[user]\n\tname = Invented\n\temail = invented@example.invalid\n"
                         "[init]\n\tdefaultBranch = main\n")
        os.makedirs(os.path.join(self.repo, "src"))
        self.put("src/shared.txt", "one\ntwo\nthree\n")
        self.put("README.md", "invented fixture\n")
        self.git("init", "-q")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "base")
        self.base = self.git("rev-parse", "HEAD").strip()
        self.brief = os.path.join(self.tmp, "brief.md")
        with open(self.brief, "w", encoding="utf-8") as handle:
            handle.write("Invented brief for fake writers.\n")
        self.count = 0

    def put(self, rel, content, root=None):
        path = os.path.join(root or self.repo, rel)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8") as handle:
            handle.write(content)

    def git(self, *args, cwd=None):
        return subprocess.run(["git", *args], cwd=cwd or self.repo, env=self.env, check=True,
                              capture_output=True, text=True).stdout

    def writer(self, key, owned, *worker_args, preds=(), timeout=20.0, grace=0.5, attempts=1):
        return {"key": key, "command": [sys.executable, WORKER, *worker_args],
                "ownedPaths": list(owned), "predecessors": list(preds),
                "timeoutSeconds": timeout, "graceSeconds": grace, "maxAttempts": attempts}

    def request(self, tasks, concurrency=2, integration=None):
        self.count += 1
        run_id = "run-%d" % self.count
        with open(self.brief, "rb") as handle:
            brief_sha = hashlib.sha256(handle.read()).hexdigest()
        req = {"schemaVersion": 1, "runId": run_id, "repository": self.repo,
               "baseCommit": self.base, "brief": {"path": self.brief, "sha256": brief_sha},
               "worktreeRoot": os.path.join(self.tmp, "worktrees"),
               "runDir": os.path.join(self.tmp, "evidence", run_id),
               "maxConcurrency": concurrency, "tasks": tasks}
        if integration is not None:
            req["integration"] = integration
        path = os.path.join(self.tmp, run_id + ".json")
        with open(path, "w", encoding="utf-8") as handle:
            json.dump(req, handle)
        return path, req

    def supervise(self, *args):
        proc = subprocess.run([sys.executable, SUPERVISOR, *args], env=self.env,
                              capture_output=True, text=True, timeout=120)
        return proc

    def run(self, tasks, **kw):
        path, req = self.request(tasks, **kw)
        proc = self.supervise("run", path)
        record_path = os.path.join(req["runDir"], "run.json")
        record = None
        if os.path.exists(record_path):
            with open(record_path, encoding="utf-8") as handle:
                record = json.load(handle)
        return proc, record, path, req

    def lead_status(self):
        return self.git("status", "--porcelain", "--untracked-files=all", "--ignored")

    def close(self):
        shutil.rmtree(self.tmp, ignore_errors=True)


class SupervisorSafety(unittest.TestCase):
    def setUp(self):
        self.s = Scratch()

    def tearDown(self):
        self.s.close()

    def attempt(self, record, key, index=-1):
        return record["tasks"][key]["attempts"][index]

    def assert_settled(self, attempt):
        self.assertTrue(attempt["stop"]["confirmed"], attempt)
        self.assertEqual(group_members(attempt["pgid"]), [], attempt)

    def assert_lead_untouched(self):
        self.assertEqual(self.s.lead_status(), "")
        self.assertEqual(self.s.git("rev-parse", "HEAD").strip(), self.s.base)

    def assert_retained(self, attempt, rel=None, content=None):
        self.assertTrue(os.path.isdir(attempt["worktree"]), attempt)
        self.assertTrue(attempt["worktree"].startswith(os.path.join(self.s.tmp, "worktrees") + os.sep))
        if rel is not None:
            with open(os.path.join(attempt["worktree"], rel), encoding="utf-8") as handle:
                self.assertEqual(handle.read(), content)
        self.assertEqual(sha256(attempt["evidencePatch"]), attempt["evidencePatchSha256"])

    def test_independent_writers(self):
        proc, record, path, req = self.s.run([
            self.s.writer("a", ["src/a/"], "--write", "src/a/out.txt=A\n", "--sleep", "1.5"),
            self.s.writer("b", ["src/b.txt"], "--write", "src/b.txt=B\n", "--sleep", "1.5"),
        ])
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertEqual(record["status"], "complete")
        a, b = self.attempt(record, "a"), self.attempt(record, "b")
        self.assertLess(a["startedEpoch"], b["endedEpoch"])
        self.assertLess(b["startedEpoch"], a["endedEpoch"], "writers did not overlap in time")
        self.assertNotEqual(a["worktree"], b["worktree"])
        for att, rel, body in ((a, "src/a/out.txt", "A\n"), (b, "src/b.txt", "B\n")):
            self.assertEqual(att["status"], "captured")
            self.assertTrue(att["accepted"])
            self.assert_settled(att)
            self.assertEqual(att["base"], self.s.base)
            self.assertNotEqual(os.path.realpath(att["worktree"]), os.path.realpath(self.s.repo))
            self.assertEqual(sha256(att["patch"]), att["patchSha256"])
            self.assertEqual(att["files"], {rel: hashlib.sha256(body.encode()).hexdigest()})
            with open(att["patch"], encoding="utf-8") as handle:
                self.assertIn(rel, handle.read())
        self.assert_lead_untouched()
        # Integration is disabled by default and needs a separate explicit grant.
        refused = self.s.supervise("integrate", path)
        self.assertNotEqual(refused.returncode, 0)
        self.assertIn("integration-disabled", refused.stdout + refused.stderr)
        self.assert_lead_untouched()

    def test_a_to_b_order(self):
        integ = os.path.join(self.s.tmp, "integration")
        self.s.git("worktree", "add", "-q", "--detach", integ, self.s.base)
        proc, record, path, req = self.s.run([
            self.s.writer("b", ["src/b.txt"], "--require", "src/a.txt=from A\n",
                        "--require-input", "a", "--write", "src/b.txt=B saw A\n", preds=["a"]),
            self.s.writer("a", ["src/a.txt"], "--write", "src/a.txt=from A\n", "--sleep", "1"),
        ], integration={"enabled": True, "worktree": integ, "head": self.s.base})
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        a, b = self.attempt(record, "a"), self.attempt(record, "b")
        self.assertGreaterEqual(b["startedEpoch"], a["capturedEpoch"], "B started before A was verified")
        self.assertEqual(b["inputs"][0]["task"], "a")
        self.assertEqual(b["inputs"][0]["patchSha256"], a["patchSha256"])
        self.assertEqual(list(b["files"]), ["src/b.txt"], "B capture must exclude predecessor input")
        # Without the explicit grant flag nothing is applied, even when enabled.
        self.assertNotEqual(self.s.supervise("integrate", path).returncode, 0)
        self.assertFalse(os.path.exists(os.path.join(integ, "src/a.txt")))
        done = self.s.supervise("integrate", path, "--grant-integration")
        self.assertEqual(done.returncode, 0, done.stdout + done.stderr)
        with open(os.path.join(req["runDir"], "integration-1.json"), encoding="utf-8") as handle:
            steps = json.load(handle)["steps"]
        self.assertEqual([step["task"] for step in steps], ["a", "b"])
        with open(os.path.join(integ, "src/b.txt"), encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "B saw A\n")
        self.assertEqual(self.s.git("rev-parse", "HEAD", cwd=integ).strip(), self.s.base, "no commits")
        self.assert_lead_untouched()
        # A failed predecessor blocks its dependent, which never starts.
        proc, record, _, _ = self.s.run([
            self.s.writer("a", ["src/a.txt"], "--result", "none"),
            self.s.writer("b", ["src/b.txt"], "--write", "src/b.txt=B\n", preds=["a"]),
        ])
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(record["tasks"]["b"]["state"], "blocked")
        self.assertEqual(record["tasks"]["b"]["attempts"], [])

    def test_timeout(self):
        proc, record, _, _ = self.s.run([
            self.s.writer("slow", ["src/slow.txt"], "--write", "src/slow.txt=partial\n",
                        "--ignore-sigterm", "--sleep", "60", timeout=1.0, grace=0.5),
        ])
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        att = self.attempt(record, "slow")
        self.assertEqual(att["status"], "timeout")
        self.assertFalse(att["accepted"])
        self.assertEqual(att["stop"]["signals"], ["SIGTERM", "SIGKILL"])
        self.assert_settled(att)
        self.assert_retained(att, "src/slow.txt", "partial\n")
        self.assertIsNone(att.get("patch"))
        self.assertEqual(record["tasks"]["slow"]["state"], "failed")
        self.assertIn(att["worktree"], record["retainedWorktrees"])
        self.assert_lead_untouched()

    def test_delayed_descendant(self):
        proc, record, _, _ = self.s.run([
            self.s.writer("d", ["src/d.txt", "src/late.txt"], "--write", "src/d.txt=D\n",
                        "--spawn-delayed", "src/late.txt=late\n", "--delay", "3", grace=0.5),
        ])
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        att = self.attempt(record, "d")
        self.assertEqual(att["status"], "live-descendant")
        self.assertFalse(att["accepted"])
        self.assert_settled(att)
        time.sleep(3.5)
        self.assertFalse(os.path.exists(os.path.join(att["worktree"], "src/late.txt")),
                         "descendant writer survived settlement")
        self.assert_retained(att, "src/d.txt", "D\n")
        self.assert_lead_untouched()

    def test_missing_result(self):
        proc, record, _, _ = self.s.run([
            self.s.writer("none", ["src/n.txt"], "--write", "src/n.txt=N\n", "--result", "none"),
            self.s.writer("bad", ["src/m.txt"], "--write", "src/m.txt=M\n", "--result", "malformed"),
            self.s.writer("wrong", ["src/w.txt"], "--write", "src/w.txt=W\n", "--result", "wrong-base"),
            self.s.writer("failed", ["src/f.txt"], "--write", "src/f.txt=F\n", "--result", "failed"),
        ])
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        want = {"none": "missing-result", "bad": "malformed-result",
                "wrong": "binding-mismatch", "failed": "worker-reported-failure"}
        for key, reason in want.items():
            att = self.attempt(record, key)
            self.assertEqual(att["status"], reason, att)
            self.assertFalse(att["accepted"])
            self.assert_settled(att)
            self.assert_retained(att)
        self.assert_lead_untouched()

    def test_ownership_violation(self):
        proc, record, path, req = self.s.run([
            self.s.writer("o", ["src/o.txt"], "--write", "src/o.txt=O\n", "--write", "README.md=stolen\n"),
        ])
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        att = self.attempt(record, "o")
        self.assertEqual(att["status"], "ownership-violation")
        self.assertIn("README.md", att["details"]["nonOwned"])
        self.assertFalse(att["accepted"])
        self.assert_retained(att, "README.md", "stolen\n")
        self.assert_lead_untouched()
        # Request-level claims are validated before any worktree exists.
        for owned in (["../escape.txt"], ["/abs.txt"], [".git/config"], ["src/./x"]):
            bad_path, bad = self.s.request([self.s.writer("x", owned, "--write", "x=1")])
            out = self.s.supervise("run", bad_path)
            self.assertEqual(out.returncode, 2, owned)
            self.assertFalse(os.path.exists(bad["runDir"]))
        overlap_path, overlap = self.s.request([
            self.s.writer("x", ["src/"], "--write", "src/x=1"),
            self.s.writer("y", ["src/y.txt"], "--write", "src/y.txt=1")])
        out = self.s.supervise("run", overlap_path)
        self.assertEqual(out.returncode, 2)
        self.assertIn("overlap", out.stdout + out.stderr)
        cycle_path, _ = self.s.request([
            self.s.writer("x", ["x"], preds=["y"]), self.s.writer("y", ["y"], preds=["x"])])
        self.assertEqual(self.s.supervise("validate", cycle_path).returncode, 2)
        # Immutable run binding: the same run cannot be replayed over its evidence.
        self.assertEqual(self.s.supervise("run", path).returncode, 2)
        self.assertFalse(os.path.exists(os.path.join(self.s.tmp, "worktrees", "run-1", "o", "attempt-2")))

    def test_dirty_result(self):
        proc, record, _, _ = self.s.run([
            self.s.writer("staged", ["src/s.txt"], "--write", "src/s.txt=S\n", "--stage"),
            self.s.writer("committed", ["src/c.txt"], "--write", "src/c.txt=C\n", "--commit"),
            self.s.writer("unreported", ["src/u/"], "--write", "src/u/ok.txt=ok\n",
                        "--unreported", "src/u/scratch.tmp=junk\n"),
            self.s.writer("clean", ["src/k.txt"], "--write", "src/k.txt=K\n"),
        ], concurrency=1)
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        want = {"staged": "staged-residue", "committed": "head-drift",
                "unreported": "unreported-change"}
        for key, reason in want.items():
            att = self.attempt(record, key)
            self.assertEqual(att["status"], reason, att)
            self.assertFalse(att["accepted"])
            self.assert_retained(att)
        # Intended uncommitted edits inside ownership are the capture, not dirt.
        clean = self.attempt(record, "clean")
        self.assertEqual(clean["status"], "captured")
        self.assertTrue(clean["accepted"])
        self.assert_lead_untouched()

    def test_merge_conflict(self):
        integ = os.path.join(self.s.tmp, "integration")
        self.s.git("worktree", "add", "-q", "-b", "integration", integ, self.s.base)
        self.s.put("src/shared.txt", "one\nINTEGRATION\nthree\n", root=integ)
        self.s.git("commit", "-q", "-am", "integration head moved", cwd=integ)
        head = self.s.git("rev-parse", "HEAD", cwd=integ).strip()
        proc, record, path, req = self.s.run([
            self.s.writer("m", ["src/shared.txt"], "--write", "src/shared.txt=one\nWRITER\nthree\n"),
        ], integration={"enabled": True, "worktree": integ, "head": head})
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        att = self.attempt(record, "m")
        out = self.s.supervise("integrate", path, "--grant-integration")
        self.assertNotEqual(out.returncode, 0)
        with open(os.path.join(req["runDir"], "integration-1.json"), encoding="utf-8") as handle:
            report = json.load(handle)
        self.assertEqual(report["status"], "merge-conflict")
        self.assertEqual(report["task"], "m")
        with open(os.path.join(integ, "src/shared.txt"), encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "one\nINTEGRATION\nthree\n", "conflict mutated integration")
        self.assertEqual(self.s.git("status", "--porcelain", cwd=integ), "")
        self.assertEqual(sha256(att["patch"]), att["patchSha256"], "captured patch not retained")
        self.assert_retained(att, "src/shared.txt", "one\nWRITER\nthree\n")
        # A drifted integration head is refused before any apply.
        bad_path, _ = self.s.request([self.s.writer("n", ["src/n.txt"], "--write", "src/n.txt=N\n")],
                                     integration={"enabled": True, "worktree": integ, "head": self.s.base})
        self.assertEqual(self.s.supervise("run", bad_path).returncode, 0)
        drift = self.s.supervise("integrate", bad_path, "--grant-integration")
        self.assertNotEqual(drift.returncode, 0)
        self.assertIn("head-drift", drift.stdout + drift.stderr)
        self.assertFalse(os.path.exists(os.path.join(integ, "src/n.txt")))

    def faulted_run(self, tasks, configure):
        """Inject an observation fault, never a product fault flag, around real workers."""
        spec = importlib.util.spec_from_file_location("cx_fault_supervisor", SUPERVISOR)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        path, req = self.s.request(tasks, integration={
            "enabled": True, "worktree": self.s.repo, "head": self.s.base})
        started = []
        original = subprocess.Popen

        def launch(*args, **kwargs):
            proc = original(*args, **kwargs)
            if kwargs.get("start_new_session"):
                started.append(proc)
            return proc

        with mock.patch.dict(os.environ, self.s.env), mock.patch.object(mod.subprocess, "Popen", launch):
            run = mod.Run(mod.load_request(path))
            run.prepare()
            try:
                with configure(mod, run, started):
                    code = run.execute()
                with open(os.path.join(req["runDir"], "run.json"), encoding="utf-8") as handle:
                    record = json.load(handle)
                return code, record, path, req
            finally:
                # Red-test failure must not leave a writer behind or erase live work.
                for proc in started:
                    if group_members(proc.pid):
                        os.killpg(proc.pid, 9)
                    proc.wait(timeout=5)
                    self.assertEqual(group_members(proc.pid), [])

    def test_stop_unknown(self):
        def configure(mod, run, started):
            original = mod.stop_group

            def uncertain(proc, pgid, grace):
                stop = original(proc, pgid, grace)
                stop["confirmed"] = False  # observer cannot prove settlement
                return stop

            return mock.patch.object(mod, "stop_group", uncertain)

        code, record, path, _ = self.faulted_run([
            self.s.writer("unknown", ["src/u.txt"], "--write", "src/u.txt=partial\n",
                          "--sleep", "3", timeout=0.3, grace=0.1, attempts=2),
            self.s.writer("peer", ["src/p.txt"], "--write", "src/p.txt=peer\n",
                          "--sleep", "1", grace=0.1),
            self.s.writer("dependent", ["src/d.txt"], preds=["unknown"]),
        ], configure)
        self.assertEqual(code, 3)
        self.assertEqual(record["status"], "blocked")
        att = self.attempt(record, "unknown")
        self.assertEqual(att["status"], "stop-unknown")
        self.assertFalse(att["stop"]["confirmed"])
        self.assertFalse(att["accepted"])
        self.assertNotIn("patch", att)
        self.assertNotIn("evidencePatch", att)
        self.assertEqual(len(record["tasks"]["unknown"]["attempts"]), 1)
        self.assertEqual(record["tasks"]["dependent"]["attempts"], [])
        peer = self.attempt(record, "peer")
        self.assertFalse(peer["accepted"], "unknown state must block peer capture")
        self.assertIn("stop", peer)
        self.assertTrue(os.path.isdir(att["worktree"]))
        refused = self.s.supervise("integrate", path, "--grant-integration")
        self.assertNotEqual(refused.returncode, 0)
        self.assertIn("incomplete-run", refused.stdout)
        self.assertEqual(self.s.supervise("run", path).returncode, 2, "no replay over unknown evidence")
        self.assert_lead_untouched()

    def test_supervisor_error_settles_workers(self):
        for fault in (RuntimeError, KeyboardInterrupt):
            with self.subTest(fault=fault.__name__):
                def configure(mod, run, started):
                    raised = False

                    def sleep(seconds):
                        nonlocal raised
                        if len(started) == 2 and not raised:
                            raised = True
                            raise fault("invented supervisor fault with two live workers")
                        time.sleep(seconds)

                    return mock.patch.object(mod, "time", SimpleNamespace(
                        sleep=sleep, monotonic=time.monotonic, time=time.time))

                code, record, _, _ = self.faulted_run([
                    self.s.writer("a", ["src/a.txt"], "--write", "src/a.txt=partial\n",
                                  "--sleep", "1", grace=0.1),
                    self.s.writer("b", ["src/b.txt"], "--sleep", "1", grace=0.1),
                ], configure)
                self.assertEqual(code, 3)
                self.assertEqual(record["status"], "blocked")
                for key in ("a", "b"):
                    att = self.attempt(record, key)
                    self.assert_settled(att)
                    self.assertFalse(att["accepted"])
                    self.assertNotIn("patch", att)
                    self.assertTrue(os.path.isdir(att["worktree"]))
                    self.assertIn(att["worktree"], record["retainedWorktrees"])
                self.assert_lead_untouched()

        # Drive actual CLI interruption while both workers have written partial data.
        for sig in (signal.SIGTERM, signal.SIGINT):
            with self.subTest(signal=sig.name):
                path, req = self.s.request([
                    self.s.writer("a", ["src/a.txt"], "--write", "src/a.txt=partial\n",
                                  "--sleep", "60", grace=0.1),
                    self.s.writer("b", ["src/b.txt"], "--write", "src/b.txt=partial\n",
                                  "--sleep", "60", grace=0.1),
                ])
                proc = subprocess.Popen([sys.executable, SUPERVISOR, "run", path],
                                        env=self.s.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                try:
                    deadline = time.monotonic() + 5
                    while not all(os.path.isfile(os.path.join(req["worktreeRoot"], req["runId"],
                                  key, "attempt-1", "src", key + ".txt")) for key in ("a", "b")):
                        self.assertLess(time.monotonic(), deadline, "workers never started")
                        time.sleep(0.02)
                    proc.send_signal(sig)
                    out, err = proc.communicate(timeout=10)
                    self.assertEqual(proc.returncode, 3, (out, err))
                    with open(os.path.join(req["runDir"], "run.json"), encoding="utf-8") as handle:
                        record = json.load(handle)
                    self.assertEqual(record["status"], "blocked")
                    for key in ("a", "b"):
                        att = self.attempt(record, key)
                        self.assert_settled(att)
                        self.assertFalse(att["accepted"])
                        self.assertNotIn("patch", att)
                        with open(os.path.join(att["worktree"], "src", key + ".txt"), encoding="utf-8") as handle:
                            self.assertEqual(handle.read(), "partial\n")
                finally:
                    if proc.poll() is None:
                        proc.send_signal(signal.SIGTERM)
                        proc.communicate(timeout=10)
                self.assert_lead_untouched()

    def test_input_apply_failed(self):
        def configure(mod, run, started):
            original = mod.git

            def git(cwd, *args, **kwargs):
                if args and args[0] == "apply":
                    return subprocess.CompletedProcess(args, 1, b"", b"invented apply failure")
                return original(cwd, *args, **kwargs)

            return mock.patch.object(mod, "git", git)

        code, record, _, _ = self.faulted_run([
            self.s.writer("a", ["src/a.txt"], "--write", "src/a.txt=A\n"),
            self.s.writer("b", ["src/b.txt"], preds=["a"], attempts=2),
            self.s.writer("c", ["src/c.txt"], preds=["b"]),
        ], configure)
        self.assertEqual(code, 1)
        self.assertEqual(record["status"], "failed")
        a, b = self.attempt(record, "a"), self.attempt(record, "b")
        self.assertEqual(a["status"], "captured")
        self.assertEqual(sha256(a["patch"]), a["patchSha256"])
        self.assertEqual(b["status"], "input-apply-failed")
        self.assertEqual(b["details"]["task"], "a")
        self.assertIn("invented apply failure", b["details"]["stderr"])
        self.assertEqual(len(record["tasks"]["b"]["attempts"]), 1)
        self.assertNotIn("pid", b)
        self.assertFalse(b["accepted"])
        self.assertNotIn("patch", b)
        self.assertTrue(os.path.isdir(b["worktree"]))
        self.assertEqual(record["tasks"]["c"]["attempts"], [])
        self.assert_lead_untouched()

    def test_retry_preserves_failed_attempt(self):
        proc, record, _, _ = self.s.run([
            self.s.writer("r", ["src/r.txt"], "--require-absent", "src/r.txt",
                        "--write", "src/r.txt=R\n", "--fail-attempts", "1", attempts=2),
        ])
        # The first attempt writes src/r.txt and exits without a result; it must
        # not leak into attempt 2, which requires a fresh tree from the base.
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        first, second = record["tasks"]["r"]["attempts"]
        self.assertEqual(first["status"], "worker-exit-nonzero")
        self.assertFalse(first["accepted"])
        self.assert_retained(first, "src/r.txt", "R\n")
        self.assertEqual(second["status"], "captured")
        self.assertNotEqual(first["worktree"], second["worktree"])
        self.assertNotEqual(first["branch"], second["branch"])
        self.assertEqual(second["base"], self.s.base)
        self.assertIn(first["worktree"], record["retainedWorktrees"])
        # Retries are bounded: the default single attempt never retries.
        proc, record, _, _ = self.s.run([
            self.s.writer("once", ["src/o.txt"], "--write", "src/o.txt=O\n", "--fail-attempts", "5"),
        ])
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(len(record["tasks"]["once"]["attempts"]), 1)
        self.assert_lead_untouched()


if __name__ == "__main__":
    unittest.main()

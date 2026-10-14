"""Public CLI tests for both standalone native worktree allocators, stdlib only."""
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
CX_SCRIPT = os.path.join(os.path.dirname(HERE), "scripts", "prepare-worktrees.py")
OC_SCRIPT = os.path.join(os.path.dirname(os.path.dirname(HERE)),
                         "plan-execute-parallel-oc", "scripts", "prepare-worktrees.py")


class Scratch:
    def __init__(self):
        self.tmp = tempfile.mkdtemp(prefix="native-worktree-allocation-")
        self.repo = os.path.join(self.tmp, "repo")
        os.makedirs(self.repo)
        self.env = dict(os.environ, GIT_CONFIG_NOSYSTEM="1",
                        GIT_CONFIG_GLOBAL=os.path.join(self.tmp, "gitconfig"),
                        PYTHONDONTWRITEBYTECODE="1")
        with open(self.env["GIT_CONFIG_GLOBAL"], "w", encoding="utf-8") as handle:
            handle.write("[user]\nname = Invented\nemail = invented@example.invalid\n[init]\ndefaultBranch = main\n")
        with open(os.path.join(self.repo, "README.md"), "w", encoding="utf-8") as handle:
            handle.write("Invented allocation fixture\n")
        self.git("init", "-q")
        self.git("add", "README.md")
        self.git("commit", "-q", "-m", "base")
        self.base = self.git("rev-parse", "HEAD").strip()
        self.brief = os.path.join(self.tmp, "brief.md")
        with open(self.brief, "wb") as handle:
            handle.write(b"Invented native allocation grant\n")
        self.count = 0

    def git(self, *args, cwd=None):
        return subprocess.run(["git", *args], cwd=cwd or self.repo, env=self.env,
                              check=True, capture_output=True, text=True).stdout

    def request(self, tasks):
        self.count += 1
        req = {"schemaVersion": 1, "runId": "run-%d" % self.count,
               "repository": self.repo, "baseCommit": self.base,
               "brief": {"path": self.brief,
                         "sha256": hashlib.sha256(b"Invented native allocation grant\n").hexdigest()},
               "worktreeRoot": os.path.join(self.tmp, "worktrees"),
               "runDir": os.path.join(self.tmp, "evidence", "run-%d" % self.count), "tasks": tasks}
        path = os.path.join(self.tmp, "request-%d.json" % self.count)
        self.save(path, req)
        return path, req

    def save(self, path, req):
        with open(path, "w", encoding="utf-8") as handle:
            json.dump(req, handle)

    def close(self):
        shutil.rmtree(self.tmp)


class PrepareWorktreesCX(unittest.TestCase):
    script = CX_SCRIPT

    def setUp(self):
        self.s = Scratch()
        # Only the standalone payload is installed in this invented package.
        # No sibling skills, supervisor or support module can satisfy an import.
        package = os.path.join(self.s.tmp, "standalone-package")
        os.makedirs(package)
        self.installed_script = os.path.join(package, "prepare-worktrees.py")
        shutil.copyfile(self.script, self.installed_script)

    def tearDown(self):
        self.s.close()

    def run_cli(self, path, *args):
        return subprocess.run([sys.executable, self.installed_script, path, *args], env=self.s.env,
                              capture_output=True, text=True, timeout=30)

    def test_allocates_commandless_native_bindings(self):
        path, req = self.s.request([{"key": "a", "ownedPaths": ["src/a/"]},
                                    {"key": "b", "ownedPaths": ["src/b.txt"]}])
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        report = json.loads(proc.stdout)
        self.assertEqual((report["mode"], report["status"]), ("worktrees-only", "allocated"))
        with open(os.path.join(req["runDir"], "allocation.json"), encoding="utf-8") as handle:
            self.assertEqual(json.load(handle), report)
        self.assertEqual(os.listdir(req["runDir"]), ["allocation.json"])
        for key, binding in report["tasks"].items():
            self.assertEqual(binding["status"], "allocated")
            self.assertEqual(binding["base"], self.s.base)
            self.assertEqual(self.s.git("rev-parse", "HEAD", cwd=binding["root"]).strip(), self.s.base)
            self.assertEqual(self.s.git("branch", "--show-current", cwd=binding["root"]).strip(), binding["branch"])
        self.assertEqual(self.s.git("status", "--porcelain"), "")

    def assert_no_allocation(self, req):
        self.assertFalse(os.path.exists(req["runDir"]))
        self.assertFalse(os.path.exists(req["worktreeRoot"]))
        self.assertEqual(self.s.git("status", "--porcelain"), "")

    def test_execution_shaped_fields_are_rejected_without_worker_launch(self):
        sentinel = os.path.join(self.s.tmp, "agent-was-launched")
        path, req = self.s.request([
            {"key": "a", "ownedPaths": ["src/a"],
             "command": [sys.executable, "-c", "open(%r, 'w').write('not allowed')" % sentinel]},
        ])
        checked = self.run_cli(path, "--validate")
        self.assertEqual(checked.returncode, 2, checked.stdout + checked.stderr)
        self.assertIn("command", checked.stdout)
        self.assert_no_allocation(req)
        self.assertFalse(os.path.exists(sentinel))

    def test_invalid_branches_and_existing_branches_preflight_all_tasks(self):
        for bad in ("bad..name", "already-there"):
            with self.subTest(key=bad):
                path, req = self.s.request([
                    {"key": "a", "ownedPaths": ["src/a"]},
                    {"key": bad, "ownedPaths": ["src/b"]},
                ])
                if bad == "already-there":
                    self.s.git("branch", "patronus-allocation/%s/%s" % (req["runId"], bad))
                proc = self.run_cli(path)
                self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
                self.assert_no_allocation(req)

    def test_invalid_claims_and_execution_semantics_refused(self):
        for tasks, integration in (
            ([{"key": "a", "ownedPaths": ["src/"]}, {"key": "b", "ownedPaths": ["src/b"]}], None),
            ([{"key": "a", "ownedPaths": ["../outside"]}], None),
            ([{"key": "a", "ownedPaths": [".git/config"]}], None),
            ([{"key": "a", "ownedPaths": ["src/a"], "predecessors": ["b"]}], None),
            ([{"key": "a", "ownedPaths": ["src/a"]}], {"enabled": True}),
            ([{"key": "a", "ownedPaths": ["src/a"], "timeoutSeconds": 3}], None),
            ([{"key": "a", "ownedPaths": ["src/a"], "maxAttempts": 2}], None),
        ):
            with self.subTest(tasks=tasks, integration=integration):
                path, req = self.s.request(tasks)
                if integration is not None:
                    req["integration"] = integration
                    self.s.save(path, req)
                proc = self.run_cli(path)
                self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
                self.assert_no_allocation(req)

    def test_slashless_ancestor_claim_is_rejected_before_allocation(self):
        path, req = self.s.request([
            {"key": "a", "ownedPaths": ["src"]},
            {"key": "b", "ownedPaths": ["src/b"]},
        ])
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assert_no_allocation(req)

    def test_symlink_destination_and_base_claim_refused(self):
        path, req = self.s.request([{"key": "a", "ownedPaths": ["src/a"]}])
        outside = os.path.join(self.s.tmp, "outside")
        os.makedirs(outside)
        os.symlink(outside, req["worktreeRoot"])
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertEqual(os.listdir(outside), [])
        self.assertFalse(os.path.exists(req["runDir"]))
        os.unlink(req["worktreeRoot"])
        os.symlink("../outside", os.path.join(self.s.repo, "escape"))
        self.s.git("add", "escape")
        self.s.git("commit", "-q", "-m", "invented symlink")
        self.s.base = self.s.git("rev-parse", "HEAD").strip()
        path, req = self.s.request([{"key": "a", "ownedPaths": ["escape/out.txt"]}])
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assert_no_allocation(req)

    def test_destination_inside_other_checkout_refused(self):
        other = os.path.join(self.s.tmp, "another-checkout")
        self.s.git("worktree", "add", "-q", "--detach", other, self.s.base)
        path, req = self.s.request([{"key": "a", "ownedPaths": ["src/a"]}])
        req["worktreeRoot"] = os.path.join(other, "allocations")
        self.s.save(path, req)
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assert_no_allocation(req)
        self.assertEqual(self.s.git("status", "--porcelain", cwd=other), "")

    def test_partial_failure_is_retained_not_completed(self):
        hook = os.path.join(self.s.repo, ".git", "hooks", "post-checkout")
        with open(hook, "w", encoding="utf-8") as handle:
            handle.write("#!/bin/sh\nexit 1\n")
        os.chmod(hook, 0o755)
        path, req = self.s.request([{"key": "a", "ownedPaths": ["src/a"]}])
        proc = self.run_cli(path)
        self.assertEqual(proc.returncode, 3, proc.stdout + proc.stderr)
        report = json.loads(proc.stdout)
        self.assertEqual(report["status"], "blocked")
        self.assertTrue(os.path.isdir(report["tasks"]["a"]["root"]))
        self.assertEqual(os.listdir(req["runDir"]), ["allocation.json"])
        with open(os.path.join(req["runDir"], "allocation.json"), encoding="utf-8") as handle:
            self.assertEqual(json.load(handle), report)
        self.assertEqual(self.s.git("status", "--porcelain"), "")


class PrepareWorktreesOC(PrepareWorktreesCX):
    script = OC_SCRIPT


class PackagedMirror(unittest.TestCase):
    def test_both_payloads_match_the_canonical_allocator(self):
        with open(CX_SCRIPT, "rb") as cx, open(OC_SCRIPT, "rb") as oc:
            self.assertEqual(cx.read(), oc.read())


if __name__ == "__main__":
    unittest.main()

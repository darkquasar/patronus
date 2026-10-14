"""Invented-data fixtures for scripts/opencode-worker-precheck.sh.

Historical, not run by the development gates (same status as the supervisor
fake-worker suite). Every repository, agent, hook registration, guard script and
opencode.json below is invented in a scratch directory; no real guard script or
user configuration is read. Run with:

    python3 artifacts/skills/isolated-writer-supervisor/fixtures/test_opencode_worker_precheck.py -v
"""
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "opencode-worker-precheck.sh"

AGENT = "---\nname: patronus-writer-oc\ndescription: invented writer role\nmode: subagent\n---\n\nInvented body.\n"
RUNNER = "// invented hook runner placeholder\nexport const PatronusHookRunner = async () => ({});\n"
GUARD = "#!/usr/bin/env python3\n# invented guard placeholder: fixture-not-a-secret\nraise SystemExit(0)\n"
HOOKS = {
    "schemaVersion": 1,
    "hooks": [
        {"patronusId": "hook:block-secrets-oc:1.0.0", "event": "tool.execute.before", "command": "patronus/hooks/block-secrets-oc.py"},
        {"patronusId": "hook:gitleaks-guard-oc:1.0.0", "event": "tool.execute.before", "command": "patronus/hooks/gitleaks-guard-oc.py"},
    ],
}
CONFIG = {"$schema": "https://opencode.ai/config.json", "permission": {"edit": {"*": "allow", "*.env": "deny", ".env.*": "deny"}}}

# Relative paths of the six identities, project form (tracked) and global form.
IDENTITIES = {
    "agent": ("agents/patronus-writer-oc.md", AGENT),
    "runner": ("plugins/patronus-hook-runner.ts", RUNNER),
    "hooks": ("patronus/hooks.json", json.dumps(HOOKS, indent=2) + "\n"),
    "guard1": ("patronus/hooks/block-secrets-oc.py", GUARD),
    "guard2": ("patronus/hooks/gitleaks-guard-oc.py", GUARD),
    "config": ("opencode.json", json.dumps(CONFIG, indent=2) + "\n"),
}


def write(root, rel, text, executable=False):
    path = root / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    if executable:
        path.chmod(path.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    return path


def place(root, keys, project=True, hooks_text=None):
    """Write the selected identities under root (.opencode/ prefix for project form)."""
    for key in keys:
        rel, text = IDENTITIES[key]
        if key == "hooks" and hooks_text is not None:
            text = hooks_text
        if key != "config" and project:
            rel = ".opencode/" + rel
        write(root, rel, text, executable=key in ("guard1", "guard2"))


def git(repo, *args):
    env = dict(os.environ, GIT_AUTHOR_NAME="fixture", GIT_AUTHOR_EMAIL="fixture@example.invalid",
               GIT_COMMITTER_NAME="fixture", GIT_COMMITTER_EMAIL="fixture@example.invalid",
               GIT_CONFIG_GLOBAL="/dev/null", GIT_CONFIG_NOSYSTEM="1")
    return subprocess.run(["git", "-C", str(repo)] + list(args), check=True, capture_output=True, text=True, env=env, timeout=10)


def commit_all(repo):
    git(repo, "add", "-A")
    git(repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "invented fixture commit")
    return git(repo, "rev-parse", "HEAD").stdout.strip()


def run_precheck(cwd, base, global_mode=False, env=None):
    argv = ["sh", str(SCRIPT), base] + (["--global"] if global_mode else [])
    merged = dict(os.environ)
    merged.pop("OPENCODE_CONFIG_DIR", None)
    merged.pop("XDG_CONFIG_HOME", None)
    if env:
        merged.update(env)
    return subprocess.run(argv, cwd=str(cwd), capture_output=True, text=True, env=merged, timeout=10)


class TrackedWorkingSet(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp(prefix="worker-precheck-"))
        self.addCleanup(shutil.rmtree, self.tmp, True)
        self.repo = self.tmp / "repo"
        self.repo.mkdir()
        git(self.repo, "init", "-q")
        git(self.repo, "config", "core.fileMode", "true")

    def test_complete_tracked_set_passes(self):
        place(self.repo, IDENTITIES.keys())
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("OK", result.stdout)

    def test_agent_only_fails_and_names_the_other_five(self):
        place(self.repo, ["agent"])
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode, result.stdout)
        named = [line for line in result.stderr.splitlines() if line.startswith("worker-precheck: missing:")]
        self.assertEqual(5, len(named), result.stderr)
        for rel in (".opencode/plugins/patronus-hook-runner.ts", ".opencode/patronus/hooks.json",
                    ".opencode/patronus/hooks/block-secrets-oc.py", ".opencode/patronus/hooks/gitleaks-guard-oc.py",
                    "opencode.json"):
            self.assertIn(rel, result.stderr)
        self.assertNotIn("patronus-writer-oc.md", result.stderr)

    def test_config_only_fails(self):
        place(self.repo, ["config"])
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertIn(".opencode/agents/patronus-writer-oc.md", result.stderr)
        self.assertNotIn("opencode.json", result.stderr)

    def test_hooks_json_without_guard_scripts_fails(self):
        place(self.repo, ["agent", "runner", "hooks", "config"])
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertIn("block-secrets-oc.py", result.stderr)
        self.assertIn("gitleaks-guard-oc.py", result.stderr)
        self.assertNotIn("hooks.json", result.stderr)

    def test_unrelated_tracked_file_under_opencode_fails(self):
        write(self.repo, ".opencode/commands/invented.md", "# invented command\n")
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertEqual(6, len([l for l in result.stderr.splitlines() if l.startswith("worker-precheck: missing:")]), result.stderr)

    def test_guard_without_executable_bit_fails(self):
        place(self.repo, IDENTITIES.keys())
        plain = self.repo / ".opencode/patronus/hooks/gitleaks-guard-oc.py"
        plain.chmod(0o644)
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertIn("gitleaks-guard-oc.py executable bit", result.stderr)

    def test_hooks_json_missing_one_guard_registration_fails(self):
        hooks = {"hooks": [{"patronusId": "hook:block-secrets-oc:1.0.0"}]}
        place(self.repo, IDENTITIES.keys(), hooks_text=json.dumps(hooks))
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertIn("patronusId entry for gitleaks-guard-oc", result.stderr)
        self.assertNotIn("patronusId entry for block-secrets-oc", result.stderr)

    def test_scalar_permission_container_fails(self):
        place(self.repo, IDENTITIES.keys())
        write(self.repo, "opencode.json", json.dumps({"permission": "allow"}))
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode, "invalid permission container must refuse launch")
        self.assertIn("permission.edit object", result.stderr)
        self.assertNotIn("OK", result.stdout)
        self.assertNotIn("Traceback", result.stderr)

    def test_null_and_list_permission_containers_fail(self):
        place(self.repo, IDENTITIES.keys())
        for permission in (None, [], ["allow"]):
            with self.subTest(permission=permission):
                write(self.repo, "opencode.json", json.dumps({"permission": permission}))
                base = commit_all(self.repo)
                result = run_precheck(self.repo, base)
                self.assertEqual(1, result.returncode)
                self.assertIn("permission.edit object", result.stderr)
                self.assertNotIn("Traceback", result.stderr)
                self.assertNotIn("OK", result.stdout)

    def test_malformed_json_fails_without_payload_diagnostics(self):
        place(self.repo, IDENTITIES.keys())
        for rel in (".opencode/patronus/hooks.json", "opencode.json"):
            with self.subTest(identity=rel):
                original = (self.repo / rel).read_text(encoding="utf-8")
                write(self.repo, rel, "{invented-private-payload")
                base = commit_all(self.repo)
                result = run_precheck(self.repo, base)
                self.assertEqual(1, result.returncode)
                self.assertIn(rel + " is not valid JSON", result.stderr)
                self.assertNotIn("invented-private-payload", result.stderr)
                self.assertNotIn("Traceback", result.stderr)
                write(self.repo, rel, original)

    def validator_failure(self, target, status=7):
        """Invented executable fails one JSON checker with no stdout."""
        place(self.repo, IDENTITIES.keys())
        base = commit_all(self.repo)
        real_python = shutil.which("python3")
        self.assertIsNotNone(real_python)
        bindir = self.tmp / "bin"
        condition = '[ "${3:-}" = block-secrets-oc ]' if target == "hooks" else 'case "$2" in *permission*) true;; *) false;; esac'
        wrapper = "#!/bin/sh\nif " + condition + "; then\n  echo invented-private-diagnostic >&2\n  exit " + str(status) + "\nfi\nexec " + json.dumps(real_python) + ' "$@"\n'
        write(bindir, "python3", wrapper, executable=True)
        result = run_precheck(self.repo, base, env={"PATH": str(bindir) + os.pathsep + os.environ["PATH"]})
        self.assertEqual(1, result.returncode, "checker failure without stdout must refuse launch")
        self.assertIn("validator failed", result.stderr)
        self.assertIn("hooks.json" if target == "hooks" else "opencode.json", result.stderr)
        self.assertNotIn("invented-private-diagnostic", result.stderr)
        self.assertNotIn("OK", result.stdout)

    def test_hook_validator_failure_without_stdout_fails(self):
        self.validator_failure("hooks")

    def test_permission_validator_failure_without_stdout_fails(self):
        self.validator_failure("permission")

    def test_empty_missing_diagnostics_and_other_validator_statuses_fail(self):
        for target in ("hooks", "permission"):
            for status in (1, 2, 127):
                with self.subTest(target=target, status=status):
                    self.repo = self.tmp / (target + str(status))
                    self.repo.mkdir()
                    git(self.repo, "init", "-q")
                    self.validator_failure(target, status)

    def test_config_without_env_keys_fails(self):
        place(self.repo, IDENTITIES.keys())
        write(self.repo, "opencode.json", json.dumps({"permission": {"edit": {"*": "allow"}}}))
        base = commit_all(self.repo)
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)
        self.assertIn("permission.edit key *.env", result.stderr)
        self.assertIn("permission.edit key .env.*", result.stderr)

    def test_untracked_working_set_is_not_enough(self):
        write(self.repo, "README.md", "invented\n")
        base = commit_all(self.repo)
        place(self.repo, IDENTITIES.keys())  # present on disk, not tracked at base
        result = run_precheck(self.repo, base)
        self.assertEqual(1, result.returncode)

    def test_unknown_base_ref_exits_2(self):
        write(self.repo, "README.md", "invented\n")
        commit_all(self.repo)
        result = run_precheck(self.repo, "0" * 40)
        self.assertEqual(2, result.returncode)


class GlobalWorkingSet(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp(prefix="worker-precheck-global-"))
        self.addCleanup(shutil.rmtree, self.tmp, True)
        self.repo = self.tmp / "repo"
        self.repo.mkdir()
        git(self.repo, "init", "-q")
        write(self.repo, "README.md", "invented\n")
        self.base = commit_all(self.repo)

    def test_opencode_config_dir_complete_passes(self):
        root = self.tmp / "cfgdir"
        place(root, IDENTITIES.keys(), project=False)
        result = run_precheck(self.repo, self.base, global_mode=True, env={"OPENCODE_CONFIG_DIR": str(root)})
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn(str(root), result.stdout)

    def test_opencode_config_dir_partial_fails(self):
        root = self.tmp / "cfgdir"
        place(root, ["agent", "runner", "config"], project=False)
        result = run_precheck(self.repo, self.base, global_mode=True, env={"OPENCODE_CONFIG_DIR": str(root)})
        self.assertEqual(1, result.returncode)
        self.assertIn(str(root / "patronus/hooks.json"), result.stderr)
        self.assertIn(str(root / "patronus/hooks/block-secrets-oc.py"), result.stderr)
        self.assertIn(str(root / "patronus/hooks/gitleaks-guard-oc.py"), result.stderr)

    def test_xdg_config_home_complete_passes(self):
        xdg = self.tmp / "xdg"
        place(xdg / "opencode", IDENTITIES.keys(), project=False)
        result = run_precheck(self.repo, self.base, global_mode=True, env={"XDG_CONFIG_HOME": str(xdg)})
        self.assertEqual(0, result.returncode, result.stderr)

    def test_xdg_config_home_partial_fails(self):
        xdg = self.tmp / "xdg"
        place(xdg / "opencode", ["hooks", "guard1", "guard2"], project=False)
        result = run_precheck(self.repo, self.base, global_mode=True, env={"XDG_CONFIG_HOME": str(xdg)})
        self.assertEqual(1, result.returncode)
        self.assertIn("patronus-writer-oc.md", result.stderr)
        self.assertIn("opencode.json", result.stderr)

    def test_opencode_config_dir_wins_over_xdg(self):
        xdg = self.tmp / "xdg"
        place(xdg / "opencode", IDENTITIES.keys(), project=False)
        empty = self.tmp / "empty"
        empty.mkdir()
        result = run_precheck(self.repo, self.base, global_mode=True,
                              env={"XDG_CONFIG_HOME": str(xdg), "OPENCODE_CONFIG_DIR": str(empty)})
        self.assertEqual(1, result.returncode)
        self.assertIn(str(empty), result.stderr)

    def test_global_agent_without_subagent_mode_fails(self):
        root = self.tmp / "cfgdir"
        place(root, IDENTITIES.keys(), project=False)
        write(root, "agents/patronus-writer-oc.md", "---\nname: patronus-writer-oc\nmode: primary\n---\n")
        result = run_precheck(self.repo, self.base, global_mode=True, env={"OPENCODE_CONFIG_DIR": str(root)})
        self.assertEqual(1, result.returncode)
        self.assertIn("mode: subagent", result.stderr)


if __name__ == "__main__":
    if not SCRIPT.is_file():
        sys.exit("precheck script not found at %s" % SCRIPT)
    unittest.main()

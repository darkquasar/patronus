"""Run the shipped smoke plan against invented catalogs and a built public CLI."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[2]


class ClaudeSmokeDriver(unittest.TestCase):
    def setUp(self):
        binary = os.environ.get("PATRONUS_TEST_BIN", "")
        if not Path(binary).is_file():
            self.skipTest("PATRONUS_TEST_BIN must name an already-built public CLI")
        self.temp = tempfile.TemporaryDirectory(prefix="claude-smoke-driver-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "repo"
        self.root.mkdir()
        (self.root / "adapters").mkdir()
        (self.root / "go.mod").write_text("module invented.example/smoke\n\ngo 1.25\n")
        (self.root / "adapters/claude.yaml").write_text(
            "apiVersion: patronus/v2\nfamily: adapter\ntool: claude\n"
            "layout:\n  skill:\n    global: '~/.claude/skills/{name}/SKILL.md'\n"
            "    project: '.claude/skills/{name}/SKILL.md'\n")
        for name, targets in (("go-style-uber", "[codex, opencode]"),
                              ("go-style-uber-cl", "[claude]"),
                              ("invented-override-cl", "[claude]")):
            package = self.root / "artifacts/skills" / name
            package.mkdir(parents=True)
            (package / "patronus.yaml").write_text(
                "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\n"
                f"role: capability\nname: {name}\nversion: 1.0.0\ndescription: invented smoke fixture\n"
                f"entry: SKILL.md\nfiles: []\ntargets: {targets}\n")
            (package / "SKILL.md").write_text(
                f"---\nname: {name}\ndescription: invented smoke fixture\n---\nInvented.\n")
        # The build command is an external boundary. Reuse the already-built real
        # Patronus CLI instead of invoking a compiler inside the invented checkout.
        tools = Path(self.temp.name) / "tools"
        tools.mkdir()
        go = tools / "go"
        go.write_text('#!/bin/sh\n[ "$1" = build ] && [ "$2" = -o ] || exit 2\n'
                      'cp "$PATRONUS_TEST_BIN" "$3"\n')
        go.chmod(0o755)
        home = Path(self.temp.name) / "caller-home"
        home.mkdir()
        self.env = dict(os.environ, PATRONUS_TEST_BIN=str(Path(binary).resolve()),
                        PATRONUS_REPO=str(self.root), HOME=str(home),
                        PATH=str(tools) + os.pathsep + os.environ["PATH"])
        self.env.pop("PATRONUS_SMOKE_ITEM", None)
        control = subprocess.run([str(Path(binary).resolve()), "install", "go-style-uber-cl",
                                  "--target", "claude", "--global", "--local-registry"],
                                 cwd=self.root, env=self.env, capture_output=True, text=True)
        self.assertEqual(control.returncode, 0, "fixture control: " + control.stdout + control.stderr)
        rejected = subprocess.run([str(Path(binary).resolve()), "install", "go-style-uber",
                                   "--target", "claude", "--global", "--local-registry"],
                                  cwd=self.root, env=self.env, capture_output=True, text=True)
        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn('does not target tool "claude"', rejected.stderr)

    def run_plan(self, bundle, item=None):
        driver = Path(self.temp.name) / (bundle + ".sh")
        shutil.copyfile(SOURCE / "artifacts/skills" / bundle / "driver.sh", driver)
        env = dict(self.env)
        if item:
            env["PATRONUS_SMOKE_ITEM"] = item
        return subprocess.run(["bash", str(driver), "plan"], cwd=self.root,
                              env=env, capture_output=True, text=True, timeout=30)

    def test_default_plan_selects_a_claude_compatible_item(self):
        for bundle in ("run-patronus", "run-patronus-cl"):
            with self.subTest(bundle=bundle):
                result = self.run_plan(bundle)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn("declares itself a dry run", result.stdout)
                self.assertIn("wrote nothing", result.stdout)

    def test_explicit_compatible_override_is_preserved(self):
        for bundle in ("run-patronus", "run-patronus-cl"):
            with self.subTest(bundle=bundle):
                result = self.run_plan(bundle, "invented-override-cl")
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_incompatible_explicit_override_still_fails(self):
        for bundle in ("run-patronus", "run-patronus-cl"):
            with self.subTest(bundle=bundle):
                result = self.run_plan(bundle, "go-style-uber")
                self.assertNotEqual(result.returncode, 0)

    def test_canonical_item_can_deploy_through_the_public_cli(self):
        result = subprocess.run([self.env["PATRONUS_TEST_BIN"], "install", "go-style-uber-cl",
                                 "--target", "claude", "--global", "--local-registry", "--deploy"],
                                cwd=self.root, env=self.env, capture_output=True, text=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        installed = Path(self.env["HOME"]) / ".claude/skills/go-style-uber-cl/SKILL.md"
        self.assertEqual(installed.read_bytes(),
                         (self.root / "artifacts/skills/go-style-uber-cl/SKILL.md").read_bytes())


if __name__ == "__main__":
    unittest.main()

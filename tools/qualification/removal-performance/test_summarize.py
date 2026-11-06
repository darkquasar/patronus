#!/usr/bin/env python3
"""Regression for test2json splitting the benchmark name from its metrics."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class SummarizeTest(unittest.TestCase):
    def summarize(self, outputs):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "go-test.jsonl"
            source.write_text("".join(json.dumps({"Output": item}) + "\n" for item in outputs))
            return subprocess.run(
                [sys.executable, str(Path(__file__).with_name("summarize.py")), str(source)],
                capture_output=True, text=True, check=False, timeout=10,
            )

    def test_split_output_events(self):
        result = self.summarize([
            "BenchmarkRemovalRealPersistence/N100\n",
            "BenchmarkRemovalRealPersistence/N100 \t",
            " 1\t123 ns/op\t456 B/op\t7 allocs/op\n",
        ])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)["measurements"], [{
            "benchmark": "BenchmarkRemovalRealPersistence/N100", "iterations": 1,
            "metrics": {"ns/op": 123.0, "B/op": 456.0, "allocs/op": 7.0},
        }])

    def test_unsplit_line(self):
        result = self.summarize(["BenchmarkRemovalRealPersistence/N100 1 123 ns/op\n"])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(json.loads(result.stdout)["measurements"]), 1)

    def test_missing_measurement_fails_closed(self):
        result = self.summarize(["PASS\n"])
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(json.loads(result.stdout)["measurements"], [])


if __name__ == "__main__":
    unittest.main()

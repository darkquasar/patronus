#!/usr/bin/env python3
"""Extract Go benchmark metrics, not test/build/setup wall time (stdlib only)."""
import json
import re
import sys

rows = []
with open(sys.argv[1], encoding="utf-8") as stream:
    # test2json may split a single benchmark line across several Output events.
    output = "".join(json.loads(line).get("Output", "") for line in stream)
for line in output.splitlines():
    match = re.match(r"^(Benchmark\S+)\s+(\d+)\s+(.+)$", line.strip())
    if not match:
        continue
    fields = match[3].split()
    metrics = {fields[i + 1]: float(fields[i]) for i in range(0, len(fields), 2)}
    rows.append({"benchmark": match[1], "iterations": int(match[2]), "metrics": metrics})
json.dump({"scope": "service operation only; setup and discovery acknowledgement excluded", "measurements": rows}, sys.stdout, indent=2)
print()
if not rows:
    print("no completed benchmark measurements", file=sys.stderr)
    sys.exit(2)

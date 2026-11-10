#!/usr/bin/env python3
"""
Report each Go package's init() clock time (GODEBUG=inittrace=1) on stderr and, on Linux,
fail if the total init time exceeds --max-ms.

The total is judged by its minimum over several runs, since load from parallel tests
inflates individual runs but slow inits are slow on every run. While the total is over the
threshold, more runs are added (up to --max-runs) to ride out bursts of load.
"""

import argparse
import os
import re
import subprocess
import sys

INIT_LINE = re.compile(r"^init (\S+) @\S+ ms, (\S+) ms clock")


def measure(command):
    env = dict(os.environ, GODEBUG="inittrace=1")
    cp = subprocess.run(command, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
    if cp.returncode != 0:
        sys.exit(f"command failed with exit code {cp.returncode}:\n{cp.stderr}")
    clocks = {m.group(1): float(m.group(2)) for m in map(INIT_LINE.match, cp.stderr.splitlines()) if m}
    if not clocks:
        sys.exit(f"no inittrace output from {command}:\n{cp.stderr}")
    return clocks


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=5)
    parser.add_argument("--max-runs", type=int, default=20)
    parser.add_argument("--max-ms", type=float, required=True)
    parser.add_argument("command", nargs="+")
    args = parser.parse_args()

    runs = [measure(args.command) for _ in range(args.runs)]
    while len(runs) < args.max_runs and min(sum(r.values()) for r in runs) > args.max_ms:
        runs.append(measure(args.command))

    total = min(sum(r.values()) for r in runs)
    clocks = {pkg: min(r[pkg] for r in runs) for pkg in runs[0]}
    print(f"Minimum init() clock time in ms over {len(runs)} runs:", file=sys.stderr)
    print(f"{total:5.2f} total", file=sys.stderr)
    for pkg, ms in sorted(clocks.items(), key=lambda kv: (-kv[1], kv[0])):
        print(f"{ms:5.2f} {pkg}", file=sys.stderr)

    # Clocks elsewhere are too coarse or noisy to enforce (on Windows they can be quantized
    # to the ~15.6 ms system tick).
    if total > args.max_ms and sys.platform == "linux":
        print(
            f"FAIL: package init() takes {total:.2f} ms in total, over {args.max_ms:g} ms; this delays every CLI invocation."
        )
        print("To investigate:")
        print("  GODEBUG=inittrace=1 databricks --version 2>&1 | sort -k5 -n -r | head")
        print("  go mod why -m <module>")
        sys.exit(1)
    print(f"OK: total package init() time is within {args.max_ms:g} ms")


if __name__ == "__main__":
    main()

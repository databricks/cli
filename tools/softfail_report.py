#!/usr/bin/env python3
"""
Scan a gotestsum/`go test -json` file for SOFTFAIL markers and print a markdown
report. Acceptance tests emit `SOFTFAIL <file>` followed by a unified diff via
t.Logf when a soft-failed golden drifts (see doComparison in acceptance_test.go).

Those tests stay green, so the drift is invisible unless we surface it. This report
is written to the GitHub step summary on every run so oncall can refresh the golden
on a cadence without the build ever turning red.

Usage: softfail_report.py test-output.json [test-output.json ...]
Exits 0 whether or not markers are found (soft-fail is non-blocking by design).
"""

import json
import sys
from collections import defaultdict

MARKER = "SOFTFAIL "

# Prefixes of go test status/control lines that the framework emits between a
# test's own output and its terminating action (e.g. "    --- PASS: TestAccept/x").
# The unified diff's own "--- <path>" / "+++ <path>" headers lack the colon, so
# they are not mistaken for these.
CONTROL_PREFIXES = ("--- PASS:", "--- FAIL:", "--- SKIP:", "--- BENCH:", "=== ")


def collect_softfails(events):
    """Group SOFTFAIL diff blocks by test from `go test -json` output events.

    Each SOFTFAIL marker line opens a block that captures the unified diff that
    follows it, closing on the next non-output action or go test status line so the
    "--- PASS:" the framework emits for a passing soft-failed test stays out of the diff.

    >>> evs = [
    ...     {"Action": "output", "Test": "TestAccept/x", "Output": "    foo.go:1: SOFTFAIL out.drifty.txt\\n"},
    ...     {"Action": "output", "Test": "TestAccept/x", "Output": "        --- a\\n"},
    ...     {"Action": "output", "Test": "TestAccept/x", "Output": "        +++ b\\n"},
    ...     {"Action": "output", "Test": "TestAccept/x", "Output": "    --- PASS: TestAccept/x (0.50s)\\n"},
    ...     {"Action": "pass", "Test": "TestAccept/x"},
    ... ]
    >>> result = collect_softfails(evs)
    >>> result["TestAccept/x"][0][0]
    'out.drifty.txt'
    >>> "PASS" in "".join(result["TestAccept/x"][0][1])
    False
    """
    blocks = defaultdict(list)
    open_block = {}
    for ev in events:
        if ev.get("Action") != "output":
            open_block.pop(ev.get("Test"), None)
            continue
        test = ev.get("Test")
        out = ev.get("Output", "")
        idx = out.find(MARKER)
        if idx != -1:
            relpath = out[idx + len(MARKER) :].strip()
            block = [relpath, []]
            blocks[test].append(block)
            open_block[test] = block
        elif test in open_block:
            if out.lstrip().startswith(CONTROL_PREFIXES):
                open_block.pop(test)
                continue
            open_block[test][1].append(out)
    return blocks


def render(blocks):
    """Render collected SOFTFAIL blocks as a markdown report."""
    if not blocks:
        return "No soft-failed acceptance goldens drifted in this run.\n"

    total = sum(len(v) for v in blocks.values())
    lines = [
        f"## Soft-failed acceptance goldens ({total})",
        "",
        "These goldens drifted but did not fail the build. Refresh with `./task test-update`.",
        "",
    ]
    for test in sorted(blocks):
        for relpath, diff in blocks[test]:
            lines.append(f"<details><summary><code>{relpath}</code> — {test}</summary>")
            lines.append("")
            lines.append("```diff")
            lines.append("".join(diff).rstrip("\n"))
            lines.append("```")
            lines.append("</details>")
            lines.append("")
    return "\n".join(lines)


def read_events(paths):
    for path in paths:
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    yield json.loads(line)
                except json.JSONDecodeError:
                    continue


def main(argv):
    if len(argv) < 2:
        sys.stderr.write(f"usage: {argv[0]} test-output.json [...]\n")
        return 2
    blocks = collect_softfails(read_events(argv[1:]))
    sys.stdout.write(render(blocks))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))

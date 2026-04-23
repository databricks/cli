#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Deadcode checker for the Databricks CLI.

Runs the 'deadcode' tool (golang.org/x/tools/cmd/deadcode) to find functions
that are unreachable from main() or test entry points. Since the CLI is a
product (not a library), any unreachable function is dead code.

Suppression mechanisms
======================

1. Directory exclusions (EXCLUDED_DIRS below):
   Entire directories can be excluded. Use this for directories where
   everything is a false positive. Example: libs/gorules/ contains lint
   rule definitions loaded by golangci-lint's ruleguard engine, not
   through Go's call graph.

2. Inline comments:
   Add "//deadcode:allow <reason>" above a function to suppress a
   specific finding. The comment can appear on the line directly above
   the func keyword, or above a doc comment block. The script scans
   up to 5 lines above the reported line, stopping at a blank line.

   Example:

       //deadcode:allow loaded by golangci-lint ruleguard, not via Go imports
       // ProcessRule applies a lint rule.
       func MyLintRule(m dsl.Matcher) {

   This matches the //nolint: pattern Go developers already know.
"""

import re
import subprocess
import sys

# Directories to exclude entirely. Each entry is matched as a substring
# of the file path in deadcode output.
EXCLUDED_DIRS = [
    "libs/gorules/",  # Lint rule definitions loaded by golangci-lint's ruleguard
    "bundle/internal/tf/schema/",  # Generated from Terraform provider schema
]

ALLOW_COMMENT = "//deadcode:allow"


def main():
    result = subprocess.run(
        ["go", "tool", "-modfile=tools/go.mod", "deadcode", "-test", "./..."],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0 and not result.stdout.strip():
        print("deadcode failed:\n", file=sys.stderr)
        print(result.stderr, file=sys.stderr)
        sys.exit(1)

    output = result.stdout.strip()
    if not output:
        print("No dead code found.")
        return

    lines = output.split("\n")
    violations = []

    for line in lines:
        if any(line.startswith(d) or ("/" + d) in line for d in EXCLUDED_DIRS):
            continue

        match = re.match(r"(.+?):(\d+):\d+:", line)
        if not match:
            violations.append(line)
            continue

        filepath = match.group(1)
        lineno = int(match.group(2))

        try:
            with open(filepath) as f:
                file_lines = f.readlines()
            # Walk backward from the func line; stop at the first blank
            # line so we only see this function's own comment block.
            suppressed = False
            for i in range(lineno - 2, max(-1, lineno - 8), -1):
                stripped = file_lines[i].strip()
                if not stripped:
                    break
                if ALLOW_COMMENT in stripped:
                    suppressed = True
                    break
            if suppressed:
                continue
        except (OSError, IndexError):
            pass

        violations.append(line)

    if not violations:
        print("No dead code found.")
        return

    print("Dead code found:\n")
    for v in violations:
        print(f"  {v}")
    print(f"\n{len(violations)} unreachable function(s) found.")
    print("\nTo suppress, add a comment on the line above the function:")
    print("  //deadcode:allow <reason>")
    print("\nOr add a directory exclusion in tools/check_deadcode.py.")
    sys.exit(1)


if __name__ == "__main__":
    main()

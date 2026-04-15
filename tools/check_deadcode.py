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
   Add "//deadcode:allow <reason>" on the line directly above a function
   to suppress a specific finding. Example:

       //deadcode:allow loaded by golangci-lint ruleguard, not via Go imports
       func MyLintRule(m dsl.Matcher) {

   This matches the //nolint: pattern Go developers already know.
"""
import re
import subprocess
import sys

# Directories to exclude entirely. Everything in these directories is a
# false positive. Each entry is matched as a prefix of the file path in
# deadcode output.
EXCLUDED_DIRS = [
    "libs/gorules/",
]

ALLOW_COMMENT = "//deadcode:allow"


def main():
    result = subprocess.run(
        ["go", "tool", "-modfile=tools/go.mod", "deadcode", "-test", "./..."],
        capture_output=True,
        text=True,
    )
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
            if lineno >= 2 and ALLOW_COMMENT in file_lines[lineno - 2]:
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

#!/usr/bin/env python3
#
# Finds unused Go functions in the codebase.
#
# It works by:
# 1. Finding all function declarations using `git grep`.
# 2. For each function, it searches for its usages, again with `git grep`.
# 3. If a function's name only appears on its declaration line, it's
#    considered unused.
#
# This script has limitations and may produce false positives or negatives:
# - It can be confused by methods with the same name on different types.
# - It doesn't understand interfaces. If a method is only used via an
#   interface, it might be reported as unused.
# - It's based on simple text matching, not Go's type system.

import subprocess
import re


def find_funcs():
    """Returns a list of function definition lines."""
    # `-n` adds line numbers to simplify comparison later.
    result = subprocess.run(
        ["git", "grep", "-n", "-w", "^func", "--", "*.go"],
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    if result.returncode == 2:  # git grep error (e.g., bad pattern)
        raise RuntimeError(result.stderr)
    return result.stdout.strip().split("\n") if result.stdout else []


def extract_func_name(line):
    """Return function name from `git grep` result line."""
    # line is like: path/to/file.go:123:func (r *Receiver) MyFunc(...) {
    # we take the part after the file path and line number
    parts = line.split(":", 2)
    if len(parts) < 3:
        return None
    decl = parts[2]

    match = re.search(r"^func(?:\s+\([^)]+\))?\s+([a-zA-Z0-9_]+)", decl)
    if match:
        return match.group(1)


def check_usage(func_name):
    """Return all `git grep` matches for *func_name* (may be empty)."""
    result = subprocess.run(
        ["git", "grep", "-n", "-w", func_name],
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    if result.returncode > 1:  # git grep error
        raise RuntimeError(result.stderr)
    return result.stdout.strip().split("\n") if result.stdout else []


def main():
    func_defs = find_funcs()
    print(f"{len(func_defs)} function definitions", flush=True)

    if not func_defs:
        return

    printed_header = False

    for def_line in func_defs:
        if not def_line:
            continue

        parts = def_line.split(":", 2)
        if len(parts) < 3:
            continue
        filepath = parts[0]

        func_name = extract_func_name(def_line)
        if not func_name:
            continue

        # Ignore tests and benchmarks in test files
        if filepath.endswith("_test.go"):
            if func_name.startswith("Test") or func_name.startswith("Benchmark"):
                continue

        usages = check_usage(func_name)

        # Only its declaration found -> unused.
        if len(usages) == 1:
            assert usages[0] == def_line, (usages, def_line)
            if not printed_header:
                print("Potentially unused functions:", flush=True)
                printed_header = True
            print(f"- {def_line}", flush=True)
        else:
            assert usages, func_name

    if not printed_header:
        print("No unused functions found.", flush=True)


if __name__ == "__main__":
    main()

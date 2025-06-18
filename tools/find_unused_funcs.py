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


# ----------------------------- helpers -----------------------------------


def _run_git_grep(pattern):
    """Return git-grep output lines for *pattern* inside *.go files."""
    result = subprocess.run(
        ["git", "grep", "-n", "-w", pattern, "--", "*.go"],
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    if result.returncode == 2:  # git grep error (e.g., bad pattern)
        raise RuntimeError(result.stderr)
    return result.stdout.strip().split("\n") if result.stdout else []


def find_defs(keyword):
    """Return all lines that *start* with *keyword* in Go sources."""
    return _run_git_grep(f"^{keyword}")


# Regex patterns for extracting identifiers from a declaration line.
_EXTRACTORS = {
    "func": re.compile(r"^func(?:\s+\([^)]+\))?\s+([a-zA-Z0-9_]+)"),
    "type": re.compile(r"^type\s+([a-zA-Z0-9_]+)"),
    "var": re.compile(r"^var\s+([a-zA-Z0-9_]+)"),
    "const": re.compile(r"^const\s+([a-zA-Z0-9_]+)"),
}


def extract_name(keyword, line):
    """Extract declared identifier name from *line* that starts with *keyword*."""
    parts = line.split(":", 2)
    if len(parts) < 3:
        return None
    decl = parts[2]
    m = _EXTRACTORS[keyword].search(decl)
    if m:
        return m.group(1)


def check_usage(ident):
    """Return all `git grep` matches for *ident* (may be empty)."""
    result = subprocess.run(
        ["git", "grep", "-n", "-w", ident],
        stdout=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    if result.returncode > 1:  # git grep error
        raise RuntimeError(result.stderr)
    return result.stdout.strip().split("\n") if result.stdout else []


def _analyze(keyword, skip_decl_pred):
    """Return (unused, test_only) lists for *keyword* declarations."""
    unused = []
    test_only = []

    for def_line in find_defs(keyword):
        if not def_line:
            continue

        parts = def_line.split(":", 2)
        if len(parts) < 3:
            continue

        filepath = parts[0]
        name = extract_name(keyword, def_line)
        if not name:
            continue

        if skip_decl_pred(filepath, name):
            continue

        usages = check_usage(name)
        references = [u for u in usages if u != def_line]

        if not references:
            unused.append(def_line)
            continue

        if all(r.split(":", 1)[0].endswith("_test.go") for r in references):
            # For functions, only track if the declaration itself is not in a test file
            # and not inside acceptance/internal package.
            if not (filepath.endswith("_test.go") or "acceptance/internal" in filepath):
                test_only.append(def_line)

    return unused, test_only


def main():
    categories = [
        ("func", lambda path, name: path.endswith("_test.go") and name.startswith(("Test", "Benchmark"))),
        ("type", lambda path, _n: "bundle/internal/tf/schema" in path),
        ("var", lambda _p, _n: False),
        ("const", lambda _p, _n: False),
    ]

    overall_unused = {}
    overall_test_only = {}

    for keyword, skip_pred in categories:
        unused, test_only = _analyze(keyword, skip_pred)
        if unused:
            overall_unused[keyword] = unused
        if test_only:
            overall_test_only[keyword] = test_only

    if overall_unused:
        print("Potentially unused identifiers:")
        for kw, lines in overall_unused.items():
            header = {
                "func": "functions",
                "type": "types",
                "var": "variables",
                "const": "consts",
            }[kw]
            print(f"  {header}:")
            for l in lines:
                print(f"    - {l}")

    if overall_test_only:
        if overall_unused:
            print()
        print("Identifiers only referenced from tests:")
        for kw, lines in overall_test_only.items():
            header = {
                "func": "functions",
                "type": "types",
                "var": "variables",
                "const": "consts",
            }[kw]
            print(f"  {header}:")
            for l in lines:
                print(f"    - {l}")

    if not overall_unused and not overall_test_only:
        print("No unused or test-only identifiers found.")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
Read the replacements applied to the test output from $ACC_REPLS.

The file holds two kinds of lines:
 - JSON objects written by the test harness, where "Old" is a regular expression;
 - "<value>:<NAME>" records appended by add_repl.py, where <value> is a literal
   that is replaced with [<NAME>].
"""

import json
import os
import re
import sys
from pathlib import Path

# Order of the replacements added by the scripts. Matches loadUserReplacements in
# acceptance_test.go, which applies them before the ones written by the harness.
USER_ORDER = -100


def read_repls():
    """Return (pattern, replacement) pairs in the order they must be applied."""
    entries = []

    for line in Path(os.environ["ACC_REPLS"]).read_text().splitlines():
        line = line.strip()
        if not line:
            continue

        if line.startswith("{"):
            item = json.loads(line)
            # "Distinct" is not honoured here; unlike the harness, we do not number the matches.
            entries.append((item.get("Order", 0), item["Old"], item["New"]))
            continue

        value, name = line.rsplit(":", 1)
        # Set() in libs/testdiff also registers the JSON-encoded form of the value, so that
        # values with quotes or backslashes are replaced inside JSON output as well.
        encoded = json.dumps(value, ensure_ascii=False)[1:-1]
        if encoded != value:
            entries.append((USER_ORDER, re.escape(encoded), f"[{name}]"))
        entries.append((USER_ORDER, re.escape(value), f"[{name}]"))

    # Stable sort: replacements with the same order are applied in the order they were added.
    entries.sort(key=lambda entry: entry[0])

    return [(old, new) for _, old, new in entries]


def compile_repls():
    """Same as read_repls(), with patterns compiled. Invalid patterns are reported and skipped."""
    result = []
    for old, new in read_repls():
        try:
            result.append((re.compile(old), new))
        except re.error as e:
            print(f"Regex error for pattern {old}: {e}", file=sys.stderr)
    return result


def replace_all(patterns, s):
    for comp, new in patterns:
        s = comp.sub(new, s)
    return s

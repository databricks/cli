#!/usr/bin/env python3
"""
Read the replacements applied to the test output from $ACC_REPLS.

Every line is one replacement encoded as a JSON object: "Old" is a regular expression
(written by the test harness), "Literal" is a value to replace verbatim (appended by
add_repl.py). "New" is the replacement, "Order" defines the order they are applied in.
"""

import json
import os
import re
import sys
from pathlib import Path

# Order of the replacements added by the scripts, so that they are applied before the ones
# from the harness: a job id must become [MY_JOB] even if a broader harness pattern matches it.
USER_ORDER = -100


def read_entries():
    """Return the raw entries of $ACC_REPLS."""
    result = []
    for line in Path(os.environ["ACC_REPLS"]).read_text().splitlines():
        line = line.strip()
        if line:
            result.append(json.loads(line))
    return result


def read_repls():
    """Return (pattern, replacement, distinct) tuples in application order."""
    entries = []

    for item in read_entries():
        order = item.get("Order", 0)
        new = item["New"]
        literal = item.get("Literal")

        if literal is None:
            entries.append((order, item["Old"], new, item["Distinct"]))
            continue

        # Set() in libs/testdiff also registers the JSON-encoded form of the value, so that
        # values with quotes or backslashes are replaced inside JSON output as well.
        encoded = json.dumps(literal, ensure_ascii=False)[1:-1]
        if encoded != literal:
            entries.append((order, re.escape(encoded), new, False))
        entries.append((order, re.escape(literal), new, False))

    # Stable sort: replacements with the same order are applied in the order they were added.
    entries.sort(key=lambda entry: entry[0])

    return [(old, new, distinct) for _, old, new, distinct in entries]


def compile_repls():
    """Same as read_repls(), with patterns compiled. Invalid patterns are reported and skipped."""
    result = []
    for old, new, distinct in read_repls():
        try:
            result.append((re.compile(old), new, distinct))
        except re.error as e:
            print(f"Regex error for pattern {old}: {e}", file=sys.stderr)
    return result


def replace_all(patterns, s):
    r"""Apply replacements with the same per-output numbering as the Go harness.

    >>> distinct = [(re.compile(r"item-\w+"), "[ITEM]", True)]
    >>> replace_all(distinct, "item-b item-a item-b")
    '[ITEM][0] [ITEM][1] [ITEM][0]'
    >>> replace_all(distinct, "item-a item-a")
    '[ITEM] [ITEM]'
    >>> replace_all(distinct, "no matches")
    'no matches'
    >>> replace_all(distinct, "[ITEM][0] [ITEM][1] [ITEM][0]")
    '[ITEM][0] [ITEM][1] [ITEM][0]'
    >>> replace_all([(re.compile(r"item-\w+"), "[ITEM]", False)], "item-a item-b")
    '[ITEM] [ITEM]'
    """
    for comp, new, distinct in patterns:
        if distinct:
            seen = {}
            for match in comp.finditer(s):
                value = match.group(0)
                if value not in seen:
                    seen[value] = len(seen)
            if len(seen) > 1:
                s = comp.sub(lambda match, new=new, seen=seen: match.expand(new) + f"[{seen[match.group(0)]}]", s)
                continue
        s = comp.sub(new, s)
    return s

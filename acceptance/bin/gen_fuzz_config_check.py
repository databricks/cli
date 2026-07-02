#!/usr/bin/env python3
"""
Contract check for gen_fuzz_config.to_yaml: every scalar is on its own line as
`key: <json>`. edit_fuzz_config.py relies on this to edit a field by regex, not a YAML
parser. Prints each case's YAML (diffed by the harness) and exits non-zero on a violation.
"""

import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from edit_fuzz_config import FIELD_RE
from gen_fuzz_config import to_yaml

# Tricky shapes: strings with ':' and '"', nested maps, lists of dicts, empty containers.
CASES = [
    {"comment": "value: with a colon", "description": 'quote " and : colon'},
    {"resources": {"jobs": {"j": {"name": "n", "tags": {"team": "jobs"}}}}},
    {"tasks": [{"description": "d", "timeout_seconds": 3600}, {"comment": "c"}]},
    {"nums": [0, 1, 2], "flag": True, "ratio": 1.5, "empty_map": {}, "empty_list": []},
]

HEADER = re.compile(r"[\w.\-]+:$")  # non-empty container: `key:`
SCALAR = re.compile(r"[\w.\-]+: (.+)$")  # `key: <json>`


def check_line(line):
    rest = line.lstrip(" ")
    rest = rest.removeprefix("- ")
    if HEADER.fullmatch(rest):
        return  # container header; value is on following lines
    m = SCALAR.fullmatch(rest)
    if m:
        json.loads(m.group(1))  # value must be single-line JSON
        return
    json.loads(rest)  # bare list scalar: `- <json>`


def main():
    failed = False
    for case in CASES:
        text = to_yaml(case)
        sys.stdout.write(text)
        for line in text.splitlines():
            if not line.strip():
                continue
            try:
                check_line(line)
            except ValueError:
                sys.stderr.write(f"contract violation: not `key: <json>`: {line!r}\n")
                failed = True

    # edit_fuzz_config's FIELD_RE must still match when the value contains a colon (CASES[0]).
    if not any(FIELD_RE.match(line) for line in to_yaml(CASES[0]).splitlines()):
        sys.stderr.write("FIELD_RE did not match a comment/description line\n")
        failed = True

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
